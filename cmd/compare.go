package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/filter"
)

var flagCompareCount int

type compareStoreResult struct {
	Rank         int     `json:"rank"`
	Number       string  `json:"number"`
	Name         string  `json:"name"`
	City         string  `json:"city"`
	State        string  `json:"state"`
	Distance     string  `json:"distance"`
	MatchedDeals int     `json:"matchedDeals"`
	BogoDeals    int     `json:"bogoDeals"`
	Score        float64 `json:"score"`
	TopDeal      string  `json:"topDeal"`
}

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare nearby stores by filtered deal quality",
	Example: `  pubcli compare --zip 33101
  pubcli compare --zip 33101 --category produce --sort savings
  pubcli compare --zip 33101 --bogo --json`,
	RunE: runCompare,
}

func init() {
	rootCmd.AddCommand(compareCmd)

	registerDealFilterFlags(compareCmd.Flags())
	compareCmd.Flags().IntVar(&flagCompareCount, "count", 5, "Number of nearby stores to compare (1-10)")
}

func runCompare(cmd *cobra.Command, _ []string) error {
	if err := validateSortMode(); err != nil {
		return err
	}
	if flagZip == "" {
		return invalidArgsError(
			"--zip is required for compare",
			"pubcli compare --zip 33101",
			"pubcli compare --zip 33101 --category produce",
		)
	}
	if flagCompareCount < 1 || flagCompareCount > 10 {
		return invalidArgsError(
			"--count must be between 1 and 10",
			"pubcli compare --zip 33101 --count 5",
		)
	}

	client := api.NewClient()
	stores, err := client.FetchStores(cmd.Context(), flagZip, flagCompareCount)
	if err != nil {
		return upstreamError("fetching stores", err)
	}
	if len(stores) == 0 {
		return notFoundError(
			fmt.Sprintf("no stores found near %s", flagZip),
			"Try a nearby ZIP code.",
		)
	}

	results := make([]compareStoreResult, 0, len(stores))
	errCount := 0
	for _, store := range stores {
		storeNumber := api.StoreNumber(store.Key)
		resp, fetchErr := client.FetchSavings(cmd.Context(), storeNumber)
		if fetchErr != nil {
			errCount++
			continue
		}

		items := filter.Apply(resp.Savings, filter.Options{
			BOGO:       flagBogo,
			Category:   flagCategory,
			Department: flagDepartment,
			Query:      flagQuery,
			Sort:       flagSort,
			Limit:      flagLimit,
		})
		if len(items) == 0 {
			continue
		}

		bogoDeals := 0
		score := 0.0
		for _, item := range items {
			if filter.ContainsIgnoreCase(item.Categories, "bogo") {
				bogoDeals++
			}
			score += filter.DealScore(item)
		}

		results = append(results, compareStoreResult{
			Number:       storeNumber,
			Name:         store.Name,
			City:         store.City,
			State:        store.State,
			Distance:     strings.TrimSpace(store.Distance),
			MatchedDeals: len(items),
			BogoDeals:    bogoDeals,
			Score:        score,
			TopDeal:      topDealTitle(items[0]),
		})
	}

	if len(results) == 0 {
		if errCount == len(stores) {
			return upstreamError("fetching deals", fmt.Errorf("all %d store lookups failed", len(stores)))
		}
		return notFoundError(
			"no stores have deals matching your filters",
			"Relax filters like --category/--department/--query.",
		)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].MatchedDeals != results[j].MatchedDeals {
			return results[i].MatchedDeals > results[j].MatchedDeals
		}
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return parseDistance(results[i].Distance) < parseDistance(results[j].Distance)
	})
	for i := range results {
		results[i].Rank = i + 1
	}

	if flagJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(results)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nStore comparison near %s (%d matching store(s))\n\n", flagZip, len(results))
	for _, r := range results {
		fmt.Fprintf(
			cmd.OutOrStdout(),
			"%d. #%s %s (%s, %s)\n   matches: %d | bogo: %d | score: %.1f | distance: %s mi\n   top: %s\n\n",
			r.Rank,
			r.Number,
			r.Name,
			r.City,
			r.State,
			r.MatchedDeals,
			r.BogoDeals,
			r.Score,
			emptyIf(r.Distance, "?"),
			r.TopDeal,
		)
	}
	if errCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "note: skipped %d store(s) due to upstream fetch errors.\n", errCount)
	}
	return nil
}

func topDealTitle(item api.SavingItem) string {
	if title := filter.CleanText(filter.Deref(item.Title)); title != "" {
		return title
	}
	if desc := filter.CleanText(filter.Deref(item.Description)); desc != "" {
		return desc
	}
	if item.ID != "" {
		return "Deal " + item.ID
	}
	return "Untitled deal"
}

func parseDistance(raw string) float64 {
	for _, token := range strings.Fields(raw) {
		clean := strings.Trim(token, ",")
		if d, err := strconv.ParseFloat(clean, 64); err == nil {
			return d
		}
	}
	return 999999
}

func emptyIf(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

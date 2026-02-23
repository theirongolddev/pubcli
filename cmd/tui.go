package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/display"
	"github.com/tayloree/publix-deals/internal/filter"
	"golang.org/x/term"
)

const tuiPageSize = 10

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Browse deals interactively in the terminal",
	Example: `  pubcli tui --zip 33101
  pubcli tui --store 1425 --category produce --sort ending`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	registerDealFilterFlags(tuiCmd.Flags())
}

func runTUI(cmd *cobra.Command, _ []string) error {
	if err := validateSortMode(); err != nil {
		return err
	}
	if !flagJSON && !isInteractiveSession(cmd.InOrStdin(), cmd.OutOrStdout()) {
		return invalidArgsError(
			"`pubcli tui` requires an interactive terminal",
			"Use `pubcli --zip 33101 --json` in pipelines.",
		)
	}

	client := api.NewClient()
	storeNumber, err := resolveStore(cmd, client)
	if err != nil {
		return err
	}

	resp, err := client.FetchSavings(cmd.Context(), storeNumber)
	if err != nil {
		return upstreamError("fetching deals", err)
	}
	if len(resp.Savings) == 0 {
		return notFoundError(
			fmt.Sprintf("no deals found for store #%s", storeNumber),
			"Try another store with --store.",
		)
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
		return notFoundError(
			"no deals match your filters",
			"Relax filters like --category/--department/--query.",
		)
	}

	if flagJSON {
		return display.PrintDealsJSON(cmd.OutOrStdout(), items)
	}

	return runTUILoop(cmd.OutOrStdout(), cmd.InOrStdin(), storeNumber, items)
}

func isInteractiveSession(stdin io.Reader, stdout io.Writer) bool {
	inputFile, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	if !term.IsTerminal(int(inputFile.Fd())) {
		return false
	}
	return isTTY(stdout)
}

func runTUILoop(out io.Writer, in io.Reader, storeNumber string, items []api.SavingItem) error {
	reader := bufio.NewReader(in)
	page := 0
	totalPages := (len(items)-1)/tuiPageSize + 1

	for {
		renderTUIPage(out, storeNumber, items, page, totalPages)

		line, err := reader.ReadString('\n')
		if err != nil {
			return nil
		}
		cmd := strings.TrimSpace(strings.ToLower(line))

		switch cmd {
		case "q", "quit", "exit":
			return nil
		case "n", "next":
			if page < totalPages-1 {
				page++
			}
		case "p", "prev", "previous":
			if page > 0 {
				page--
			}
		default:
			idx, convErr := strconv.Atoi(cmd)
			if convErr != nil || idx < 1 || idx > len(items) {
				continue
			}
			renderDealDetail(out, items[idx-1], idx, len(items))
			if _, err := reader.ReadString('\n'); err != nil {
				return nil
			}
		}
	}
}

func renderTUIPage(out io.Writer, storeNumber string, items []api.SavingItem, page, totalPages int) {
	fmt.Fprint(out, "\033[H\033[2J")
	fmt.Fprintf(out, "pubcli tui | store #%s | %d deals | page %d/%d\n\n", storeNumber, len(items), page+1, totalPages)

	start := page * tuiPageSize
	end := minInt(start+tuiPageSize, len(items))
	for i := start; i < end; i++ {
		item := items[i]
		title := topDealTitle(item)
		savings := filter.CleanText(filter.Deref(item.Savings))
		if savings == "" {
			savings = "-"
		}
		fmt.Fprintf(out, "%2d. %s [%s]\n", i+1, title, savings)
	}
	fmt.Fprintf(out, "\ncommands: number=details | n=next | p=prev | q=quit\n> ")
}

func renderDealDetail(out io.Writer, item api.SavingItem, index, total int) {
	fmt.Fprint(out, "\033[H\033[2J")
	fmt.Fprintf(out, "deal %d/%d\n\n", index, total)
	display.PrintDeals(out, []api.SavingItem{item})
	fmt.Fprint(out, "press Enter to return\n")
}

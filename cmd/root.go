package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/display"
	"github.com/tayloree/publix-deals/internal/filter"
)

var (
	flagStore      string
	flagZip        string
	flagCategory   string
	flagDepartment string
	flagBogo       bool
	flagQuery      string
	flagSort       string
	flagLimit      int
	flagJSON       bool
)

var rootCmd = &cobra.Command{
	Use:   "pubcli",
	Short: "Fetch current Publix weekly ad deals",
	Long: "CLI tool that fetches the current week's sale items from the Publix API.\n" +
		"Requires a store number or zip code to find deals for your local store.\n\n" +
		"Agent-friendly mode: minor syntax issues are auto-corrected when intent is clear " +
		"(for example: -zip 33101, zip=33101, --ziip 33101).",
	Example: `  pubcli --zip 33101
  pubcli --store 1425 --bogo
  pubcli --zip 33101 --sort savings
  pubcli categories --zip 33101
  pubcli stores --zip 33101 --json
  pubcli compare --zip 33101 --category produce`,
	RunE: runDeals,
}

func init() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&flagStore, "store", "s", "", "Publix store number (e.g., 1425)")
	pf.StringVarP(&flagZip, "zip", "z", "", "Zip code to find nearby stores")
	pf.BoolVar(&flagJSON, "json", false, "Output as JSON")

	registerDealFilterFlags(rootCmd.Flags())
}

// Execute runs the root command.
func Execute() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	resetCLIState()

	normalizedArgs, notes := normalizeCLIArgs(args)
	for _, note := range notes {
		fmt.Fprintf(stderr, "note: %s\n", note)
	}

	if len(normalizedArgs) == 0 {
		if err := printQuickStart(stdout, !isTTY(stdout)); err != nil {
			cliErr := classifyCLIError(err)
			fmt.Fprintln(stderr, formatCLIErrorText(cliErr))
			return cliErr.ExitCode
		}
		return ExitSuccess
	}

	if shouldAutoJSON(normalizedArgs, isTTY(stdout)) {
		normalizedArgs = append(normalizedArgs, "--json")
	}

	setCommandIO(rootCmd, stdout, stderr)
	rootCmd.SetArgs(normalizedArgs)

	if err := rootCmd.Execute(); err != nil {
		cliErr := classifyCLIError(err)
		if hasJSONPreference(normalizedArgs) {
			if jerr := printCLIErrorJSON(stderr, cliErr); jerr != nil {
				fmt.Fprintln(stderr, formatCLIErrorText(classifyCLIError(jerr)))
				return ExitInternal
			}
		} else {
			fmt.Fprintln(stderr, formatCLIErrorText(cliErr))
		}
		return cliErr.ExitCode
	}
	return ExitSuccess
}

func setCommandIO(cmd *cobra.Command, stdout, stderr io.Writer) {
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	for _, child := range cmd.Commands() {
		setCommandIO(child, stdout, stderr)
	}
}

func resetCLIState() {
	flagStore = ""
	flagZip = ""
	flagCategory = ""
	flagDepartment = ""
	flagBogo = false
	flagQuery = ""
	flagSort = ""
	flagLimit = 0
	flagCompareCount = 5
	flagJSON = false
}

func registerDealFilterFlags(f *pflag.FlagSet) {
	f.StringVarP(&flagCategory, "category", "c", "", "Filter by category (e.g., bogo, meat, produce)")
	f.StringVarP(&flagDepartment, "department", "d", "", "Filter by department (e.g., Meat, Deli)")
	f.BoolVar(&flagBogo, "bogo", false, "Show only BOGO deals")
	f.StringVarP(&flagQuery, "query", "q", "", "Search deals by keyword in title/description")
	f.StringVar(&flagSort, "sort", "", "Sort deals by relevance, savings, or ending")
	f.IntVarP(&flagLimit, "limit", "n", 0, "Limit number of results (0 = all)")
}

func validateSortMode() error {
	switch strings.ToLower(strings.TrimSpace(flagSort)) {
	case "", "relevance", "savings", "ending", "end", "expiry", "expiration":
		return nil
	default:
		return invalidArgsError(
			"invalid value for --sort (use relevance, savings, or ending)",
			"pubcli --zip 33101 --sort savings",
			"pubcli --zip 33101 --sort ending",
		)
	}
}

func resolveStore(cmd *cobra.Command, client *api.Client) (string, error) {
	if flagStore != "" {
		return flagStore, nil
	}
	if flagZip == "" {
		return "", invalidArgsError(
			"please provide --store NUMBER or --zip ZIPCODE",
			"pubcli --zip 33101",
			"pubcli --store 1425",
		)
	}

	stores, err := client.FetchStores(cmd.Context(), flagZip, 1)
	if err != nil {
		return "", upstreamError("finding stores", err)
	}
	if len(stores) == 0 {
		return "", notFoundError(
			fmt.Sprintf("no Publix stores found near %s", flagZip),
			"Try a nearby ZIP code.",
		)
	}

	num := api.StoreNumber(stores[0].Key)
	if !flagJSON {
		display.PrintStoreContext(cmd.OutOrStdout(), stores[0])
	}
	return num, nil
}

func runDeals(cmd *cobra.Command, _ []string) error {
	if err := validateSortMode(); err != nil {
		return err
	}

	client := api.NewClient()

	storeNumber, err := resolveStore(cmd, client)
	if err != nil {
		return err
	}

	data, err := client.FetchSavings(cmd.Context(), storeNumber)
	if err != nil {
		return upstreamError("fetching deals", err)
	}

	items := data.Savings
	if len(items) == 0 {
		return notFoundError(
			fmt.Sprintf("no deals found for store #%s", storeNumber),
			"Try another store with --store.",
		)
	}

	items = filter.Apply(items, filter.Options{
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
	display.PrintDeals(cmd.OutOrStdout(), items)
	return nil
}

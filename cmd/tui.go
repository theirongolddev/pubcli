package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/display"
	"github.com/tayloree/publix-deals/internal/filter"
	"golang.org/x/term"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Browse deals in a full-screen interactive terminal UI",
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

	initialOpts := filter.Options{
		BOGO:       flagBogo,
		Category:   flagCategory,
		Department: flagDepartment,
		Query:      flagQuery,
		Sort:       flagSort,
		Limit:      flagLimit,
	}

	if flagJSON {
		_, _, rawItems, err := loadTUIData(cmd.Context(), flagStore, flagZip)
		if err != nil {
			return err
		}
		items := filter.Apply(rawItems, initialOpts)
		if len(items) == 0 {
			return notFoundError(
				"no deals match your filters",
				"Relax filters like --category/--department/--query.",
			)
		}
		return display.PrintDealsJSON(cmd.OutOrStdout(), items)
	}

	if !isInteractiveSession(cmd.InOrStdin(), cmd.OutOrStdout()) {
		return invalidArgsError(
			"`pubcli tui` requires an interactive terminal",
			"Use `pubcli --zip 33101 --json` in pipelines.",
		)
	}

	model := newLoadingDealsTUIModel(tuiLoadConfig{
		ctx:         cmd.Context(),
		storeNumber: flagStore,
		zipCode:     flagZip,
		initialOpts: initialOpts,
	})

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithInput(cmd.InOrStdin()),
		tea.WithOutput(cmd.OutOrStdout()),
	)

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("running tui: %w", err)
	}
	if finalState, ok := finalModel.(dealsTUIModel); ok && finalState.fatalErr != nil {
		return finalState.fatalErr
	}
	return nil
}

func resolveStoreForTUI(ctx context.Context, client *api.Client, storeNumber, zipCode string) (resolvedStoreNumber, storeLabel string, err error) {
	if storeNumber != "" {
		return storeNumber, "#" + storeNumber, nil
	}
	if zipCode == "" {
		return "", "", invalidArgsError(
			"please provide --store NUMBER or --zip ZIPCODE",
			"pubcli tui --zip 33101",
			"pubcli tui --store 1425",
		)
	}

	stores, err := client.FetchStores(ctx, zipCode, 1)
	if err != nil {
		return "", "", upstreamError("finding stores", err)
	}
	if len(stores) == 0 {
		return "", "", notFoundError(
			fmt.Sprintf("no Publix stores found near %s", zipCode),
			"Try a nearby ZIP code.",
		)
	}

	store := stores[0]
	resolvedStoreNumber = api.StoreNumber(store.Key)
	storeLabel = fmt.Sprintf("#%s — %s (%s, %s)", resolvedStoreNumber, store.Name, store.City, store.State)
	return resolvedStoreNumber, storeLabel, nil
}

func loadTUIData(ctx context.Context, storeNumber, zipCode string) (resolvedStoreNumber, storeLabel string, items []api.SavingItem, err error) {
	client := api.NewClient()

	resolvedStoreNumber, storeLabel, err = resolveStoreForTUI(ctx, client, storeNumber, zipCode)
	if err != nil {
		return "", "", nil, err
	}

	resp, err := client.FetchSavings(ctx, resolvedStoreNumber)
	if err != nil {
		return "", "", nil, upstreamError("fetching deals", err)
	}
	if len(resp.Savings) == 0 {
		return "", "", nil, notFoundError(
			fmt.Sprintf("no deals found for store #%s", resolvedStoreNumber),
			"Try another store with --store.",
		)
	}

	return resolvedStoreNumber, storeLabel, resp.Savings, nil
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

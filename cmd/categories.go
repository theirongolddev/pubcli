package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/display"
	"github.com/tayloree/publix-deals/internal/filter"
)

var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List available categories for the current week",
	Example: `  pubcli categories --store 1425
  pubcli categories -z 33101 --json`,
	RunE: runCategories,
}

func init() {
	rootCmd.AddCommand(categoriesCmd)
}

func runCategories(cmd *cobra.Command, _ []string) error {
	client := api.NewClient()

	storeNumber, err := resolveStore(cmd, client)
	if err != nil {
		return err
	}

	data, err := client.FetchSavings(cmd.Context(), storeNumber)
	if err != nil {
		return upstreamError("fetching deals", err)
	}

	if len(data.Savings) == 0 {
		return notFoundError(
			fmt.Sprintf("no deals found for store #%s", storeNumber),
			"Try another store with --store.",
		)
	}

	cats := filter.Categories(data.Savings)

	if flagJSON {
		return display.PrintCategoriesJSON(cmd.OutOrStdout(), cats)
	}
	display.PrintCategories(cmd.OutOrStdout(), cats, storeNumber)
	return nil
}

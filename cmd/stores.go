package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/display"
)

var storesCmd = &cobra.Command{
	Use:   "stores",
	Short: "List nearby Publix stores",
	Long:  "Find Publix stores near a zip code. Use this to discover store numbers for fetching deals.",
	Example: `  pubcli stores --zip 33101
  pubcli stores -z 32801 --json`,
	RunE: runStores,
}

func init() {
	rootCmd.AddCommand(storesCmd)
}

func runStores(cmd *cobra.Command, _ []string) error {
	if flagZip == "" {
		return invalidArgsError(
			"--zip is required for store lookup",
			"pubcli stores --zip 33101",
			"pubcli stores -z 33101 --json",
		)
	}

	client := api.NewClient()
	stores, err := client.FetchStores(cmd.Context(), flagZip, 5)
	if err != nil {
		return upstreamError("fetching stores", err)
	}
	if len(stores) == 0 {
		return notFoundError(
			fmt.Sprintf("no stores found near %s", flagZip),
			"Try a nearby ZIP code.",
		)
	}

	if flagJSON {
		return display.PrintStoresJSON(cmd.OutOrStdout(), stores)
	}
	display.PrintStores(cmd.OutOrStdout(), stores, flagZip)
	return nil
}

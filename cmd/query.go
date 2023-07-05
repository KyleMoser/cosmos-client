package cmd

import (
	"github.com/spf13/cobra"
)

// queryCmd represents the query command tree.
func queryCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "query things about a chain",
	}

	cmd.AddCommand(bankQueryCmd(a))
	return cmd
}

// bankQueryCmd  returns the transaction commands for this module
func bankQueryCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bank",
		Aliases: []string{"b"},
		Short:   "Querying commands for the bank module",
	}

	cmd.AddCommand(
		bankBalanceCmd(a),
		bankTotalSupplyCmd(a),
		bankDenomsMetadataCmd(a),
	)

	return cmd
}

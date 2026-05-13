package cli

import "github.com/spf13/cobra"

func landCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "land",
		Short: "Land the bottom-most PR in the stack.",
		Long:  `Squash-merges the bottom PR and rebases the rest of the stack.`,
	}
}

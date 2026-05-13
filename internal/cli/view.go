package cli

import "github.com/spf13/cobra"

func viewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Safely inspect the current stack.",
		Long:  `Does not modify commits or push branches. May fetch/prune the remote.`,
	}
}

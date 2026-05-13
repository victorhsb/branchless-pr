package cli

import "github.com/spf13/cobra"

func abandonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abandon",
		Short: "Remove stack metadata and delete generated branches.",
		Long:  `Deletes local and remote generated branches and strips stack metadata from commits.`,
	}
}

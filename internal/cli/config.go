package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/config"
)

func configCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "config <section>.<key>=<value>",
		Short: "Create or update a config setting.",
		Long:  `Expects exactly one argument in the form <section>.<key>=<value>.`,
		Args:  cobra.ExactArgs(1),
		// config does not need git checks or gh username.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			section, key, value, err := config.ParseConfigArg(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				os.Exit(1)
			}
			// Reload from disk to avoid overwriting concurrent changes.
			cfgPath, err := config.FilePath()
			if err != nil {
				return err
			}
			current, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			current.Set(section, key, value)
			if err := current.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("%s.%s = %s\n", section, key, value)
			return nil
		},
	}
}

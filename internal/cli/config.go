package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/config"
)

func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config [init | set <section>.<key>=<value> | <section>.<key>=<value>]",
		Short: "Manage or update stack-pr configuration.",
		Long: `Without a subcommand keyword, the argument is treated as a set operation
(<section>.<key>=<value>), preserving backward compatibility.`,
		// Config command does not need git checks or gh username.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Usage()
			}
			if args[0] == "init" {
				return runConfigInit(cmd)
			}
			if args[0] == "set" {
				if len(args) != 2 {
					return fmt.Errorf("usage: config set <section>.<key>=<value>")
				}
				return runConfigSet(cmd, args[1])
			}
			// Legacy inline syntax: config <section>.<key>=<value>
			if len(args) == 1 {
				return runConfigSet(cmd, args[0])
			}
			return cmd.Usage()
		},
	}
}

func runConfigInit(cmd *cobra.Command) error {
	path, err := config.FilePath()
	if err != nil {
		return err
	}
	if err := config.WriteDefaults(path); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
	return nil
}

func runConfigSet(cmd *cobra.Command, arg string) error {
	section, key, value, err := config.ParseConfigArg(arg)
	if err != nil {
		return err
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
	fmt.Fprintf(cmd.OutOrStdout(), "%s.%s = %s\n", section, key, value)
	return nil
}

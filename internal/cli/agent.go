package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	prompt "github.com/victorhsb/branchless-pr/internal/agent"
	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/diagnose"
)

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Emit artifacts for LLM agents.",
		Long:  "Commands under agent emit static, side-effect-free artifacts for LLM agents and do not require a git repository or gh authentication.",
	}
	cmd.AddCommand(agentPromptCmd())
	cmd.AddCommand(agentDiagnoseCmd())
	return cmd
}

func agentDiagnoseCmd() *cobra.Command {
	var (
		format string
		online bool
	)

	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Read-only, best-effort diagnostic report for agents.",
		Long:  "Inspect repository and stack state without mutating Git or GitHub. Reportable outcomes always exit 0; severity is surfaced in the report payload.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("unknown agent diagnose format %q: expected \"text\" or \"json\"", format)
			}

			cfg := config.Defaults()
			if p, err := config.FilePath(); err == nil {
				if loaded, err := config.Load(p); err == nil {
					loaded.Merge(cfg)
					cfg = loaded
				}
			}
			ca := ResolveSharedArgs(cfg, flagBase, flagHead, flagRemote, flagTarget, nil, nil, flagBranchTemplate, nil)
			report := diagnose.Run(diagnose.Options{
				Remote:             ca.Remote,
				Target:             ca.Target,
				Base:               ca.Base,
				Head:               ca.Head,
				BranchNameTemplate: ca.BranchNameTemplate,
				Online:             online,
			})

			switch format {
			case "text":
				_, err := fmt.Fprint(cmd.OutOrStdout(), diagnose.RenderText(report))
				return err
			case "json":
				out, err := diagnose.RenderJSON(report)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", `Output format: "text" or "json"`)
	cmd.Flags().BoolVar(&online, "online", false, "Allow optional GitHub network checks for availability and live PR state")
	return cmd
}

func agentPromptCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "prompt [topic]",
		Short: "Emit static guidance for LLM agents using stack-pr.",
		Long:  fmt.Sprintf("Emit deterministic prompt guidance for LLM agents. Supported topics: %s.", prompt.AllowedTopicsString()),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := prompt.TopicAll
			if len(args) > 0 {
				topic = args[0]
			}
			if err := prompt.ValidateTopic(topic); err != nil {
				return err
			}

			switch format {
			case "text":
				out, err := prompt.RenderText(topic)
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), out)
				return err
			case "json":
				out, err := prompt.RenderJSON(topic)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return err
			default:
				return fmt.Errorf("unknown agent prompt format %q: expected \"text\" or \"json\"", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", `Output format: "text" or "json"`)
	return cmd
}

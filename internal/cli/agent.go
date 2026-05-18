package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	prompt "github.com/victorhsb/branchless-pr/internal/agent"
)

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Emit artifacts for LLM agents.",
		Long:  "Commands under agent emit static, side-effect-free artifacts for LLM agents and do not require a git repository or gh authentication.",
	}
	cmd.AddCommand(agentPromptCmd())
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

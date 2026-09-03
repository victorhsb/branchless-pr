package cli

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/invocation"
	"github.com/victorhsb/branchless-pr/internal/shell"
)

var (
	flagRemote         string
	flagBase           string
	flagHead           string
	flagTarget         string
	flagHyperlinks     bool
	flagNoHyperlinks   bool
	flagVerbose        bool
	flagBranchTemplate string
	flagShowTips       bool
	flagNoShowTips     bool
	flagStash          bool
)

type appContextKey struct{}

var ctxKey appContextKey

// Execute is the entrypoint called from main.go.  progName is the command
// name shown in CLI help, error messages, and completions (e.g. "stack-pr"
// or "bpr").
func Execute(progName string) error {
	args := []string{"--help"}
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}
	root, err := newRootCommand(progName, args, shell.Default{})
	if err != nil {
		return err
	}
	return root.Execute()
}

func newRootCommand(progName string, args []string, runner shell.Runner) (*cobra.Command, error) {
	cobra.EnableCommandSorting = false

	bootstrap, err := invocation.NewBootstrap(runner, argsSelectAgent(args))
	if err != nil {
		return nil, err
	}
	cfg := bootstrap.Config

	root := &cobra.Command{
		Use:           progName,
		Short:         "Create, update, view, abandon, and land stacked GitHub pull requests.",
		Version:       Version(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			policy := invocation.PolicyFor(cmd.Name(), commandInSubtree(cmd, "agent"), commandInSubtree(cmd, "config"))
			if policy.AgentOnly {
				app, err := bootstrap.Start(invocation.BootstrapOptions{Policy: policy})
				if err != nil {
					return err
				}
				cmd.SetContext(newContextFromApp(app))
				return nil
			}

			// Resolve shared args
			var hyperlinks *bool
			if cmd.Flags().Changed("hyperlinks") {
				hyperlinks = &flagHyperlinks
			} else if cmd.Flags().Changed("no-hyperlinks") {
				t := false
				hyperlinks = &t
			}
			var verbose *bool
			if cmd.Flags().Changed("verbose") {
				verbose = &flagVerbose
			}
			var showTips *bool
			if cmd.Flags().Changed("show-tips") {
				showTips = &flagShowTips
			} else if cmd.Flags().Changed("no-show-tips") {
				t := false
				showTips = &t
			}
			headExplicit := cmd.Flags().Changed("head")

			ca := ResolveSharedArgs(cfg, flagBase, flagHead, flagRemote, flagTarget, hyperlinks, verbose, flagBranchTemplate, showTips)
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			appCtx, err := bootstrap.Start(invocation.BootstrapOptions{
				Args:         ca,
				Policy:       policy,
				HeadExplicit: headExplicit,
				Stash:        flagStash,
				DryRun:       dryRun,
				Stderr:       cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}

			cmd.SetContext(newContextFromApp(appCtx))
			return nil
		},
	}

	// --- persistent flags ---
	root.PersistentFlags().StringVarP(&flagRemote, "remote", "R", "", "Remote name; default from config repo.remote or origin")
	root.PersistentFlags().StringVarP(&flagBase, "base", "B", "", "Local base revision; default deduced via git merge-base")
	root.PersistentFlags().StringVarP(&flagHead, "head", "H", "", "Local head revision; default HEAD")
	root.PersistentFlags().StringVarP(&flagTarget, "target", "T", "", "Remote target branch; default from config repo.target or main")
	root.PersistentFlags().BoolVar(&flagHyperlinks, "hyperlinks", true, "Enable terminal hyperlinks")
	root.PersistentFlags().BoolVar(&flagNoHyperlinks, "no-hyperlinks", false, "Disable terminal hyperlinks")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "V", false, "Show verbose Git/GH subprocess output")
	root.PersistentFlags().StringVarP(&flagBranchTemplate, "branch-name-template", "b", "", "Generated branch template; default $USERNAME/stack")
	root.PersistentFlags().BoolVar(&flagShowTips, "show-tips", true, "Show post-command guidance")
	root.PersistentFlags().BoolVar(&flagNoShowTips, "no-show-tips", false, "Suppress post-command guidance")

	// Add subcommands
	root.AddCommand(submitCmd()) // submit has alias "export"
	root.AddCommand(viewCmd())
	root.AddCommand(fixCmd())
	root.AddCommand(commentsCmd())
	root.AddCommand(checksCmd())

	// Land is only registered when land.style != disable (SPEC §6.2)
	landStyle := cfg.Get("land", "style")
	if landStyle != "disable" {
		root.AddCommand(landCmd())
	}

	root.AddCommand(abandonCmd())
	root.AddCommand(configCmd(bootstrap.RepoRoot()))
	root.AddCommand(agentCmd())

	root.SetArgs(args)
	return root, nil
}

func argsSelectAgent(args []string) bool {
	valueFlags := map[string]struct{}{
		"-R": {}, "--remote": {},
		"-B": {}, "--base": {},
		"-H": {}, "--head": {},
		"-T": {}, "--target": {},
		"-b": {}, "--branch-name-template": {},
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return false
		}
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "=") {
				continue
			}
			if _, takesValue := valueFlags[arg]; takesValue {
				i++
			}
			continue
		}
		return arg == "agent"
	}
	return false
}

func commandInSubtree(cmd *cobra.Command, name string) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == name {
			return true
		}
	}
	return false
}

func newContextFromApp(app *AppContext) context.Context {
	return context.WithValue(context.Background(), ctxKey, app)
}

// FromContext extracts the AppContext from a Go context.
func FromContext(ctx context.Context) (*AppContext, bool) {
	v, ok := ctx.Value(ctxKey).(*AppContext)
	return v, ok
}

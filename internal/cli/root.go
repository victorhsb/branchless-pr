package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/stack"
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

var ctxKey struct{}

// Execute is the entrypoint called from main.go.
func Execute() error {
	cobra.EnableCommandSorting = false

	// Pre-resolve config so subcommands can be conditionally added.
	cfgPath, err := config.FilePath()
	if err != nil {
		return fmt.Errorf("unable to locate repo root: %w", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}
	defaults := config.Defaults()
	cfg.Merge(defaults)

	root := &cobra.Command{
		Use:     "stack-pr",
		Short:   "Create, update, view, abandon, and land stacked GitHub pull requests.",
		Version: Version(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Merge defaults fresh so multiple invocations in tests work.
			cfg.Merge(defaults)

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

			// Validate branch template
			tmpl := stack.ParseTemplate(ca.BranchNameTemplate)
			if !tmpl.HasID {
				return fmt.Errorf("branch name template must contain $ID (or be one that appends /$ID): got %q", ca.BranchNameTemplate)
			}

			// Verbosity
			if ca.Verbose {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
			}

			// Check gh
			if err := git.CheckGHInstalled(); err != nil {
				return err
			}

			// Repo root
			repoRoot, err := git.RepoRoot()
			if err != nil {
				return err
			}

			// Current branch
			origBranch, err := git.CurrentBranchName()
			if err != nil {
				return err
			}

			// When running in a git-branchless stack, HEAD may point at a middle
			// commit while higher commits are descendants. The original stack-pr
			// discovers BASE..HEAD, so use the branchless stack top as the default
			// head unless the user explicitly supplied --head.
			if !headExplicit {
				if branchlessHead, ok := git.BranchlessStackHead(repoRoot); ok {
					ca.Head = branchlessHead
				}
			}

			// Username
			username, err := git.GetGHUsername()
			if err != nil {
				return err
			}

			appCtx := &AppContext{
				Config:     cfg,
				Args:       ca,
				RepoRoot:   repoRoot,
				Username:   username,
				OrigBranch: origBranch,
			}

			// --- Steps below are skipped for config command ---
			if cmd.Name() == "config" {
				cmd.SetContext(newContextFromApp(appCtx))
				return nil
			}

			// Stash (submit/export only, before clean check)
			if (cmd.Name() == "submit" || cmd.Name() == "export") && flagStash {
				stashed, err := git.StashSave("stack-pr auto-stash")
				if err != nil {
					return fmt.Errorf("failed to stash changes: %w", err)
				}
				appCtx.StashCreated = stashed
			}

			// Require clean repo (all except view/config)
			if cmd.Name() != "view" && cmd.Name() != "config" {
				if err := RequireCleanRepo(); err != nil {
					return err
				}
			}

			// Check that REMOTE/TARGET exists
			if err := git.TargetExists(ca.Remote, ca.Target); err != nil {
				if ca.Target == "main" {
					if e := git.TargetExists(ca.Remote, "master"); e == nil {
						fmt.Fprintln(os.Stderr, "Hint: target branch 'main' not found, but 'master' exists on remote. Use --target master if applicable.")
					}
				}
				return err
			}

			// Deduce base if missing
			if ca.Base == "" {
				mb, err := git.MergeBase(ca.Head, ca.Remote+"/"+ca.Target)
				if err != nil {
					return fmt.Errorf("unable to deduce base merge-base: %w", err)
				}
				appCtx.Args.Base = mb
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
	root.PersistentFlags().StringVar(&flagBranchTemplate, "branch-name-template", "", "Generated branch template; default $USERNAME/stack")
	root.PersistentFlags().BoolVar(&flagShowTips, "show-tips", true, "Show post-command guidance")
	root.PersistentFlags().BoolVar(&flagNoShowTips, "no-show-tips", false, "Suppress post-command guidance")

	// Add subcommands
	root.AddCommand(submitCmd()) // submit has alias "export"
	root.AddCommand(viewCmd())

	// Land is only registered when land.style != disable (SPEC §6.2)
	landStyle := cfg.Get("land", "style")
	if landStyle != "disable" {
		root.AddCommand(landCmd())
	}

	root.AddCommand(abandonCmd())
	root.AddCommand(configCmd())

	// No command: show help and exit
	root.SetArgs([]string{"--help"})
	if len(os.Args) > 1 {
		root.SetArgs(os.Args[1:])
	}
	return root.Execute()
}

func newContextFromApp(app *AppContext) context.Context {
	return context.WithValue(context.Background(), ctxKey, app)
}

// FromContext extracts the AppContext from a Go context.
func FromContext(ctx context.Context) (*AppContext, bool) {
	v, ok := ctx.Value(ctxKey).(*AppContext)
	return v, ok
}

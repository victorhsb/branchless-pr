package invocation

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/shell"
)

// Bootstrap owns the dependencies and configuration shared by one CLI
// construction. Repository discovery and config loading happen once, before
// Cobra commands are registered; Start resolves the command-specific state
// after flags have been parsed.
type Bootstrap struct {
	repo     *git.Repo
	client   *pr.Client
	Config   *config.Config
	repoRoot string
}

// RepoRoot returns the repository root discovered while loading config.
func (b *Bootstrap) RepoRoot() string {
	return b.repoRoot
}

// BootstrapOptions controls command-specific invocation initialization.
type BootstrapOptions struct {
	Args         CommonArgs
	Policy       CommandPolicy
	HeadExplicit bool
	Stash        bool
	DryRun       bool
	Stderr       io.Writer
}

// NewBootstrap prepares config and repository discovery for a root command.
// Agent commands deliberately skip both so they remain usable outside a repo.
func NewBootstrap(run shell.Runner, agentOnly bool) (*Bootstrap, error) {
	if run == nil {
		run = shell.Default{}
	}
	b := &Bootstrap{
		repo:   git.New("", run),
		client: pr.NewClient(run),
		Config: config.Defaults(),
	}
	if agentOnly {
		return b, nil
	}

	if os.Getenv("STACKPR_CONFIG") == "" {
		root, err := b.repo.RepoRoot()
		if err != nil {
			return nil, fmt.Errorf("unable to locate repo root: %w", err)
		}
		b.repoRoot = root
		b.repo.Dir = root
	}
	cfgPath, err := config.FilePath(b.repoRoot)
	if err != nil {
		return nil, fmt.Errorf("unable to locate config: %w", err)
	}
	loaded, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("unable to load config: %w", err)
	}
	loaded.Merge(b.Config)
	b.Config = loaded
	return b, nil
}

// Start resolves runtime state for one command invocation.
func (b *Bootstrap) Start(opts BootstrapOptions) (_ *AppContext, err error) {
	if opts.Policy.AgentOnly {
		return &AppContext{
			Config: b.Config,
			Git:    b.repo,
			PR:     b.client,
		}, nil
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	if err := git.ValidateRemoteName(opts.Args.Remote); err != nil {
		return nil, err
	}
	if err := git.ValidateRefName("target branch", opts.Args.Target); err != nil {
		return nil, err
	}
	if opts.Args.Verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(opts.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	if err := b.client.CheckGHInstalled(); err != nil {
		return nil, err
	}
	repoRoot := b.repoRoot
	if repoRoot == "" {
		repoRoot, err = b.repo.RepoRoot()
		if err != nil {
			return nil, err
		}
		b.repoRoot = repoRoot
		b.repo.Dir = repoRoot
	}
	origBranch, err := b.repo.CurrentBranchName()
	if err != nil {
		return nil, err
	}
	if !opts.HeadExplicit {
		if branchlessHead, ok := b.repo.BranchlessStackHead(); ok {
			opts.Args.Head = branchlessHead
		}
	}
	username, err := b.client.GetGHUsername()
	if err != nil {
		return nil, err
	}

	app := &AppContext{
		Config:     b.Config,
		Args:       opts.Args,
		Git:        b.repo,
		PR:         b.client,
		RepoRoot:   repoRoot,
		Username:   username,
		OrigBranch: origBranch,
	}
	if opts.Policy.ConfigOnly {
		return app, nil
	}

	if opts.Policy.UsesStash && opts.Stash && !opts.DryRun {
		stash, err := b.repo.StashSave("stack-pr auto-stash")
		if err != nil {
			return nil, fmt.Errorf("failed to stash changes: %w", err)
		}
		app.AutomaticStash = stash
	}
	if !app.AutomaticStash.IsZero() {
		defer func() {
			if err == nil {
				return
			}
			if restoreErr := app.RestoreStash(); restoreErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to restore automatic stash after initialization error: %w", restoreErr))
			}
		}()
	}
	if !opts.Policy.AllowsDirty {
		if err := RequireCleanRepo(b.repo); err != nil {
			return nil, err
		}
	}
	if opts.Policy.RequiresTarget {
		if err := b.repo.TargetExists(opts.Args.Remote, opts.Args.Target); err != nil {
			if opts.Args.Target == "main" {
				if masterErr := b.repo.TargetExists(opts.Args.Remote, "master"); masterErr == nil {
					fmt.Fprintln(opts.Stderr, "Hint: target branch 'main' not found, but 'master' exists on remote. Use --target master if applicable.")
				}
			}
			return nil, err
		}
	}
	if opts.Policy.RequiresTarget && opts.Args.Base == "" {
		mergeBase, err := b.repo.MergeBase(opts.Args.Head, opts.Args.Remote+"/"+opts.Args.Target)
		if err != nil {
			return nil, fmt.Errorf("unable to deduce base merge-base: %w", err)
		}
		app.Args.Base = mergeBase
	}
	return app, nil
}

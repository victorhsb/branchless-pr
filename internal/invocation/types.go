package invocation

import (
	"fmt"
	"os"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
)

// CommonArgs holds resolved shared arguments across commands.
type CommonArgs struct {
	Base               string
	Head               string
	Remote             string
	Target             string
	Hyperlinks         bool
	Verbose            bool
	BranchNameTemplate string
	ShowTips           bool
}

// AppContext holds runtime state shared between commands.
type AppContext struct {
	Config         *config.Config
	Args           CommonArgs
	Git            *git.Repo
	PR             *pr.Client
	RepoRoot       string
	Username       string
	OrigBranch     string
	AutomaticStash git.StashRef
}

// RestoreStash restores the automatic stash recorded for this invocation. A
// successful restore consumes the recorded state so later lifecycle cleanup
// cannot pop another stash entry.
func (a *AppContext) RestoreStash() error {
	if a == nil || a.AutomaticStash.IsZero() {
		return nil
	}
	if err := a.Git.StashRestore(a.AutomaticStash); err != nil {
		return err
	}
	a.AutomaticStash = git.StashRef{}
	return nil
}

// ResolveSharedArgs reads defaults from config and merges CLI overrides
// to build a fully-qualified CommonArgs struct.
func ResolveSharedArgs(cfg *config.Config, base, head, remote, target string, hyperlinks, verbose *bool, tmpl string, showTips *bool) CommonArgs {
	ca := CommonArgs{}

	if remote != "" {
		ca.Remote = remote
	} else if v := cfg.Get("repo", "remote"); v != "" {
		ca.Remote = v
	} else {
		ca.Remote = "origin"
	}

	if target != "" {
		ca.Target = target
	} else if v := cfg.Get("repo", "target"); v != "" {
		ca.Target = v
	} else {
		ca.Target = "main"
	}

	ca.Base = base
	if head != "" {
		ca.Head = head
	} else {
		ca.Head = "HEAD"
	}

	if hyperlinks != nil {
		ca.Hyperlinks = *hyperlinks
	} else if cfg.Has("common", "hyperlinks") {
		b, _ := cfg.GetBool("common", "hyperlinks")
		ca.Hyperlinks = b
	} else {
		ca.Hyperlinks = true
	}

	if verbose != nil {
		ca.Verbose = *verbose
	} else if cfg.Has("common", "verbose") {
		b, _ := cfg.GetBool("common", "verbose")
		ca.Verbose = b
	} else {
		ca.Verbose = false
	}

	if tmpl != "" {
		ca.BranchNameTemplate = tmpl
	} else if v := cfg.Get("repo", "branch_name_template"); v != "" {
		ca.BranchNameTemplate = v
	} else {
		ca.BranchNameTemplate = "$USERNAME/stack"
	}

	if showTips != nil {
		ca.ShowTips = *showTips
	} else if cfg.Has("common", "show_tips") {
		b, _ := cfg.GetBool("common", "show_tips")
		ca.ShowTips = b
	} else {
		ca.ShowTips = true
	}

	return ca
}

// DefaultReviewer returns the reviewer string with precedence:
// 1. arg, 2. STACK_PR_DEFAULT_REVIEWER env, 3. config repo.reviewer.
func DefaultReviewer(cfg *config.Config, arg string) string {
	if arg != "" {
		return arg
	}
	if v := os.Getenv("STACK_PR_DEFAULT_REVIEWER"); v != "" {
		return v
	}
	if v := cfg.Get("repo", "reviewer"); v != "" {
		return v
	}
	return ""
}

// RequireCleanRepo exits with an error if the working tree has tracked changes.
func RequireCleanRepo(repo *git.Repo) error {
	dirty, err := repo.HasTrackedChanges()
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("ERROR: working tree is not clean; tracked/staged/unstaged changes exist")
	}
	return nil
}

// WithRecovery runs fn and ensures the original branch and recorded automatic
// stash are restored on any error or panic. It should wrap the main body of
// commands that mutate local state.
func WithRecovery(app *AppContext, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			_ = app.Git.CheckoutBranch(app.OrigBranch)
			_ = app.RestoreStash()
		}
	}()
	return fn()
}

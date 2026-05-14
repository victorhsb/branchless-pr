package cli

import (
	"fmt"
	"os"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
)

// CommonArgs holds resolved shared arguments across commands (SPEC §9.3).
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
	Config       *config.Config
	Args         CommonArgs
	RepoRoot     string
	Username     string
	OrigBranch   string
	StashCreated bool
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
func RequireCleanRepo() error {
	changes, err := git.UncommittedChanges()
	if err != nil {
		return err
	}
	for status := range changes {
		// Ignore untracked files (status starts with "??")
		if status != "??" {
			return fmt.Errorf("ERROR: working tree is not clean; tracked/staged/unstaged changes exist")
		}
	}
	return nil
}

// WithRecovery runs fn and ensures the original branch is restored (and hidden
// stash popped) on any error or panic. It should wrap the main body of
// commands that mutate local state.
func WithRecovery(app *AppContext, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			// Restore original branch best-effort.
			_ = git.CheckoutBranch(app.OrigBranch)
			if app.StashCreated {
				_ = git.StashPop()
			}
		}
	}()
	return fn()
}

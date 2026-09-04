package git

import (
	"fmt"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// BranchExists reports whether a local branch exists.
func (r *Repo) BranchExists(branch string) (bool, error) {
	if err := ValidateRefName("branch name", branch); err != nil {
		return false, &Error{Op: "branch_exists", Err: err}
	}
	args := []string{"git", "show-ref", "-q", "refs/heads/" + branch}
	opts := r.opts(shell.RunOpts{Quiet: true, Check: false})
	_, _, err := r.runner().Run(args, opts)
	if err == nil {
		return true, nil
	}
	if code, ok := shell.ExitCode(err); ok && code == 1 {
		return false, nil
	}
	return false, &Error{Op: "branch_exists", Err: err}
}

// CurrentBranchName returns the name of the current branch.
func (r *Repo) CurrentBranchName() (string, error) {
	args := []string{"git", "rev-parse", "--abbrev-ref", "HEAD"}
	out, err := r.runner().Output(args, r.opts(shell.RunOpts{}))
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil && exitErr.ExitCode() == 128 {
			err = exitErr
		}
		return "", &Error{Op: "current_branch_name", Err: err}
	}
	return out, nil
}

// RepoRoot returns the absolute path of the repository root.
func (r *Repo) RepoRoot() (string, error) {
	args := []string{"git", "rev-parse", "--show-toplevel"}
	out, err := r.runner().Output(args, r.opts(shell.RunOpts{}))
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil && exitErr.ExitCode() == 128 {
			err = exitErr
		}
		return "", &Error{Op: "repo_root", Err: err}
	}
	return out, nil
}

// HasTrackedChanges reports whether porcelain status contains staged or
// unstaged tracked changes. Untracked files do not make the repository dirty.
func (r *Repo) HasTrackedChanges() (bool, error) {
	out, err := r.runner().Output([]string{"git", "status", "--porcelain"}, r.opts(shell.RunOpts{}))
	if err != nil {
		return false, &Error{Op: "uncommitted_changes", Err: err}
	}
	for _, line := range strings.Split(out, "\n") {
		if line != "" && !strings.HasPrefix(line, "??") {
			return true, nil
		}
	}
	return false, nil
}

// Checkout creates or resets branch from startPoint.
func (r *Repo) Checkout(branch, startPoint string) error {
	if err := ValidateRevisionArg("start point", startPoint); err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	if err := ValidateRefName("branch name", branch); err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	_, _, err := r.runner().Run(
		[]string{"git", "checkout", startPoint, "-B", branch},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	return nil
}

// ForceUpdateBranch creates or resets branch from startPoint without switching the worktree.
func (r *Repo) ForceUpdateBranch(branch, startPoint string) error {
	if err := ValidateRefName("branch name", branch); err != nil {
		return &Error{Op: "force_update_branch", Err: err}
	}
	if err := ValidateRevisionArg("start point", startPoint); err != nil {
		return &Error{Op: "force_update_branch", Err: err}
	}
	current, currentErr := r.CurrentBranchName()
	if currentErr == nil && current == branch {
		currentSHA, err := r.RevParse("HEAD")
		if err != nil {
			return &Error{Op: "force_update_branch", Err: err}
		}
		targetSHA, err := r.RevParse(startPoint)
		if err != nil {
			return &Error{Op: "force_update_branch", Err: err}
		}
		if currentSHA == targetSHA {
			return nil
		}
		return &Error{
			Op: "force_update_branch",
			Err: fmt.Errorf(
				"cannot reset currently checked out branch %q from %s to %s; switch to a non-generated branch and retry",
				branch,
				currentSHA,
				targetSHA,
			),
		}
	}
	_, _, err := r.runner().Run(
		[]string{"git", "branch", "-f", branch, startPoint},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return &Error{Op: "force_update_branch", Err: err}
	}
	return nil
}

// CheckoutBranch switches to branch without -B (used for post-op restore).
func (r *Repo) CheckoutBranch(branch string) error {
	if err := ValidateRefName("branch name", branch); err != nil {
		return &Error{Op: "checkout_branch", Err: err}
	}
	_, _, err := r.runner().Run(
		[]string{"git", "checkout", branch},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return &Error{Op: "checkout_branch", Err: err}
	}
	return nil
}

// DeleteLocalBranches deletes local branches (best-effort, ignores failure).
func (r *Repo) DeleteLocalBranches(branches ...string) {
	if len(branches) == 0 {
		return
	}
	args := append([]string{"git", "branch", "-D"}, branches...)
	_, _ = r.runner().Output(args, r.opts(shell.RunOpts{Quiet: true, Check: false}))
}

// CommitAmend amends HEAD with a new message from stdin.
func (r *Repo) CommitAmend(msg []byte) error {
	_, _, err := r.runner().Run(
		[]string{"git", "commit", "--amend", "-F", "-"},
		r.opts(shell.RunOpts{Stdin: msg}),
	)
	if err != nil {
		return &Error{Op: "commit_amend", Err: err}
	}
	return nil
}

// CommitMsg returns the current commit message for HEAD.
func (r *Repo) CommitMsg() (string, error) {
	out, err := r.runner().Output(
		[]string{"git", "log", "-1", "--pretty=%B"},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return "", &Error{Op: "commit_msg", Err: err}
	}
	return out, nil
}

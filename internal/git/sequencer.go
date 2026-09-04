package git

import (
	"os"
	"path/filepath"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// IsRebaseInProgress reports whether a rebase is currently active.
func (r *Repo) IsRebaseInProgress() bool {
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if r.operationPathExists(name) {
			return true
		}
	}
	return false
}

// IsMergeInProgress reports whether a merge is currently active.
func (r *Repo) IsMergeInProgress() bool {
	return r.operationPathExists("MERGE_HEAD")
}

// IsCherryPickInProgress reports whether a cherry-pick is currently active.
func (r *Repo) IsCherryPickInProgress() bool {
	for _, name := range []string{"sequencer/todo", "CHERRY_PICK_HEAD"} {
		if r.operationPathExists(name) {
			return true
		}
	}
	return false
}

// AnySequencerInProgress reports whether any rebase, merge, or cherry-pick is active.
func (r *Repo) AnySequencerInProgress() bool {
	return r.IsRebaseInProgress() || r.IsMergeInProgress() || r.IsCherryPickInProgress()
}

// operationPathExists asks Git to resolve an operation marker in the supplied
// repository context. Git owns this mapping because metadata may live outside
// a worktree's .git path (for example in linked worktrees or submodules).
func (r *Repo) operationPathExists(marker string) bool {
	opts := r.opts(shell.RunOpts{Quiet: true})
	path, err := r.runner().Output([]string{"git", "rev-parse", "--git-path", marker}, opts)
	if err != nil || path == "" {
		return false
	}
	if opts.Dir != "" && !filepath.IsAbs(path) {
		path = filepath.Join(opts.Dir, path)
	}
	_, err = os.Stat(path)
	return err == nil
}

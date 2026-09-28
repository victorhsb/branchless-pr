package git

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// TrackedChangeCount counts porcelain records for tracked changes, excluding
// untracked files. A staged and unstaged change to one file counts once.
func (r *Repo) TrackedChangeCount() (int, error) {
	out, err := r.runner().Output([]string{"git", "status", "--porcelain"}, r.opts(shell.RunOpts{}))
	if err != nil {
		return 0, &Error{Op: "uncommitted_changes", Err: err}
	}
	count := 0
	for _, line := range strings.Split(out, "\n") {
		if len(line) >= 2 && !strings.HasPrefix(line, "??") {
			count++
		}
	}
	return count, nil
}

// RebaseInProgress preserves resolution failures so best-effort callers can
// distinguish an unknown repository state from an inactive rebase.
func (r *Repo) RebaseInProgress() (bool, error) {
	dir, err := r.runner().Output([]string{"git", "rev-parse", "--git-dir"}, r.opts(shell.RunOpts{}))
	if err != nil {
		return false, &Error{Op: "rebase_in_progress", Err: err}
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(r.Dir, dir)
	}
	for _, marker := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true, nil
		}
	}
	return false, nil
}

package git

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestRebaseInProgressInspection(t *testing.T) {
	for _, marker := range []string{"", "rebase-merge", "rebase-apply"} {
		t.Run("marker="+marker, func(t *testing.T) {
			dir := t.TempDir()
			if marker != "" {
				if err := os.MkdirAll(filepath.Join(dir, ".git", marker), 0755); err != nil {
					t.Fatal(err)
				}
			}
			run := shelltest.New(t, shelltest.Response{Match: shelltest.Exact("git", "rev-parse", "--git-dir"), Stdout: ".git"})
			active, err := New(dir, run).RebaseInProgress()
			if err != nil || active != (marker != "") {
				t.Fatalf("active=%v, err=%v", active, err)
			}
			if run.Calls()[0].Opts.Dir != dir {
				t.Fatal("repository working directory not passed to runner")
			}
		})
	}
}

func TestInspectRepositoryErrorsPreserveCause(t *testing.T) {
	cause := errors.New("repository inaccessible")
	for _, op := range []string{"status", "rebase"} {
		t.Run(op, func(t *testing.T) {
			run := shelltest.New(t, shelltest.Response{Err: cause})
			repo := New("", run)
			var err error
			if op == "status" {
				_, err = repo.TrackedChangeCount()
			} else {
				_, err = repo.RebaseInProgress()
			}
			if !errors.Is(err, cause) {
				t.Fatalf("lost cause: %v", err)
			}
		})
	}
}

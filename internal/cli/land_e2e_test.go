package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestLandE2E(t *testing.T) {
	for _, whole := range []bool{false, true} {
		t.Run(fmt.Sprintf("whole-stack=%v", whole), func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.mustInvoke("submit")
			tree := f.gitRun(f.repo, "rev-parse", "HEAD^{tree}")
			main := f.gitRun(f.remote, "rev-parse", "main")
			if whole {
				f.mustInvoke("land", "--whole-stack")
				if f.prs[1].BaseRefName != "main" || f.prs[0].State != "OPEN" || f.prs[1].State != "OPEN" {
					t.Fatal("whole-stack did not queue tip correctly")
				}
				if f.gitRun(f.remote, "rev-parse", "main") != main {
					t.Fatal("queued merge advanced target prematurely")
				}
				last := f.calls[len(f.calls)-1]
				if strings.Join(last, " ") != "gh pr merge "+f.prs[1].URL+" --rebase --auto" {
					t.Fatalf("queued command=%v", last)
				}
			} else {
				f.mustInvoke("land")
				if f.prs[0].State != "MERGED" || f.prs[1].State != "OPEN" || f.prs[1].BaseRefName != "main" {
					t.Fatal("bottom landing did not merge and retarget correctly")
				}
				if f.gitRun(f.remote, "rev-parse", "main") == main {
					t.Fatal("bottom landing did not advance target")
				}
				if f.gitRun(f.repo, "rev-list", "--count", "main..feature") != "1" {
					t.Fatal("bottom landing did not remove landed commit")
				}
				// Cleanup may recreate the feature commit with a different committer
				// date. Verify the published history and content rather than OID equality.
				remoteHead := f.prs[1].HeadRefName
				if f.gitRun(f.remote, "rev-parse", remoteHead+"^") != f.gitRun(f.remote, "rev-parse", "main") {
					t.Fatal("remaining remote commit was not rebased onto the merged target")
				}
				if f.gitRun(f.remote, "rev-parse", remoteHead+"^{tree}") != f.gitRun(f.repo, "rev-parse", "HEAD^{tree}") {
					t.Fatal("rebased remaining commit content was not pushed")
				}
			}
			f.assertRestored()
			if f.gitRun(f.repo, "rev-parse", "HEAD^{tree}") != tree {
				t.Fatal("landing changed feature content")
			}
		})
	}
}

func TestLandSinglePRE2E(t *testing.T) {
	f := newLifecycleFixture(t)
	f.gitRun(f.repo, "reset", "--hard", "HEAD~1")
	f.mustInvoke("submit")
	f.mustInvoke("land")
	f.assertRestored()
	target := f.gitRun(f.remote, "rev-parse", "main")
	for _, branch := range []string{"main", "feature"} {
		if got := f.gitRun(f.repo, "rev-parse", branch); got != target {
			t.Fatalf("%s=%s, want landed target %s", branch, got, target)
		}
	}
	if f.gitRun(f.repo, "branch", "--format=%(refname:short)") != "feature\nmain" {
		t.Fatal("single-PR landing left generated branches")
	}
}

func TestLandRejectsDirtyTreeE2E(t *testing.T) {
	assertDirtyTreeRejectedE2E(t, "land")
}
func TestLandEmptyStackE2E(t *testing.T) {
	assertEmptyStackUnchangedE2E(t, "land")
}

func TestLandNativeSafetyE2E(t *testing.T) {
	for _, whole := range []bool{false, true} {
		t.Run(fmt.Sprintf("native landing whole-stack=%v", whole), func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.mustInvoke("submit")
			f.enableNative()
			before := f.snapshot()
			args := []string{"land"}
			if whole {
				args = append(args, "--whole-stack")
			}
			err := f.invoke(args...)
			if err == nil || !strings.Contains(err.Error(), "landing is not supported") {
				t.Fatalf("error=%v", err)
			}
			if f.snapshot() != before {
				t.Fatal("native landing refusal changed refs")
			}
			f.assertRestored()
		})
	}
}

func TestLandMergeFailureE2E(t *testing.T) {
	for _, whole := range []bool{false, true} {
		t.Run(fmt.Sprintf("whole-stack=%v", whole), func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.mustInvoke("submit")
			main := f.gitRun(f.remote, "rev-parse", "main")
			f.fail = "pr merge"
			args := []string{"land"}
			if whole {
				args = append(args, "--whole-stack")
			}
			if err := f.invoke(args...); err == nil || !strings.Contains(err.Error(), "injected GitHub failure") {
				t.Fatalf("error=%v", err)
			}
			f.assertRestored()
			if f.gitRun(f.remote, "rev-parse", "main") != main {
				t.Fatal("failed operation advanced target")
			}
		})
	}
}
func TestLandRejectsDisabledMergeQueueE2E(t *testing.T) {
	f := newLifecycleFixture(t)
	f.mustInvoke("submit")
	before := f.snapshot()
	f.queue = false
	if err := f.invoke("land", "--whole-stack"); err == nil || !strings.Contains(err.Error(), "merge queue enabled") {
		t.Fatalf("error=%v", err)
	}
	if f.snapshot() != before {
		t.Fatal("queue preflight changed refs")
	}
	f.assertRestored()
}

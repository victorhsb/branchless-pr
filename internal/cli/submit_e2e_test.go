package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubmitE2E(t *testing.T) {
	for _, engine := range []string{"legacy", "experimental"} {
		t.Run(engine, func(t *testing.T) {
			f := newLifecycleFixture(t)
			if engine == "experimental" {
				t.Setenv(experimentalSubmitEngineEnv, "1")
			}
			before := f.snapshot()
			f.mustInvoke("submit", "--dry-run")
			if f.snapshot() != before || len(f.prs) != 0 {
				t.Fatal("dry-run mutated refs or created PRs")
			}
			f.mustInvoke("submit")
			f.assertRestored()
			if len(f.prs) != 2 {
				t.Fatalf("PR count=%d", len(f.prs))
			}
			if f.prs[0].BaseRefName != "main" || f.prs[1].BaseRefName != f.prs[0].HeadRefName {
				t.Fatalf("incorrect PR bases: %+v", f.prs)
			}
			for i, p := range f.prs {
				sha := f.gitRun(f.remote, "rev-parse", p.HeadRefName)
				local := f.gitRun(f.repo, "rev-parse", fmt.Sprintf("HEAD~%d", 1-i))
				if sha != local {
					t.Fatalf("PR #%d remote commit %s != local %s", p.Number, sha, local)
				}
				msg := f.gitRun(f.repo, "show", "-s", "--format=%B", sha)
				if !strings.Contains(msg, "stack-info: PR: "+p.URL+", branch: "+p.HeadRefName) {
					t.Fatalf("missing metadata: %s", msg)
				}
				if !strings.Contains(p.Body, []string{"first body", "second body"}[i]) {
					t.Fatalf("missing commit body: %s", p.Body)
				}
			}
			f.write("second.txt", "updated content\n")
			f.gitRun(f.repo, "add", "second.txt")
			f.gitRun(f.repo, "commit", "--amend", "--no-edit")
			f.mustInvoke("export")
			f.assertRestored()
			if len(f.prs) != 2 {
				t.Fatal("republish created duplicate PRs")
			}
			if f.gitRun(f.remote, "rev-parse", f.prs[1].HeadRefName) != f.gitRun(f.repo, "rev-parse", "HEAD") {
				t.Fatal("republish did not update remote tip")
			}
		})
	}
}

func TestSubmitPreservesDraftStatesE2E(t *testing.T) {
	for _, engine := range []string{"legacy", "experimental"} {
		t.Run(engine, func(t *testing.T) {
			f := newLifecycleFixture(t)
			if engine == "experimental" {
				t.Setenv(experimentalSubmitEngineEnv, "1")
			}
			f.mustInvoke("submit", "--draft-bitmask", "10")
			f.mustInvoke("submit")
			f.assertRestored()
			if !f.prs[0].IsDraft || f.prs[1].IsDraft {
				t.Fatalf("republish changed draft states: bottom=%v tip=%v", f.prs[0].IsDraft, f.prs[1].IsDraft)
			}
		})
	}
}

func TestSubmitRestoresStashE2E(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("create fails=%v", fail), func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.write("tracked.txt", "prior user stash\n")
			f.gitRun(f.repo, "stash", "push", "-m", "user stash")
			userStash := f.gitRun(f.repo, "rev-parse", "refs/stash")
			f.write("tracked.txt", "unsaved work\n")
			if fail {
				f.fail = "pr create"
			}
			err := f.invoke("submit", "--stash")
			if fail {
				if err == nil || !strings.Contains(err.Error(), "injected GitHub failure") {
					t.Fatalf("error=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if f.gitRun(f.repo, "branch", "--show-current") != "feature" {
				t.Fatal("stash recovery did not restore original branch")
			}
			data, err := os.ReadFile(filepath.Join(f.repo, "tracked.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "unsaved work\n" {
				t.Fatalf("restored contents=%q", data)
			}
			if got := f.gitRun(f.repo, "stash", "list", "--format=%H"); got != userStash {
				t.Fatalf("stash list=%q, want preserved user stash %q", got, userStash)
			}
		})
	}
}

func TestSubmitRejectsDirtyTreeE2E(t *testing.T) {
	assertDirtyTreeRejectedE2E(t, "submit")
}
func TestSubmitEmptyStackE2E(t *testing.T) {
	assertEmptyStackUnchangedE2E(t, "submit")
}

func TestSubmitCreateFailureE2E(t *testing.T) {
	f := newLifecycleFixture(t)
	main := f.gitRun(f.remote, "rev-parse", "main")
	f.fail = "pr create"
	if err := f.invoke("submit"); err == nil || !strings.Contains(err.Error(), "injected GitHub failure") {
		t.Fatalf("error=%v", err)
	}
	f.assertRestored()
	if f.gitRun(f.remote, "rev-parse", "main") != main {
		t.Fatal("failed operation advanced target")
	}
}

package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestAbandonE2E(t *testing.T) {
	f := newLifecycleFixture(t)
	f.mustInvoke("submit")
	tree := f.gitRun(f.repo, "rev-parse", "HEAD^{tree}")
	f.gitRun(f.repo, "push", "origin", "main:refs/heads/alice/stack/unrelated")
	f.mustInvoke("abandon")
	f.assertRestored()
	if f.gitRun(f.repo, "rev-parse", "HEAD^{tree}") != tree {
		t.Fatal("abandon changed file content")
	}
	messages := f.gitRun(f.repo, "log", "main..HEAD", "--format=%B")
	if strings.Contains(messages, "stack-info:") || !strings.Contains(messages, "first body") || !strings.Contains(messages, "second body") {
		t.Fatalf("metadata stripping damaged commit messages: %s", messages)
	}
	refs := f.remoteRefs()
	if !strings.Contains(refs, "refs/heads/alice/stack/unrelated") {
		t.Fatal("abandon deleted unrelated matching branch")
	}
	for _, p := range f.prs {
		if strings.Contains(refs, "refs/heads/"+p.HeadRefName+"\n") || strings.HasSuffix(refs, "refs/heads/"+p.HeadRefName) {
			t.Fatalf("generated remote branch survived: %s", p.HeadRefName)
		}
	}
	if f.gitRun(f.repo, "branch", "--format=%(refname:short)") != "feature\nmain" {
		t.Fatal("generated local branches survived abandon")
	}
}

func TestAbandonRewriteFailureE2E(t *testing.T) {
	f := newLifecycleFixture(t)
	f.mustInvoke("submit")
	before := f.remoteRefs()
	original := f.gitRun(f.repo, "rev-parse", "feature")
	f.failGit = "commit --amend"
	if err := f.invoke("abandon"); err == nil || !strings.Contains(err.Error(), "injected Git failure") {
		t.Fatalf("error=%v", err)
	}
	f.assertRestored()
	if f.remoteRefs() != before || f.gitRun(f.repo, "rev-parse", "feature") != original {
		t.Fatal("failed metadata rewrite deleted remote branches or changed original branch")
	}
}

func TestAbandonNativeUnstackFailureE2E(t *testing.T) {
	f := newLifecycleFixture(t)
	f.mustInvoke("submit")
	f.enableNative()
	before := f.snapshot()
	f.fail = "api --include --method POST"
	if err := f.invoke("abandon"); err == nil || !strings.Contains(err.Error(), "injected GitHub failure") {
		t.Fatalf("error=%v", err)
	}
	if f.snapshot() != before {
		t.Fatal("failed native unstack allowed branch cleanup")
	}
	f.assertRestored()
}

func TestAbandonRejectsDirtyTreeE2E(t *testing.T) {
	assertDirtyTreeRejectedE2E(t, "abandon")
}
func TestAbandonEmptyStackE2E(t *testing.T) {
	assertEmptyStackUnchangedE2E(t, "abandon")
}

func TestAbandonNativeSafetyE2E(t *testing.T) {
	for _, incomplete := range []bool{false, true} {
		t.Run(fmt.Sprintf("native abandon incomplete=%v", incomplete), func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.mustInvoke("submit")
			f.enableNative()
			before := f.snapshot()
			f.incompleteUnstack = incomplete
			err := f.invoke("abandon")
			if incomplete {
				if err == nil || !strings.Contains(err.Error(), "left unmerged PR") {
					t.Fatalf("error=%v", err)
				}
				if f.snapshot() != before {
					t.Fatal("incomplete unstack allowed local or remote cleanup")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(f.remoteRefs(), "refs/heads/alice/stack/") {
					t.Fatal("unstack success did not permit cleanup")
				}
			}
			f.assertRestored()
		})
	}
}

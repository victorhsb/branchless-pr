package stack

import (
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/pr"
)

func TestVerifyWithInfoUsesCachedPRState(t *testing.T) {
	e := verifyTestEntry("https://github.com/foo/bar/pull/42", "alice/stack/1", "main")
	lookupCalls := 0

	err := VerifyWithInfo(Stack{e}, true, func(prRef string) (*pr.Info, bool) {
		lookupCalls++
		return &pr.Info{
			BaseRefName:      "main",
			HeadRefName:      "alice/stack/1",
			Number:           42,
			State:            "OPEN",
			MergeStateStatus: "CLEAN",
		}, true
	})
	if err != nil {
		t.Fatalf("VerifyWithInfo returned error: %v", err)
	}
	if lookupCalls != 1 {
		t.Fatalf("lookup calls = %d, want 1", lookupCalls)
	}
}

func TestVerifyWithInfoPreservesValidationFailures(t *testing.T) {
	e := verifyTestEntry("https://github.com/foo/bar/pull/42", "alice/stack/1", "main")

	err := VerifyWithInfo(Stack{e}, true, func(prRef string) (*pr.Info, bool) {
		return &pr.Info{
			BaseRefName:      "wrong-base",
			HeadRefName:      "alice/stack/1",
			Number:           42,
			State:            "OPEN",
			MergeStateStatus: "CLEAN",
		}, true
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "base branch mismatch") {
		t.Fatalf("error = %q, want base branch mismatch", err.Error())
	}
}

func verifyTestEntry(prURL, head, base string) *Entry {
	e := &Entry{Commit: &Header{
		SHA:   "0123456789abcdef0123456789abcdef01234567",
		Title: "Test commit",
	}}
	e.SetPR(prURL)
	e.SetHead(head)
	e.SetBase(base)
	return e
}

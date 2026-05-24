package stack

import (
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/pr"
)

func TestVerifyWithProviderUsesCachedInfo(t *testing.T) {
	e := &Entry{Commit: &Header{SHA: "0123456789abcdef0123456789abcdef01234567", Title: "Title"}}
	e.SetPR("https://github.com/acme/repo/pull/42")
	e.SetHead("alice/stack/1")
	e.SetBase("main")

	calls := 0
	err := VerifyWithProvider(Stack{e}, true, func(prRef string) (*pr.Info, error) {
		calls++
		return &pr.Info{
			BaseRefName:      "main",
			HeadRefName:      "alice/stack/1",
			Number:           42,
			State:            "OPEN",
			MergeStateStatus: "CLEAN",
		}, nil
	})
	if err != nil {
		t.Fatalf("VerifyWithProvider returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls = %d, want 1", calls)
	}
}

func TestVerifyWithProviderPreservesValidationFailures(t *testing.T) {
	e := &Entry{Commit: &Header{SHA: "0123456789abcdef0123456789abcdef01234567", Title: "Title"}}
	e.SetPR("https://github.com/acme/repo/pull/42")
	e.SetHead("alice/stack/1")
	e.SetBase("main")

	err := VerifyWithProvider(Stack{e}, true, func(prRef string) (*pr.Info, error) {
		return &pr.Info{
			BaseRefName:      "develop",
			HeadRefName:      "alice/stack/1",
			Number:           42,
			State:            "OPEN",
			MergeStateStatus: "CLEAN",
		}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "base branch mismatch") {
		t.Fatalf("error = %v, want base branch mismatch", err)
	}
}

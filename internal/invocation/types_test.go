package invocation

import (
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
)

func TestResolveSharedArgsDefaultsHeadToHEAD(t *testing.T) {
	args := ResolveSharedArgs(config.Defaults(), "", "", "", "", nil, nil, "", nil)
	if args.Head != "HEAD" {
		t.Fatalf("Head = %q, want HEAD", args.Head)
	}
}

func TestResolveSharedArgsHonorsExplicitHead(t *testing.T) {
	args := ResolveSharedArgs(config.Defaults(), "", "feature-tip", "", "", nil, nil, "", nil)
	if args.Head != "feature-tip" {
		t.Fatalf("Head = %q, want feature-tip", args.Head)
	}
}

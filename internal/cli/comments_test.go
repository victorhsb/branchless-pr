package cli

import (
	"os"
	"strings"
	"testing"
)

func TestCommentsCmdExposesFlags(t *testing.T) {
	cmd := commentsCmd()
	if got := cmd.Use; got != "comments" {
		t.Fatalf("Use = %q, want comments", got)
	}
	for _, name := range []string{"format", "unresolved-only", "kind", "author"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("--%s flag not registered", name)
		}
	}
}

func TestRootCleanCheckExemptsComments(t *testing.T) {
	rootData, err := os.ReadFile("root.go")
	if err != nil {
		t.Fatal(err)
	}
	bootstrapData, err := os.ReadFile("../invocation/bootstrap.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rootData), `invocation.PolicyFor`) ||
		!strings.Contains(string(rootData), `Policy:       policy`) ||
		!strings.Contains(string(bootstrapData), `!opts.Policy.AllowsDirty`) {
		t.Fatal("root clean check does not use invocation policy")
	}
}

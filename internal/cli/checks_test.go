package cli

import (
	"os"
	"strings"
	"testing"
)

func TestChecksCmdExposesFlags(t *testing.T) {
	cmd := checksCmd()
	if got := cmd.Use; got != "checks" {
		t.Fatalf("Use = %q, want checks", got)
	}
	for _, name := range []string{"format", "failed-only", "required-only", "verbose", "pr", "commit"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("--%s flag not registered", name)
		}
	}
}

func TestRootCleanCheckExemptsChecks(t *testing.T) {
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

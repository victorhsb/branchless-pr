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
	data, err := os.ReadFile("root.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `invocation.PolicyFor`) || !strings.Contains(string(data), `!policy.AllowsDirty`) {
		t.Fatal("root clean check does not use invocation policy")
	}
}

package cli

import "testing"

func TestCIWatchCmdExposesFlagsAndDefaults(t *testing.T) {
	cmd := ciWatchCmd()
	if got := cmd.Use; got != "ci-watch" {
		t.Fatalf("Use = %q, want ci-watch", got)
	}
	for _, name := range []string{"interval", "timeout", "live", "no-live"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("--%s flag not registered", name)
		}
	}
	if got := cmd.Flags().Lookup("interval").DefValue; got != "1m0s" {
		t.Fatalf("interval default = %q, want 1m0s", got)
	}
	if got := cmd.Flags().Lookup("timeout").DefValue; got != "15m0s" {
		t.Fatalf("timeout default = %q, want 15m0s", got)
	}
	if got := cmd.Flags().Lookup("live").DefValue; got != "true" {
		t.Fatalf("live default = %q, want true", got)
	}
}

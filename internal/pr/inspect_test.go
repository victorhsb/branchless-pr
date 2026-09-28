package pr

import (
	"errors"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestProbeAvailabilityPreservesFailureEvidence(t *testing.T) {
	cause := errors.New("request failed")
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("gh", "api", "/rate_limit"),
		Stdout: "HTTP 503", Stderr: "Service unavailable", Err: cause,
	})
	detail, err := NewClient(run).ProbeAvailability()
	if !errors.Is(err, cause) {
		t.Fatalf("lost cause: %v", err)
	}
	for _, want := range []string{"HTTP 503", "Service unavailable", "request failed"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in %q", want, detail)
		}
	}
	if !run.Calls()[0].Opts.Quiet {
		t.Fatal("probe output must remain in the report")
	}
}

func TestInspectPRRejectsUnsafeReferenceBeforeExecution(t *testing.T) {
	if _, err := NewClient(shelltest.New(t)).InspectPR("--help"); err == nil {
		t.Fatal("accepted option-like PR reference")
	}
}

func TestLookPathDoesNotExecuteGH(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := NewClient(shelltest.New(t)).LookPath(); err == nil {
		t.Fatal("expected missing gh")
	}
}

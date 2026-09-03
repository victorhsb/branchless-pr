package shelltest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

func TestFakeRecordsCallsAndReturnsOutput(t *testing.T) {
	fake := New(t, Response{
		Name:   "rev-parse",
		Match:  Exact("git", "rev-parse", "HEAD"),
		Stdout: "abc123\n",
	})

	got, err := fake.Output(
		[]string{"git", "rev-parse", "HEAD"},
		shell.RunOpts{Dir: "/repo"},
	)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if got != "abc123" {
		t.Fatalf("Output() = %q, want %q", got, "abc123")
	}

	calls := fake.Calls()
	if len(calls) != 1 {
		t.Fatalf("len(Calls()) = %d, want 1", len(calls))
	}
	if calls[0].Opts.Dir != "/repo" {
		t.Fatalf("call Dir = %q, want /repo", calls[0].Opts.Dir)
	}
	if !calls[0].Opts.Quiet || !calls[0].Opts.Check {
		t.Fatalf("Output call opts = %+v, want Quiet and Check", calls[0].Opts)
	}
}

func TestFakeReturnsProcessFreeExitCode(t *testing.T) {
	fake := New(t, Response{
		Match:    Prefix("git", "show-ref"),
		ExitCode: 1,
	})

	_, _, err := fake.Run(
		[]string{"git", "show-ref", "-q", "refs/heads/missing"},
		shell.RunOpts{Quiet: true, Check: true},
	)
	if err == nil {
		t.Fatal("Run() error = nil, want exit error")
	}
	code, ok := shell.ExitCode(err)
	if !ok || code != 1 {
		t.Fatalf("shell.ExitCode(error) = (%d, %v), want (1, true)", code, ok)
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error %T does not wrap *shelltest.ExitError", err)
	}
}

func TestFakeHonorsExplicitOutputBuffers(t *testing.T) {
	fake := New(t, Response{
		Match:  Exact("command"),
		Stdout: "out",
		Stderr: "err",
	})
	var stdout, stderr bytes.Buffer

	out, errOut, err := fake.Run(
		[]string{"command"},
		shell.RunOpts{Quiet: true, Stdout: &stdout, Stderr: &stderr},
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out != nil || errOut != nil {
		t.Fatalf("Run() returned (%q, %q), want output in explicit buffers", out, errOut)
	}
	if stdout.String() != "out" || stderr.String() != "err" {
		t.Fatalf("buffers = (%q, %q), want (out, err)", stdout.String(), stderr.String())
	}
}

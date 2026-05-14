package shell

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunQuietFalseCapturesNothing(t *testing.T) {
	_, _, err := Run([]string{"sh", "-c", "echo hi"}, RunOpts{Quiet: false, Check: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunQuietTrueCapturesStdout(t *testing.T) {
	var out bytes.Buffer
	_, _, err := Run([]string{"sh", "-c", "echo hello"}, RunOpts{Quiet: true, Check: true, Stdout: &out})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected stdout to contain 'hello', got %q", out.String())
	}
}

func TestRunQuietTrueCapturesStderrOnFailure(t *testing.T) {
	_, errBuf, err := Run([]string{"sh", "-c", "echo boom >&2; exit 7"}, RunOpts{Quiet: true, Check: true})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(string(errBuf), "boom") {
		t.Fatalf("expected captured stderr to contain 'boom', got %q", string(errBuf))
	}
	if AsExitError(err) == nil {
		t.Fatalf("expected exit error in chain")
	}
}

func TestRunCheckFalseReturnsNoError(t *testing.T) {
	_, _, err := Run([]string{"sh", "-c", "exit 3"}, RunOpts{Quiet: true, Check: false})
	if err == nil {
		t.Fatalf("expected raw exec.ExitError to be returned")
	}
	if AsExitError(err) == nil {
		t.Fatalf("expected exit error in chain")
	}
}

func TestOutputStripsTrailingWhitespace(t *testing.T) {
	got, err := Output([]string{"sh", "-c", "printf 'hello\\n\\n'"}, RunOpts{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

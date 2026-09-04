package shell

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRunQuietFalseCapturesNothing(t *testing.T) {
	_, _, err := (Default{}).Run([]string{"sh", "-c", "echo hi"}, RunOpts{Quiet: false, Check: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunQuietTrueCapturesStdout(t *testing.T) {
	var out bytes.Buffer
	_, _, err := (Default{}).Run([]string{"sh", "-c", "echo hello"}, RunOpts{Quiet: true, Check: true, Stdout: &out})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected stdout to contain 'hello', got %q", out.String())
	}
}

func TestRunQuietTrueCapturesStderrOnFailure(t *testing.T) {
	_, errBuf, err := (Default{}).Run([]string{"sh", "-c", "echo boom >&2; exit 7"}, RunOpts{Quiet: true, Check: true})
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
	_, _, err := (Default{}).Run([]string{"sh", "-c", "exit 3"}, RunOpts{Quiet: true, Check: false})
	if err == nil {
		t.Fatalf("expected raw exec.ExitError to be returned")
	}
	if AsExitError(err) == nil {
		t.Fatalf("expected exit error in chain")
	}
}

func TestOutputStripsTrailingWhitespace(t *testing.T) {
	got, err := (Default{}).Output([]string{"sh", "-c", "printf 'hello\\n\\n'"}, RunOpts{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestExitCodeFindsWrappedExitCoder(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", exitCodeError(17))
	code, ok := ExitCode(err)
	if !ok || code != 17 {
		t.Fatalf("ExitCode() = (%d, %v), want (17, true)", code, ok)
	}
}

func TestExitCodeRejectsOrdinaryError(t *testing.T) {
	if code, ok := ExitCode(fmt.Errorf("ordinary")); ok {
		t.Fatalf("ExitCode() = (%d, true), want (_, false)", code)
	}
}

type exitCodeError int

func (e exitCodeError) Error() string { return fmt.Sprintf("exit status %d", e) }
func (e exitCodeError) ExitCode() int { return int(e) }

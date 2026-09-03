// Package shell wraps subprocess execution, matching the semantics of the
// Python stack-pr tool (https://github.com/modular/stack-pr).
package shell

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// RunOpts configures a single invocation of Run.
type RunOpts struct {
	Quiet bool
	Check bool
	Dir   string
	// Stdin is passed through to the underlying command when non-nil.
	Stdin []byte
	// Explicit overrides for stdout/stderr. If nil, behaviour is governed by Quiet.
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}

// Runner executes subprocesses.
type Runner interface {
	Run(args []string, opts RunOpts) ([]byte, []byte, error)
	Output(args []string, opts RunOpts) (string, error)
}

// Default executes subprocesses through os/exec.
type Default struct{}

var _ Runner = Default{}

// Run executes a command given as a slice of string arguments.
//
// A non-zero exit status always returns an error. When opts.Check is true, the
// error is additionally wrapped with the command argv; the zero value is
// false. If opts.Quiet is true, stdout and stderr are captured to bytes.Buffer
// unless the caller provides explicit buffers. If opts.Quiet is false, the
// command inherits the process stdout/stderr.
func Run(args []string, opts RunOpts) ([]byte, []byte, error) {
	return Default{}.Run(args, opts)
}

// Run executes a command given as a slice of string arguments.
func (Default) Run(args []string, opts RunOpts) ([]byte, []byte, error) {
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("shell.Run: empty command")
	}

	cmd := exec.Command(args[0], args[1:]...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	var outBuf, errBuf bytes.Buffer
	if opts.Quiet {
		if opts.Stdout != nil {
			cmd.Stdout = opts.Stdout
		} else {
			cmd.Stdout = &outBuf
		}
		if opts.Stderr != nil {
			cmd.Stderr = opts.Stderr
		} else {
			cmd.Stderr = &errBuf
		}
	} else {
		if opts.Stdout != nil {
			cmd.Stdout = opts.Stdout
		}
		if opts.Stderr != nil {
			cmd.Stderr = opts.Stderr
		}
		// otherwise inherited from os.Stdout / os.Stderr implicitly
	}

	if opts.Stdin != nil {
		cmd.Stdin = bytes.NewReader(opts.Stdin)
	}

	slog.Debug("shell.Run", "cmd", args, "dir", opts.Dir, "quiet", opts.Quiet)

	err := cmd.Run()
	if err != nil && opts.Check {
		return outBuf.Bytes(), errBuf.Bytes(), fmt.Errorf("shell.Run %v: %w", args, err)
	}
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// ExitError is an error carrying a subprocess exit code.
type ExitError interface {
	error
	ExitCode() int
}

// AsExitError extracts an ExitError from an error chain.
func AsExitError(err error) ExitError {
	if err == nil {
		return nil
	}
	var exitErr ExitError
	if errors.As(err, &exitErr) {
		return exitErr
	}
	return nil
}

// ExitCode returns the process exit code carried by err.
//
// It accepts any error in the chain that implements ExitCode, including
// *exec.ExitError and process-free test errors.
func ExitCode(err error) (int, bool) {
	exitErr := AsExitError(err)
	if exitErr == nil {
		return 0, false
	}
	return exitErr.ExitCode(), true
}

// Output executes a command and returns its stdout as a UTF-8 string with
// trailing whitespace stripped (rtrim). It always enables Quiet and Check.
func Output(args []string, opts RunOpts) (string, error) {
	return Default{}.Output(args, opts)
}

// Output executes a command and returns its stdout as a UTF-8 string with
// trailing whitespace stripped (rtrim). It always enables Quiet and Check.
func (d Default) Output(args []string, opts RunOpts) (string, error) {
	opts.Quiet = true
	opts.Check = true
	out, _, err := d.Run(args, opts)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), " \t\n\r"), nil
}

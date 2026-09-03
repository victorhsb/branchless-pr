// Package shelltest provides process-free test implementations of shell.Runner.
package shelltest

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// Matcher reports whether a command matches an expected invocation.
type Matcher func(args []string, opts shell.RunOpts) bool

// Response describes one expected runner call and its result.
type Response struct {
	Name     string
	Match    Matcher
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Call records one runner invocation.
type Call struct {
	Args []string
	Opts shell.RunOpts
}

// Fake is a process-free, concurrency-safe shell.Runner.
//
// Responses are matched in order. Unmatched calls fail the test unless
// AllowUnmatched is set, in which case they return success.
type Fake struct {
	t testing.TB

	mu        sync.Mutex
	responses []Response
	calls     []Call
	next      int

	AllowUnmatched bool
}

var _ shell.Runner = (*Fake)(nil)

// New returns a Fake with the supplied ordered responses.
func New(t testing.TB, responses ...Response) *Fake {
	t.Helper()
	f := &Fake{
		t:         t,
		responses: append([]Response(nil), responses...),
	}
	t.Cleanup(func() {
		if !t.Failed() {
			f.Verify()
		}
	})
	return f
}

// Exact matches a complete argv slice.
func Exact(want ...string) Matcher {
	expected := append([]string(nil), want...)
	return func(args []string, _ shell.RunOpts) bool {
		if len(args) != len(expected) {
			return false
		}
		for i := range expected {
			if args[i] != expected[i] {
				return false
			}
		}
		return true
	}
}

// Prefix matches the beginning of an argv slice.
func Prefix(want ...string) Matcher {
	expected := append([]string(nil), want...)
	return func(args []string, _ shell.RunOpts) bool {
		if len(args) < len(expected) {
			return false
		}
		for i := range expected {
			if args[i] != expected[i] {
				return false
			}
		}
		return true
	}
}

// Run implements shell.Runner.
func (f *Fake) Run(args []string, opts shell.RunOpts) ([]byte, []byte, error) {
	f.t.Helper()

	call := Call{
		Args: append([]string(nil), args...),
		Opts: cloneOpts(opts),
	}

	f.mu.Lock()
	f.calls = append(f.calls, call)
	if f.next >= len(f.responses) {
		allow := f.AllowUnmatched
		f.mu.Unlock()
		if allow {
			return nil, nil, nil
		}
		err := fmt.Errorf("shelltest: unexpected command: %q", args)
		f.t.Errorf("%v", err)
		return nil, nil, err
	}
	response := f.responses[f.next]
	f.next++
	responseNumber := f.next
	f.mu.Unlock()

	if response.Match != nil && !response.Match(args, opts) {
		name := response.Name
		if name == "" {
			name = fmt.Sprintf("response %d", responseNumber)
		}
		err := fmt.Errorf("shelltest: command %q did not match %s", args, name)
		f.t.Errorf("%v", err)
		return nil, nil, err
	}

	stdout, stderr := capture(response, opts)
	err := response.Err
	if err == nil && response.ExitCode != 0 {
		err = &ExitError{Code: response.ExitCode}
	}
	if err != nil && opts.Check {
		err = fmt.Errorf("shell.Run %v: %w", args, err)
	}
	return stdout, stderr, err
}

// Output implements shell.Runner. It always enables Quiet and Check.
func (f *Fake) Output(args []string, opts shell.RunOpts) (string, error) {
	opts.Quiet = true
	opts.Check = true
	out, _, err := f.Run(args, opts)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), " \t\n\r"), nil
}

// Calls returns a snapshot of all recorded calls.
func (f *Fake) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()

	calls := make([]Call, len(f.calls))
	for i, call := range f.calls {
		calls[i] = Call{
			Args: append([]string(nil), call.Args...),
			Opts: cloneOpts(call.Opts),
		}
	}
	return calls
}

// Verify fails the test if an expected response was not consumed.
func (f *Fake) Verify() {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.next != len(f.responses) {
		f.t.Errorf("shelltest: %d of %d expected commands were not called", len(f.responses)-f.next, len(f.responses))
	}
}

// ExitError is an in-memory subprocess failure.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// ExitCode returns the configured process exit code.
func (e *ExitError) ExitCode() int {
	return e.Code
}

func cloneOpts(opts shell.RunOpts) shell.RunOpts {
	opts.Stdin = append([]byte(nil), opts.Stdin...)
	return opts
}

func capture(response Response, opts shell.RunOpts) ([]byte, []byte) {
	stdout := []byte(response.Stdout)
	stderr := []byte(response.Stderr)

	if opts.Stdout != nil {
		_, _ = opts.Stdout.Write(stdout)
		stdout = nil
	} else if !opts.Quiet {
		stdout = nil
	}
	if opts.Stderr != nil {
		_, _ = opts.Stderr.Write(stderr)
		stderr = nil
	} else if !opts.Quiet {
		stderr = nil
	}
	return stdout, stderr
}

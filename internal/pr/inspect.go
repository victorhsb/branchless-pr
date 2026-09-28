package pr

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// LookPath checks CLI installation without executing gh or accessing GitHub.
func (c *Client) LookPath() error {
	_, err := exec.LookPath("gh")
	return err
}

// AuthStatus checks the current gh authentication.
func (c *Client) AuthStatus() error {
	_, _, err := c.runner().Run([]string{"gh", "auth", "status"}, shell.RunOpts{Quiet: true})
	return err
}

// ProbeAvailability returns the complete failure detail for best-effort outage
// classification. Authentication failures are not necessarily service outages.
func (c *Client) ProbeAvailability() (string, error) {
	out, stderr, err := c.runner().Run([]string{"gh", "api", "/rate_limit"}, shell.RunOpts{Quiet: true})
	if err == nil {
		return "", nil
	}
	detail := strings.TrimSpace(strings.Join([]string{string(out), string(stderr), err.Error()}, " "))
	return detail, err
}

// DecodeError distinguishes malformed GitHub responses from query failures.
type DecodeError struct{ Err error }

func (e *DecodeError) Error() string { return fmt.Sprintf("parse PR state: %v", e.Err) }
func (e *DecodeError) Unwrap() error { return e.Err }

// InspectPR reads only the fields required for diagnostic state checks.
func (c *Client) InspectPR(ref string) (*Info, error) {
	if err := ValidateRef(ref); err != nil {
		return nil, err
	}
	out, err := c.runner().Output([]string{"gh", "pr", "view", ref, "--json", "baseRefName,headRefName,number,state,mergeStateStatus,isDraft"}, shell.RunOpts{})
	if err != nil {
		return nil, err
	}
	var info Info
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return nil, &DecodeError{Err: err}
	}
	return &info, nil
}

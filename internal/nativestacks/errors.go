package nativestacks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// FeatureUnavailable is returned when the GitHub native Stacks feature is not
// enabled for the repository or account.
type FeatureUnavailable struct {
	Msg string
}

func (e *FeatureUnavailable) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "GitHub native Stacks is unavailable for this repository"
}

// IsFeatureUnavailable reports whether err signals that native Stacks is
// unavailable as opposed to an authentication/authorization/transport error.
func IsFeatureUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var fu *FeatureUnavailable
	if errors.As(err, &fu) {
		return true
	}
	// The gh-stack extension documents exit code 9 for unavailable native stacks.
	exit := shell.AsExitError(err)
	if exit != nil && exit.ExitCode() == 9 {
		return true
	}
	return false
}

// IsAuthenticationError reports whether err is an auth or authorization failure.
func IsAuthenticationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAnyLower(msg, []string{
		"authentication",
		"unauthorized",
		"401",
		"403",
		"forbidden",
	}) && !IsFeatureUnavailable(err)
}

// ClassifyExtensionError converts a gh stack command failure into a stable
// error type. It avoids parsing human-readable status text.
func ClassifyExtensionError(err error, stderr []byte) error {
	if err == nil {
		return nil
	}
	exit := shell.AsExitError(err)
	if exit != nil {
		switch exit.ExitCode() {
		case 9:
			return &FeatureUnavailable{Msg: "GitHub native Stacks is unavailable (gh-stack exit code 9)"}
		default:
			return fmt.Errorf("gh-stack failed (exit %d): %w: %s", exit.ExitCode(), err, string(stderr))
		}
	}
	return fmt.Errorf("gh-stack failed: %w: %s", err, string(stderr))
}

func containsAnyLower(s string, subs []string) bool {
	s = strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

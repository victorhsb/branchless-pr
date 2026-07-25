package nativestacks

import (
	"errors"
	"fmt"
	"strings"
)

// FeatureUnavailable is returned only when ordinary repository access succeeds
// and the repository-level Stacks endpoint reports that the preview is absent.
type FeatureUnavailable struct {
	Msg string
}

func (e *FeatureUnavailable) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "GitHub native Stacks is unavailable for this repository"
}

// IsFeatureUnavailable reports whether err is the repository-level preview
// availability result.
func IsFeatureUnavailable(err error) bool {
	var unavailable *FeatureUnavailable
	return errors.As(err, &unavailable)
}

// StackNotFound distinguishes a missing numbered Stack from a repository where
// the preview feature is unavailable.
type StackNotFound struct {
	Number int
	Err    error
}

func (e *StackNotFound) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("native Stack #%d not found: %v", e.Number, e.Err)
	}
	return fmt.Sprintf("native Stack #%d not found", e.Number)
}

func (e *StackNotFound) Unwrap() error { return e.Err }

// IsStackNotFound reports whether err identifies a missing numbered Stack.
func IsStackNotFound(err error) bool {
	var notFound *StackNotFound
	return errors.As(err, &notFound)
}

// APIError preserves REST operation context and whether a failed write may
// have reached GitHub.
type APIError struct {
	Method         string
	Endpoint       string
	Status         int
	Message        string
	Headers        string
	OutcomeUnknown bool
	Err            error
}

func (e *APIError) Error() string {
	var parts []string
	if e.Method != "" || e.Endpoint != "" {
		parts = append(parts, strings.TrimSpace(e.Method+" "+e.Endpoint))
	}
	if e.Status != 0 {
		parts = append(parts, fmt.Sprintf("HTTP %d", e.Status))
	}
	if strings.TrimSpace(e.Message) != "" {
		parts = append(parts, strings.TrimSpace(e.Message))
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	if len(parts) == 0 {
		return "GitHub API request failed"
	}
	return strings.Join(parts, ": ")
}

func (e *APIError) Unwrap() error { return e.Err }

// IsAPIStatus reports whether an error chain contains the given HTTP status.
func IsAPIStatus(err error, status int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == status
}

// IsAuthenticationError reports whether err is an authentication or
// authorization failure.
func IsAuthenticationError(err error) bool {
	return IsAPIStatus(err, 401) || IsAPIStatus(err, 403)
}

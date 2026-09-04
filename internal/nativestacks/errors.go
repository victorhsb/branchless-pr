package nativestacks

import (
	"errors"
	"fmt"

	"github.com/victorhsb/branchless-pr/internal/pr"
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

// IsAuthenticationError reports whether err is an authentication or
// authorization failure.
func IsAuthenticationError(err error) bool {
	return pr.IsAPIStatus(err, 401) || pr.IsAPIStatus(err, 403)
}

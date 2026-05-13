package git

import "fmt"

const (
	// NotARepo is the git exit code when run outside a repository.
	NotARepo = 128
	// SHALength is the length of a full Git SHA.
	SHALength = 40
)

// Error is returned for selected Git / gh helper failures.
type Error struct {
	Op  string // what operation failed
	Err error  // underlying error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("git: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("git: %s", e.Op)
}

func (e *Error) Unwrap() error { return e.Err }

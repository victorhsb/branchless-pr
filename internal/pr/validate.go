package pr

import (
	"fmt"
	"strings"
	"unicode"
)

// PR references originate from `stack-info:` metadata embedded in commit
// messages, which the stack-info regex captures without validation. That value
// reaches `gh pr view/edit/merge/ready <ref>` as a positional, so a ref
// beginning with `-` would be parsed by gh as a flag. A legitimate ref is a
// PR URL or a bare number, so rejecting leading dashes, whitespace, and
// control characters rejects nothing legitimate.

// ValidateRef reports whether ref is usable as a positional `gh pr` argument.
func ValidateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("pull request reference is empty")
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("invalid pull request reference %q: must not begin with '-' (gh would parse it as a flag)", ref)
	}
	for _, r := range ref {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("invalid pull request reference %q: must not contain whitespace or control characters", ref)
		}
	}
	return nil
}

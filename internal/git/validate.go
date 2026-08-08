package git

import (
	"fmt"
	"strings"
	"unicode"
)

// Values that reach `git` and `gh` argument vectors as positionals come from
// sources the user does not necessarily control: `.stack-pr.cfg` (which can be
// checked into a repository) and `stack-info:` metadata embedded in commit
// messages. Git accepts a transport URL anywhere it accepts a remote name and
// parses leading-dash positionals as options, so an unvalidated value such as
// `--upload-pack=<cmd>` or `ext::sh -c '<cmd>'` is code execution rather than a
// failed lookup. These helpers keep such values positional.
//
// Callers also pass `--` before remote positionals; validation and termination
// are complementary, since `--` does not stop a value that git reads as a URL.

// ValidateRemoteName reports whether remote is usable as a git remote *name*.
//
// Name-only is already the effective contract elsewhere in this package:
// RepoSlug runs `git remote get-url <remote>` and TargetExists builds
// `<remote>/<target>` and rev-parses it, neither of which works for a URL.
func ValidateRemoteName(remote string) error {
	if remote == "" {
		return fmt.Errorf("remote name is empty")
	}
	if strings.HasPrefix(remote, "-") {
		return fmt.Errorf("invalid remote name %q: must not begin with '-' (git would parse it as an option)", remote)
	}
	if strings.HasPrefix(remote, "/") || strings.HasPrefix(remote, ".") {
		return fmt.Errorf("invalid remote name %q: looks like a path, expected a configured remote name such as \"origin\"", remote)
	}
	if strings.Contains(remote, ":") {
		return fmt.Errorf("invalid remote name %q: looks like a URL, expected a configured remote name such as \"origin\"", remote)
	}
	if err := rejectControlOrSpace("remote name", remote); err != nil {
		return err
	}
	return nil
}

// ValidateRefName reports whether ref is usable as a positional branch or
// refspec component. It rejects the option-injection and whitespace cases; git
// itself enforces the remaining ref syntax rules via check-ref-format.
func ValidateRefName(kind, ref string) error {
	if ref == "" {
		return fmt.Errorf("%s is empty", kind)
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("invalid %s %q: must not begin with '-' (git would parse it as an option)", kind, ref)
	}
	return rejectControlOrSpace(kind, ref)
}

// validateRemoteAndRefs checks a remote name plus the branch names that will be
// built into refspec positionals alongside it, wrapping failures as an Error
// tagged with the calling operation.
func validateRemoteAndRefs(op, remote string, refs []string) error {
	if err := ValidateRemoteName(remote); err != nil {
		return &Error{Op: op, Err: err}
	}
	for _, r := range refs {
		if err := ValidateRefName("branch name", r); err != nil {
			return &Error{Op: op, Err: err}
		}
	}
	return nil
}

func rejectControlOrSpace(kind, s string) error {
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("invalid %s %q: must not contain whitespace or control characters", kind, s)
		}
	}
	return nil
}

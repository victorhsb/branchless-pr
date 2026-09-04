package git

import (
	"fmt"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// StashRef identifies one exact stash commit. Its zero value means that no
// stash was created.
type StashRef struct {
	OID string
}

// IsZero reports whether the reference identifies no stash.
func (s StashRef) IsZero() bool { return s.OID == "" }

// StashSave stashes tracked changes with an optional message and returns the
// exact created stash identity. Creation is detected from refs/stash rather
// than localized command output.
func (r *Repo) StashSave(msg string) (StashRef, error) {
	if msg == "" {
		msg = "stack-pr auto-stash"
	}
	before, err := r.stashHead()
	if err != nil {
		return StashRef{}, err
	}
	_, _, err = r.runner().Run(
		[]string{"git", "stash", "push", "-m", msg},
		r.opts(shell.RunOpts{Quiet: true}),
	)
	if err != nil {
		return StashRef{}, &Error{Op: "stash_save", Err: err}
	}
	after, err := r.stashHead()
	if err != nil {
		return StashRef{}, err
	}
	if after == before {
		return StashRef{}, nil
	}
	if after == "" {
		return StashRef{}, &Error{Op: "stash_save", Err: fmt.Errorf("refs/stash disappeared after Git reported success")}
	}
	return StashRef{OID: after}, nil
}

// StashRestore applies one exact stash commit and drops only its matching
// reflog entry. Apply failures leave the stash available for manual recovery.
func (r *Repo) StashRestore(ref StashRef) error {
	if ref.IsZero() {
		return nil
	}
	if _, err := r.stashSelector(ref); err != nil {
		return err
	}
	_, _, err := r.runner().Run(
		[]string{"git", "stash", "apply", "--quiet", ref.OID},
		r.opts(shell.RunOpts{Quiet: true}),
	)
	if err != nil {
		return &Error{
			Op:  "stash_apply",
			Err: fmt.Errorf("automatic stash %s could not be applied and was kept for manual recovery: %w", ref.OID, err),
		}
	}
	selector, err := r.stashSelector(ref)
	if err != nil {
		return &Error{
			Op:  "stash_drop",
			Err: fmt.Errorf("automatic stash %s was applied but its reflog entry could not be found for removal: %w", ref.OID, err),
		}
	}
	_, _, err = r.runner().Run(
		[]string{"git", "stash", "drop", "--quiet", selector},
		r.opts(shell.RunOpts{Quiet: true}),
	)
	if err != nil {
		return &Error{
			Op:  "stash_drop",
			Err: fmt.Errorf("automatic stash %s was applied but could not be removed; drop %s manually: %w", ref.OID, selector, err),
		}
	}
	return nil
}

func (r *Repo) stashHead() (string, error) {
	out, _, err := r.runner().Run(
		[]string{"git", "rev-parse", "--verify", "--quiet", "refs/stash^{commit}"},
		r.opts(shell.RunOpts{Quiet: true}),
	)
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	if code, ok := shell.ExitCode(err); ok && code == 1 {
		return "", nil
	}
	return "", &Error{Op: "stash_ref", Err: err}
}

func (r *Repo) stashSelector(ref StashRef) (string, error) {
	out, err := r.runner().Output(
		[]string{"git", "stash", "list", "--format=%H%x00%gd"},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return "", &Error{Op: "stash_list", Err: err}
	}
	for _, line := range strings.Split(out, "\n") {
		oid, selector, ok := strings.Cut(line, "\x00")
		if ok && oid == ref.OID && selector != "" {
			return selector, nil
		}
	}
	return "", &Error{
		Op:  "stash_find",
		Err: fmt.Errorf("automatic stash %s is no longer present; no other stash was changed", ref.OID),
	}
}

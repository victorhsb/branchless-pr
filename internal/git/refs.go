package git

import (
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

const lowerHex = "0123456789abcdef"

// IsFullSHA reports whether s is exactly 40 lowercase hexadecimal characters.
func IsFullSHA(s string) bool {
	if len(s) != SHALength {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune(lowerHex, r) {
			return false
		}
	}
	return true
}

// MergeBase returns the common ancestor of a and b.
func (r *Repo) MergeBase(a, b string) (string, error) {
	if err := ValidateRevisionArg("first revision", a); err != nil {
		return "", &Error{Op: "merge_base", Err: err}
	}
	if err := ValidateRevisionArg("second revision", b); err != nil {
		return "", &Error{Op: "merge_base", Err: err}
	}
	out, err := r.runner().Output([]string{"git", "merge-base", a, b}, r.opts(shell.RunOpts{}))
	if err != nil {
		return "", &Error{Op: "merge_base", Err: err}
	}
	return out, nil
}

// RevListHeaders returns NUL-delimited commit headers for base..head.
func (r *Repo) RevListHeaders(base, head string) (string, error) {
	if err := ValidateRevisionArg("base revision", base); err != nil {
		return "", &Error{Op: "rev_list_headers", Err: err}
	}
	if err := ValidateRevisionArg("head revision", head); err != nil {
		return "", &Error{Op: "rev_list_headers", Err: err}
	}
	out, err := r.runner().Output(
		[]string{"git", "rev-list", "--header", "^" + base, head},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return "", &Error{Op: "rev_list_headers", Err: err}
	}
	return out, nil
}

// BranchlessStackHead returns the top commit in the current git-branchless
// stack. The boolean is false when git-branchless is unavailable, the repo is
// not initialized for branchless, or the command returns no valid commits.
func (r *Repo) BranchlessStackHead() (string, bool) {
	opts := r.opts(shell.RunOpts{Quiet: true, Check: false})
	out, _, err := r.runner().Run([]string{"git", "branchless", "query", "-r", "stack()"}, opts)
	if err != nil {
		return "", false
	}

	var top string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !IsFullSHA(line) {
			return "", false
		}
		top = line
	}
	if top == "" {
		return "", false
	}
	return top, true
}

// IsAncestor reports whether a is an ancestor of b.
func (r *Repo) IsAncestor(a, b string) (bool, error) {
	if err := ValidateRevisionArg("ancestor revision", a); err != nil {
		return false, &Error{Op: "is_ancestor", Err: err}
	}
	if err := ValidateRevisionArg("descendant revision", b); err != nil {
		return false, &Error{Op: "is_ancestor", Err: err}
	}
	_, _, err := r.runner().Run(
		[]string{"git", "merge-base", "--is-ancestor", a, b},
		r.opts(shell.RunOpts{Quiet: true, Check: false}),
	)
	if err == nil {
		return true, nil
	}
	if code, ok := shell.ExitCode(err); ok && code == 1 {
		return false, nil
	}
	return false, &Error{Op: "is_ancestor", Err: err}
}

// Rebase runs git rebase with optional extra arguments before upstream.
// If branch is empty it rebases the current branch.
func (r *Repo) Rebase(upstream, branch string, extras ...string) error {
	if err := ValidateRevisionArg("upstream revision", upstream); err != nil {
		return &Error{Op: "rebase", Err: err}
	}
	if branch != "" {
		if err := ValidateRefName("branch name", branch); err != nil {
			return &Error{Op: "rebase", Err: err}
		}
	}
	args := []string{"git", "rebase"}
	args = append(args, extras...)
	args = append(args, upstream)
	if branch != "" {
		args = append(args, branch)
	}
	_, _, err := r.runner().Run(args, r.opts(shell.RunOpts{}))
	if err != nil {
		return &Error{Op: "rebase", Err: err}
	}
	return nil
}

// RebaseWithAuthorDate is like Rebase but with --committer-date-is-author-date.
func (r *Repo) RebaseWithAuthorDate(upstream, branch string) error {
	return r.Rebase(upstream, branch, "--committer-date-is-author-date")
}

// RevParse resolves a ref to its full 40-char SHA.
func (r *Repo) RevParse(ref string) (string, error) {
	if err := ValidateRevisionArg("revision", ref); err != nil {
		return "", &Error{Op: "rev_parse", Err: err}
	}
	out, err := r.runner().Output([]string{"git", "rev-parse", "--verify", ref}, r.opts(shell.RunOpts{}))
	if err != nil {
		return "", &Error{Op: "rev_parse", Err: err}
	}
	return out, nil
}

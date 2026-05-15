package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

var lowerHex = "0123456789abcdef"

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

// BranchExists reports whether a local branch exists.
func BranchExists(branch string, repoDir ...string) (bool, error) {
	args := []string{"git", "show-ref", "-q", "refs/heads/" + branch}
	opts := shell.RunOpts{Quiet: true, Check: false}
	if len(repoDir) > 0 && repoDir[0] != "" {
		opts.Dir = repoDir[0]
	}
	_, _, err := shell.Run(args, opts)
	if err == nil {
		return true, nil
	}
	if exitErr := shell.AsExitError(err); exitErr != nil {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}
	return false, &Error{Op: "branch_exists", Err: err}
}

// CurrentBranchName returns the name of the current branch.
func CurrentBranchName(repoDir ...string) (string, error) {
	args := []string{"git", "rev-parse", "--abbrev-ref", "HEAD"}
	opts := shell.RunOpts{}
	if len(repoDir) > 0 && repoDir[0] != "" {
		opts.Dir = repoDir[0]
	}
	out, err := shell.Output(args, opts)
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil {
			if exitErr.ExitCode() == NotARepo {
				return "", &Error{Op: "current_branch_name", Err: exitErr}
			}
		}
		return "", &Error{Op: "current_branch_name", Err: err}
	}
	return out, nil
}

// RepoRoot returns the absolute path of the repository root.
func RepoRoot(repoDir ...string) (string, error) {
	args := []string{"git", "rev-parse", "--show-toplevel"}
	opts := shell.RunOpts{}
	if len(repoDir) > 0 && repoDir[0] != "" {
		opts.Dir = repoDir[0]
	}
	out, err := shell.Output(args, opts)
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil {
			if exitErr.ExitCode() == NotARepo {
				return "", &Error{Op: "repo_root", Err: exitErr}
			}
		}
		return "", &Error{Op: "repo_root", Err: err}
	}
	return out, nil
}

// UncommittedChanges parses `git status --porcelain` and returns a map keyed
// by the first two status characters, with values from column 4 onward.
func UncommittedChanges(repoDir ...string) (map[string]string, error) {
	args := []string{"git", "status", "--porcelain"}
	opts := shell.RunOpts{}
	if len(repoDir) > 0 && repoDir[0] != "" {
		opts.Dir = repoDir[0]
	}
	out, err := shell.Output(args, opts)
	if err != nil {
		return nil, &Error{Op: "uncommitted_changes", Err: err}
	}
	result := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		path := strings.TrimSpace(line[2:])
		result[status] = path
	}
	return result, nil
}

// CheckGHInstalled verifies that `gh` is on PATH.
func CheckGHInstalled() error {
	_, err := shell.Output([]string{"gh"}, shell.RunOpts{})
	if err != nil {
		return &Error{
			Op:  "check_gh_installed",
			Err: fmt.Errorf("gh does not appear to be installed; see https://cli.github.com/: %w", err),
		}
	}
	return nil
}

var loginRe = regexp.MustCompile(`"login"\s*:\s*"([^"]+)"`)

// GetGHUsername returns the current GitHub login name.
func GetGHUsername() (string, error) {
	if u := gitConfig.UsernameOverride(); u != nil {
		return *u, nil
	}
	out, err := shell.Output([]string{"gh", "api", "graphql", "-f", "query=query{viewer{login}}"}, shell.RunOpts{})
	if err != nil {
		return "", &Error{Op: "get_gh_username", Err: err}
	}
	m := loginRe.FindStringSubmatch(out)
	if m == nil {
		return "", &Error{Op: "get_gh_username", Err: fmt.Errorf("could not parse login from gh response")}
	}
	return m[1], nil
}

// GetChangedFiles returns the paths of files changed between base and HEAD.
// If base is empty, it defaults to "main".
func GetChangedFiles(base string, repoDir ...string) ([]string, error) {
	if base == "" {
		base = "main"
	}
	args := []string{"git", "diff", "--name-only", base + "...HEAD"}
	opts := shell.RunOpts{}
	if len(repoDir) > 0 && repoDir[0] != "" {
		opts.Dir = repoDir[0]
	}
	out, err := shell.Output(args, opts)
	if err != nil {
		return nil, &Error{Op: "get_changed_files", Err: err}
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// GetChangedDirs returns the set of top-level directories that contain changed files.
func GetChangedDirs(base string, repoDir ...string) (map[string]struct{}, error) {
	files, err := GetChangedFiles(base, repoDir...)
	if err != nil {
		return nil, err
	}
	dirs := make(map[string]struct{})
	for _, f := range files {
		dir := filepath.Dir(f)
		if dir == "." {
			dir = ""
		}
		dirs[dir] = struct{}{}
	}
	return dirs, nil
}

// IsRebaseInProgress reports whether a rebase is currently active.
// Per SPEC §11, with repoDir it checks repoDir/.git/rebase-*;
// without it, it checks ./.git/rebase-*.
func IsRebaseInProgress(repoDir ...string) bool {
	gitDir := ".git"
	if len(repoDir) > 0 && repoDir[0] != "" {
		gitDir = filepath.Join(repoDir[0], ".git")
	}
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(gitDir, name)); err == nil {
			return true
		}
	}
	return false
}

// MergeBase returns the common ancestor of a and b.
func MergeBase(a, b string) (string, error) {
	out, err := shell.Output([]string{"git", "merge-base", a, b}, shell.RunOpts{})
	if err != nil {
		return "", &Error{Op: "merge_base", Err: err}
	}
	return out, nil
}

// BranchlessStackHead returns the top commit in the current git-branchless
// stack. The boolean is false when git-branchless is unavailable, the repo is
// not initialized for branchless, or the command returns no valid commits.
func BranchlessStackHead(repoDir ...string) (string, bool) {
	opts := shell.RunOpts{Quiet: true, Check: false}
	if len(repoDir) > 0 && repoDir[0] != "" {
		opts.Dir = repoDir[0]
	}
	out, _, err := shell.Run([]string{"git", "branchless", "query", "-r", "stack()"}, opts)
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
func IsAncestor(a, b string) (bool, error) {
	_, _, err := shell.Run(
		[]string{"git", "merge-base", "--is-ancestor", a, b},
		shell.RunOpts{Quiet: true, Check: false},
	)
	if err == nil {
		return true, nil
	}
	if exitErr := shell.AsExitError(err); exitErr != nil {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}
	return false, &Error{Op: "is_ancestor", Err: err}
}

// Fetch runs git fetch --prune on the given remote.
func Fetch(remote string) error {
	_, err := shell.Output([]string{"git", "fetch", "--prune", remote}, shell.RunOpts{})
	if err != nil {
		return &Error{Op: "fetch", Err: err}
	}
	return nil
}

// Checkout creates or resets branch from startPoint.
func Checkout(startPoint, branch string) error {
	_, _, err := shell.Run(
		[]string{"git", "checkout", startPoint, "-B", branch},
		shell.RunOpts{},
	)
	if err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	return nil
}

// CheckoutBranch switches to branch without -B (used for post-op restore).
func CheckoutBranch(branch string) error {
	_, _, err := shell.Run(
		[]string{"git", "checkout", branch},
		shell.RunOpts{},
	)
	if err != nil {
		return &Error{Op: "checkout_branch", Err: err}
	}
	return nil
}

// ForcePush force-pushes local branches to remote (ref:ref format).
func ForcePush(remote string, refs ...string) error {
	args := []string{"git", "push", "-f", remote}
	for _, r := range refs {
		args = append(args, r+":"+r)
	}
	_, _, err := shell.Run(args, shell.RunOpts{})
	if err != nil {
		return &Error{Op: "force_push", Err: err}
	}
	return nil
}

// DeleteRemoteBranches deletes branches on the remote via empty ref.
func DeleteRemoteBranches(remote string, branches ...string) error {
	args := []string{"git", "push", "-f", remote}
	for _, b := range branches {
		args = append(args, ":"+b)
	}
	_, _, err := shell.Run(args, shell.RunOpts{})
	if err != nil {
		return &Error{Op: "delete_remote_branches", Err: err}
	}
	return nil
}

// DeleteLocalBranches deletes local branches (best-effort, ignores failure).
func DeleteLocalBranches(branches ...string) {
	if len(branches) == 0 {
		return
	}
	args := append([]string{"git", "branch", "-D"}, branches...)
	_, _ = shell.Output(args, shell.RunOpts{Quiet: true, Check: false})
}

// Rebase runs git rebase with optional extra args between onto/upstream.
// If branch is empty it rebases the current branch.
func Rebase(onto, branch string, extras ...string) error {
	args := []string{"git", "rebase"}
	args = append(args, extras...)
	args = append(args, onto)
	if branch != "" {
		args = append(args, branch)
	}
	_, _, err := shell.Run(args, shell.RunOpts{})
	if err != nil {
		return &Error{Op: "rebase", Err: err}
	}
	return nil
}

// RebaseWithAuthorDate is like Rebase but with --committer-date-is-author-date.
func RebaseWithAuthorDate(onto, branch string) error {
	return Rebase(onto, branch, "--committer-date-is-author-date")
}

// StashSave stashes changes with an optional message.
// It returns true if anything was actually stashed, false if working tree was clean.
func StashSave(msg string) (bool, error) {
	if msg == "" {
		msg = "stack-pr auto-stash"
	}
	out, _, err := shell.Run(
		[]string{"git", "stash", "save", msg},
		shell.RunOpts{Quiet: true, Check: false},
	)
	if err != nil {
		return false, &Error{Op: "stash_save", Err: err}
	}
	// "No local changes to save" means nothing was stashed.
	if strings.Contains(string(out), "No local changes to save") {
		return false, nil
	}
	return true, nil
}

// StashPop pops the most recent stash entry.
func StashPop() error {
	_, _, err := shell.Run(
		[]string{"git", "stash", "pop"},
		shell.RunOpts{},
	)
	if err != nil {
		return &Error{Op: "stash_pop", Err: err}
	}
	return nil
}

// RevParse resolves a ref to its full 40-char SHA.
func RevParse(ref string) (string, error) {
	out, err := shell.Output([]string{"git", "rev-parse", "--verify", ref}, shell.RunOpts{})
	if err != nil {
		return "", &Error{Op: "rev_parse", Err: err}
	}
	return out, nil
}

// CommitAmend amends HEAD with a new message from stdin.
func CommitAmend(msg []byte) error {
	_, _, err := shell.Run(
		[]string{"git", "commit", "--amend", "-F", "-"},
		shell.RunOpts{Stdin: msg},
	)
	if err != nil {
		return &Error{Op: "commit_amend", Err: err}
	}
	return nil
}

// TargetExists returns nil if remote/target is a valid ref.
func TargetExists(remote, target string) error {
	ref := remote + "/" + target
	_, err := shell.Output(
		[]string{"git", "rev-parse", "--verify", ref},
		shell.RunOpts{Quiet: true, Check: false},
	)
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil {
			if exitErr.ExitCode() == 128 {
				return fmt.Errorf("target branch %s does not exist on remote %s", target, remote)
			}
		}
		return &Error{Op: "target_exists", Err: err}
	}
	return nil
}

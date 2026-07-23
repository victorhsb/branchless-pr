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
func IsRebaseInProgress(repoDir ...string) bool {
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if operationPathExists(name, repoDir...) {
			return true
		}
	}
	return false
}

// operationPathExists asks Git to resolve an operation marker in the supplied
// repository context. Git owns this mapping because metadata may live outside
// a worktree's .git path (for example in linked worktrees or submodules).
func operationPathExists(marker string, repoDir ...string) bool {
	opts := shell.RunOpts{Quiet: true}
	if len(repoDir) > 0 && repoDir[0] != "" {
		opts.Dir = repoDir[0]
	}
	path, err := shell.Output([]string{"git", "rev-parse", "--git-path", marker}, opts)
	if err != nil || path == "" {
		return false
	}
	if opts.Dir != "" && !filepath.IsAbs(path) {
		path = filepath.Join(opts.Dir, path)
	}
	_, err = os.Stat(path)
	return err == nil
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

// ForceUpdateBranch creates or resets branch from startPoint without switching the worktree.
func ForceUpdateBranch(branch, startPoint string) error {
	current, currentErr := CurrentBranchName()
	if currentErr == nil && current == branch {
		currentSHA, err := RevParse("HEAD")
		if err != nil {
			return &Error{Op: "force_update_branch", Err: err}
		}
		targetSHA, err := RevParse(startPoint)
		if err != nil {
			return &Error{Op: "force_update_branch", Err: err}
		}
		if currentSHA == targetSHA {
			return nil
		}
		return &Error{
			Op: "force_update_branch",
			Err: fmt.Errorf(
				"cannot reset currently checked out branch %q from %s to %s; switch to a non-generated branch and retry",
				branch,
				currentSHA,
				targetSHA,
			),
		}
	}
	_, _, err := shell.Run(
		[]string{"git", "branch", "-f", branch, startPoint},
		shell.RunOpts{},
	)
	if err != nil {
		return &Error{Op: "force_update_branch", Err: err}
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

// ResolveRemoteRefs returns the current OID for each remote ref, or an empty
// string if the ref does not exist.
func ResolveRemoteRefs(remote string, refs ...string) (map[string]string, error) {
	result := make(map[string]string, len(refs))
	for _, r := range refs {
		out, err := shell.Output([]string{"git", "rev-parse", remote + "/" + r}, shell.RunOpts{Quiet: true, Check: false})
		if err != nil {
			continue // ref does not exist
		}
		result[r] = strings.TrimSpace(out)
	}
	return result, nil
}

// ForcePushWithLease force-pushes with atomic force-with-lease expectations.
// leases maps branch name to expected remote OID; an empty expectation means
// the branch must not exist.
func ForcePushWithLease(remote string, leases map[string]string, refs ...string) error {
	args := []string{"git", "push", "--force-with-lease", remote}
	for _, r := range refs {
		expect := leases[r]
		if expect == "" {
			args = append(args, r+":"+r)
		} else {
			args = append(args, fmt.Sprintf("%s:%s^%s", r, r, expect))
		}
	}
	_, _, err := shell.Run(args, shell.RunOpts{})
	if err != nil {
		return &Error{Op: "force_push_with_lease", Err: err}
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
func StashSave(msg string) (StashRef, error) {
	if msg == "" {
		msg = "stack-pr auto-stash"
	}
	before, err := stashHead()
	if err != nil {
		return StashRef{}, err
	}
	_, _, err = shell.Run(
		[]string{"git", "stash", "push", "-m", msg},
		shell.RunOpts{Quiet: true},
	)
	if err != nil {
		return StashRef{}, &Error{Op: "stash_save", Err: err}
	}
	after, err := stashHead()
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
func StashRestore(ref StashRef) error {
	if ref.IsZero() {
		return nil
	}
	if _, err := stashSelector(ref); err != nil {
		return err
	}
	_, _, err := shell.Run(
		[]string{"git", "stash", "apply", "--quiet", ref.OID},
		shell.RunOpts{Quiet: true},
	)
	if err != nil {
		return &Error{
			Op:  "stash_apply",
			Err: fmt.Errorf("automatic stash %s could not be applied and was kept for manual recovery: %w", ref.OID, err),
		}
	}
	selector, err := stashSelector(ref)
	if err != nil {
		return &Error{
			Op:  "stash_drop",
			Err: fmt.Errorf("automatic stash %s was applied but its reflog entry could not be found for removal: %w", ref.OID, err),
		}
	}
	_, _, err = shell.Run(
		[]string{"git", "stash", "drop", "--quiet", selector},
		shell.RunOpts{Quiet: true},
	)
	if err != nil {
		return &Error{
			Op:  "stash_drop",
			Err: fmt.Errorf("automatic stash %s was applied but could not be removed; drop %s manually: %w", ref.OID, selector, err),
		}
	}
	return nil
}

func stashHead() (string, error) {
	out, _, err := shell.Run(
		[]string{"git", "rev-parse", "--verify", "--quiet", "refs/stash^{commit}"},
		shell.RunOpts{Quiet: true},
	)
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	if exitErr := shell.AsExitError(err); exitErr != nil && exitErr.ExitCode() == 1 {
		return "", nil
	}
	return "", &Error{Op: "stash_ref", Err: err}
}

func stashSelector(ref StashRef) (string, error) {
	out, err := shell.Output(
		[]string{"git", "stash", "list", "--format=%H%x00%gd"},
		shell.RunOpts{},
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

// RepoSlug returns the (owner, repo) pair for the named remote by parsing the
// URL reported by `git remote get-url <remote>`. It accepts both HTTPS
// (`https://github.com/owner/repo[.git]`) and SSH (`git@github.com:owner/repo[.git]`)
// forms.
func RepoSlug(remote string) (owner, repo string, err error) {
	if remote == "" {
		return "", "", fmt.Errorf("remote name is empty")
	}
	url, err := shell.Output([]string{"git", "remote", "get-url", remote}, shell.RunOpts{Quiet: true})
	if err != nil {
		return "", "", &Error{Op: "repo_slug", Err: err}
	}
	return parseRepoSlug(url)
}

func parseRepoSlug(url string) (owner, repo string, err error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", "", fmt.Errorf("empty remote url")
	}
	var path string
	switch {
	case strings.HasPrefix(url, "git@"):
		// git@github.com:owner/repo.git
		_, p, ok := strings.Cut(url, ":")
		if !ok {
			return "", "", fmt.Errorf("invalid ssh remote url: %q", url)
		}
		path = p
	case strings.HasPrefix(url, "ssh://"):
		// ssh://git@github.com/owner/repo.git
		rest := strings.TrimPrefix(url, "ssh://")
		_, p, ok := strings.Cut(rest, "/")
		if !ok {
			return "", "", fmt.Errorf("invalid ssh remote url: %q", url)
		}
		path = p
	case strings.HasPrefix(url, "https://"), strings.HasPrefix(url, "http://"):
		// https://github.com/owner/repo[.git]
		rest := url
		rest = strings.TrimPrefix(rest, "https://")
		rest = strings.TrimPrefix(rest, "http://")
		_, p, ok := strings.Cut(rest, "/")
		if !ok {
			return "", "", fmt.Errorf("invalid https remote url: %q", url)
		}
		path = p
	default:
		return "", "", fmt.Errorf("unsupported remote url scheme: %q", url)
	}
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cannot parse owner/repo from remote url: %q", url)
	}
	return parts[0], parts[1], nil
}

// IsMergeInProgress reports whether a merge is currently active.
func IsMergeInProgress(repoDir ...string) bool {
	return operationPathExists("MERGE_HEAD", repoDir...)
}

// IsCherryPickInProgress reports whether a cherry-pick is currently active.
func IsCherryPickInProgress(repoDir ...string) bool {
	for _, name := range []string{"sequencer/todo", "CHERRY_PICK_HEAD"} {
		if operationPathExists(name, repoDir...) {
			return true
		}
	}
	return false
}

// AnySequencerInProgress reports whether any rebase, merge, or cherry-pick is active.
func AnySequencerInProgress(repoDir ...string) bool {
	return IsRebaseInProgress(repoDir...) || IsMergeInProgress(repoDir...) || IsCherryPickInProgress(repoDir...)
}

// CommitMsg returns the current commit message for HEAD.
func CommitMsg() (string, error) {
	out, err := shell.Output([]string{"git", "log", "-1", "--pretty=%B"}, shell.RunOpts{})
	if err != nil {
		return "", &Error{Op: "commit_msg", Err: err}
	}
	return out, nil
}
func TargetExists(remote, target string) error {
	ref := remote + "/" + target
	_, err := shell.Output(
		[]string{"git", "rev-parse", "--verify", ref},
		shell.RunOpts{Quiet: true, Check: false},
	)
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil {
			if exitErr.ExitCode() == 128 || exitErr.ExitCode() == 1 {
				return fmt.Errorf("target branch %s does not exist on remote %s", target, remote)
			}
		}
		return &Error{Op: "target_exists", Err: err}
	}
	return nil
}

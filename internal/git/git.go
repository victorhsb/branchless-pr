package git

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

const lowerHex = "0123456789abcdef"

// Repo runs Git commands in one working directory through an injectable runner.
// An empty Dir uses the process working directory.
type Repo struct {
	Dir string
	run shell.Runner
}

// New returns a Git repository command boundary.
func New(dir string, run shell.Runner) *Repo {
	if run == nil {
		run = shell.Default{}
	}
	return &Repo{Dir: dir, run: run}
}

func (r *Repo) runner() shell.Runner {
	return r.run
}

func (r *Repo) opts(opts shell.RunOpts) shell.RunOpts {
	if r != nil {
		opts.Dir = r.Dir
	}
	return opts
}

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
func (r *Repo) BranchExists(branch string) (bool, error) {
	if err := ValidateRefName("branch name", branch); err != nil {
		return false, &Error{Op: "branch_exists", Err: err}
	}
	args := []string{"git", "show-ref", "-q", "refs/heads/" + branch}
	opts := r.opts(shell.RunOpts{Quiet: true, Check: false})
	_, _, err := r.runner().Run(args, opts)
	if err == nil {
		return true, nil
	}
	if code, ok := shell.ExitCode(err); ok && code == 1 {
		return false, nil
	}
	return false, &Error{Op: "branch_exists", Err: err}
}

// CurrentBranchName returns the name of the current branch.
func (r *Repo) CurrentBranchName() (string, error) {
	args := []string{"git", "rev-parse", "--abbrev-ref", "HEAD"}
	out, err := r.runner().Output(args, r.opts(shell.RunOpts{}))
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil && exitErr.ExitCode() == NotARepo {
			return "", &Error{Op: "current_branch_name", Err: exitErr}
		}
		return "", &Error{Op: "current_branch_name", Err: err}
	}
	return out, nil
}

// RepoRoot returns the absolute path of the repository root.
func (r *Repo) RepoRoot() (string, error) {
	args := []string{"git", "rev-parse", "--show-toplevel"}
	out, err := r.runner().Output(args, r.opts(shell.RunOpts{}))
	if err != nil {
		if exitErr := shell.AsExitError(err); exitErr != nil && exitErr.ExitCode() == NotARepo {
			return "", &Error{Op: "repo_root", Err: exitErr}
		}
		return "", &Error{Op: "repo_root", Err: err}
	}
	return out, nil
}

// UncommittedChanges parses `git status --porcelain` and returns a map keyed
// by the first two status characters, with values from column 4 onward.
func (r *Repo) UncommittedChanges() (map[string]string, error) {
	args := []string{"git", "status", "--porcelain"}
	out, err := r.runner().Output(args, r.opts(shell.RunOpts{}))
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

// IsRebaseInProgress reports whether a rebase is currently active.
func (r *Repo) IsRebaseInProgress() bool {
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if r.operationPathExists(name) {
			return true
		}
	}
	return false
}

// operationPathExists asks Git to resolve an operation marker in the supplied
// repository context. Git owns this mapping because metadata may live outside
// a worktree's .git path (for example in linked worktrees or submodules).
func (r *Repo) operationPathExists(marker string) bool {
	opts := r.opts(shell.RunOpts{Quiet: true})
	path, err := r.runner().Output([]string{"git", "rev-parse", "--git-path", marker}, opts)
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
func (r *Repo) MergeBase(a, b string) (string, error) {
	if err := ValidateRevisionArg("first revision", a); err != nil {
		return "", &Error{Op: "merge_base", Err: err}
	}
	if err := ValidateRevisionArg("second revision", b); err != nil {
		return "", &Error{Op: "merge_base", Err: err}
	}
	out, err := r.runner().Output(
		[]string{"git", "merge-base", a, b},
		r.opts(shell.RunOpts{}),
	)
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

// Fetch runs git fetch --prune on the given remote.
func (r *Repo) Fetch(remote string) error {
	if err := ValidateRemoteName(remote); err != nil {
		return &Error{Op: "fetch", Err: err}
	}
	_, err := r.runner().Output(
		[]string{"git", "fetch", "--prune", "--", remote},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return &Error{Op: "fetch", Err: err}
	}
	return nil
}

// Checkout creates or resets branch from startPoint.
func (r *Repo) Checkout(startPoint, branch string) error {
	if err := ValidateRevisionArg("start point", startPoint); err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	if err := ValidateRefName("branch name", branch); err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	_, _, err := r.runner().Run(
		[]string{"git", "checkout", startPoint, "-B", branch},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return &Error{Op: "checkout", Err: err}
	}
	return nil
}

// ForceUpdateBranch creates or resets branch from startPoint without switching the worktree.
func (r *Repo) ForceUpdateBranch(branch, startPoint string) error {
	if err := ValidateRefName("branch name", branch); err != nil {
		return &Error{Op: "force_update_branch", Err: err}
	}
	if err := ValidateRevisionArg("start point", startPoint); err != nil {
		return &Error{Op: "force_update_branch", Err: err}
	}
	current, currentErr := r.CurrentBranchName()
	if currentErr == nil && current == branch {
		currentSHA, err := r.RevParse("HEAD")
		if err != nil {
			return &Error{Op: "force_update_branch", Err: err}
		}
		targetSHA, err := r.RevParse(startPoint)
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
	_, _, err := r.runner().Run(
		[]string{"git", "branch", "-f", branch, startPoint},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return &Error{Op: "force_update_branch", Err: err}
	}
	return nil
}

// CheckoutBranch switches to branch without -B (used for post-op restore).
func (r *Repo) CheckoutBranch(branch string) error {
	if err := ValidateRefName("branch name", branch); err != nil {
		return &Error{Op: "checkout_branch", Err: err}
	}
	_, _, err := r.runner().Run(
		[]string{"git", "checkout", branch},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return &Error{Op: "checkout_branch", Err: err}
	}
	return nil
}

// ForcePush force-pushes local branches to remote (ref:ref format).
func (r *Repo) ForcePush(remote string, refs ...string) error {
	if err := validateRemoteAndRefs("force_push", remote, refs); err != nil {
		return err
	}
	args := []string{"git", "push", "-f", "--", remote}
	for _, r := range refs {
		args = append(args, r+":"+r)
	}
	_, _, err := r.runner().Run(args, r.opts(shell.RunOpts{}))
	if err != nil {
		return &Error{Op: "force_push", Err: err}
	}
	return nil
}

// ResolveRemoteRefs reads current branch OIDs from the remote itself. Missing
// refs are omitted from the result.
func (r *Repo) ResolveRemoteRefs(remote string, refs ...string) (map[string]string, error) {
	if err := validateRemoteAndRefs("resolve_remote_refs", remote, refs); err != nil {
		return nil, err
	}
	args := []string{"git", "ls-remote", "--heads", "--", remote}
	for _, r := range refs {
		args = append(args, "refs/heads/"+r)
	}
	out, err := r.runner().Output(args, r.opts(shell.RunOpts{Quiet: true}))
	if err != nil {
		return nil, &Error{Op: "resolve_remote_refs", Err: err}
	}
	return parseRemoteBranchRefs(out), nil
}

// RemoteBranches returns all branch names advertised by remote.
func (r *Repo) RemoteBranches(remote string) ([]string, error) {
	if err := ValidateRemoteName(remote); err != nil {
		return nil, &Error{Op: "remote_branches", Err: err}
	}
	out, err := r.runner().Output(
		[]string{"git", "ls-remote", "--heads", "--", remote},
		r.opts(shell.RunOpts{Quiet: true}),
	)
	if err != nil {
		return nil, &Error{Op: "remote_branches", Err: err}
	}
	refs := parseRemoteBranchRefs(out)
	branches := make([]string, 0, len(refs))
	for branch := range refs {
		branches = append(branches, branch)
	}
	sort.Strings(branches)
	return branches, nil
}

func parseRemoteBranchRefs(out string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		const prefix = "refs/heads/"
		if strings.HasPrefix(fields[1], prefix) {
			result[strings.TrimPrefix(fields[1], prefix)] = fields[0]
		}
	}
	return result
}

// ForcePushWithLease force-pushes with atomic force-with-lease expectations.
// leases maps branch name to expected remote OID; an empty expectation means
// the branch must not exist. It uses the --force-with-lease=<ref>:<expect>
// option form (supported since git 2.0) rather than the refspec ^ notation
// (introduced in git 2.44) for broad compatibility.
func (r *Repo) ForcePushWithLease(remote string, leases map[string]string, refs ...string) error {
	if err := validateRemoteAndRefs("force_push_with_lease", remote, refs); err != nil {
		return err
	}
	args := []string{"git", "push", "--atomic"}
	for _, r := range refs {
		expect := leases[r]
		args = append(args, fmt.Sprintf("--force-with-lease=refs/heads/%s:%s", r, expect))
	}
	args = append(args, "--", remote)
	for _, r := range refs {
		args = append(args, r+":refs/heads/"+r)
	}
	_, _, err := r.runner().Run(args, r.opts(shell.RunOpts{}))
	if err != nil {
		return &Error{Op: "force_push_with_lease", Err: err}
	}
	return nil
}

// DeleteRemoteBranches deletes branches on the remote via empty ref.
func (r *Repo) DeleteRemoteBranches(remote string, branches ...string) error {
	if err := validateRemoteAndRefs("delete_remote_branches", remote, branches); err != nil {
		return err
	}
	args := []string{"git", "push", "-f", "--", remote}
	for _, b := range branches {
		args = append(args, ":"+b)
	}
	_, _, err := r.runner().Run(args, r.opts(shell.RunOpts{}))
	if err != nil {
		return &Error{Op: "delete_remote_branches", Err: err}
	}
	return nil
}

// DeleteLocalBranches deletes local branches (best-effort, ignores failure).
func (r *Repo) DeleteLocalBranches(branches ...string) {
	if len(branches) == 0 {
		return
	}
	args := append([]string{"git", "branch", "-D"}, branches...)
	_, _ = r.runner().Output(args, r.opts(shell.RunOpts{Quiet: true, Check: false}))
}

// Rebase runs git rebase with optional extra args between onto/upstream.
// If branch is empty it rebases the current branch.
func (r *Repo) Rebase(onto, branch string, extras ...string) error {
	if err := ValidateRevisionArg("upstream revision", onto); err != nil {
		return &Error{Op: "rebase", Err: err}
	}
	if branch != "" {
		if err := ValidateRefName("branch name", branch); err != nil {
			return &Error{Op: "rebase", Err: err}
		}
	}
	args := []string{"git", "rebase"}
	args = append(args, extras...)
	args = append(args, onto)
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
func (r *Repo) RebaseWithAuthorDate(onto, branch string) error {
	return r.Rebase(onto, branch, "--committer-date-is-author-date")
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

// RevParse resolves a ref to its full 40-char SHA.
func (r *Repo) RevParse(ref string) (string, error) {
	if err := ValidateRevisionArg("revision", ref); err != nil {
		return "", &Error{Op: "rev_parse", Err: err}
	}
	out, err := r.runner().Output(
		[]string{"git", "rev-parse", "--verify", ref},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return "", &Error{Op: "rev_parse", Err: err}
	}
	return out, nil
}

// CommitAmend amends HEAD with a new message from stdin.
func (r *Repo) CommitAmend(msg []byte) error {
	_, _, err := r.runner().Run(
		[]string{"git", "commit", "--amend", "-F", "-"},
		r.opts(shell.RunOpts{Stdin: msg}),
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
func (r *Repo) RepoSlug(remote string) (owner, repo string, err error) {
	if err := ValidateRemoteName(remote); err != nil {
		return "", "", err
	}
	url, err := r.runner().Output(
		[]string{"git", "remote", "get-url", "--", remote},
		r.opts(shell.RunOpts{Quiet: true}),
	)
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
func (r *Repo) IsMergeInProgress() bool {
	return r.operationPathExists("MERGE_HEAD")
}

// IsCherryPickInProgress reports whether a cherry-pick is currently active.
func (r *Repo) IsCherryPickInProgress() bool {
	for _, name := range []string{"sequencer/todo", "CHERRY_PICK_HEAD"} {
		if r.operationPathExists(name) {
			return true
		}
	}
	return false
}

// AnySequencerInProgress reports whether any rebase, merge, or cherry-pick is active.
func (r *Repo) AnySequencerInProgress() bool {
	return r.IsRebaseInProgress() || r.IsMergeInProgress() || r.IsCherryPickInProgress()
}

// CommitMsg returns the current commit message for HEAD.
func (r *Repo) CommitMsg() (string, error) {
	out, err := r.runner().Output(
		[]string{"git", "log", "-1", "--pretty=%B"},
		r.opts(shell.RunOpts{}),
	)
	if err != nil {
		return "", &Error{Op: "commit_msg", Err: err}
	}
	return out, nil
}
func (r *Repo) TargetExists(remote, target string) error {
	if err := ValidateRemoteName(remote); err != nil {
		return &Error{Op: "target_exists", Err: err}
	}
	if err := ValidateRefName("target branch", target); err != nil {
		return &Error{Op: "target_exists", Err: err}
	}
	// No `--` here: in rev-parse it separates revisions from paths, so it would
	// make the ref be read as a pathname. Validation above keeps it positional.
	ref := remote + "/" + target
	_, err := r.runner().Output(
		[]string{"git", "rev-parse", "--verify", ref},
		r.opts(shell.RunOpts{Quiet: true, Check: false}),
	)
	if err != nil {
		if code, ok := shell.ExitCode(err); ok && (code == 128 || code == 1) {
			return fmt.Errorf("target branch %s does not exist on remote %s", target, remote)
		}
		return &Error{Op: "target_exists", Err: err}
	}
	return nil
}

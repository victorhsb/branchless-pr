package git

import (
	"fmt"
	"sort"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// Fetch runs git fetch --prune on the given remote.
func (r *Repo) Fetch(remote string) error {
	if err := ValidateRemoteName(remote); err != nil {
		return &Error{Op: "fetch", Err: err}
	}
	_, err := r.runner().Output([]string{"git", "fetch", "--prune", "--", remote}, r.opts(shell.RunOpts{}))
	if err != nil {
		return &Error{Op: "fetch", Err: err}
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
		_, p, ok := strings.Cut(url, ":")
		if !ok {
			return "", "", fmt.Errorf("invalid ssh remote url: %q", url)
		}
		path = p
	case strings.HasPrefix(url, "ssh://"):
		rest := strings.TrimPrefix(url, "ssh://")
		_, p, ok := strings.Cut(rest, "/")
		if !ok {
			return "", "", fmt.Errorf("invalid ssh remote url: %q", url)
		}
		path = p
	case strings.HasPrefix(url, "https://"), strings.HasPrefix(url, "http://"):
		rest := strings.TrimPrefix(url, "https://")
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

// TargetExists verifies that the named remote-tracking branch resolves to a
// Git revision.
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
		r.opts(shell.RunOpts{Quiet: true}),
	)
	if err != nil {
		if code, ok := shell.ExitCode(err); ok && (code == 128 || code == 1) {
			return fmt.Errorf("target branch %s does not exist on remote %s", target, remote)
		}
		return &Error{Op: "target_exists", Err: err}
	}
	return nil
}

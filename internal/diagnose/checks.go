package diagnose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

type Options struct {
	Remote             string
	Target             string
	Base               string
	Head               string
	BranchNameTemplate string
	Online             bool
	WorkDir            string
	Runner             Runner
}

type inspector struct {
	opts   Options
	run    Runner
	report Report
	st     stack.Stack
}

func Run(opts Options) Report {
	if opts.Runner == nil {
		opts.Runner = DefaultRunner{}
	}
	if opts.Head == "" {
		opts.Head = "HEAD"
	}
	i := &inspector{opts: opts, run: opts.Runner}
	i.report = Report{
		SchemaVersion: SchemaVersion,
		Status:        StatusUnknown,
		Repo: RepoContext{
			Remote:             opts.Remote,
			Target:             opts.Target,
			Base:               opts.Base,
			Head:               opts.Head,
			BranchNameTemplate: opts.BranchNameTemplate,
			Online:             opts.Online,
		},
	}

	i.add("git_repository", i.checkGitRepository)
	i.add("gh_installed", i.checkGHInstalled)
	i.add("github_authentication", i.checkGitHubAuth)
	i.add("github_availability", i.checkGitHubAvailability)
	i.add("working_tree_clean", i.checkWorkingTreeClean)
	i.add("rebase_in_progress", i.checkRebaseInProgress)
	i.add("base_head_resolution", i.checkBaseHeadResolution)
	i.add("target_branch_exists", i.checkTargetBranchExists)
	i.add("branch_name_template", i.checkBranchNameTemplate)
	i.add("stack_discovery", i.checkStackDiscovery)
	i.add("pr_base_coherence", i.checkPRBaseCoherence)
	i.add("local_base_behind_remote_target", i.checkLocalBaseBehindRemoteTarget)
	i.add("online_pr_state", i.checkOnlinePRState)
	i.report.finalize()
	return i.report
}

func (i *inspector) add(id string, fn func() (CheckEntry, error)) {
	entry := CheckEntry{ID: id, Status: StatusUnknown, Message: "check did not complete"}
	defer func() {
		if r := recover(); r != nil {
			entry = CheckEntry{ID: id, Status: StatusUnknown, Message: fmt.Sprintf("check panicked: %v", r)}
		}
		if entry.ID == "" {
			entry.ID = id
		}
		i.report.Checks = append(i.report.Checks, entry)
	}()

	out, err := fn()
	if err != nil {
		entry = CheckEntry{ID: id, Status: StatusUnknown, Message: err.Error()}
		return
	}
	entry = out
	entry.ID = id
}

func (i *inspector) gitOutput(args ...string) (string, error) {
	return i.run.Output(append([]string{"git"}, args...), shell.RunOpts{Dir: i.opts.WorkDir})
}

func (i *inspector) gitRun(args ...string) ([]byte, []byte, error) {
	return i.run.Run(append([]string{"git"}, args...), shell.RunOpts{Dir: i.opts.WorkDir, Quiet: true, Check: false})
}

func (i *inspector) checkGitRepository() (CheckEntry, error) {
	root, err := i.gitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		return CheckEntry{Status: StatusBlocking, Message: "current directory is not inside a Git repository", Blocks: []string{"view", "submit", "land", "abandon"}, SuggestedFix: "Change into a Git repository that contains the stack you want to inspect."}, nil
	}
	i.report.Repo.Root = root
	branch, err := i.gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		i.report.Repo.CurrentBranch = branch
	}
	return CheckEntry{Status: StatusOK, Message: "inside a Git repository"}, nil
}

func (i *inspector) checkGHInstalled() (CheckEntry, error) {
	if _, err := i.run.LookPath("gh"); err != nil {
		return CheckEntry{Status: StatusWarning, Message: "gh CLI was not found on PATH"}, nil
	}
	return CheckEntry{Status: StatusOK, Message: "gh CLI is installed"}, nil
}

func (i *inspector) checkGitHubAuth() (CheckEntry, error) {
	if !i.opts.Online {
		return CheckEntry{Status: StatusUnknown, Message: "GitHub authentication was not checked because --online was not specified"}, nil
	}
	if _, err := i.run.LookPath("gh"); err != nil {
		return CheckEntry{Status: StatusUnknown, Message: "GitHub authentication cannot be checked because gh is not installed"}, nil
	}
	_, _, err := i.run.Run([]string{"gh", "auth", "status"}, shell.RunOpts{Dir: i.opts.WorkDir, Quiet: true, Check: false})
	if err != nil {
		return CheckEntry{Status: StatusWarning, Message: fmt.Sprintf("gh authentication check failed: %v", err)}, nil
	}
	return CheckEntry{Status: StatusOK, Message: "gh authentication appears available"}, nil
}

func (i *inspector) checkGitHubAvailability() (CheckEntry, error) {
	if !i.opts.Online {
		return CheckEntry{Status: StatusUnknown, Message: "GitHub availability was not checked because --online was not specified"}, nil
	}
	if _, err := i.run.LookPath("gh"); err != nil {
		return CheckEntry{Status: StatusUnknown, Message: "GitHub availability cannot be checked because gh is not installed"}, nil
	}

	out, stderr, err := i.run.Run([]string{"gh", "api", "/rate_limit"}, shell.RunOpts{Dir: i.opts.WorkDir, Quiet: true, Check: false})
	if err == nil {
		return CheckEntry{Status: StatusOK, Message: "GitHub appears reachable via gh"}, nil
	}

	detail := strings.TrimSpace(strings.Join([]string{string(out), string(stderr), err.Error()}, " "))
	if isGitHubAuthFailure(detail) {
		return CheckEntry{Status: StatusUnknown, Message: "GitHub availability was not classified as an outage because gh reported an authentication or authorization failure"}, nil
	}
	if isLikelyGitHubOutage(detail) {
		return CheckEntry{
			Status:       StatusBlocking,
			Message:      fmt.Sprintf("GitHub appears unavailable via gh: %s", detail),
			Blocks:       []string{"submit", "land", "abandon"},
			SuggestedFix: "Wait for GitHub availability to recover, then rerun stack-pr agent diagnose --online before mutating stack state.",
		}, nil
	}
	return CheckEntry{Status: StatusUnknown, Message: fmt.Sprintf("GitHub availability could not be determined via gh: %s", detail)}, nil
}

func (i *inspector) checkWorkingTreeClean() (CheckEntry, error) {
	if !i.inGitRepo() {
		return CheckEntry{Status: StatusUnknown, Message: "working tree cleanliness cannot be checked outside a Git repository"}, nil
	}
	out, err := i.gitOutput("status", "--porcelain")
	if err != nil {
		return CheckEntry{}, err
	}
	var dirty []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 || strings.HasPrefix(line, "??") {
			continue
		}
		dirty = append(dirty, strings.TrimSpace(line))
	}
	if len(dirty) > 0 {
		return CheckEntry{Status: StatusBlocking, Message: fmt.Sprintf("working tree has %d tracked staged or unstaged change(s)", len(dirty)), Blocks: []string{"submit", "land", "abandon"}, SuggestedFix: "Commit, stash, or revert tracked changes before running mutating stack-pr commands."}, nil
	}
	return CheckEntry{Status: StatusOK, Message: "working tree has no tracked staged or unstaged changes"}, nil
}

func (i *inspector) checkRebaseInProgress() (CheckEntry, error) {
	if !i.inGitRepo() {
		return CheckEntry{Status: StatusUnknown, Message: "rebase state cannot be checked outside a Git repository"}, nil
	}
	gitDir, err := i.gitOutput("rev-parse", "--git-dir")
	if err != nil {
		return CheckEntry{}, err
	}
	if !filepath.IsAbs(gitDir) {
		base := i.opts.WorkDir
		if base == "" {
			base, _ = os.Getwd()
		}
		gitDir = filepath.Join(base, gitDir)
	}
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(gitDir, name)); err == nil {
			return CheckEntry{Status: StatusBlocking, Message: "a Git rebase is in progress", Blocks: []string{"submit", "land", "abandon"}, SuggestedFix: "Finish the rebase with git rebase --continue or abort it with git rebase --abort before running stack-pr operations."}, nil
		}
	}
	return CheckEntry{Status: StatusOK, Message: "no rebase is in progress"}, nil
}

func (i *inspector) checkBaseHeadResolution() (CheckEntry, error) {
	if !i.inGitRepo() {
		return CheckEntry{Status: StatusUnknown, Message: "base/head cannot be resolved outside a Git repository"}, nil
	}
	head := i.opts.Head
	if _, err := i.gitOutput("rev-parse", "--verify", head); err != nil {
		return CheckEntry{Status: StatusBlocking, Message: fmt.Sprintf("head revision %q could not be resolved", head), Blocks: []string{"view", "submit", "land", "abandon"}, SuggestedFix: "Pass a valid --head revision or repair the repository state."}, nil
	}
	base := i.opts.Base
	if base == "" {
		mb, err := i.gitOutput("merge-base", head, i.opts.Remote+"/"+i.opts.Target)
		if err != nil {
			return CheckEntry{Status: StatusBlocking, Message: "base revision could not be deduced from HEAD and the remote target", Blocks: []string{"view", "submit", "land", "abandon"}, SuggestedFix: "Fetch or configure the target branch, or pass an explicit --base revision."}, nil
		}
		base = mb
		i.report.Repo.Base = mb
	} else if _, err := i.gitOutput("rev-parse", "--verify", base); err != nil {
		return CheckEntry{Status: StatusBlocking, Message: fmt.Sprintf("base revision %q could not be resolved", base), Blocks: []string{"view", "submit", "land", "abandon"}, SuggestedFix: "Pass a valid --base revision."}, nil
	}
	i.report.Repo.Head = head
	return CheckEntry{Status: StatusOK, Message: "base and head revisions resolved"}, nil
}

func (i *inspector) checkTargetBranchExists() (CheckEntry, error) {
	if !i.inGitRepo() {
		return CheckEntry{Status: StatusUnknown, Message: "target branch cannot be checked outside a Git repository"}, nil
	}
	ref := i.opts.Remote + "/" + i.opts.Target
	if _, err := i.gitOutput("rev-parse", "--verify", ref); err != nil {
		return CheckEntry{Status: StatusBlocking, Message: fmt.Sprintf("target branch %s is not available locally", ref), Blocks: []string{"view", "submit", "land"}, SuggestedFix: "Ensure the configured remote target exists locally, or pass --remote/--target for the correct target."}, nil
	}
	return CheckEntry{Status: StatusOK, Message: fmt.Sprintf("target branch %s is available locally", ref)}, nil
}

func (i *inspector) checkBranchNameTemplate() (CheckEntry, error) {
	if strings.TrimSpace(i.opts.BranchNameTemplate) == "" {
		return CheckEntry{Status: StatusBlocking, Message: "branch-name template is empty", Blocks: []string{"submit"}, SuggestedFix: "Configure repo.branch_name_template or pass --branch-name-template."}, nil
	}
	tmpl := stack.ParseTemplate(i.opts.BranchNameTemplate)
	if !tmpl.HasID {
		return CheckEntry{Status: StatusBlocking, Message: "branch-name template must contain or imply $ID", Blocks: []string{"submit"}, SuggestedFix: "Use a template such as $USERNAME/stack or $USERNAME/$BRANCH/$ID."}, nil
	}
	return CheckEntry{Status: StatusOK, Message: "branch-name template is valid"}, nil
}

func (i *inspector) checkStackDiscovery() (CheckEntry, error) {
	if !i.inGitRepo() {
		return CheckEntry{Status: StatusUnknown, Message: "stack cannot be discovered outside a Git repository"}, nil
	}
	base := i.report.Repo.Base
	if base == "" {
		base = i.opts.Base
	}
	if base == "" {
		return CheckEntry{Status: StatusUnknown, Message: "stack discovery skipped because base could not be resolved"}, nil
	}
	st, err := i.discover(base, i.opts.Head)
	if err != nil {
		return CheckEntry{}, err
	}
	withPR := 0
	for _, e := range st {
		if e.ReadMetadata() && e.HasPR() {
			withPR++
		}
	}
	st.AssignBases(i.opts.Target)
	i.st = st
	i.report.Stack = StackSummary{Size: len(st), EntriesWithPR: withPR, EntriesMissingPR: len(st) - withPR}
	if len(st) == 0 {
		return CheckEntry{Status: StatusWarning, Message: "stack range contains no commits"}, nil
	}
	if withPR < len(st) {
		return CheckEntry{Status: StatusWarning, Message: fmt.Sprintf("stack has %d commit(s); %d with PR metadata and %d missing PR metadata", len(st), withPR, len(st)-withPR)}, nil
	}
	return CheckEntry{Status: StatusOK, Message: fmt.Sprintf("stack has %d commit(s), all with PR metadata", len(st))}, nil
}

func (i *inspector) checkPRBaseCoherence() (CheckEntry, error) {
	if len(i.st) == 0 {
		return CheckEntry{Status: StatusUnknown, Message: "PR base coherence cannot be checked before stack metadata is available"}, nil
	}
	withPR := 0
	for idx, e := range i.st {
		if !e.HasPR() {
			continue
		}
		withPR++
		if !e.HasHead() {
			return CheckEntry{Status: StatusBlocking, Message: fmt.Sprintf("stack entry %d has PR metadata without a head branch", idx), Blocks: []string{"submit", "land"}, SuggestedFix: "Refresh stack metadata by running stack-pr submit --dry-run and then an approved submit if appropriate."}, nil
		}
	}
	if withPR == 0 {
		return CheckEntry{Status: StatusUnknown, Message: "no PR metadata is present, so PR base coherence cannot be evaluated"}, nil
	}
	if !i.opts.Online {
		return CheckEntry{Status: StatusUnknown, Message: "live PR base coherence was not checked because --online was not specified"}, nil
	}
	// Online base relationships are evaluated with the same gh queries as online_pr_state.
	return CheckEntry{Status: StatusOK, Message: "PR metadata is locally coherent; live PR state is checked separately"}, nil
}

func (i *inspector) checkLocalBaseBehindRemoteTarget() (CheckEntry, error) {
	if !i.inGitRepo() {
		return CheckEntry{Status: StatusUnknown, Message: "base/remote relationship cannot be checked outside a Git repository"}, nil
	}
	base := i.report.Repo.Base
	if base == "" {
		return CheckEntry{Status: StatusUnknown, Message: "local base behind remote target cannot be checked because base is unresolved"}, nil
	}
	remoteTarget := i.opts.Remote + "/" + i.opts.Target
	baseHash, err := i.gitOutput("rev-parse", "--verify", base)
	if err != nil {
		return CheckEntry{}, err
	}
	targetHash, err := i.gitOutput("rev-parse", "--verify", remoteTarget)
	if err != nil {
		return CheckEntry{Status: StatusUnknown, Message: fmt.Sprintf("remote target %s could not be resolved locally", remoteTarget)}, nil
	}
	_, _, ancErr := i.gitRun("merge-base", "--is-ancestor", base, remoteTarget)
	if ancErr == nil && baseHash != targetHash {
		return CheckEntry{Status: StatusWarning, Message: fmt.Sprintf("local base %s is behind %s", base, remoteTarget)}, nil
	}
	return CheckEntry{Status: StatusOK, Message: "local base is not behind the remote target"}, nil
}

func (i *inspector) checkOnlinePRState() (CheckEntry, error) {
	if !i.opts.Online {
		return CheckEntry{Status: StatusUnknown, Message: "online PR state was not checked because --online was not specified"}, nil
	}
	if c, ok := findCheck(i.report.Checks, "github_availability"); ok && c.Status == StatusBlocking {
		return CheckEntry{Status: StatusUnknown, Message: "live PR state was not trusted because GitHub appears unavailable"}, nil
	}
	if len(i.st) == 0 {
		return CheckEntry{Status: StatusUnknown, Message: "online PR state cannot be checked before stack metadata is available"}, nil
	}
	checked := 0
	for idx, e := range i.st {
		if !e.HasPR() {
			continue
		}
		checked++
		out, err := i.run.Output([]string{"gh", "pr", "view", e.PR(), "--json", "baseRefName,headRefName,number,state,mergeStateStatus,isDraft"}, shell.RunOpts{Dir: i.opts.WorkDir})
		if err != nil {
			return CheckEntry{Status: StatusUnknown, Message: fmt.Sprintf("could not query PR state for entry %d: %v", idx, err)}, nil
		}
		var info struct {
			BaseRefName string `json:"baseRefName"`
			HeadRefName string `json:"headRefName"`
			Number      int    `json:"number"`
			State       string `json:"state"`
		}
		if err := json.Unmarshal([]byte(out), &info); err != nil {
			return CheckEntry{Status: StatusUnknown, Message: fmt.Sprintf("could not parse PR state for entry %d: %v", idx, err)}, nil
		}
		if info.State != "OPEN" {
			return CheckEntry{Status: StatusWarning, Message: fmt.Sprintf("PR #%d for entry %d is not open (state=%s)", info.Number, idx, info.State)}, nil
		}
		if info.HeadRefName != "" && e.HasHead() && info.HeadRefName != e.Head() {
			return CheckEntry{Status: StatusWarning, Message: fmt.Sprintf("PR #%d head is %q but metadata head is %q", info.Number, info.HeadRefName, e.Head())}, nil
		}
		if info.BaseRefName != "" && e.HasBase() && info.BaseRefName != e.Base() {
			return CheckEntry{Status: StatusWarning, Message: fmt.Sprintf("PR #%d base is %q but expected %q", info.Number, info.BaseRefName, e.Base())}, nil
		}
	}
	if checked == 0 {
		return CheckEntry{Status: StatusUnknown, Message: "no PR metadata is available for online PR state checks"}, nil
	}
	return CheckEntry{Status: StatusOK, Message: fmt.Sprintf("queried live state for %d PR(s)", checked)}, nil
}

func isLikelyGitHubOutage(detail string) bool {
	detail = strings.ToLower(detail)
	needles := []string{
		"500",
		"502",
		"503",
		"504",
		"bad gateway",
		"connection refused",
		"connection reset",
		"connection timed out",
		"context deadline exceeded",
		"could not resolve host",
		"gateway timeout",
		"i/o timeout",
		"internal server error",
		"network is unreachable",
		"no such host",
		"service unavailable",
		"temporarily unavailable",
		"temporary failure in name resolution",
		"timed out",
		"timeout",
		"tls handshake timeout",
	}
	for _, needle := range needles {
		if strings.Contains(detail, needle) {
			return true
		}
	}
	return false
}

func isGitHubAuthFailure(detail string) bool {
	detail = strings.ToLower(detail)
	needles := []string{
		"401",
		"403",
		"authentication",
		"authorization",
		"forbidden",
		"gh auth login",
		"not logged in",
		"oauth",
		"permission denied",
		"requires authentication",
		"unauthorized",
	}
	for _, needle := range needles {
		if strings.Contains(detail, needle) {
			return true
		}
	}
	return false
}

func (i *inspector) inGitRepo() bool {
	c, ok := findCheck(i.report.Checks, "git_repository")
	return ok && c.Status == StatusOK
}

func (i *inspector) discover(base, head string) (stack.Stack, error) {
	out, err := i.gitOutput("rev-list", "--header", "^"+base, head)
	if err != nil {
		return nil, fmt.Errorf("rev-list: %w", err)
	}
	// git rev-list --header separates commits with NUL bytes.
	var blocks []string
	for _, block := range strings.Split(out, "\x00") {
		block = strings.TrimSpace(block)
		if block != "" {
			blocks = append(blocks, strings.TrimSuffix(block, "\n"))
		}
	}
	var st stack.Stack
	for idx := len(blocks) - 1; idx >= 0; idx-- {
		h, err := stack.ParseHeader(strings.TrimSuffix(blocks[idx], "\n"))
		if err != nil {
			return nil, fmt.Errorf("parse header: %w", err)
		}
		st = append(st, &stack.Entry{Commit: h})
	}
	return st, nil
}

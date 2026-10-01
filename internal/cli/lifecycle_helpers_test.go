package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/shell"
)

// These tests exercise the complete Cobra invocation, bootstrap, recovery, and
// real Git mutations. Only the GitHub boundary is simulated: no network or
// credentials are required, and unexpected GitHub commands fail the test.
type lifecycleFixture struct {
	t                 *testing.T
	repo, remote, git string
	prs               []*pr.Info
	calls             [][]string
	fail              string
	queue             bool
	incompleteUnstack bool
	failGit           string
}

func newLifecycleFixture(t *testing.T) *lifecycleFixture {
	t.Helper()
	repo, git := setupStashLifecycleRepo(t)
	f := &lifecycleFixture{t: t, repo: repo, remote: filepath.Join(filepath.Dir(repo), "remote.git"), git: git, queue: true}
	f.gitRun(repo, "checkout", "--", "tracked.txt")
	f.write(".stack-pr.cfg", "[github]\nnative_stacks = off\n[repo]\nbranch_name_template = $USERNAME/stack/$ID\n")
	f.gitRun(repo, "add", ".stack-pr.cfg")
	f.gitRun(repo, "commit", "-m", "test configuration")
	f.gitRun(repo, "push", "origin", "main")
	f.gitRun(repo, "checkout", "-b", "feature")
	for _, name := range []string{"first", "second"} {
		f.write(name+".txt", name+" content\n")
		f.gitRun(repo, "add", name+".txt")
		f.gitRun(repo, "commit", "-m", name+" change", "-m", name+" body")
	}
	chdirForTest(t, repo)
	t.Setenv(experimentalSubmitEngineEnv, "")
	t.Setenv("STACK_PR_DEFAULT_REVIEWER", "")
	return f
}

func (f *lifecycleFixture) write(name, contents string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.repo, name), []byte(contents), 0o644); err != nil {
		f.t.Fatal(err)
	}
}
func (f *lifecycleFixture) gitRun(dir string, args ...string) string {
	f.t.Helper()
	return gitOutputForStashTest(f.t, f.git, dir, args...)
}
func (f *lifecycleFixture) invoke(args ...string) error {
	f.t.Helper()
	args = append(args, "--base", "main", "--no-hyperlinks", "--no-show-tips")
	_, err := executeRootForTest(args, f)
	return err
}
func (f *lifecycleFixture) mustInvoke(args ...string) {
	f.t.Helper()
	if err := f.invoke(args...); err != nil {
		f.t.Fatalf("%v: %v", args, err)
	}
}
func (f *lifecycleFixture) assertRestored() {
	f.t.Helper()
	if got := f.gitRun(f.repo, "branch", "--show-current"); got != "feature" {
		f.t.Fatalf("current branch = %q", got)
	}
	if got := f.gitRun(f.repo, "status", "--porcelain"); got != "" {
		f.t.Fatalf("dirty worktree: %s", got)
	}
}
func (f *lifecycleFixture) remoteRefs() string { return f.gitRun(f.remote, "show-ref", "--heads") }
func (f *lifecycleFixture) snapshot() string {
	return f.gitRun(f.repo, "show-ref", "--heads") + "\n" + f.remoteRefs()
}
func (f *lifecycleFixture) Run(args []string, opts shell.RunOpts) ([]byte, []byte, error) {
	if args[0] != "gh" {
		if f.failGit != "" && strings.HasPrefix(strings.Join(args[1:], " "), f.failGit) {
			return nil, nil, fmt.Errorf("injected Git failure: %s", f.failGit)
		}
		return (shell.Default{}).Run(args, opts)
	}
	out, err := f.github(args, opts)
	return []byte(out), nil, err
}
func (f *lifecycleFixture) Output(args []string, opts shell.RunOpts) (string, error) {
	if equalArgs(args, "git", "remote", "get-url", "--", "origin") {
		// Repository identity belongs to the simulated GitHub service; Git
		// fetches and pushes still use the real local bare remote.
		return "https://github.com/acme/widget.git", nil
	}
	if args[0] != "gh" {
		if f.failGit != "" && strings.HasPrefix(strings.Join(args[1:], " "), f.failGit) {
			return "", fmt.Errorf("injected Git failure: %s", f.failGit)
		}
		return (shell.Default{}).Output(args, opts)
	}
	return f.github(args, opts)
}
func (f *lifecycleFixture) github(args []string, opts shell.RunOpts) (string, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if len(args) == 1 {
		return "", nil
	}
	if f.fail != "" && strings.HasPrefix(strings.Join(args[1:], " "), f.fail) {
		err := fmt.Errorf("injected GitHub failure: %s", f.fail)
		if len(args) > 2 && args[1] == "api" && args[2] == "--include" {
			return `HTTP/2.0 403 Forbidden` + "\n\n" + `{"message":"injected GitHub failure"}`, err
		}
		return "", err
	}
	flag := func(name string) string {
		for i, arg := range args {
			if arg == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	encode := func(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
	if args[1] == "api" {
		if args[2] == "--include" {
			return f.nativeAPI(args, flag("--method"))
		}
		if args[2] == "graphql" {
			if strings.Contains(flag("-f"), "viewer") {
				return `{"data":{"viewer":{"login":"alice"}}}`, nil
			}
			if strings.Contains(strings.Join(args, " "), "rebaseMergeAllowed") {
				return `{"data":{"repository":{"rebaseMergeAllowed":true}}}`, nil
			}
		}
		if strings.Contains(args[2], "/rules/branches/") {
			if f.queue {
				return `[{"type":"merge_queue"}]`, nil
			}
			return `[]`, nil
		}
	}
	if len(args) < 3 || args[1] != "pr" {
		f.t.Fatalf("unexpected GitHub command: %v", args)
	}
	if args[2] == "create" {
		n := len(f.prs) + 1
		p := &pr.Info{Number: n, URL: fmt.Sprintf("https://github.com/acme/widget/pull/%d", n), BaseRefName: flag("-B"), HeadRefName: flag("-H"), Title: flag("-t"), Body: string(opts.Stdin), State: "OPEN", MergeStateStatus: "CLEAN", IsDraft: false}
		// --draft is a boolean flag, so detect presence directly.
		for _, a := range args {
			if a == "--draft" {
				p.IsDraft = true
			}
		}
		f.prs = append(f.prs, p)
		return p.URL, nil
	}
	if len(args) < 4 {
		f.t.Fatalf("missing PR reference: %v", args)
	}
	n, err := strconv.Atoi(args[3][strings.LastIndex(args[3], "/")+1:])
	if err != nil || n < 1 || n > len(f.prs) {
		f.t.Fatalf("unknown PR: %v", args)
	}
	p := f.prs[n-1]
	switch args[2] {
	case "view":
		p.HeadRefOid = f.gitRun(f.remote, "rev-parse", "refs/heads/"+p.HeadRefName)
		return encode(p)
	case "edit":
		if v := flag("-B"); v != "" {
			p.BaseRefName = v
		}
		if v := flag("-t"); v != "" {
			p.Title = v
		}
		if flag("-F") == "-" {
			p.Body = string(opts.Stdin)
		}
	case "ready":
		p.IsDraft = false
		for _, a := range args {
			if a == "--undo" {
				p.IsDraft = true
			}
		}
	case "merge":
		if flag("-t") == "" {
			return "", nil
		} // whole-stack only queues; no local merge yet
		// Model GitHub's squash merge in a separate clone of the bare remote.
		dir := filepath.Join(f.t.TempDir(), "merger")
		f.gitRun(f.repo, "clone", "--branch", "main", f.remote, dir)
		f.gitRun(dir, "config", "user.name", "Test User")
		f.gitRun(dir, "config", "user.email", "test@example.com")
		f.gitRun(dir, "merge", "--squash", "origin/"+p.HeadRefName)
		f.gitRun(dir, "commit", "-m", flag("-t"), "-m", string(opts.Stdin))
		f.gitRun(dir, "push", "origin", "main")
		p.State = "MERGED"
	default:
		f.t.Fatalf("unexpected GitHub command: %v", args)
	}
	return "", nil
}

func (f *lifecycleFixture) nativeAPI(args []string, method string) (string, error) {
	endpoint := args[5]
	response := func(status int, body string) (string, error) {
		return fmt.Sprintf("HTTP/2.0 %d Test\ncontent-type: application/json\n\n%s", status, body), nil
	}
	switch {
	case endpoint == "repos/acme/widget":
		return response(200, `{}`)
	case strings.Contains(endpoint, "stacks?per_page="):
		return response(200, `[]`)
	case strings.Contains(endpoint, "/pulls/"):
		n, err := strconv.Atoi(endpoint[strings.LastIndex(endpoint, "/")+1:])
		if err != nil || n < 1 || n > len(f.prs) {
			f.t.Fatalf("unexpected native PR endpoint: %s", endpoint)
		}
		p := f.prs[n-1]
		body := fmt.Sprintf(`{"number":%d,"state":"open","draft":false,"merged_at":null,"head":{"ref":%q,"sha":"head-sha","repo":{"full_name":"acme/widget"}},"base":{"ref":%q,"sha":"base-sha","repo":{"full_name":"acme/widget"}},"stack":{"id":7,"number":7,"size":2,"position":%d,"base":{"ref":"main","sha":"base-sha"}}}`, n, p.HeadRefName, p.BaseRefName, n)
		return response(200, body)
	case endpoint == "repos/acme/widget/stacks/7":
		return response(200, f.nativeStackBody())
	case endpoint == "repos/acme/widget/stacks/7/unstack" && method == "POST":
		if f.incompleteUnstack {
			return response(200, f.nativeStackBody())
		}
		return response(204, "")
	default:
		f.t.Fatalf("unexpected native API: %v", args)
	}
	return "", nil
}

func (f *lifecycleFixture) nativeStackBody() string {
	members := make([]string, len(f.prs))
	for i, p := range f.prs {
		members[i] = fmt.Sprintf(`{"number":%d,"state":"open","draft":false,"merged_at":null,"head":{"ref":%q,"sha":"head-sha"}}`, p.Number, p.HeadRefName)
	}
	return fmt.Sprintf(`{"id":7,"number":7,"node_id":"stack-node","url":"https://api.github.com/repos/acme/widget/stacks/7","base":{"ref":"main"},"open":true,"created_at":"2026-07-25T10:00:00Z","pull_requests":[%s]}`, strings.Join(members, ","))
}

func (f *lifecycleFixture) enableNative() {
	f.t.Helper()
	f.write(".stack-pr.cfg", "[github]\nnative_stacks = required\n[repo]\nbranch_name_template = $USERNAME/stack/$ID\n")
	f.gitRun(f.repo, "add", ".stack-pr.cfg")
	f.gitRun(f.repo, "commit", "--amend", "--no-edit")
}

func assertDirtyTreeRejectedE2E(t *testing.T, command string) {
	t.Helper()
	f := newLifecycleFixture(t)
	before := f.snapshot()
	f.write("first.txt", "unsaved work\n")
	if err := f.invoke(command); err == nil || !strings.Contains(err.Error(), "working tree is not clean") {
		t.Fatalf("error=%v", err)
	}
	if f.snapshot() != before || len(f.prs) != 0 {
		t.Fatal("dirty-tree refusal mutated refs or PRs")
	}
	data, err := os.ReadFile(filepath.Join(f.repo, "first.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unsaved work\n" {
		t.Fatal("lost unsaved work")
	}
}
func assertEmptyStackUnchangedE2E(t *testing.T, command string) {
	t.Helper()
	f := newLifecycleFixture(t)
	f.gitRun(f.repo, "reset", "--hard", "main")
	before := f.snapshot()
	f.mustInvoke(command)
	f.assertRestored()
	if f.snapshot() != before || len(f.prs) != 0 {
		t.Fatal("empty stack mutated refs or PRs")
	}
}

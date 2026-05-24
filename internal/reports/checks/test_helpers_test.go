package checks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func createCommentsTestRepo(t *testing.T) (repoDir, base, head string) {
	t.Helper()
	repoDir = t.TempDir()
	runGitCmd(t, repoDir, "init", "-b", "main")
	runGitCmd(t, repoDir, "config", "user.name", "Test User")
	runGitCmd(t, repoDir, "config", "user.email", "test@example.com")
	writeTestFile(t, repoDir, "file.txt", "base\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "base")
	base = gitOutput(t, repoDir, "rev-parse", "HEAD")

	writeTestFile(t, repoDir, "file.txt", "base\none\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "missing metadata")

	writeTestFile(t, repoDir, "file.txt", "base\none\ntwo\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "with pr", "-m", "stack-info: PR: https://github.com/acme/widgets/pull/7, branch: alice/stack/7")

	writeTestFile(t, repoDir, "file.txt", "base\none\ntwo\nthree\n")
	runGitCmd(t, repoDir, "add", "file.txt")
	runGitCmd(t, repoDir, "commit", "-m", "failed pr", "-m", "stack-info: PR: https://github.com/acme/widgets/pull/8, branch: alice/stack/8")
	head = gitOutput(t, repoDir, "rev-parse", "HEAD")

	bareRemote := filepath.Join(t.TempDir(), "remote.git")
	runGitCmd(t, "", "init", "--bare", bareRemote)
	runGitCmd(t, repoDir, "remote", "add", "origin", bareRemote)
	return repoDir, base, head
}

func commentsTestApp(repoDir, base, head string) *AppContext {
	return &AppContext{
		Args: CommonArgs{
			Base:               base,
			Head:               head,
			Remote:             "origin",
			Target:             "main",
			BranchNameTemplate: "$USERNAME/stack",
		},
		RepoRoot:   repoDir,
		Username:   "alice",
		OrigBranch: "main",
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func writeTestFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

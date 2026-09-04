package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	runGitForTest(t, repo, "config", "user.email", "test@example.com")
	return repo
}

func commitTestFile(t *testing.T, repo, name, contents string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	runGitForTest(t, repo, "add", name)
	runGitForTest(t, repo, "commit", "-m", contents)
	out, err := (shell.Default{}).Output([]string{"git", "rev-parse", "HEAD"}, shell.RunOpts{Dir: repo})
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return out
}

func runGitForTest(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := append([]string{"git"}, args...)
	if _, err := (shell.Default{}).Output(cmd, shell.RunOpts{Dir: repo}); err != nil {
		t.Fatalf("%v: %v", cmd, err)
	}
}

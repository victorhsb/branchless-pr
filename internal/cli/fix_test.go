package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
)

func TestFixCmdExposesFlags(t *testing.T) {
	cmd := fixCmd()

	if got := cmd.Use; got != "fix" {
		t.Fatalf("fix Use = %q, want fix", got)
	}

	for _, name := range []string{"pr", "replace", "dry-run"} {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("--%s flag not registered on fix command", name)
		}
		if f.Value.Type() != "bool" && f.Value.Type() != "int" {
			t.Fatalf("--%s flag type = %q, want bool or int", name, f.Value.Type())
		}
	}
}

func TestFixDryRunReportsNoAmend(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "git.log")
	gitPath := filepath.Join(binDir, "git")
	gitScript := `#!/bin/sh
printf '%s\n' "$*" >> "$GIT_LOG"
if [ "$1" = "log" ] && [ "$2" = "-1" ]; then
	echo "Hello world"
fi
if [ "$1" = "rev-parse" ] && [ "$2" = "--verify" ]; then
	echo "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
fi
if [ "$1" = "status" ] && [ "$2" = "--porcelain" ]; then
	echo ""
fi
`
	if err := os.WriteFile(gitPath, []byte(gitScript), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	t.Setenv("GIT_LOG", logPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ghPath := filepath.Join(binDir, "gh")
	ghScript := `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
	echo '{"url":"https://github.com/test/repo/pull/42","headRefName":"feature","baseRefName":"main","headRefOid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","number":42,"state":"OPEN","body":"","title":"Test PR","mergeStateStatus":"CLEAN","isDraft":false}'
	exit 0
fi
if [ "$1" = "api" ]; then
	echo '{"data":{"viewer":{"login":"testuser"}}}'
	exit 0
fi
`
	if err := os.WriteFile(ghPath, []byte(ghScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

	repoDir := t.TempDir()
	if err := runGitForTest(repoDir, "init"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	chdirForTest(t, repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "add", "file.txt"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}

	// We need origin/main to exist for stack.Discover in advisory check
	if err := runGitForTest(repoDir, "remote", "add", "origin", "/dev/null"); err != nil {
		t.Fatal(err)
	}
	headSHA, _ := git.RevParse("HEAD")
	if err := runGitForTest(repoDir, "update-ref", "refs/remotes/origin/main", headSHA); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		app := &AppContext{
			Config: config.Defaults(),
			Args: CommonArgs{
				Base:   headSHA,
				Head:   "HEAD",
				Remote: "origin",
				Target: "main",
			},
			RepoRoot:   repoDir,
			Username:   "testuser",
			OrigBranch: "main",
		}
		err := fixImpl(app, fixOptions{PRNumber: 42, DryRun: true})
		if err != nil {
			t.Fatalf("fixImpl returned error: %v", err)
		}
	})

	if !strings.Contains(out, "PR URL:") {
		t.Fatalf("dry-run output missing PR URL, got:\n%s", out)
	}
	if !strings.Contains(out, "No commit was changed") {
		t.Fatalf("dry-run output missing 'No commit was changed', got:\n%s", out)
	}

	// Verify no amend was attempted in fake git log
	log := readTestFile(t, logPath)
	if strings.Contains(log, "commit --amend") {
		t.Fatalf("dry-run should not amend; git log:\n%s", log)
	}
}

func TestFixAlreadyFixedReportsNoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	repoDir := t.TempDir()
	if err := runGitForTest(repoDir, "init"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	chdirForTest(t, repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "add", "file.txt"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "commit", "-m", "initial\n\nstack-info: PR: https://github.com/test/repo/pull/42, branch: feature"); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	ghScript := `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
	echo '{"url":"https://github.com/test/repo/pull/42","headRefName":"feature","baseRefName":"main","headRefOid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","number":42,"state":"OPEN","body":"","title":"Test PR","mergeStateStatus":"CLEAN","isDraft":false}'
	exit 0
fi
if [ "$1" = "api" ]; then
	echo '{"data":{"viewer":{"login":"testuser"}}}'
	exit 0
fi
`
	if err := os.WriteFile(ghPath, []byte(ghScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Set up origin/main
	if err := runGitForTest(repoDir, "remote", "add", "origin", "/dev/null"); err != nil {
		t.Fatal(err)
	}
	headSHA, _ := git.RevParse("HEAD")
	// update-ref requires a real git remote ref
	if err := runGitForTest(repoDir, "update-ref", "refs/remotes/origin/main", headSHA); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		app := &AppContext{
			Config: config.Defaults(),
			Args: CommonArgs{
				Base:   headSHA,
				Head:   "HEAD",
				Remote: "origin",
				Target: "main",
			},
			RepoRoot:   repoDir,
			Username:   "testuser",
			OrigBranch: "main",
		}
		err := fixImpl(app, fixOptions{PRNumber: 42})
		if err != nil {
			t.Fatalf("fixImpl returned error: %v", err)
		}
	})

	if !strings.Contains(out, "already fixed") {
		t.Fatalf("expected 'already fixed' in output, got:\n%s", out)
	}
}

func TestFixRefusesDifferentMetadataWithoutReplace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	repoDir := t.TempDir()
	if err := runGitForTest(repoDir, "init"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	chdirForTest(t, repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "add", "file.txt"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "commit", "-m", "initial\n\nstack-info: PR: https://github.com/test/repo/pull/1, branch: old-branch"); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	ghScript := `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
	echo '{"url":"https://github.com/test/repo/pull/42","headRefName":"feature","baseRefName":"main","headRefOid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","number":42,"state":"OPEN","body":"","title":"Test PR","mergeStateStatus":"CLEAN","isDraft":false}'
	exit 0
fi
if [ "$1" = "api" ]; then
	echo '{"data":{"viewer":{"login":"testuser"}}}'
	exit 0
fi
`
	if err := os.WriteFile(ghPath, []byte(ghScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := runGitForTest(repoDir, "remote", "add", "origin", "/dev/null"); err != nil {
		t.Fatal(err)
	}
	headSHA, _ := git.RevParse("HEAD")
	if err := runGitForTest(repoDir, "update-ref", "refs/remotes/origin/main", headSHA); err != nil {
		t.Fatal(err)
	}

	err := fixImpl(&AppContext{
		Config: config.Defaults(),
		Args: CommonArgs{
			Base:   headSHA,
			Head:   "HEAD",
			Remote: "origin",
			Target: "main",
		},
		RepoRoot:   repoDir,
		Username:   "testuser",
		OrigBranch: "main",
	}, fixOptions{PRNumber: 42})
	if err == nil {
		t.Fatal("expected error for different metadata without --replace")
	}
	if !strings.Contains(err.Error(), "already has different stack metadata") {
		t.Fatalf("expected different metadata error, got: %v", err)
	}
}

func TestFixReplaceOverwritesDifferentMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh/git is Unix-only")
	}
	repoDir := t.TempDir()
	if err := runGitForTest(repoDir, "init"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	chdirForTest(t, repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "add", "file.txt"); err != nil {
		t.Fatal(err)
	}
	if err := runGitForTest(repoDir, "commit", "-m", "initial\n\nstack-info: PR: https://github.com/test/repo/pull/1, branch: old-branch"); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "git.log")
	// Fake git that logs amends
	gitScript := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> "%s"
if [ "$1" = "commit" ] && [ "$2" = "--amend" ]; then
	exit 0
fi
exec /usr/bin/git "$@"
`, logPath)
	gitPath := filepath.Join(binDir, "git")
	if err := os.WriteFile(gitPath, []byte(gitScript), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	t.Setenv("GIT_LOG", logPath)

	ghPath := filepath.Join(binDir, "gh")
	ghScript := `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
	echo '{"url":"https://github.com/test/repo/pull/42","headRefName":"feature","baseRefName":"main","headRefOid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","number":42,"state":"OPEN","body":"","title":"Test PR","mergeStateStatus":"CLEAN","isDraft":false}'
	exit 0
fi
if [ "$1" = "api" ]; then
	echo '{"data":{"viewer":{"login":"testuser"}}}'
	exit 0
fi
`
	if err := os.WriteFile(ghPath, []byte(ghScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := runGitForTest(repoDir, "remote", "add", "origin", "/dev/null"); err != nil {
		t.Fatal(err)
	}
	headSHA, _ := git.RevParse("HEAD")
	if err := runGitForTest(repoDir, "update-ref", "refs/remotes/origin/main", headSHA); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		app := &AppContext{
			Config: config.Defaults(),
			Args: CommonArgs{
				Base:   headSHA,
				Head:   "HEAD",
				Remote: "origin",
				Target: "main",
			},
			RepoRoot:   repoDir,
			Username:   "testuser",
			OrigBranch: "main",
		}
		err := fixImpl(app, fixOptions{PRNumber: 42, Replace: true})
		if err != nil {
			t.Fatalf("fixImpl returned error: %v", err)
		}
	})

	if !strings.Contains(out, "Fixed stack metadata") {
		t.Fatalf("expected success output, got:\n%s", out)
	}

	log := readTestFile(t, logPath)
	if !strings.Contains(log, "commit --amend") {
		t.Fatalf("expected amend in git log:\n%s", log)
	}
}

func TestBuildFixedMessageAppendsMetadata(t *testing.T) {
	msg := buildFixedMessage("Hello world\n", "https://github.com/test/repo/pull/42", "feature")
	want := "Hello world\n\nstack-info: PR: https://github.com/test/repo/pull/42, branch: feature\n"
	if msg != want {
		t.Fatalf("buildFixedMessage = %q, want %q", msg, want)
	}
}

func TestBuildFixedMessageReplacesExisting(t *testing.T) {
	msg := buildFixedMessage("Hello world\n\nstack-info: PR: https://old, branch: old\n", "https://github.com/test/repo/pull/42", "feature")
	want := "Hello world\n\nstack-info: PR: https://github.com/test/repo/pull/42, branch: feature\n"
	if msg != want {
		t.Fatalf("buildFixedMessage = %q, want %q", msg, want)
	}
}

func TestPluralize(t *testing.T) {
	if pluralize(1, "y is", "ies are") != "y is" {
		t.Fatal("pluralize(1) != y is")
	}
	if pluralize(2, "y is", "ies are") != "ies are" {
		t.Fatal("pluralize(2) != entries are")
	}
}

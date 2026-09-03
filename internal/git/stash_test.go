package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

func TestStashSaveCleanTreeDoesNotClaimExistingUserStash(t *testing.T) {
	repo := initTestRepo(t)
	commitTestFile(t, repo, "tracked.txt", "original")
	gitRepo := New(repo, shell.Default{})

	writeStashTestFile(t, repo, "tracked.txt", "user stash")
	runGitForTest(t, repo, "stash", "push", "-m", "user stash")
	before := stashOIDsForTest(t, repo)

	ref, err := gitRepo.StashSave("automatic")
	if err != nil {
		t.Fatalf("StashSave: %v", err)
	}
	if !ref.IsZero() {
		t.Fatalf("clean tree returned stash identity %q", ref.OID)
	}
	if after := stashOIDsForTest(t, repo); !equalStrings(before, after) {
		t.Fatalf("user stash changed: before=%v after=%v", before, after)
	}
}

func TestStashSaveIgnoresUnexpectedHumanOutput(t *testing.T) {
	repo := initTestRepo(t)
	commitTestFile(t, repo, "tracked.txt", "original")
	writeStashTestFile(t, repo, "tracked.txt", "automatic changes")
	gitRepo := New(repo, noisyStashRunner{})

	ref, err := gitRepo.StashSave("automatic")
	if err != nil {
		t.Fatalf("StashSave: %v", err)
	}
	if ref.IsZero() {
		t.Fatal("unexpected human output caused a created stash to be missed")
	}
	if got := stashOIDsForTest(t, repo); len(got) != 1 || got[0] != ref.OID {
		t.Fatalf("stash list = %v, want exact automatic stash %s", got, ref.OID)
	}
}

func TestStashRestorePreservesOlderAndNewerUserStashes(t *testing.T) {
	repo := initTestRepo(t)
	commitTestFile(t, repo, "older.txt", "original older")
	commitTestFile(t, repo, "automatic.txt", "original automatic")
	commitTestFile(t, repo, "newer.txt", "original newer")
	gitRepo := New(repo, shell.Default{})

	writeStashTestFile(t, repo, "older.txt", "older user changes")
	runGitForTest(t, repo, "stash", "push", "-m", "older user stash")
	olderOID := stashOIDsForTest(t, repo)[0]

	writeStashTestFile(t, repo, "automatic.txt", "automatic changes")
	automatic, err := gitRepo.StashSave("automatic")
	if err != nil {
		t.Fatalf("StashSave: %v", err)
	}

	writeStashTestFile(t, repo, "newer.txt", "newer user changes")
	runGitForTest(t, repo, "stash", "push", "-m", "newer user stash")
	newerOID := stashOIDsForTest(t, repo)[0]

	if err := gitRepo.StashRestore(automatic); err != nil {
		t.Fatalf("StashRestore: %v", err)
	}
	if got := readStashTestFile(t, repo, "automatic.txt"); got != "automatic changes" {
		t.Fatalf("automatic changes = %q, want restored contents", got)
	}
	gotOIDs := stashOIDsForTest(t, repo)
	if len(gotOIDs) != 2 || gotOIDs[0] != newerOID || gotOIDs[1] != olderOID {
		t.Fatalf("remaining stashes = %v, want newer %s then older %s", gotOIDs, newerOID, olderOID)
	}
}

func TestStashRestoreConflictKeepsExactStash(t *testing.T) {
	repo := initTestRepo(t)
	commitTestFile(t, repo, "tracked.txt", "base")
	gitRepo := New(repo, shell.Default{})

	writeStashTestFile(t, repo, "tracked.txt", "automatic changes")
	automatic, err := gitRepo.StashSave("automatic")
	if err != nil {
		t.Fatalf("StashSave: %v", err)
	}
	writeStashTestFile(t, repo, "tracked.txt", "conflicting committed changes")
	runGitForTest(t, repo, "add", "tracked.txt")
	runGitForTest(t, repo, "commit", "-m", "conflict")

	err = gitRepo.StashRestore(automatic)
	if err == nil {
		t.Fatal("expected stash application conflict")
	}
	for _, want := range []string{"stash_apply", automatic.OID, "kept for manual recovery"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
	if got := stashOIDsForTest(t, repo); len(got) != 1 || got[0] != automatic.OID {
		t.Fatalf("conflicting automatic stash was not preserved: %v", got)
	}
}

func TestStashRestoreMissingEntryDoesNotChangeUserStashes(t *testing.T) {
	repo := initTestRepo(t)
	commitTestFile(t, repo, "tracked.txt", "base")
	gitRepo := New(repo, shell.Default{})
	writeStashTestFile(t, repo, "tracked.txt", "user changes")
	runGitForTest(t, repo, "stash", "push", "-m", "user stash")
	before := stashOIDsForTest(t, repo)

	missing := StashRef{OID: strings.Repeat("a", 40)}
	err := gitRepo.StashRestore(missing)
	if err == nil || !strings.Contains(err.Error(), "no longer present") {
		t.Fatalf("error = %v, want actionable missing-stash error", err)
	}
	if after := stashOIDsForTest(t, repo); !equalStrings(before, after) {
		t.Fatalf("unrelated stash changed: before=%v after=%v", before, after)
	}
}

func stashOIDsForTest(t *testing.T, repo string) []string {
	t.Helper()
	out, err := shell.Output([]string{"git", "stash", "list", "--format=%H"}, shell.RunOpts{Dir: repo})
	if err != nil {
		t.Fatalf("list stashes: %v", err)
	}
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func writeStashTestFile(t *testing.T, repo, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readStashTestFile(t *testing.T, repo, name string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(repo, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}

type noisyStashRunner struct {
	shell.Default
}

func (r noisyStashRunner) Run(args []string, opts shell.RunOpts) ([]byte, []byte, error) {
	stdout, stderr, err := r.Default.Run(args, opts)
	if err == nil && len(args) >= 3 && args[0] == "git" && args[1] == "stash" && args[2] == "push" {
		stdout = []byte("sortie localisee inattendue: aucune phrase stable\n")
	}
	return stdout, stderr, err
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

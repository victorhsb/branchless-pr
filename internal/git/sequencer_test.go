package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

func TestGitOperationDetectionRecognizesMarkers(t *testing.T) {
	tests := []struct {
		name      string
		marker    string
		directory bool
		detect    func(*Repo) bool
	}{
		{name: "rebase merge", marker: "rebase-merge", directory: true, detect: func(r *Repo) bool { return r.IsRebaseInProgress() }},
		{name: "rebase apply", marker: "rebase-apply", directory: true, detect: func(r *Repo) bool { return r.IsRebaseInProgress() }},
		{name: "merge", marker: "MERGE_HEAD", detect: func(r *Repo) bool { return r.IsMergeInProgress() }},
		{name: "cherry-pick", marker: "CHERRY_PICK_HEAD", detect: func(r *Repo) bool { return r.IsCherryPickInProgress() }},
		{name: "sequencer", marker: "sequencer/todo", detect: func(r *Repo) bool { return r.IsCherryPickInProgress() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := initTestRepo(t)
			writeOperationMarker(t, repo, tt.marker, tt.directory)
			gitRepo := New(repo, nil)

			if !tt.detect(gitRepo) {
				t.Fatalf("%s was not detected in %s", tt.marker, repo)
			}
			if !gitRepo.AnySequencerInProgress() {
				t.Fatalf("aggregate operation detection missed %s", tt.marker)
			}
		})
	}
}

func TestAnySequencerInProgressReturnsFalseWithoutMarkers(t *testing.T) {
	repo := initTestRepo(t)
	if New(repo, nil).AnySequencerInProgress() {
		t.Fatal("operation reported active in repository without operation markers")
	}
}

func TestGitOperationDetectionSupportsRepositoryLayouts(t *testing.T) {
	t.Run("repository subdirectory", func(t *testing.T) {
		repo := initTestRepo(t)
		subdir := filepath.Join(repo, "nested", "directory")
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeOperationMarker(t, repo, "rebase-merge", true)
		if !New(subdir, nil).IsRebaseInProgress() {
			t.Fatal("rebase was not detected from a repository subdirectory")
		}
	})

	t.Run("linked worktree", func(t *testing.T) {
		repo := initTestRepo(t)
		commitTestFile(t, repo, "initial.txt", "initial")
		linked := filepath.Join(t.TempDir(), "linked")
		runGitForTest(t, repo, "worktree", "add", "-b", "linked", linked)
		writeOperationMarker(t, linked, "MERGE_HEAD", false)

		if !New(linked, nil).IsMergeInProgress() {
			t.Fatal("merge was not detected in a linked worktree")
		}
	})

	t.Run("submodule", func(t *testing.T) {
		source := initTestRepo(t)
		commitTestFile(t, source, "initial.txt", "initial")
		super := initTestRepo(t)
		child := filepath.Join(super, "modules", "child")
		runGitForTest(t, super, "-c", "protocol.file.allow=always", "submodule", "add", source, "modules/child")
		writeOperationMarker(t, child, "CHERRY_PICK_HEAD", false)

		if !New(child, nil).IsCherryPickInProgress() {
			t.Fatal("cherry-pick was not detected in a submodule")
		}
	})

	t.Run("separate git directory", func(t *testing.T) {
		parent := t.TempDir()
		worktree := filepath.Join(parent, "worktree")
		gitDir := filepath.Join(parent, "metadata")
		runGitForTest(t, parent, "init", "-b", "main", "--separate-git-dir", gitDir, worktree)
		writeOperationMarker(t, worktree, "sequencer/todo", false)

		if !New(worktree, nil).IsCherryPickInProgress() {
			t.Fatal("sequencer was not detected with a separate Git directory")
		}
	})
}

func writeOperationMarker(t *testing.T, repo, marker string, directory bool) {
	t.Helper()
	path, err := (shell.Default{}).Output(
		[]string{"git", "rev-parse", "--git-path", marker},
		shell.RunOpts{Dir: repo},
	)
	if err != nil {
		t.Fatalf("resolve Git path %s: %v", marker, err)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(repo, path)
	}
	if directory {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create operation directory %s: %v", path, err)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create operation marker parent: %v", err)
	}
	if err := os.WriteFile(path, []byte("test operation marker\n"), 0o644); err != nil {
		t.Fatalf("write operation marker %s: %v", path, err)
	}
}

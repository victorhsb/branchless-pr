package pr

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCreateCapturesPRURL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
cat >/dev/null
printf 'https://github.com/acme/repo/pull/123\n'
`
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := Create(CreateOptions{
		Base:  "main",
		Head:  "alice/stack/1",
		Title: "Test PR",
		Body:  []byte("body"),
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	want := "https://github.com/acme/repo/pull/123"
	if got != want {
		t.Fatalf("Create URL = %q, want %q", got, want)
	}
}

func TestLoadForSubmitFetchesEachPR(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "gh.log")
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$GH_LOG"
printf '{"baseRefName":"main","headRefName":"alice/stack/1","number":42,"state":"OPEN","body":"body","title":"Title","url":"https://github.com/acme/repo/pull/42","mergeStateStatus":"CLEAN","isDraft":false}\n'
`
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("GH_LOG", logPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	infos, err := LoadForSubmit([]string{"1", "2"})
	if err != nil {
		t.Fatalf("LoadForSubmit returned error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("infos len = %d, want 2", len(infos))
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if got := strings.Count(string(log), "pr view"); got != 2 {
		t.Fatalf("pr view calls = %d, want 2; log=%s", got, log)
	}
}

func TestParseCreateOutputFindsLastPullURL(t *testing.T) {
	out := []byte("Creating pull request\nhttps://github.com/acme/repo/pull/122\n✓ Created https://github.com/acme/repo/pull/123\nLearn more at https://docs.github.com/\n")
	got, err := parseCreateOutput(out)
	if err != nil {
		t.Fatalf("parseCreateOutput returned error: %v", err)
	}
	want := "https://github.com/acme/repo/pull/123"
	if got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}

func TestParseCreateOutputRejectsEmptyOutput(t *testing.T) {
	_, err := parseCreateOutput([]byte("\n\t "))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unexpected empty output") {
		t.Fatalf("error = %q, want unexpected empty output", err.Error())
	}
}

func TestViewManyFetchesEachPRRef(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "gh.log")
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
echo "$@" >> ` + logPath + `
case "$3" in
  41)
    printf '{"baseRefName":"main","headRefName":"alice/stack/1","number":41,"state":"OPEN","body":"body 41","title":"Title 41","url":"https://github.com/acme/repo/pull/41","mergeStateStatus":"CLEAN","isDraft":false}\n'
    ;;
  42)
    printf '{"baseRefName":"alice/stack/1","headRefName":"alice/stack/2","number":42,"state":"OPEN","body":"body 42","title":"Title 42","url":"https://github.com/acme/repo/pull/42","mergeStateStatus":"CLEAN","isDraft":true}\n'
    ;;
  *)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := ViewMany([]string{"41", "42"})
	if err != nil {
		t.Fatalf("ViewMany returned error: %v", err)
	}
	if got["41"].Title != "Title 41" || got["42"].Title != "Title 42" || !got["42"].IsDraft {
		t.Fatalf("ViewMany result = %+v", got)
	}
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if count := strings.Count(string(logBytes), "pr view"); count != 2 {
		t.Fatalf("gh view calls = %d, want 2\n%s", count, string(logBytes))
	}
}

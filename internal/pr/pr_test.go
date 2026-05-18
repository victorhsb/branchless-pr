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

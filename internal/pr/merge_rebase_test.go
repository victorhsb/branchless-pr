package pr

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMergeRebaseInvokesGh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "gh.log")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$GH_LOG"
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("GH_LOG", logPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := MergeRebase("https://github.com/acme/repo/pull/42"); err != nil {
		t.Fatalf("MergeRebase returned error: %v", err)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	got := strings.TrimSpace(string(log))
	want := "pr merge https://github.com/acme/repo/pull/42 --rebase"
	if got != want {
		t.Fatalf("gh invocation = %q, want %q", got, want)
	}
}

func TestRebaseMergeAllowedReturnsTrue(t *testing.T) {
	got, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		if !strings.Contains(query, "rebaseMergeAllowed") {
			t.Fatalf("query missing rebaseMergeAllowed: %q", query)
		}
		if fields["owner"] != "acme" || fields["repo"] != "widget" {
			t.Fatalf("fields = %v, want acme/widget", fields)
		}
		return []byte(`{"data":{"repository":{"rebaseMergeAllowed":true}}}`), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatalf("expected true, got false")
	}
}

func TestRebaseMergeAllowedReturnsFalse(t *testing.T) {
	got, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return []byte(`{"data":{"repository":{"rebaseMergeAllowed":false}}}`), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("expected false, got true")
	}
}

func TestRebaseMergeAllowedPropagatesAPIError(t *testing.T) {
	want := errors.New("boom")
	_, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return nil, want
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error containing %q, got %v", "boom", err)
	}
}

func TestRebaseMergeAllowedSurfacesGraphQLErrors(t *testing.T) {
	_, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return []byte(`{"errors":[{"message":"Could not resolve to a Repository"}]}`), nil
	})
	if err == nil || !strings.Contains(err.Error(), "Could not resolve") {
		t.Fatalf("expected graphql error, got %v", err)
	}
}

func TestRebaseMergeAllowedSurfacesParseErrors(t *testing.T) {
	_, err := rebaseMergeAllowedWith("acme", "widget", func(query string, fields map[string]string) ([]byte, error) {
		return []byte(`{not-json}`), nil
	})
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

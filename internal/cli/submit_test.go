package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

// TestSubmitCmdExposesDryRunFlag verifies that the --dry-run flag is wired up
// on the submit command and therefore also reachable through its `export`
// alias (Cobra aliases share the same flag set).
func TestSubmitCmdExposesDryRunFlag(t *testing.T) {
	cmd := submitCmd()

	if got := cmd.Use; got != "submit" {
		t.Fatalf("submit Use = %q, want submit", got)
	}
	foundExport := false
	for _, a := range cmd.Aliases {
		if a == "export" {
			foundExport = true
			break
		}
	}
	if !foundExport {
		t.Fatalf("submit command missing export alias, got aliases %v", cmd.Aliases)
	}

	f := cmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Fatalf("--dry-run flag not registered on submit command")
	}
	if f.Value.Type() != "bool" {
		t.Fatalf("--dry-run flag type = %q, want bool", f.Value.Type())
	}
	if f.DefValue != "false" {
		t.Fatalf("--dry-run default = %q, want false", f.DefValue)
	}
	if !strings.Contains(strings.ToLower(f.Usage), "preview") {
		t.Fatalf("--dry-run usage should describe preview behavior, got %q", f.Usage)
	}
}

func TestResolveDraftFlagsAllDraft(t *testing.T) {
	got, ok := resolveDraftFlags(true, "", 3)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	for i, v := range got {
		if !v {
			t.Errorf("index %d: draft = false, want true", i)
		}
	}
}

func TestResolveDraftFlagsBitmask(t *testing.T) {
	got, ok := resolveDraftFlags(false, "101", 3)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	want := []bool{true, false, true}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("index %d: got %v, want %v", i, got[i], v)
		}
	}
}

func TestResolveDraftFlagsRejectsLengthMismatch(t *testing.T) {
	got := captureStderr(t, func() {
		if _, ok := resolveDraftFlags(false, "11", 3); ok {
			t.Errorf("expected ok=false for length mismatch")
		}
	})
	if !strings.Contains(got, "draft bitmask length") {
		t.Errorf("stderr missing length error, got %q", got)
	}
}

func TestResolveDraftFlagsRejectsInvalidChar(t *testing.T) {
	got := captureStderr(t, func() {
		if _, ok := resolveDraftFlags(false, "1x0", 3); ok {
			t.Errorf("expected ok=false for invalid char")
		}
	})
	if !strings.Contains(got, "draft bitmask must contain only 0 or 1") {
		t.Errorf("stderr missing invalid-char error, got %q", got)
	}
}

// TestPrintDryRunPlanRendersAllRequiredFields verifies the plan output covers
// every per-entry field required by the spec for a non-empty stack:
// commit title, generated head, computed base, create/update action,
// existing PR URL when present, draft state for new PRs, and a metadata-add
// indication. The closing no-changes note must always be printed.
func TestPrintDryRunPlanRendersAllRequiredFields(t *testing.T) {
	e1 := &stack.Entry{Commit: &stack.Header{
		SHA:   "0123456789abcdef0123456789abcdef01234567",
		Title: "First commit",
	}}
	e1.SetHead("alice/stack/1")
	e1.SetBase("main")

	e2 := &stack.Entry{Commit: &stack.Header{
		SHA:   "abcdefabcdefabcdefabcdefabcdefabcdef0123",
		Title: "Second commit",
	}}
	e2.SetHead("alice/stack/2")
	e2.SetBase("alice/stack/1")
	e2.SetPR("https://github.com/foo/bar/pull/42")

	st := stack.Stack{e1, e2}
	needsMeta := []bool{true, false}
	isDraft := []bool{true, false}

	out := captureStdout(t, func() {
		printDryRunPlan(st, needsMeta, isDraft)
	})

	mustContain(t, out, "create PR")
	mustContain(t, out, "update PR")
	mustContain(t, out, "First commit")
	mustContain(t, out, "Second commit")
	mustContain(t, out, "alice/stack/1")
	mustContain(t, out, "alice/stack/2")
	mustContain(t, out, "main")
	mustContain(t, out, "https://github.com/foo/bar/pull/42")
	mustContain(t, out, "draft")
	mustContain(t, out, "stack-info commit metadata")
	mustContain(t, out, dryRunNoChangesNote)
}

// TestPrintDryRunPlanMarksReadyForNonDraftNewPR ensures the per-entry draft
// state is rendered as "ready" when the entry is new and not draft.
func TestPrintDryRunPlanMarksReadyForNonDraftNewPR(t *testing.T) {
	e := &stack.Entry{Commit: &stack.Header{
		SHA:   "0123456789abcdef0123456789abcdef01234567",
		Title: "Only commit",
	}}
	e.SetHead("alice/stack/1")
	e.SetBase("main")

	st := stack.Stack{e}
	out := captureStdout(t, func() {
		printDryRunPlan(st, []bool{false}, []bool{false})
	})
	mustContain(t, out, "ready")
}

// TestDryRunNoChangesNoteWording locks in the precise wording so changes are
// reviewed deliberately — the spec requires this note explicitly.
func TestDryRunNoChangesNoteWording(t *testing.T) {
	want := "No local Git changes, remote pushes, or GitHub PR changes were made."
	if dryRunNoChangesNote != want {
		t.Fatalf("dryRunNoChangesNote = %q, want %q", dryRunNoChangesNote, want)
	}
}

func TestUseExperimentalSubmitEngineSelection(t *testing.T) {
	t.Setenv("STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE", "")
	app := &AppContext{Config: config.Defaults()}
	if useExperimentalSubmitEngine(app) {
		t.Fatalf("default engine selection = experimental, want legacy")
	}

	t.Setenv("STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE", "1")
	if !useExperimentalSubmitEngine(app) {
		t.Fatalf("env opt-in did not select experimental engine")
	}

	t.Setenv("STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE", "")
	cfg := config.Defaults()
	cfg.Set("submit", "experimental_engine", "true")
	app.Config = cfg
	if !useExperimentalSubmitEngine(app) {
		t.Fatalf("config opt-in did not select experimental engine")
	}
}

func TestTempDraftAndResetBasesOptimizedSkipsNoOps(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	logPath := installFakeGHLogger(t)

	e := &stack.Entry{}
	e.SetPR("https://github.com/acme/repo/pull/1")
	cache := &submitPRStateCache{infos: map[string]*pr.Info{
		e.PR(): {BaseRefName: "main", IsDraft: true},
	}}

	tmp, err := tempDraftAndResetBasesOptimized(stack.Stack{e}, "main", cache)
	if err != nil {
		t.Fatalf("tempDraftAndResetBasesOptimized returned error: %v", err)
	}
	if len(tmp) != 0 {
		t.Fatalf("tmp draft PRs = %v, want none", tmp)
	}
	if got := readTestFile(t, logPath); got != "" {
		t.Fatalf("gh commands = %q, want none", got)
	}
}

func TestTempDraftAndResetBasesOptimizedMutatesOnlyWhenNeeded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is Unix-only")
	}
	logPath := installFakeGHLogger(t)

	e := &stack.Entry{}
	e.SetPR("https://github.com/acme/repo/pull/2")
	cache := &submitPRStateCache{infos: map[string]*pr.Info{
		e.PR(): {BaseRefName: "feature", IsDraft: false},
	}}

	tmp, err := tempDraftAndResetBasesOptimized(stack.Stack{e}, "main", cache)
	if err != nil {
		t.Fatalf("tempDraftAndResetBasesOptimized returned error: %v", err)
	}
	if len(tmp) != 1 || tmp[0] != e.PR() {
		t.Fatalf("tmp draft PRs = %v, want %s", tmp, e.PR())
	}
	if !e.IsTmpDraft {
		t.Fatalf("entry was not marked temporary draft")
	}
	info := cache.infos[e.PR()]
	if !info.IsDraft || info.BaseRefName != "main" {
		t.Fatalf("cached info = %+v, want draft with base main", info)
	}
	log := readTestFile(t, logPath)
	mustContain(t, log, "pr ready https://github.com/acme/repo/pull/2 --undo")
	mustContain(t, log, "pr edit https://github.com/acme/repo/pull/2 -B main")
}

func TestSubmitPREditNeededComparesTitleBodyAndBase(t *testing.T) {
	info := &pr.Info{Title: "Title", Body: "body", BaseRefName: "main"}
	if submitPREditNeeded(info, "Title", "main", []byte("body")) {
		t.Fatalf("unchanged title/body/base should not need edit")
	}
	for _, tt := range []struct {
		name  string
		title string
		base  string
		body  []byte
	}{
		{"title", "New", "main", []byte("body")},
		{"base", "Title", "develop", []byte("body")},
		{"body", "Title", "main", []byte("new body")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !submitPREditNeeded(info, tt.title, tt.base, tt.body) {
				t.Fatalf("changed %s should need edit", tt.name)
			}
		})
	}
}

func TestSubmitPRStateCacheReusesCachedBody(t *testing.T) {
	cache := &submitPRStateCache{infos: map[string]*pr.Info{
		"https://github.com/acme/repo/pull/3": {Body: "existing body"},
	}}
	info, err := cache.get("https://github.com/acme/repo/pull/3")
	if err != nil {
		t.Fatalf("cache.get returned error: %v", err)
	}
	if info.Body != "existing body" {
		t.Fatalf("body = %q, want cached body", info.Body)
	}
}

func TestAmendCommitMetadataChangedReportsNoMetadataChanges(t *testing.T) {
	changed, err := amendCommitMetadataChanged(stack.Stack{&stack.Entry{}}, []bool{false})
	if err != nil {
		t.Fatalf("amendCommitMetadataChanged returned error: %v", err)
	}
	if changed {
		t.Fatalf("changed = true, want false")
	}
}

// --- helpers ---

func installFakeGHLogger(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "gh.log")
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$GH_LOG"
`
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("GH_LOG", logPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("output missing %q:\n%s", needle, haystack)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stdout = orig
	<-done
	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stderr = orig
	<-done
	return buf.String()
}

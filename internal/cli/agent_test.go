package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	prompt "github.com/victorhsb/branchless-pr/internal/agent"
)

func TestUserFacingCommandsHaveAgentRegistryEntries(t *testing.T) {
	cmd, err := newRootCommand("stack-pr", []string{"agent", "prompt"})
	if err != nil {
		t.Fatal(err)
	}

	allowedExclusions := map[string]string{
		"agent":  "agent-facing artifact group, not a stack operation",
		"config": "configuration writer intentionally excluded from stack-operation prompt guidance",
	}
	for _, child := range cmd.Commands() {
		name := child.Name()
		if _, ok := allowedExclusions[name]; ok {
			continue
		}
		if _, ok := prompt.Commands[name]; !ok {
			t.Fatalf("command %q missing from agent registry", name)
		}
		if name == "submit" {
			if _, ok := prompt.Commands["submit --dry-run"]; !ok {
				t.Fatalf("submit --dry-run missing from agent registry")
			}
		}
	}
}

func TestAgentPromptDefaultEmitsAllText(t *testing.T) {
	out, err := executeRootForTest([]string{"agent", "prompt"})
	if err != nil {
		t.Fatal(err)
	}
	for _, heading := range []string{"Overview", "View", "Submit", "Land", "Abandon", "Fix", "Recovery"} {
		if !strings.Contains(out, "# stack-pr agent prompt: "+heading) {
			t.Fatalf("output missing %s heading:\n%s", heading, out)
		}
	}
}

func TestAgentPromptSubmitScopedMarkdown(t *testing.T) {
	out, err := executeRootForTest([]string{"agent", "prompt", "submit"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# stack-pr agent prompt: Submit") {
		t.Fatalf("missing submit heading:\n%s", out)
	}
	if !strings.Contains(out, "stack-pr submit --dry-run") || !strings.Contains(out, "stack-pr submit") {
		t.Fatalf("missing submit command guidance:\n%s", out)
	}
	if strings.Contains(out, "# stack-pr agent prompt: Abandon") || strings.Contains(out, "# stack-pr agent prompt: Recovery") {
		t.Fatalf("submit output includes unrelated topic body:\n%s", out)
	}
}

func TestAgentPromptSubmitJSON(t *testing.T) {
	out, err := executeRootForTest([]string{"agent", "prompt", "submit", "--format", "json"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		ID       string `json:"id"`
		Audience string `json:"audience"`
		Commands []struct {
			SideEffects *bool `json:"side_effects"`
		} `json:"commands"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json %q: %v", out, err)
	}
	if payload.ID != "stack-pr.prompt.submit.v1" {
		t.Fatalf("id = %q", payload.ID)
	}
	if payload.Audience != "llm-agent" {
		t.Fatalf("audience = %q", payload.Audience)
	}
	if len(payload.Commands) == 0 {
		t.Fatal("commands empty")
	}
	for i, cmd := range payload.Commands {
		if cmd.SideEffects == nil {
			t.Fatalf("commands[%d] missing side_effects", i)
		}
	}
}

func TestAgentPromptDefaultJSONAllOrder(t *testing.T) {
	out, err := executeRootForTest([]string{"agent", "prompt", "--format", "json"})
	if err != nil {
		t.Fatal(err)
	}
	var payload []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(payload) != len(prompt.TopicOrder) {
		t.Fatalf("len = %d, want %d", len(payload), len(prompt.TopicOrder))
	}
	for i, topic := range prompt.TopicOrder {
		want := "stack-pr.prompt." + topic + ".v1"
		if payload[i].ID != want {
			t.Fatalf("payload[%d].id = %q, want %q", i, payload[i].ID, want)
		}
	}
}

func TestAgentPromptRejectsUnknownTopic(t *testing.T) {
	_, err := executeRootForTest([]string{"agent", "prompt", "nope"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown agent prompt topic") || !strings.Contains(err.Error(), "overview, view, submit, land, abandon, fix, recovery, all") {
		t.Fatalf("unclear error: %v", err)
	}
}

func TestAgentPromptRejectsUnknownFormat(t *testing.T) {
	_, err := executeRootForTest([]string{"agent", "prompt", "--format", "yaml"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unknown agent prompt format "yaml"`) {
		t.Fatalf("unclear error: %v", err)
	}
}

func TestAgentDiagnoseFlagParsing(t *testing.T) {
	out, err := executeRootForTest([]string{"agent", "diagnose", "--format", "json", "--online"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Repo          struct {
			Online bool `json:"online"`
		} `json:"repo"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid diagnose json: %v\n%s", err, out)
	}
	if payload.SchemaVersion != "1" || !payload.Repo.Online {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestAgentDiagnoseRejectsUnknownFormat(t *testing.T) {
	_, err := executeRootForTest([]string{"agent", "diagnose", "--format", "yaml"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unknown agent diagnose format "yaml"`) {
		t.Fatalf("unclear error: %v", err)
	}
}

func TestAgentDiagnoseExitZeroForReportableOutcomes(t *testing.T) {
	if _, err := executeRootForTest([]string{"agent", "diagnose", "--format", "json"}); err != nil {
		t.Fatalf("clean/current repository diagnose returned error: %v", err)
	}

	blockingRepo := t.TempDir()
	if err := runGitForTest(blockingRepo, "init"); err != nil {
		t.Fatal(err)
	}
	chdirForTest(t, blockingRepo)
	if _, err := executeRootForTest([]string{"agent", "diagnose", "--format", "json"}); err != nil {
		t.Fatalf("blocking repository diagnose returned error: %v", err)
	}
}

func TestAgentDiagnoseRunsOutsideGitRepository(t *testing.T) {
	chdirForTest(t, t.TempDir())
	out, err := executeRootForTest([]string{"agent", "diagnose", "--format", "json"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Checks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid diagnose json: %v", err)
	}
	found := false
	for _, c := range payload.Checks {
		if c.ID == "git_repository" && c.Status == "blocking" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing blocking git_repository check: %s", out)
	}
}

func TestAgentDiagnoseTextMentionsReadOnlyRecommendation(t *testing.T) {
	out, err := executeRootForTest([]string{"agent", "diagnose"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# stack-pr agent diagnose") || !strings.Contains(out, "## Recommendation") {
		t.Fatalf("missing diagnose text sections:\n%s", out)
	}
}

func TestAgentPromptRunsOutsideGitRepository(t *testing.T) {
	chdirForTest(t, t.TempDir())
	out, err := executeRootForTest([]string{"agent", "prompt", "overview"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# stack-pr agent prompt: Overview") {
		t.Fatalf("missing overview output: %s", out)
	}
}

func TestAgentPromptRunsWithoutGHOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	out, err := executeRootForTest([]string{"agent", "prompt", "overview"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# stack-pr agent prompt: Overview") {
		t.Fatalf("missing overview output: %s", out)
	}
}

func executeRootForTest(args []string) (string, error) {
	cmd, err := newRootCommand("stack-pr", args)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	err = cmd.Execute()
	return out.String(), err
}

func runGitForTest(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
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

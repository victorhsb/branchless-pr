package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	prompt "github.com/victorhsb/branchless-pr/internal/agent"
)

func TestUserFacingCommandsHaveAgentRegistryEntries(t *testing.T) {
	cmd, err := newRootCommand([]string{"agent", "prompt"})
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
	for _, heading := range []string{"Overview", "View", "Submit", "Land", "Abandon", "Recovery"} {
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
	if !strings.Contains(err.Error(), "unknown agent prompt topic") || !strings.Contains(err.Error(), "overview, view, submit, land, abandon, recovery, all") {
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
	cmd, err := newRootCommand(args)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	err = cmd.Execute()
	return out.String(), err
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

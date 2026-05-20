package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

func TestRenderTextGoldenFiles(t *testing.T) {
	for _, topic := range append(append([]string{}, TopicOrder...), TopicAll) {
		t.Run(topic, func(t *testing.T) {
			got, err := RenderText(topic)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", "prompt_"+topic+".golden.md")
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("RenderText(%q) mismatch with %s\n--- got ---\n%s", topic, path, got)
			}
		})
	}
}

func TestRenderJSONSchema(t *testing.T) {
	versioned := regexp.MustCompile(`^stack-pr\.prompt\.[a-z]+\.v[1-9][0-9]*$`)
	for _, topic := range TopicOrder {
		t.Run(topic, func(t *testing.T) {
			data, err := RenderJSON(topic)
			if err != nil {
				t.Fatal(err)
			}
			var envelope struct {
				ID       string `json:"id"`
				Audience string `json:"audience"`
				Summary  string `json:"summary"`
				Commands []struct {
					Command     string   `json:"command"`
					SideEffects *bool    `json:"side_effects"`
					Purpose     string   `json:"purpose"`
					Effects     []string `json:"effects"`
				} `json:"commands"`
				Rules []string `json:"rules"`
			}
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			if !versioned.MatchString(envelope.ID) {
				t.Fatalf("id = %q, want stack-pr.prompt.<topic>.v<N>", envelope.ID)
			}
			if envelope.ID != "stack-pr.prompt."+topic+".v1" {
				t.Fatalf("id = %q", envelope.ID)
			}
			if envelope.Audience != "llm-agent" {
				t.Fatalf("audience = %q", envelope.Audience)
			}
			if envelope.Summary == "" || len(envelope.Commands) == 0 || len(envelope.Rules) == 0 {
				t.Fatalf("missing required content: %+v", envelope)
			}
			for _, cmd := range envelope.Commands {
				if cmd.Command == "" || cmd.Purpose == "" || cmd.SideEffects == nil {
					t.Fatalf("invalid command entry: %+v", cmd)
				}
				if *cmd.SideEffects && len(cmd.Effects) == 0 {
					t.Fatalf("mutating command %q missing effects", cmd.Command)
				}
			}
		})
	}
}

func TestRenderJSONAllShapeAndOrder(t *testing.T) {
	data, err := RenderJSON(TopicAll)
	if err != nil {
		t.Fatal(err)
	}
	var envelopes []struct {
		ID       string `json:"id"`
		Audience string `json:"audience"`
	}
	if err := json.Unmarshal(data, &envelopes); err != nil {
		t.Fatal(err)
	}
	if len(envelopes) != len(TopicOrder) {
		t.Fatalf("all len = %d, want %d", len(envelopes), len(TopicOrder))
	}
	for i, topic := range TopicOrder {
		wantID := "stack-pr.prompt." + topic + ".v1"
		if envelopes[i].ID != wantID {
			t.Fatalf("envelope %d id = %q, want %q", i, envelopes[i].ID, wantID)
		}
		if envelopes[i].Audience != "llm-agent" {
			t.Fatalf("envelope %d audience = %q", i, envelopes[i].Audience)
		}
	}
}

func TestRenderDeterministic(t *testing.T) {
	for _, topic := range append(append([]string{}, TopicOrder...), TopicAll) {
		t.Run(topic+"/text", func(t *testing.T) {
			first, err := RenderText(topic)
			if err != nil {
				t.Fatal(err)
			}
			second, err := RenderText(topic)
			if err != nil {
				t.Fatal(err)
			}
			if first != second {
				t.Fatal("text output changed between calls")
			}
		})
		t.Run(topic+"/json", func(t *testing.T) {
			first, err := RenderJSON(topic)
			if err != nil {
				t.Fatal(err)
			}
			second, err := RenderJSON(topic)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("json output changed between calls")
			}
		})
	}
}

func TestRenderJSONSideEffectMetadata(t *testing.T) {
	data, err := RenderJSON(TopicOverview)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Commands []struct {
			Command     string `json:"command"`
			SideEffects bool   `json:"side_effects"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, cmd := range envelope.Commands {
		got[cmd.Command] = cmd.SideEffects
	}
	want := map[string]bool{
		"stack-pr view":             false,
		"stack-pr comments":         false,
		"stack-pr submit --dry-run": false,
		"stack-pr submit":           true,
		"stack-pr land":             true,
		"stack-pr abandon":          true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("side effects = %#v, want %#v", got, want)
	}
}

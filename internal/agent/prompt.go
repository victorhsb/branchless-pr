package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	TopicOverview = "overview"
	TopicView     = "view"
	TopicSubmit   = "submit"
	TopicLand     = "land"
	TopicAbandon  = "abandon"
	TopicRecovery = "recovery"
	TopicAll      = "all"
)

// TopicOrder is the canonical order for rendering the full prompt pack.
var TopicOrder = []string{
	TopicOverview,
	TopicView,
	TopicSubmit,
	TopicLand,
	TopicAbandon,
	TopicRecovery,
}

var allTopics = append(append([]string{}, TopicOrder...), TopicAll)

type topicSpec struct {
	Name        string
	Title       string
	Summary     string
	Narrative   []string
	CommandKeys []string
	Rules       []string
}

type promptEnvelope struct {
	ID       string          `json:"id"`
	Audience string          `json:"audience"`
	Summary  string          `json:"summary"`
	Commands []promptCommand `json:"commands"`
	Rules    []string        `json:"rules"`
}

type promptCommand struct {
	Command     string   `json:"command"`
	SideEffects bool     `json:"side_effects"`
	Purpose     string   `json:"purpose"`
	Effects     []string `json:"effects,omitempty"`
}

var topics = map[string]topicSpec{
	TopicOverview: {
		Name:    TopicOverview,
		Title:   "Overview",
		Summary: "High-level safety model for using stack-pr as an LLM agent.",
		Narrative: []string{
			"Use stack-pr to inspect, submit, land, or abandon stacked GitHub pull requests.",
			"Prefer read-only commands first, and ask before commands that mutate Git, branches, or GitHub PRs.",
		},
		CommandKeys: CommandKeys,
		Rules: []string{
			"Run stack-pr view before recommending a mutating stack operation when state is unknown.",
			"Use stack-pr submit --dry-run to preview publishing changes before stack-pr submit.",
			"Obtain explicit user confirmation before running any command marked as having side effects.",
			"Never claim a dry-run created, updated, merged, or deleted anything.",
		},
	},
	TopicView: {
		Name:    TopicView,
		Title:   "View",
		Summary: "Guidance for read-only stack inspection.",
		Narrative: []string{
			"stack-pr view is the default inspection command for understanding the current stack.",
			"It does not modify commits or pull requests, but it may perform ordinary read operations needed for stack discovery.",
		},
		CommandKeys: []string{"view"},
		Rules: []string{
			"Use this command when the user asks what is in the stack or whether it is ready.",
			"If view reports missing metadata or missing PRs, prefer submit --dry-run before suggesting a real submit.",
			"Do not treat view as approval to run submit, land, or abandon.",
		},
	},
	TopicSubmit: {
		Name:    TopicSubmit,
		Title:   "Submit",
		Summary: "Guidance for previewing and publishing stack PR updates.",
		Narrative: []string{
			"Use submit --dry-run to show what stack-pr would create or update without applying changes.",
			"Use submit only after the user requests publishing or updating PRs, or explicitly approves the dry-run plan.",
		},
		CommandKeys: []string{"submit --dry-run", "submit"},
		Rules: []string{
			"Prefer stack-pr submit --dry-run before stack-pr submit when the user has not already approved execution.",
			"Explain that a real submit can push branches, edit PRs, and amend commits with stack metadata.",
			"Ask for explicit confirmation before running stack-pr submit unless the user already gave a clear submit instruction.",
		},
	},
	TopicLand: {
		Name:    TopicLand,
		Title:   "Land",
		Summary: "Guidance for the destructive land flow.",
		Narrative: []string{
			"stack-pr land is destructive and has side effects: it merges the bottom PR and rewrites the remaining local stack shape.",
			"Only run it when the user explicitly wants to land the bottom PR in the stack.",
		},
		CommandKeys: []string{"land"},
		Rules: []string{
			"Obtain explicit user confirmation before invoking stack-pr land.",
			"Use stack-pr view first if the current stack state is unknown.",
			"Do not use land as a readiness check or preview operation.",
		},
	},
	TopicAbandon: {
		Name:    TopicAbandon,
		Title:   "Abandon",
		Summary: "Guidance for the destructive abandon flow.",
		Narrative: []string{
			"stack-pr abandon is destructive and has side effects: it removes stack metadata and deletes generated branches.",
			"Only run it when the user explicitly wants stack-pr to stop managing the current stack.",
		},
		CommandKeys: []string{"abandon"},
		Rules: []string{
			"Obtain explicit user confirmation before invoking stack-pr abandon.",
			"Use stack-pr view first if the user is unsure what will be affected.",
			"Do not use abandon to merge PRs, close PRs, or recover automatically after unrelated errors.",
		},
	},
	TopicRecovery: {
		Name:    TopicRecovery,
		Title:   "Recovery",
		Summary: "Guidance for responding to stack-pr errors or interrupted operations.",
		Narrative: []string{
			"When a stack-pr command fails, stop and inspect the error before running another mutating command.",
			"Prefer read-only inspection and user guidance over automatic cleanup.",
		},
		CommandKeys: []string{"view", "submit --dry-run", "submit", "land", "abandon"},
		Rules: []string{
			"Do not run a destructive command as recovery unless the user explicitly asks for that recovery action.",
			"If a rebase is in progress, ask the user whether to continue, abort, or resolve conflicts before submitting again.",
			"Use stack-pr view after manual recovery steps to verify the resulting stack state.",
		},
	},
}

// RenderText returns deterministic markdown for a topic, or the full pack for all.
func RenderText(topic string) (string, error) {
	if topic == "" {
		topic = TopicAll
	}
	if topic == TopicAll {
		parts := make([]string, 0, len(TopicOrder))
		for _, t := range TopicOrder {
			part, err := RenderText(t)
			if err != nil {
				return "", err
			}
			parts = append(parts, strings.TrimRight(part, "\n"))
		}
		return strings.Join(parts, "\n\n---\n\n") + "\n", nil
	}
	spec, ok := topics[topic]
	if !ok {
		return "", UnknownTopicError(topic)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# stack-pr agent prompt: %s\n\n", spec.Title)
	fmt.Fprintf(&b, "%s\n\n", spec.Summary)
	for _, paragraph := range spec.Narrative {
		fmt.Fprintf(&b, "%s\n\n", paragraph)
	}
	b.WriteString("## Commands\n\n")
	for _, key := range spec.CommandKeys {
		cmd := Commands[key]
		fmt.Fprintf(&b, "- `%s` — %s Side effects: %s.", cmd.Name, cmd.Purpose, yesNo(cmd.SideEffects))
		if cmd.RequiresExplicitConfirmation {
			b.WriteString(" Requires explicit user confirmation.")
		}
		b.WriteString("\n")
		if len(cmd.Effects) > 0 {
			b.WriteString("  Effects:\n")
			for _, effect := range cmd.Effects {
				fmt.Fprintf(&b, "  - %s\n", effect)
			}
		}
	}
	b.WriteString("\n## Rules\n\n")
	for _, rule := range spec.Rules {
		fmt.Fprintf(&b, "- %s\n", rule)
	}
	return b.String(), nil
}

// RenderJSON returns a deterministic JSON envelope for a topic, or an ordered
// array of envelopes for all.
func RenderJSON(topic string) ([]byte, error) {
	if topic == "" {
		topic = TopicAll
	}
	if topic == TopicAll {
		envelopes := make([]promptEnvelope, 0, len(TopicOrder))
		for _, t := range TopicOrder {
			envelope, err := buildEnvelope(t)
			if err != nil {
				return nil, err
			}
			envelopes = append(envelopes, envelope)
		}
		return json.Marshal(envelopes)
	}
	envelope, err := buildEnvelope(topic)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope)
}

// ValidateTopic reports whether topic is supported.
func ValidateTopic(topic string) error {
	if topic == "" {
		return nil
	}
	for _, allowed := range allTopics {
		if topic == allowed {
			return nil
		}
	}
	return UnknownTopicError(topic)
}

// UnknownTopicError returns a clear topic validation error.
func UnknownTopicError(topic string) error {
	return fmt.Errorf("unknown agent prompt topic %q: expected one of %s", topic, AllowedTopicsString())
}

// AllowedTopicsString returns the supported topics in display order.
func AllowedTopicsString() string {
	return strings.Join(allTopics, ", ")
}

func buildEnvelope(topic string) (promptEnvelope, error) {
	spec, ok := topics[topic]
	if !ok {
		return promptEnvelope{}, UnknownTopicError(topic)
	}
	commands := make([]promptCommand, 0, len(spec.CommandKeys))
	for _, key := range spec.CommandKeys {
		cmd := Commands[key]
		commands = append(commands, promptCommand{
			Command:     cmd.Name,
			SideEffects: cmd.SideEffects,
			Purpose:     cmd.Purpose,
			Effects:     append([]string(nil), cmd.Effects...),
		})
	}
	return promptEnvelope{
		ID:       fmt.Sprintf("stack-pr.prompt.%s.v1", topic),
		Audience: "llm-agent",
		Summary:  spec.Summary,
		Commands: commands,
		Rules:    append([]string(nil), spec.Rules...),
	}, nil
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

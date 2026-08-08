// Package nativestacks implements GitHub native Stacked PR integration for
// branchless-pr. It contains typed REST models, availability probing, direct
// Stack REST writes, and a side-effect-free reconciliation planner.
//
// All subprocess calls route through internal/shell; no Go GitHub SDK is used.
package nativestacks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/config"
)

// Mode is the configured native-stacks integration mode.
type Mode = config.NativeStacksMode

const (
	ModeOff      = config.NativeStacksOff
	ModeAuto     = config.NativeStacksAuto
	ModeRequired = config.NativeStacksRequired
)

// Ref identifies a GitHub branch and its current head commit.
type Ref struct {
	Ref string `json:"ref"`
	SHA string `json:"sha,omitempty"`
}

// Repository is the repository identity nested in a pull-request head or base.
type Repository struct {
	FullName string `json:"full_name"`
}

// PullRequestRef is a direct pull-request head or base ref.
type PullRequestRef struct {
	Ref  string      `json:"ref"`
	SHA  string      `json:"sha"`
	Repo *Repository `json:"repo"`
}

// StackMembership is the nullable stack summary on a pull-request resource.
type StackMembership struct {
	ID       int64 `json:"id"`
	Number   int   `json:"number"`
	Size     int   `json:"size"`
	Position int   `json:"position"`
	Base     Ref   `json:"base"`
}

// PullRequest is the subset of a REST pull-request resource required for safe
// native Stack planning.
type PullRequest struct {
	Number          int              `json:"number"`
	State           string           `json:"state"`
	Draft           bool             `json:"draft"`
	MergedAt        *string          `json:"merged_at"`
	Head            PullRequestRef   `json:"head"`
	Base            PullRequestRef   `json:"base"`
	Stack           *StackMembership `json:"stack"`
	AutoMerge       json.RawMessage  `json:"auto_merge"`
	MergeQueueEntry json.RawMessage  `json:"merge_queue_entry"`

	mergedAtPresent bool
}

// UnmarshalJSON records presence for nullable fields that are required by the
// preview contract. The stack field is deliberately not presence-checked: a
// pull request predating the native Stacks feature omits it entirely, and an
// absent stack field is classified as unstacked exactly like "stack": null.
func (p *PullRequest) UnmarshalJSON(data []byte) error {
	type alias PullRequest
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = PullRequest(decoded)
	_, p.mergedAtPresent = raw["merged_at"]
	return nil
}

// IsQueued reports whether the preview PR resource contains a non-null merge
// queue entry.
func (p *PullRequest) IsQueued() bool {
	return nonNullJSON(p.MergeQueueEntry)
}

// IsAutoMergeEnabled reports whether auto-merge is configured for the PR.
func (p *PullRequest) IsAutoMergeEnabled() bool {
	return nonNullJSON(p.AutoMerge)
}

// Stack represents the published GitHub native Stack REST resource.
type Stack struct {
	ID        int64     `json:"id"`
	Number    int       `json:"number"`
	NodeID    string    `json:"node_id"`
	URL       string    `json:"url"`
	Base      Ref       `json:"base"`
	Open      bool      `json:"open"`
	CreatedAt string    `json:"created_at"`
	PRs       []StackPR `json:"pull_requests"`

	openPresent bool
	prsPresent  bool
}

// UnmarshalJSON records required boolean and array field presence.
func (s *Stack) UnmarshalJSON(data []byte) error {
	type alias Stack
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = Stack(decoded)
	_, s.openPresent = raw["open"]
	_, s.prsPresent = raw["pull_requests"]
	return nil
}

// StackPR is the minimal member representation in a Stack resource.
type StackPR struct {
	Number   int     `json:"number"`
	State    string  `json:"state"`
	Draft    bool    `json:"draft"`
	MergedAt *string `json:"merged_at"`
	Head     Ref     `json:"head"`

	draftPresent    bool
	mergedAtPresent bool
}

// UnmarshalJSON records required boolean and nullable field presence.
func (p *StackPR) UnmarshalJSON(data []byte) error {
	type alias StackPR
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = StackPR(decoded)
	_, p.draftPresent = raw["draft"]
	_, p.mergedAtPresent = raw["merged_at"]
	return nil
}

// IsMerged distinguishes a merged PR from a closed-but-unmerged PR.
func (p StackPR) IsMerged() bool { return p.MergedAt != nil }

// Validate checks the documented required pull-request fields and membership
// invariants. Unknown JSON fields are deliberately ignored by encoding/json.
func (p *PullRequest) Validate() error {
	if p == nil {
		return fmt.Errorf("pull request is null")
	}
	if p.Number <= 0 {
		return fmt.Errorf("pull request number must be positive")
	}
	if p.State == "" {
		return fmt.Errorf("pull request #%d is missing state", p.Number)
	}
	if !p.mergedAtPresent {
		return fmt.Errorf("pull request #%d is missing merged_at", p.Number)
	}
	if p.Head.Ref == "" || p.Head.SHA == "" {
		return fmt.Errorf("pull request #%d is missing head ref or sha", p.Number)
	}
	if p.Base.Ref == "" || p.Base.SHA == "" {
		return fmt.Errorf("pull request #%d is missing base ref or sha", p.Number)
	}
	if p.Stack != nil {
		if err := p.Stack.Validate(p.Number); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks a PR membership summary.
func (m *StackMembership) Validate(prNumber int) error {
	if m.ID <= 0 || m.Number <= 0 {
		return fmt.Errorf("pull request #%d has invalid stack id or number", prNumber)
	}
	if m.Size < 1 || m.Position < 1 || m.Position > m.Size {
		return fmt.Errorf("pull request #%d has impossible stack position %d of %d", prNumber, m.Position, m.Size)
	}
	if m.Base.Ref == "" || m.Base.SHA == "" {
		return fmt.Errorf("pull request #%d has incomplete ultimate stack base", prNumber)
	}
	return nil
}

// Validate checks the published Stack resource and member invariants.
func (s *Stack) Validate() error {
	if s == nil {
		return fmt.Errorf("stack is null")
	}
	if s.ID <= 0 || s.Number <= 0 {
		return fmt.Errorf("stack has invalid id or number")
	}
	if s.NodeID == "" || s.URL == "" || s.Base.Ref == "" || s.CreatedAt == "" {
		return fmt.Errorf("stack #%d is missing a required resource field", s.Number)
	}
	if !s.openPresent || !s.prsPresent {
		return fmt.Errorf("stack #%d is missing open or pull_requests", s.Number)
	}
	if len(s.PRs) == 0 {
		return fmt.Errorf("stack #%d contains no pull requests; a dissolved Stack must be represented by 204", s.Number)
	}
	seen := make(map[int]struct{}, len(s.PRs))
	anyOpen := false
	for i := range s.PRs {
		if err := s.PRs[i].Validate(); err != nil {
			return fmt.Errorf("stack #%d position %d: %w", s.Number, i+1, err)
		}
		if _, ok := seen[s.PRs[i].Number]; ok {
			return fmt.Errorf("stack #%d contains duplicate pull request #%d", s.Number, s.PRs[i].Number)
		}
		seen[s.PRs[i].Number] = struct{}{}
		anyOpen = anyOpen || s.PRs[i].State == "open"
	}
	if s.Open != anyOpen {
		return fmt.Errorf("stack #%d open=%t does not match member states", s.Number, s.Open)
	}
	return nil
}

// Validate checks a minimal Stack member.
func (p *StackPR) Validate() error {
	if p.Number <= 0 {
		return fmt.Errorf("member number must be positive")
	}
	if p.State != "open" && p.State != "closed" {
		return fmt.Errorf("pull request #%d has invalid state %q", p.Number, p.State)
	}
	if !p.draftPresent || !p.mergedAtPresent {
		return fmt.Errorf("pull request #%d is missing draft or merged_at", p.Number)
	}
	if p.Head.Ref == "" || p.Head.SHA == "" {
		return fmt.Errorf("pull request #%d is missing head ref or sha", p.Number)
	}
	return nil
}

func nonNullJSON(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}

// Membership is the planner-friendly membership state for one local PR.
type Membership struct {
	PRNumber    int
	StackNumber *int // nil when the PR is unstacked
	Position    int  // 1-based position when stacked; zero when unstacked
	StackBase   string
	StackSize   int
	StackHead   string
}

// IsStacked reports whether the PR belongs to a native Stack.
func (m *Membership) IsStacked() bool { return m != nil && m.StackNumber != nil }

// ActionKind is the result of classifying local PRs against remote membership.
type ActionKind string

const (
	ActionIneligible  ActionKind = "ineligible"
	ActionCreate      ActionKind = "create"
	ActionAppend      ActionKind = "append"
	ActionNoop        ActionKind = "noop"
	ActionConflict    ActionKind = "conflict"
	ActionUnavailable ActionKind = "unavailable fallback"
	ActionDisabled    ActionKind = "disabled"
)

// Result is the outcome of native-stack reconciliation planning.
type Result struct {
	Kind ActionKind

	// For create/append/noop/conflict: the local PR numbers bottom-to-top.
	LocalPRs []int

	// For append/noop/conflict: the complete remote sequence bottom-to-top.
	RemotePRs []int

	// StackNumber is the native Stack number when known.
	StackNumber int

	// SuffixPRs are the unstacked PR numbers to append for ActionAppend.
	SuffixPRs []int

	// Conflict describes why membership could not be reconciled.
	Conflict string

	// Fallback reason when auto mode skips native behavior.
	Fallback string
}

// StackSet is a collection of native Stacks keyed by stack number.
type StackSet map[int]*Stack

// IsWriteAction reports whether the action performs a native Stack write.
func (r *Result) IsWriteAction() bool {
	return r.Kind == ActionCreate || r.Kind == ActionAppend
}

// ReceiptOperation records a native-stack outcome for operation receipts.
type ReceiptOperation struct {
	Kind        ActionKind
	StackNumber int
	PRs         []int
	Fallback    string
	Err         string
}

func prSequence(s *Stack) []int {
	if s == nil {
		return nil
	}
	out := make([]int, len(s.PRs))
	for i := range s.PRs {
		out[i] = s.PRs[i].Number
	}
	return out
}

func formatSequence(xs []int) string {
	parts := make([]string, len(xs))
	for i, n := range xs {
		parts[i] = fmt.Sprintf("#%d", n)
	}
	return strings.Join(parts, ", ")
}

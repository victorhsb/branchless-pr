// Package nativestacks implements GitHub native Stacked PR integration for
// branchless-pr. It contains typed REST models, gh-stack extension wrappers,
// availability probing, and a side-effect-free reconciliation planner.
//
// All subprocess calls route through internal/shell; no Go GitHub SDK is used.
package nativestacks

import (
	"sort"

	"github.com/victorhsb/branchless-pr/internal/config"
)

// Mode is the configured native-stacks integration mode.
type Mode = config.NativeStacksMode

const (
	ModeOff      = config.NativeStacksOff
	ModeAuto     = config.NativeStacksAuto
	ModeRequired = config.NativeStacksRequired
)

// Stack represents a GitHub native Stack resource returned by the REST API.
type Stack struct {
	Number int       `json:"number"`
	Base   string    `json:"base"`
	Head   string    `json:"head"`
	Size   int       `json:"size"`
	PRs    []StackPR `json:"pull_requests"`
}

// StackPR is one PR member of a native Stack.
type StackPR struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	State       string `json:"state"`
	HeadRef     string `json:"head_ref"`
	BaseRef     string `json:"base_ref"`
	StackNumber int    `json:"stack_number"`
	Position    int    `json:"position"`
}

// Membership is the native-stack membership state for a single PR.
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
	ActionMissingExt  ActionKind = "missing extension fallback"
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

// SortedInts returns a sorted copy of ints.
func sortedInts(xs []int) []int {
	out := make([]int, len(xs))
	copy(out, xs)
	sort.Ints(out)
	return out
}

package nativestacks

import (
	"fmt"
	"strconv"
	"strings"
)

// Classify determines the native-stack reconciliation action from the local PR
// sequence (bottom-to-top), remote membership for those PRs, and the complete
// native Stacks that contain any local PR.
// It performs no side effects.
func Classify(localPRs []int, memberships map[int]*Membership, stacks StackSet) *Result {
	if len(localPRs) == 0 {
		return &Result{Kind: ActionIneligible, Conflict: "empty local stack"}
	}
	if len(localPRs) == 1 {
		return &Result{Kind: ActionIneligible, LocalPRs: localPRs, Conflict: "single PR stacks are standalone"}
	}
	if len(localPRs) > 100 {
		return &Result{Kind: ActionIneligible, LocalPRs: localPRs, Conflict: "stack exceeds 100 PRs"}
	}

	// Collect membership state and check for consistency.
	stackNumbers := make(map[int]struct{})
	stackNumberFor := make(map[int]*int)
	for i, n := range localPRs {
		m, ok := memberships[n]
		if !ok || m == nil {
			// Treat missing membership data as unstacked.
			continue
		}
		if m.IsStacked() {
			stackNumbers[*m.StackNumber] = struct{}{}
			stackNumberFor[i] = m.StackNumber
		}
	}

	allUnstacked := true
	for _, n := range localPRs {
		m, ok := memberships[n]
		if ok && m != nil && m.IsStacked() {
			allUnstacked = false
			break
		}
	}
	if allUnstacked {
		return &Result{Kind: ActionCreate, LocalPRs: localPRs}
	}

	// Multiple stacks or mixed membership is always a conflict.
	if len(stackNumbers) != 1 {
		return conflictResult(localPRs, memberships, "local PRs belong to multiple native Stacks or a suffix PR is already stacked elsewhere")
	}

	// Identify the single stack number.
	var stackNumber int
	for n := range stackNumbers {
		stackNumber = n
	}
	stack, ok := stacks[stackNumber]
	if !ok || stack == nil {
		return conflictResult(localPRs, memberships, "local PRs reference native Stack #%d but full stack data is missing", stackNumber)
	}

	// Build the full remote sequence bottom-to-top from the Stack.
	remoteSeq := make([]int, len(stack.PRs))
	for i, pr := range stack.PRs {
		remoteSeq[i] = pr.Number
	}

	// Verify every local stacked PR belongs to this stack and appears in order.
	lastPos := 0
	for _, n := range localPRs {
		m, ok := memberships[n]
		if !ok || m == nil || !m.IsStacked() {
			continue
		}
		if m.Position <= lastPos {
			return conflictResult(localPRs, memberships, "local PR order does not match native Stack order")
		}
		lastPos = m.Position
	}

	// Find first unstacked suffix index.
	suffixStart := -1
	for i, n := range localPRs {
		m, ok := memberships[n]
		if !ok || m == nil || !m.IsStacked() {
			suffixStart = i
			break
		}
	}

	if suffixStart == -1 {
		// Every local PR is stacked. They must equal the full remote sequence.
		if !intSlicesEqual(localPRs, remoteSeq) {
			return conflictResult(localPRs, memberships, "local sequence is a proper prefix of native Stack #%d", stackNumber)
		}
		return &Result{Kind: ActionNoop, LocalPRs: localPRs, RemotePRs: remoteSeq, StackNumber: stackNumber}
	}

	// Append: remote sequence must be exact proper prefix of local sequence,
	// and every suffix PR must be unstacked.
	if suffixStart > len(remoteSeq) {
		return conflictResult(localPRs, memberships, "local prefix extends beyond native Stack #%d", stackNumber)
	}
	if !intSlicesEqual(localPRs[:suffixStart], remoteSeq[:suffixStart]) {
		return conflictResult(localPRs, memberships, "local prefix does not match native Stack #%d", stackNumber)
	}
	for i := suffixStart; i < len(localPRs); i++ {
		m := memberships[localPRs[i]]
		if m != nil && m.IsStacked() {
			return conflictResult(localPRs, memberships, "suffix contains a PR already in a native Stack")
		}
	}

	suffixPRs := make([]int, len(localPRs)-suffixStart)
	copy(suffixPRs, localPRs[suffixStart:])
	return &Result{
		Kind:        ActionAppend,
		LocalPRs:    localPRs,
		RemotePRs:   remoteSeq,
		StackNumber: stackNumber,
		SuffixPRs:   suffixPRs,
	}
}

func conflictResult(localPRs []int, memberships map[int]*Membership, format string, args ...any) *Result {
	reason := fmt.Sprintf(format, args...)
	var b strings.Builder
	fmt.Fprintf(&b, "native membership conflict: %s\n", reason)
	fmt.Fprintln(&b, "local order (bottom-to-top):")
	for _, n := range localPRs {
		m, ok := memberships[n]
		extra := "unstacked"
		if ok && m != nil && m.IsStacked() {
			extra = fmt.Sprintf("stack #%d position %d", *m.StackNumber, m.Position)
		}
		fmt.Fprintf(&b, "  #%d: %s\n", n, extra)
	}
	return &Result{
		Kind:     ActionConflict,
		LocalPRs: localPRs,
		Conflict: b.String(),
	}
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// FormatPRList formats PR numbers as space-separated strings for diagnostics.
func FormatPRList(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, " ")
}

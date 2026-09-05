// Package receipts provides an opt-in machine-readable receipt for
// submit/export executions.
package receipts

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/victorhsb/branchless-pr/internal/nativestacks"
)

// SchemaVersion is the current receipt schema version.
const SchemaVersion = "1.0.0"

// Receipt is the top-level envelope for a submit/export execution.
type Receipt struct {
	SchemaVersion string       `json:"schema_version"`
	Command       string       `json:"command"`
	Status        string       `json:"status"`
	SideEffects   bool         `json:"side_effects"`
	Repo          RepoContext  `json:"repo"`
	Stack         StackContext `json:"stack"`
	Operations    []Operation  `json:"operations"`
}

// RepoContext holds repository-level invocation context.
type RepoContext struct {
	Root               string `json:"root"`
	OriginalBranch     string `json:"original_branch"`
	Remote             string `json:"remote"`
	Target             string `json:"target"`
	Base               string `json:"base"`
	Head               string `json:"head"`
	BranchNameTemplate string `json:"branch_name_template"`
}

// StackContext holds stack-level context.
type StackContext struct {
	Size    int             `json:"size"`
	Entries []StackEntryCtx `json:"entries"`
}

// StackEntryCtx is one stack entry in the receipt.
type StackEntryCtx struct {
	Commit     string `json:"commit"`
	Title      string `json:"title"`
	HeadBranch string `json:"head_branch"`
	BaseBranch string `json:"base_branch"`
	PRURL      string `json:"pr_url"`
}

// Operation records one side effect.
type Operation struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Details any    `json:"details,omitempty"`
}

// NativeStackOperation records a native-stack reconciliation outcome.
type NativeStackOperation struct {
	Action      string `json:"action"`
	StackNumber int    `json:"stack_number,omitempty"`
	PRs         []int  `json:"prs,omitempty"`
	Fallback    string `json:"fallback,omitempty"`
}

// NewReceipt creates a receipt for a real submit/export invocation.
func NewReceipt(command string, repo RepoContext, stack StackContext) *Receipt {
	return &Receipt{
		SchemaVersion: SchemaVersion,
		Command:       command,
		Status:        "ok",
		SideEffects:   true,
		Repo:          repo,
		Stack:         stack,
	}
}

// RecordNativeStack appends a native-stack operation entry.
func (r *Receipt) RecordNativeStack(op nativestacks.ReceiptOperation) {
	status := "ok"
	if op.Err != "" {
		status = "failed"
		r.Status = "partial_failure"
	}
	entry := Operation{
		Type:   "native_stack",
		Status: status,
		Details: NativeStackOperation{
			Action:      string(op.Kind),
			StackNumber: op.StackNumber,
			PRs:         op.PRs,
			Fallback:    op.Fallback,
		},
	}
	if op.Err != "" {
		entry.Error = op.Err
	}
	r.Operations = append(r.Operations, entry)
}

// RecordPush appends a push operation entry.
func (r *Receipt) RecordPush(remote string, branches []string) {
	r.Operations = append(r.Operations, Operation{
		Type:   "push",
		Status: "ok",
		Details: map[string]any{
			"remote":   remote,
			"branches": branches,
		},
	})
}

// Finalize sets the receipt status based on recorded operations.
func (r *Receipt) Finalize() {
	hasFailed := false
	hasOK := false
	for _, op := range r.Operations {
		switch op.Status {
		case "ok":
			hasOK = true
		case "failed":
			hasFailed = true
		}
	}
	switch {
	case hasFailed && hasOK:
		r.Status = "partial_failure"
	case hasFailed:
		r.Status = "failed"
	default:
		r.Status = "ok"
	}
}

// Write emits the receipt to the configured destination.
// destination "off" suppresses output, "-" writes JSON to stdout,
// and any other value is treated as a filesystem path.
func (r *Receipt) Write(destination string) error {
	switch destination {
	case "", "off":
		return nil
	case "-":
		return r.WriteJSON(os.Stdout)
	default:
		return r.WriteFile(destination)
	}
}

// WriteJSON writes the JSON receipt to w.
func (r *Receipt) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteFile writes the JSON receipt to path.
func (r *Receipt) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create receipt file: %w", err)
	}
	defer f.Close()
	return r.WriteJSON(f)
}

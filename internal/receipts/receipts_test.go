package receipts

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/nativestacks"
)

func TestReceiptRecordsNativeStackOperation(t *testing.T) {
	r := NewReceipt("stack-pr submit", RepoContext{Root: "/repo"}, StackContext{Size: 2})
	r.RecordNativeStack(nativestacks.ReceiptOperation{
		Kind:        nativestacks.ActionCreate,
		StackNumber: 7,
		PRs:         []int{10, 20},
	})
	if r.Status != "ok" {
		t.Fatalf("status = %q, want ok", r.Status)
	}
	if len(r.Operations) != 1 {
		t.Fatalf("operations = %d, want 1", len(r.Operations))
	}
	op := r.Operations[0]
	if op.Type != "native_stack" {
		t.Errorf("type = %q, want native_stack", op.Type)
	}
	if op.Status != "ok" {
		t.Errorf("status = %q, want ok", op.Status)
	}
}

func TestReceiptPartialFailure(t *testing.T) {
	r := NewReceipt("stack-pr submit", RepoContext{Root: "/repo"}, StackContext{Size: 2})
	r.RecordPush("origin", []string{"a/stack/1"})
	r.RecordNativeStack(nativestacks.ReceiptOperation{
		Kind: nativestacks.ActionConflict,
		Err:  "native membership conflict",
	})
	r.Finalize()
	if r.Status != "partial_failure" {
		t.Fatalf("status = %q, want partial_failure", r.Status)
	}
}

func TestReceiptJSONRoundTrip(t *testing.T) {
	r := NewReceipt("stack-pr submit", RepoContext{Root: "/repo"}, StackContext{Size: 1})
	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["schema_version"] != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", payload["schema_version"], SchemaVersion)
	}
	if payload["status"] != "ok" {
		t.Errorf("status = %q, want ok", payload["status"])
	}
}

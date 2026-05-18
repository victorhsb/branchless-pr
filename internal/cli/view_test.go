package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/stack"
)

func TestWriteViewStackJSON(t *testing.T) {
	st := stack.Stack{
		newViewTestEntry("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "first", "alice/stack/1", "main", ""),
		newViewTestEntry("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "second", "alice/stack/2", "alice/stack/1", "https://github.com/foo/bar/pull/42"),
	}

	var out bytes.Buffer
	if err := writeViewStack(&out, st, "json", true); err != nil {
		t.Fatal(err)
	}

	var payload []map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 2 {
		t.Fatalf("json entries = %d, want 2", len(payload))
	}
	if got := payload[0]["commit"]; got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("first commit = %#v", got)
	}
	if got := payload[0]["pr_number"]; got != float64(42) {
		t.Fatalf("first pr_number = %#v", got)
	}
	if got := payload[1]["commit"]; got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("second commit = %#v", got)
	}
	if got := payload[1]["pr_url"]; got != "" {
		t.Fatalf("second pr_url = %#v", got)
	}
	if bytes.Contains(out.Bytes(), []byte("\x1b")) {
		t.Fatalf("json output contains ANSI escape sequence: %q", out.Bytes())
	}
}

func TestWriteViewStackJSONEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := writeViewStack(&out, stack.Stack{}, "json", true); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "[]\n" {
		t.Fatalf("empty stack json = %q, want \"[]\\n\"", got)
	}
	var payload []map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 0 {
		t.Fatalf("empty stack json entries = %d, want 0", len(payload))
	}
}

func TestWriteViewStackRejectsUnknownFormat(t *testing.T) {
	var out bytes.Buffer
	err := writeViewStack(&out, nil, "yaml", true)
	if err == nil {
		t.Fatal("expected unknown format error")
	}
	want := `unknown view format "yaml": expected "text" or "json"`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func newViewTestEntry(sha, title, head, base, pr string) *stack.Entry {
	e := &stack.Entry{
		Commit: &stack.Header{
			SHA:         sha,
			Title:       title,
			Author:      "Alice Example <alice@example.com>",
			AuthorName:  "Alice Example",
			AuthorEmail: "alice@example.com",
		},
	}
	e.SetHead(head)
	e.SetBase(base)
	if pr != "" {
		e.SetPR(pr)
	}
	return e
}

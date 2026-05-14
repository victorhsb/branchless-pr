package stack

import (
	"testing"
)

func TestParseTemplateAppendsIDWhenMissing(t *testing.T) {
	bt := ParseTemplate("$USERNAME/stack")
	if !bt.HasID {
		t.Fatalf("expected HasID true after auto-append")
	}
	got := bt.Generate("alice", "feature", 3)
	want := "alice/stack/3"
	if got != want {
		t.Fatalf("Generate: got %q, want %q", got, want)
	}
}

func TestParseTemplateHonoursExplicitID(t *testing.T) {
	bt := ParseTemplate("$USERNAME/branch/$ID-thing")
	got := bt.Generate("alice", "feature", 5)
	want := "alice/branch/5-thing"
	if got != want {
		t.Fatalf("Generate: got %q, want %q", got, want)
	}
}

func TestTemplateMatchAndExtractID(t *testing.T) {
	bt := ParseTemplate("$USERNAME/stack")
	cases := []struct {
		branch string
		match  bool
		id     int
	}{
		{"alice/stack/1", true, 1},
		{"alice/stack/42", true, 42},
		{"alice/stack/notnum", false, 0},
		{"bob/stack/1", false, 0},
		{"alice/other/1", false, 0},
	}
	for _, c := range cases {
		got := bt.Match(c.branch, "alice", "feature")
		if got != c.match {
			t.Errorf("Match(%q) = %v, want %v", c.branch, got, c.match)
			continue
		}
		if c.match {
			id, err := bt.ExtractID(c.branch, "alice", "feature")
			if err != nil {
				t.Errorf("ExtractID(%q) error: %v", c.branch, err)
				continue
			}
			if id != c.id {
				t.Errorf("ExtractID(%q) = %d, want %d", c.branch, id, c.id)
			}
		}
	}
}

func TestReadMetadataParsesStackInfo(t *testing.T) {
	h := &Header{
		SHA:   "0123456789abcdef0123456789abcdef01234567",
		Title: "Some title",
		Body:  "Some details\n\nstack-info: PR: https://github.com/foo/bar/pull/42, branch: alice/stack/1\n",
	}
	e := &Entry{Commit: h}
	if !e.ReadMetadata() {
		t.Fatalf("expected metadata to be parsed")
	}
	if got := e.PR(); got != "https://github.com/foo/bar/pull/42" {
		t.Fatalf("PR = %q", got)
	}
	if got := e.Head(); got != "alice/stack/1" {
		t.Fatalf("head = %q", got)
	}
}

func TestPRNumberFromURL(t *testing.T) {
	h := &Header{SHA: "0123456789abcdef0123456789abcdef01234567"}
	e := &Entry{Commit: h}
	e.SetPR("https://github.com/foo/bar/pull/123")
	n, err := e.PRNumber()
	if err != nil {
		t.Fatal(err)
	}
	if n != 123 {
		t.Fatalf("PRNumber = %d, want 123", n)
	}
}

func TestMetadataLineFormat(t *testing.T) {
	h := &Header{SHA: "0123456789abcdef0123456789abcdef01234567"}
	e := &Entry{Commit: h}
	e.SetPR("99")
	e.SetHead("alice/stack/1")
	got := e.MetadataLine()
	want := "\nstack-info: PR: 99, branch: alice/stack/1"
	if got != want {
		t.Fatalf("MetadataLine = %q, want %q", got, want)
	}
}

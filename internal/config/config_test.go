package config

import (
	"path/filepath"
	"testing"
)

func TestRoundtripPreservesValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".stack-pr.cfg")

	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	c.Set("repo", "remote", "upstream")
	c.Set("repo", "target", "develop")
	c.Set("common", "verbose", "true")
	c.Set("land", "style", "bottom-only")
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	c2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := c2.Get("repo", "remote"); got != "upstream" {
		t.Errorf("remote = %q", got)
	}
	if got := c2.Get("repo", "target"); got != "develop" {
		t.Errorf("target = %q", got)
	}
	if b, _ := c2.GetBool("common", "verbose"); !b {
		t.Errorf("expected verbose true")
	}
	if got := c2.Get("land", "style"); got != "bottom-only" {
		t.Errorf("land.style = %q", got)
	}
}

func TestParseConfigArg(t *testing.T) {
	cases := []struct {
		in      string
		ok      bool
		s, k, v string
	}{
		{"repo.remote=origin", true, "repo", "remote", "origin"},
		{"common.verbose=true", true, "common", "verbose", "true"},
		{"missingequals", false, "", "", ""},
		{"=blank", false, "", "", ""},
		{"nodot=value", false, "", "", ""},
	}
	for _, c := range cases {
		s, k, v, err := ParseConfigArg(c.in)
		if (err == nil) != c.ok {
			t.Errorf("%q: ok = %v, err = %v", c.in, err == nil, err)
			continue
		}
		if !c.ok {
			continue
		}
		if s != c.s || k != c.k || v != c.v {
			t.Errorf("%q: got (%q,%q,%q), want (%q,%q,%q)", c.in, s, k, v, c.s, c.k, c.v)
		}
	}
}

func TestDefaultsAndMerge(t *testing.T) {
	d := Defaults()
	c := &Config{sections: map[string]map[string]string{}}
	c.Set("repo", "remote", "myremote") // override

	c.Merge(d)
	if got := c.Get("repo", "remote"); got != "myremote" {
		t.Errorf("override lost: %q", got)
	}
	if got := c.Get("repo", "target"); got != "main" {
		t.Errorf("default missing: %q", got)
	}
	if got := c.Get("land", "style"); got != "bottom-only" {
		t.Errorf("default missing: %q", got)
	}
}

func TestGetBoolParsesPythonStyle(t *testing.T) {
	c := &Config{sections: map[string]map[string]string{}}
	c.Set("x", "a", "TRUE")
	c.Set("x", "b", "no")
	c.Set("x", "c", "on")
	c.Set("x", "d", "0")
	for _, tt := range []struct {
		k string
		v bool
	}{{"a", true}, {"b", false}, {"c", true}, {"d", false}} {
		got, err := c.GetBool("x", tt.k)
		if err != nil {
			t.Errorf("%s: %v", tt.k, err)
			continue
		}
		if got != tt.v {
			t.Errorf("%s: got %v, want %v", tt.k, got, tt.v)
		}
	}
}

package pr

import "testing"

func TestValidateRef(t *testing.T) {
	cases := []struct {
		name string
		ref  string
		ok   bool
	}{
		{"pr url", "https://github.com/acme/widget/pull/12", true},
		{"bare number", "42", true},

		{"leading dash", "-R", false},
		{"gh flag injection", "--repo=attacker/evil", false},
		{"empty", "", false},
		{"embedded space", "12 --repo evil", false},
		{"newline", "12\n--repo=evil", false},
		{"escape char", "12\x1b]0;pwn\x07", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRef(tc.ref)
			if tc.ok != (err == nil) {
				t.Fatalf("ValidateRef(%q) = %v, ok=%v", tc.ref, err, tc.ok)
			}
		})
	}
}

// A hostile `stack-info:` PR ref must be rejected before reaching `gh`.
func TestPRWrappersRejectHostileRef(t *testing.T) {
	const hostile = "--repo=attacker/evil"

	if _, err := View(hostile); err == nil {
		t.Error("View accepted a hostile PR ref")
	}
	if err := EditBase(hostile, "main"); err == nil {
		t.Error("EditBase accepted a hostile PR ref")
	}
	if err := Edit(hostile, "t", "main", nil); err == nil {
		t.Error("Edit accepted a hostile PR ref")
	}
	if err := EditTitleBody(hostile, "t", nil); err == nil {
		t.Error("EditTitleBody accepted a hostile PR ref")
	}
	if err := Ready(hostile); err == nil {
		t.Error("Ready accepted a hostile PR ref")
	}
	if err := ReadyUndo(hostile); err == nil {
		t.Error("ReadyUndo accepted a hostile PR ref")
	}
	if err := MergeSquash(hostile, "t", nil); err == nil {
		t.Error("MergeSquash accepted a hostile PR ref")
	}
	if err := MergeRebase(hostile); err == nil {
		t.Error("MergeRebase accepted a hostile PR ref")
	}
	if err := MergeRebaseAuto(hostile); err == nil {
		t.Error("MergeRebaseAuto accepted a hostile PR ref")
	}
	if _, err := FetchChecks(hostile); err == nil {
		t.Error("FetchChecks accepted a hostile PR ref")
	}
}

package textsafe

import "testing"

func TestString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "looks fine", "looks fine"},
		{"keeps newlines and tabs", "a\nb\tc\nd", "a\nb\tc\nd"},
		{"crlf normalizes to lf", "a\r\nb", "a\nb"},
		{"strips bare cr used to overwrite a line", "pending\rAll checks passed", "pendingAll checks passed"},
		{"keeps non-ascii", "café — ✅ 日本語", "café — ✅ 日本語"},

		{
			"strips ansi colour",
			"\x1b[31mLGTM, merging\x1b[0m",
			"[31mLGTM, merging[0m",
		},
		{
			"strips osc window title",
			"hi\x1b]0;pwned\x07there",
			"hi]0;pwnedthere",
		},
		{
			"strips cursor movement used to overwrite prior output",
			"real\x1b[2J\x1b[Hfake summary",
			"real[2J[Hfake summary",
		},
		{"strips bare escape", "a\x1bb", "ab"},
		{"strips nul", "a\x00b", "ab"},
		{"strips bell", "ding\x07", "ding"},
		{"strips del", "a\x7fb", "ab"},
		{"strips c1 csi (single-char ESC-[)", "ab", "ab"},
		{"strips c1 osc (single-char ESC-])", "ab", "ab"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := String(tc.in); got != tc.want {
				t.Fatalf("String(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// The escape introducers must be gone; what remains is inert text.
func TestStringLeavesNoEscapeIntroducer(t *testing.T) {
	payload := "\x1b[31mred\x1b[0m\x1b]0;title\x072K"
	got := String(payload)
	for _, r := range got {
		if r == 0x1b || r == 0x9b || r == 0x9d || r == 0x07 || r == '\r' {
			t.Fatalf("String(%q) = %q still contains an escape introducer %U", payload, got, r)
		}
	}
}

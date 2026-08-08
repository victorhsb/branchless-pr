// Package textsafe strips terminal control sequences from text that originates
// outside the local repository.
//
// Report commands print GitHub-authored content — comment and review bodies,
// check-run names, PR titles, author logins — to the terminal. A body
// containing ANSI/OSC escape sequences can otherwise repaint the screen,
// relocate the cursor, retitle the window, or hide text, letting a comment on
// a pull request forge output that appears to come from the tool itself.
//
// JSON output does not need this: encoding/json escapes control characters.
package textsafe

import "strings"

// String removes control characters from s, preserving the whitespace that
// legitimately appears in rendered text.
//
// Removed: C0 controls (U+0000–U+001F) other than tab and line feed; DEL
// (U+007F); and C1 controls (U+0080–U+009F), which are what a bare ESC becomes
// once decoded and would otherwise still introduce a sequence.
//
// Carriage return is removed rather than preserved. A bare CR returns the
// cursor to column 0, so a body containing "pending\rAll checks passed" prints
// as "All checks passed" over the real text — the same forgery the escape
// sequences above enable. GitHub bodies arrive CRLF-terminated, so dropping CR
// also normalizes them to the LF the renderer already emits.
func String(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isControl(r rune) bool {
	switch r {
	case '\t', '\n':
		return false
	}
	return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f)
}

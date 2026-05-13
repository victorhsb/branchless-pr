package stack

import (
	"fmt"
	"strings"
)

// BuildPRBody constructs the PR body for a stack entry.
//
// For a multi-PR stack a table-of-contents header is prepended.  If keepBody
// is true, existingBody is searched for the delimiter "--- --- ---"; anything
// below the last occurrence of the delimiter is preserved as the body content
// instead of the commit message body.
func BuildPRBody(entry *Entry, st Stack, keepBody bool, existingBody string) []byte {
	isMulti := len(st) > 1

	// Determine the user-authored content portion.
	content := entry.Commit.Body
	if keepBody && existingBody != "" {
		if idx := strings.LastIndex(existingBody, "--- --- ---"); idx != -1 {
			content = strings.TrimLeftFunc(
				existingBody[idx+len("--- --- ---"):],
				func(r rune) bool { return r == '\n' || r == '\r' || r == '\t' || r == ' ' },
			)
		} else {
			content = existingBody
		}
	}

	var b strings.Builder
	if isMulti {
		b.WriteString("Stacked PRs:\n")
		for _, e := range st.Reverse() {
			prNum, _ := e.PRNumber()
			prefix := " * "
			if e == entry {
				prefix = " * __->__"
			}
			b.WriteString(fmt.Sprintf("%s#%d\n", prefix, prNum))
		}
		b.WriteString("\n--- --- ---\n\n")
		b.WriteString("### ")
		b.WriteString(entry.Commit.Title)
		b.WriteString("\n\n")
	}
	b.WriteString(content)
	return []byte(b.String())
}

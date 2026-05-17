package stack

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// Stack is an ordered list of stack entries (bottom-to-top).
type Stack []*Entry

// Discover loads commits from base..head via `git rev-list --header`.
// Commits are returned oldest-to-newest.
func Discover(base, head string) (Stack, error) {
	if base == "" {
		return nil, fmt.Errorf("base is required")
	}
	if head == "" {
		head = "HEAD"
	}

	args := []string{"git", "rev-list", "--header", "^" + base, head}
	out, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return nil, fmt.Errorf("rev-list: %w", err)
	}

	// Split raw output into individual commit blocks.
	// Each block starts with a 40-char SHA line.
	var blocks []string
	var current strings.Builder
	var inBlock bool
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 40 && isSHA(trimmed) {
			if inBlock {
				blocks = append(blocks, current.String())
				current.Reset()
			}
			inBlock = true
		}
		if inBlock {
			current.WriteString(line)
			current.WriteByte('\n')
		}
	}
	if inBlock {
		blocks = append(blocks, current.String())
	}

	// Parse blocks newest-to-oldest from rev-list, then reverse.
	var stack Stack
	for i := len(blocks) - 1; i >= 0; i-- {
		h, err := ParseHeader(strings.TrimSuffix(blocks[i], "\n"))
		if err != nil {
			return nil, fmt.Errorf("parse header: %w", err)
		}
		stack = append(stack, &Entry{Commit: h})
	}
	return stack, nil
}

// Reverse returns a copy ordered newest-to-oldest (for display).
func (st Stack) Reverse() Stack {
	r := make(Stack, len(st))
	for i := range st {
		r[i] = st[len(st)-1-i]
	}
	return r
}

// AssignHeads assigns generated branch names to entries missing metadata heads.
// Existing metadata takes precedence.
func (st Stack) AssignHeads(tmpl BranchTemplate, username, branchName string, remote string) error {
	nextID, err := NextID(remote, tmpl, username, branchName)
	if err != nil {
		return fmt.Errorf("next branch id: %w", err)
	}
	for _, e := range st {
		if !e.HasHead() {
			e.SetHead(tmpl.Generate(username, branchName, nextID))
			nextID++
		}
	}
	return nil
}

// AssignBases sets base branches bottom-to-top per SPEC §5.5.
func (st Stack) AssignBases(target string) {
	for i, e := range st {
		if i == 0 {
			e.SetBase(target)
		} else {
			e.SetBase(st[i-1].Head())
		}
	}
}

// PrintStack prints the stack newest-to-oldest with optional ANSI and hyperlinks.
func (st Stack) PrintStack(links, color bool) {
	for _, e := range st.Reverse() {
		fmt.Println(e.PrettyLine(links, color))
	}
}

// ToJSON returns the stack newest-to-oldest in the machine-readable view schema.
func (st Stack) ToJSON() ([]byte, error) {
	return json.Marshal(st.Reverse())
}

// IsEmpty reports whether the stack has no entries.
func (st Stack) IsEmpty() bool { return len(st) == 0 }

// Len returns the number of entries.
func (st Stack) Len() int { return len(st) }

// Bottom returns the bottom-most entry or nil if empty.
func (st Stack) Bottom() *Entry {
	if len(st) == 0 {
		return nil
	}
	return st[0]
}

// Top returns the top-most entry or nil if empty.
func (st Stack) Top() *Entry {
	if len(st) == 0 {
		return nil
	}
	return st[len(st)-1]
}

func isSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

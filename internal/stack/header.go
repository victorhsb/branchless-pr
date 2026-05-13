package stack

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/git"
)

var (
	authorEmailRe = regexp.MustCompile(`<([^>]+)>`)
)

// Header represents parsed output from git rev-list --header for a single commit.
type Header struct {
	SHA         string
	TreeSHA     string
	Parents     []string
	Author      string // Name <email> without timestamp
	AuthorName  string
	AuthorEmail string
	Title       string
	Body        string // full commit message (title + body) without metadata
	raw         string
}

// ParseHeader parses raw output from `git rev-list --header` for a single commit.
func ParseHeader(raw string) (*Header, error) {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty header")
	}
	h := &Header{raw: raw}

	// First line is the full commit SHA.
	h.SHA = strings.TrimSpace(lines[0])
	if !git.IsFullSHA(h.SHA) {
		return nil, fmt.Errorf("invalid commit SHA: %s", h.SHA)
	}

	// Parse header fields until blank line.
	var msgStart int
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			msgStart = i + 1
			break
		}
		if strings.HasPrefix(line, "tree ") {
			h.TreeSHA = strings.TrimPrefix(line, "tree ")
		} else if strings.HasPrefix(line, "parent ") {
			h.Parents = append(h.Parents, strings.TrimPrefix(line, "parent "))
		} else if strings.HasPrefix(line, "author ") {
			rest := strings.TrimPrefix(line, "author ")
			// rest looks like: "Name <email> 1234567890 +0000"
			// strip trailing " <ts> <tz>"
			parts := strings.Fields(rest)
			if len(parts) >= 3 {
				// Remove last two fields (timestamp + timezone)
				rest = strings.TrimSuffix(rest, " "+parts[len(parts)-2]+" "+parts[len(parts)-1])
			}
			h.Author = rest
			h.AuthorName, h.AuthorEmail = parseAuthor(rest)
		}
		// committer and gpg lines are ignored for our purposes
	}

	// Read indented message body lines.
	var msgLines []string
	for i := msgStart; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "    ") {
			msgLines = append(msgLines, line[4:])
		} else if line == "" {
			msgLines = append(msgLines, "")
		} else if len(msgLines) > 0 {
			// Non-indented line after message start = end of this commit's message
			break
		}
	}

	if len(msgLines) > 0 {
		h.Title = strings.TrimSpace(msgLines[0])
		if len(msgLines) > 1 {
			h.Body = strings.Join(msgLines[1:], "\n")
			h.Body = strings.TrimRight(h.Body, " \t\n\r")
		}
	}
	return h, nil
}

// ShortSHA returns an abbreviated 8-character SHA.
func (h *Header) ShortSHA() string {
	if len(h.SHA) >= 8 {
		return h.SHA[:8]
	}
	return h.SHA
}

// CommitMsg returns full commit message (title + body) without stack-info metadata.
func (h *Header) CommitMsg() string {
	if h.Body == "" {
		return h.Title
	}
	return h.Title + "\n\n" + h.Body
}

func parseAuthor(s string) (name, email string) {
	m := authorEmailRe.FindStringSubmatch(s)
	if m == nil {
		return s, ""
	}
	email = m[1]
	name = strings.TrimSpace(strings.SplitN(s, "<", 2)[0])
	return name, email
}

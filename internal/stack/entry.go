package stack

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

var (
	stackInfoRe = regexp.MustCompile(`(?m)^stack-info: PR: (.+), branch: (.+)$`)
)

// Entry represents one stack commit and its associated PR state.
type Entry struct {
	Commit     *Header
	pr         string // PR URL or ref
	headBranch string // generated branch name
	baseBranch string // target branch for PR
	IsTmpDraft bool   // true if the PR was temporarily made draft during submit
}

// MarshalJSON emits the flat representation consumed by machine-readable views.
func (e *Entry) MarshalJSON() ([]byte, error) {
	prNumber := 0
	if e.HasPR() {
		var err error
		prNumber, err = e.PRNumber()
		if err != nil {
			return nil, err
		}
	}

	type jsonEntry struct {
		Commit      string `json:"commit"`
		ShortSHA    string `json:"short_sha"`
		Title       string `json:"title"`
		Author      string `json:"author"`
		AuthorName  string `json:"author_name"`
		AuthorEmail string `json:"author_email"`
		PRURL       string `json:"pr_url"`
		PRNumber    int    `json:"pr_number"`
		HeadBranch  string `json:"head_branch"`
		BaseBranch  string `json:"base_branch"`
	}

	return json.Marshal(jsonEntry{
		Commit:      e.Commit.SHA,
		ShortSHA:    e.Commit.ShortSHA(),
		Title:       e.Commit.Title,
		Author:      e.Commit.Author,
		AuthorName:  e.Commit.AuthorName,
		AuthorEmail: e.Commit.AuthorEmail,
		PRURL:       e.pr,
		PRNumber:    prNumber,
		HeadBranch:  e.headBranch,
		BaseBranch:  e.baseBranch,
	})
}

// HasPR reports whether an associated PR is known.
func (e *Entry) HasPR() bool { return e.pr != "" }

// HasHead reports whether a head branch is assigned.
func (e *Entry) HasHead() bool { return e.headBranch != "" }

// HasBase reports whether a base branch is assigned.
func (e *Entry) HasBase() bool { return e.baseBranch != "" }

// PR returns the PR URL/ref, or panics if unset.
func (e *Entry) PR() string {
	if e.pr == "" {
		panic("PR is unset")
	}
	return e.pr
}

// Head returns the generated head branch name, or panics if unset.
func (e *Entry) Head() string {
	if e.headBranch == "" {
		panic("head is unset")
	}
	return e.headBranch
}

// Base returns the base branch name. May be empty for the bottom entry before assignment.
func (e *Entry) Base() string { return e.baseBranch }

// SetPR assigns the PR reference.
func (e *Entry) SetPR(pr string) { e.pr = pr }

// SetHead assigns the head branch.
func (e *Entry) SetHead(head string) { e.headBranch = head }

// SetBase assigns the base branch.
func (e *Entry) SetBase(base string) { e.baseBranch = base }

// HasMissingInfo reports whether PR, head, or base is missing.
func (e *Entry) HasMissingInfo() bool {
	return e.pr == "" || e.headBranch == "" || e.baseBranch == ""
}

// ReadMetadata parses the stack-info line from the commit message body and sets
// PR and head. Returns true if metadata was found.
func (e *Entry) ReadMetadata() bool {
	if e.Commit == nil {
		return false
	}
	body := e.Commit.Body
	// Also check full raw because header parser strips indentation
	if body == "" {
		body = e.Commit.CommitMsg()
	}
	m := stackInfoRe.FindStringSubmatch(body)
	if m == nil {
		return false
	}
	e.pr = strings.TrimSpace(m[1])
	e.headBranch = strings.TrimSpace(m[2])
	return true
}

// StripMetadata removes the stack-info line from the commit message body.
func (e *Entry) StripMetadata() string {
	msg := e.Commit.CommitMsg()
	return stackInfoRe.ReplaceAllString(msg, "")
}

// MetadataLine returns the stack-info line to append to a commit message.
func (e *Entry) MetadataLine() string {
	return fmt.Sprintf("\nstack-info: PR: %s, branch: %s", e.pr, e.headBranch)
}

// PRNumber extracts the numeric PR number from the PR URL or ref.
func (e *Entry) PRNumber() (int, error) {
	if e.pr == "" {
		return 0, fmt.Errorf("no PR")
	}
	// PR ref may be a URL like .../pull/123 or just "123"
	last := pathLast(e.pr, "/")
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0, fmt.Errorf("malformed PR ref %q: %w", e.pr, err)
	}
	return n, nil
}

// PrettyLine formats one stack line with optional ANSI colours and hyperlinks.
func (e *Entry) PrettyLine(links bool, color bool) string {
	if color {
		return e.prettyColor(links)
	}
	return e.prettyPlain()
}

func (e *Entry) prettyColor(links bool) string {
	sha := e.Commit.ShortSHA()
	prText := "no PR"
	if e.HasPR() {
		prNum, _ := e.PRNumber()
		prText = fmt.Sprintf("#%d", prNum)
		if links {
			prText = hyperlink(e.pr, prText)
		}
	}
	return fmt.Sprintf("* %s%s%s (%s, '%s' -> '%s'): %s",
		Bold, sha, Reset,
		prText, e.headBranch, e.baseBranch,
		e.Commit.Title,
	)
}

func (e *Entry) prettyPlain() string {
	sha := e.Commit.ShortSHA()
	prText := "no PR"
	if e.HasPR() {
		prNum, _ := e.PRNumber()
		prText = fmt.Sprintf("#%d", prNum)
	}
	return fmt.Sprintf("* %s (%s, '%s' -> '%s'): %s",
		sha, prText, e.headBranch, e.baseBranch,
		e.Commit.Title,
	)
}

func pathLast(s, sep string) string {
	parts := strings.Split(s, sep)
	return parts[len(parts)-1]
}

// --- branch name generation ---

// BranchTemplate holds the parsed branch-name-template.
type BranchTemplate struct {
	Raw        string
	HasID      bool
	BasePrefix string // everything before $ID, or the whole template if HasID
	IDSuffix   string // everything after $ID
}

// ParseTemplate prepares a template for use.
// If the template does not contain "$ID", "/$ID" is appended.
func ParseTemplate(raw string) BranchTemplate {
	if strings.Contains(raw, "$ID") {
		parts := strings.SplitN(raw, "$ID", 2)
		return BranchTemplate{Raw: raw, HasID: true, BasePrefix: parts[0], IDSuffix: parts[1]}
	}
	return BranchTemplate{Raw: raw, HasID: true, BasePrefix: raw + "/", IDSuffix: ""}
}

// Generate produces a branch name from template variables.
func (bt BranchTemplate) Generate(username, branchName string, id int) string {
	s := bt.BasePrefix
	s = strings.ReplaceAll(s, "$USERNAME", username)
	s = strings.ReplaceAll(s, "$BRANCH", branchName)
	s += strconv.Itoa(id) + bt.IDSuffix
	return strings.TrimSuffix(s, "/")
}

// Match reports whether a branch name matches this template pattern.
func (bt BranchTemplate) Match(branch, username, localBranch string) bool {
	prefix := bt.BasePrefix
	prefix = strings.ReplaceAll(prefix, "$USERNAME", username)
	prefix = strings.ReplaceAll(prefix, "$BRANCH", localBranch)
	if !strings.HasPrefix(branch, prefix) {
		return false
	}
	rest := strings.TrimPrefix(branch, prefix)
	if bt.IDSuffix == "" {
		// must be a pure integer
		_, err := strconv.Atoi(rest)
		return err == nil
	}
	if !strings.HasSuffix(rest, bt.IDSuffix) {
		return false
	}
	rest = strings.TrimSuffix(rest, bt.IDSuffix)
	_, err := strconv.Atoi(rest)
	return err == nil
}

// ExtractID pulls the numeric ID from a branch name that matches the template.
func (bt BranchTemplate) ExtractID(branch, username, localBranch string) (int, error) {
	prefix := bt.BasePrefix
	prefix = strings.ReplaceAll(prefix, "$USERNAME", username)
	prefix = strings.ReplaceAll(prefix, "$BRANCH", localBranch)
	if !strings.HasPrefix(branch, prefix) {
		return 0, fmt.Errorf("branch %q does not match template prefix %q", branch, prefix)
	}
	rest := strings.TrimPrefix(branch, prefix)
	if bt.IDSuffix != "" {
		if !strings.HasSuffix(rest, bt.IDSuffix) {
			return 0, fmt.Errorf("branch %q missing template suffix %q", branch, bt.IDSuffix)
		}
		rest = strings.TrimSuffix(rest, bt.IDSuffix)
	}
	return strconv.Atoi(rest)
}

// NextID scans remote refs for branches matching the template and returns
// the maximum existing ID plus one. If none are found, returns 1.
func NextID(remote string, tmpl BranchTemplate, username, localBranch string) (int, error) {
	args := []string{"git", "ls-remote", "--heads", remote}
	out, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return 0, fmt.Errorf("ls-remote: %w", err)
	}

	maxID := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: <sha>\trefs/heads/<branch>
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		const prefix = "refs/heads/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}
		branch := strings.TrimPrefix(ref, prefix)
		if !tmpl.Match(branch, username, localBranch) {
			continue
		}
		id, err := tmpl.ExtractID(branch, username, localBranch)
		if err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID + 1, nil
}

package pr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

const (
	CommentKindConversation  = "conversation"
	CommentKindReview        = "review"
	CommentKindReviewComment = "review_comment"
	CommentKindReviewThread  = "review_thread"
)

// CommentLocation identifies a file/line location when GitHub provides one.
type CommentLocation struct {
	Path              string `json:"path,omitempty"`
	Line              int    `json:"line,omitempty"`
	OriginalLine      int    `json:"original_line,omitempty"`
	StartLine         int    `json:"start_line,omitempty"`
	OriginalStartLine int    `json:"original_start_line,omitempty"`
	Side              string `json:"side,omitempty"`
	StartSide         string `json:"start_side,omitempty"`
	Outdated          bool   `json:"outdated,omitempty"`
}

// CommentItem is the normalized, agent-facing representation of one GitHub
// comment-like object.
type CommentItem struct {
	ID          string           `json:"id,omitempty"`
	Kind        string           `json:"kind"`
	PRNumber    int              `json:"pr_number"`
	Author      string           `json:"author,omitempty"`
	Body        string           `json:"body,omitempty"`
	URL         string           `json:"url,omitempty"`
	CreatedAt   string           `json:"created_at,omitempty"`
	UpdatedAt   string           `json:"updated_at,omitempty"`
	SubmittedAt string           `json:"submitted_at,omitempty"`
	State       string           `json:"state,omitempty"`
	Resolved    *bool            `json:"resolved,omitempty"`
	Location    *CommentLocation `json:"location,omitempty"`
	Replies     []CommentItem    `json:"replies,omitempty"`
}

// PullRequestComments contains normalized comments fetched for one PR.
type PullRequestComments struct {
	Number   int           `json:"number"`
	URL      string        `json:"url"`
	Items    []CommentItem `json:"items"`
	Warnings []string      `json:"warnings,omitempty"`
}

// AuthError marks GitHub authentication/authorization failures.
type AuthError struct {
	Err error
}

func (e *AuthError) Error() string { return e.Err.Error() }
func (e *AuthError) Unwrap() error { return e.Err }

// IsAuthError reports whether err represents a global GitHub auth failure.
func IsAuthError(err error) bool {
	var auth *AuthError
	return errors.As(err, &auth)
}

// FetchComments fetches read-only PR comments, reviews, and review threads.
func FetchComments(prRef string) (*PullRequestComments, error) {
	summary, err := fetchPRCommentSummary(prRef)
	if err != nil {
		return nil, err
	}

	report := &PullRequestComments{
		Number: summary.Number,
		URL:    summary.URL,
	}
	report.Items = append(report.Items, summary.Items...)

	owner, repo, ok := parsePullURL(summary.URL)
	if !ok {
		report.Warnings = append(report.Warnings, "could not determine repository owner/name for review thread lookup")
		return report, nil
	}

	threads, err := fetchReviewThreads(owner, repo, summary.Number)
	if err != nil {
		if IsAuthError(err) {
			return nil, err
		}
		report.Warnings = append(report.Warnings, err.Error())
		return report, nil
	}
	report.Items = append(report.Items, threads...)
	return report, nil
}

type prCommentSummary struct {
	Number int
	URL    string
	Items  []CommentItem
}

type ghAuthor struct {
	Login string `json:"login"`
}

type ghComment struct {
	ID        string    `json:"id"`
	Author    *ghAuthor `json:"author"`
	Body      string    `json:"body"`
	URL       string    `json:"url"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

type ghReviewComment struct {
	ID                string    `json:"id"`
	Author            *ghAuthor `json:"author"`
	Body              string    `json:"body"`
	URL               string    `json:"url"`
	CreatedAt         string    `json:"createdAt"`
	UpdatedAt         string    `json:"updatedAt"`
	Path              string    `json:"path"`
	Line              int       `json:"line"`
	OriginalLine      int       `json:"originalLine"`
	StartLine         int       `json:"startLine"`
	OriginalStartLine int       `json:"originalStartLine"`
}

type ghReview struct {
	ID          string            `json:"id"`
	Author      *ghAuthor         `json:"author"`
	Body        string            `json:"body"`
	URL         string            `json:"url"`
	State       string            `json:"state"`
	SubmittedAt string            `json:"submittedAt"`
	Comments    []ghReviewComment `json:"comments"`
}

type ghPRViewComments struct {
	Number   int         `json:"number"`
	URL      string      `json:"url"`
	Comments []ghComment `json:"comments"`
	Reviews  []ghReview  `json:"reviews"`
}

func fetchPRCommentSummary(prRef string) (*prCommentSummary, error) {
	out, err := runGHJSON([]string{"gh", "pr", "view", prRef, "--json", "number,url,comments,reviews"})
	if err != nil {
		return nil, fmt.Errorf("gh pr view comments %s: %w", prRef, err)
	}
	return ParsePRCommentSummary(out)
}

// ParsePRCommentSummary normalizes gh pr view JSON for tests and FetchComments.
func ParsePRCommentSummary(data []byte) (*prCommentSummary, error) {
	var raw ghPRViewComments
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse pr comments response: %w", err)
	}

	result := &prCommentSummary{Number: raw.Number, URL: raw.URL}
	for _, c := range raw.Comments {
		result.Items = append(result.Items, CommentItem{
			ID:        c.ID,
			Kind:      CommentKindConversation,
			PRNumber:  raw.Number,
			Author:    login(c.Author),
			Body:      c.Body,
			URL:       c.URL,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	for _, r := range raw.Reviews {
		result.Items = append(result.Items, CommentItem{
			ID:          r.ID,
			Kind:        CommentKindReview,
			PRNumber:    raw.Number,
			Author:      login(r.Author),
			Body:        r.Body,
			URL:         r.URL,
			State:       r.State,
			SubmittedAt: r.SubmittedAt,
		})
		for _, c := range r.Comments {
			result.Items = append(result.Items, CommentItem{
				ID:        c.ID,
				Kind:      CommentKindReviewComment,
				PRNumber:  raw.Number,
				Author:    login(c.Author),
				Body:      c.Body,
				URL:       c.URL,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
				Location:  reviewCommentLocation(c),
			})
		}
	}
	return result, nil
}

func fetchReviewThreads(owner, repo string, number int) ([]CommentItem, error) {
	query := `query($owner:String!, $repo:String!, $number:Int!) {
  repository(owner:$owner, name:$repo) {
    pullRequest(number:$number) {
      reviewThreads(first:100) {
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          originalLine
          startLine
          originalStartLine
          diffSide
          startDiffSide
          comments(first:100) {
            nodes {
              id
              author { login }
              body
              url
              createdAt
              updatedAt
              path
              line
              originalLine
            }
          }
        }
      }
    }
  }
}`
	out, err := runGHJSON([]string{
		"gh", "api", "graphql",
		"-f", "query=" + query,
		"-f", "owner=" + owner,
		"-f", "repo=" + repo,
		"-F", "number=" + strconv.Itoa(number),
	})
	if err != nil {
		return nil, fmt.Errorf("gh api graphql reviewThreads #%d: %w", number, err)
	}
	return ParseReviewThreads(number, out)
}

type ghGraphQLThreads struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					Nodes []ghReviewThread `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

type ghReviewThread struct {
	ID                string           `json:"id"`
	IsResolved        bool             `json:"isResolved"`
	IsOutdated        bool             `json:"isOutdated"`
	Path              string           `json:"path"`
	Line              int              `json:"line"`
	OriginalLine      int              `json:"originalLine"`
	StartLine         int              `json:"startLine"`
	OriginalStartLine int              `json:"originalStartLine"`
	DiffSide          string           `json:"diffSide"`
	StartDiffSide     string           `json:"startDiffSide"`
	Comments          ghThreadComments `json:"comments"`
}

type ghThreadComments struct {
	Nodes []ghReviewComment `json:"nodes"`
}

// ParseReviewThreads normalizes gh api graphql reviewThreads JSON.
func ParseReviewThreads(prNumber int, data []byte) ([]CommentItem, error) {
	var raw ghGraphQLThreads
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse review threads response: %w", err)
	}

	var items []CommentItem
	for _, t := range raw.Data.Repository.PullRequest.ReviewThreads.Nodes {
		resolved := t.IsResolved
		item := CommentItem{
			ID:       t.ID,
			Kind:     CommentKindReviewThread,
			PRNumber: prNumber,
			Resolved: &resolved,
			Location: &CommentLocation{
				Path:              t.Path,
				Line:              t.Line,
				OriginalLine:      t.OriginalLine,
				StartLine:         t.StartLine,
				OriginalStartLine: t.OriginalStartLine,
				Side:              t.DiffSide,
				StartSide:         t.StartDiffSide,
				Outdated:          t.IsOutdated,
			},
		}
		for _, c := range t.Comments.Nodes {
			reply := CommentItem{
				ID:        c.ID,
				Kind:      CommentKindReviewComment,
				PRNumber:  prNumber,
				Author:    login(c.Author),
				Body:      c.Body,
				URL:       c.URL,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
				Location:  reviewCommentLocation(c),
			}
			if item.Author == "" {
				item.Author = reply.Author
				item.Body = reply.Body
				item.URL = reply.URL
				item.CreatedAt = reply.CreatedAt
				item.UpdatedAt = reply.UpdatedAt
			}
			item.Replies = append(item.Replies, reply)
		}
		items = append(items, item)
	}
	return items, nil
}

func runGHJSON(args []string) ([]byte, error) {
	out, errOut, err := shell.Run(args, shell.RunOpts{Quiet: true, Check: true})
	if err == nil {
		return out, nil
	}
	msg := strings.TrimSpace(string(errOut))
	if msg == "" {
		msg = err.Error()
	}
	wrapped := fmt.Errorf("%w: %s", err, msg)
	if looksLikeAuthFailure(msg) {
		return nil, &AuthError{Err: wrapped}
	}
	return nil, wrapped
}

func looksLikeAuthFailure(msg string) bool {
	msg = strings.ToLower(msg)
	for _, needle := range []string{
		"authentication",
		"authenticate",
		"authorization",
		"not logged in",
		"login required",
		"http 401",
		"http 403",
		"must authenticate",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func login(a *ghAuthor) string {
	if a == nil {
		return ""
	}
	return a.Login
}

func reviewCommentLocation(c ghReviewComment) *CommentLocation {
	if c.Path == "" && c.Line == 0 && c.OriginalLine == 0 && c.StartLine == 0 && c.OriginalStartLine == 0 {
		return nil
	}
	return &CommentLocation{
		Path:              c.Path,
		Line:              c.Line,
		OriginalLine:      c.OriginalLine,
		StartLine:         c.StartLine,
		OriginalStartLine: c.OriginalStartLine,
	}
}

func parsePullURL(raw string) (owner, repo string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

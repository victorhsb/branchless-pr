package pr

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	CheckProviderGitHubActions = "github_actions"
	CheckProviderGitHubStatus  = "github_status"
	CheckProviderGitHubCheck   = "github_check"
	CheckProviderUnknown       = "unknown"

	RequiredTrue    = "true"
	RequiredFalse   = "false"
	RequiredUnknown = "unknown"
)

// Check is the normalized, agent-facing representation of one GitHub check run
// or status context.
type Check struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	ProviderID  string `json:"provider_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	CheckRunID  string `json:"check_run_id,omitempty"`
	Workflow    string `json:"workflow,omitempty"`
	Suite       string `json:"suite,omitempty"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion,omitempty"`
	Required    string `json:"required"`
	URL         string `json:"url,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// Failed reports whether a check conclusion represents a failing outcome.
func (c Check) Failed() bool {
	switch strings.ToLower(c.Conclusion) {
	case "failure", "error", "timed_out", "startup_failure", "action_required":
		return true
	default:
		return false
	}
}

// CommentSummary is a bounded, PR-level review attention summary.
type CommentSummary struct {
	ConversationCount  int              `json:"conversation_count"`
	ReviewCount        int              `json:"review_count"`
	ReviewCommentCount int              `json:"review_comment_count"`
	ReviewThreadCount  int              `json:"review_thread_count,omitempty"`
	RequestedChanges   int              `json:"requested_changes"`
	Snippets           []CommentSnippet `json:"snippets,omitempty"`
	InspectCommand     string           `json:"inspect_command,omitempty"`
}

// CommentSnippet contains a short preview of a comment or review.
type CommentSnippet struct {
	Kind   string `json:"kind"`
	Author string `json:"author,omitempty"`
	Body   string `json:"body,omitempty"`
	URL    string `json:"url,omitempty"`
}

// PullRequestChecks contains normalized checks fetched for one PR.
type PullRequestChecks struct {
	Number         int            `json:"number"`
	URL            string         `json:"url"`
	HeadRefName    string         `json:"head_ref_name,omitempty"`
	BaseRefName    string         `json:"base_ref_name,omitempty"`
	HeadSHA        string         `json:"head_sha,omitempty"`
	Checks         []Check        `json:"checks"`
	CommentSummary CommentSummary `json:"comment_summary"`
	Warnings       []string       `json:"warnings,omitempty"`
}

// FetchChecks fetches read-only PR checks and lightweight comment summary data.
func FetchChecks(prRef string) (*PullRequestChecks, error) {
	out, err := runGHJSON([]string{
		"gh", "pr", "view", prRef,
		"--json", "number,url,headRefName,baseRefName,headRefOid,statusCheckRollup,comments,reviews",
	})
	if err != nil {
		return nil, fmt.Errorf("gh pr view checks %s: %w", prRef, err)
	}
	return ParsePRChecks(out)
}

type ghPRChecks struct {
	Number            int               `json:"number"`
	URL               string            `json:"url"`
	HeadRefName       string            `json:"headRefName"`
	BaseRefName       string            `json:"baseRefName"`
	HeadRefOID        string            `json:"headRefOid"`
	StatusCheckRollup []json.RawMessage `json:"statusCheckRollup"`
	Comments          []ghComment       `json:"comments"`
	Reviews           []ghReview        `json:"reviews"`
}

// ParsePRChecks normalizes gh pr view JSON for tests and FetchChecks.
func ParsePRChecks(data []byte) (*PullRequestChecks, error) {
	var raw ghPRChecks
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse pr checks response: %w", err)
	}
	result := &PullRequestChecks{
		Number:      raw.Number,
		URL:         raw.URL,
		HeadRefName: raw.HeadRefName,
		BaseRefName: raw.BaseRefName,
		HeadSHA:     raw.HeadRefOID,
		Checks:      []Check{},
	}
	for _, item := range raw.StatusCheckRollup {
		check, ok := parseStatusCheck(item)
		if ok {
			result.Checks = append(result.Checks, check)
		}
	}
	sort.SliceStable(result.Checks, func(i, j int) bool {
		return result.Checks[i].ID < result.Checks[j].ID
	})
	result.CommentSummary = summarizePRComments(raw.Comments, raw.Reviews)
	return result, nil
}

func parseStatusCheck(data json.RawMessage) (Check, bool) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return Check{}, false
	}
	typename := firstString(m, "__typename", "type")
	name := firstString(m, "name", "context")
	if name == "" {
		name = firstString(m, "title")
	}
	workflow := firstString(m, "workflowName", "workflow", "workflow_name")
	suite := firstString(m, "checkSuite", "suite")
	status := normalizeStatus(firstString(m, "status", "state"))
	conclusion := normalizeConclusion(firstString(m, "conclusion"))
	provider := inferCheckProvider(typename, workflow, m)
	if provider == CheckProviderGitHubStatus && conclusion == "" {
		switch status {
		case "success", "failure", "error":
			conclusion = status
			status = "completed"
		}
	}
	url := firstString(m, "detailsUrl", "detailsURL", "targetUrl", "targetURL", "url")
	providerID := firstID(m, "databaseId", "id")
	checkRunID := firstID(m, "checkRunId", "check_run_id")
	runID := firstID(m, "runId", "run_id")
	required := parseRequired(m)
	check := Check{
		ID:          SemanticCheckID(provider, workflow, suite, name),
		Provider:    provider,
		ProviderID:  providerID,
		RunID:       runID,
		CheckRunID:  checkRunID,
		Workflow:    workflow,
		Suite:       suite,
		Name:        name,
		Status:      status,
		Conclusion:  conclusion,
		Required:    required,
		URL:         url,
		StartedAt:   firstString(m, "startedAt", "started_at"),
		CompletedAt: firstString(m, "completedAt", "completed_at"),
	}
	if check.Name == "" {
		check.Name = check.ID
	}
	return check, true
}

func inferCheckProvider(typename, workflow string, m map[string]any) string {
	typename = strings.ToLower(typename)
	if workflow != "" || strings.Contains(typename, "checkrun") {
		return CheckProviderGitHubActions
	}
	if strings.Contains(typename, "statuscontext") || firstString(m, "context") != "" {
		return CheckProviderGitHubStatus
	}
	if typename != "" {
		return CheckProviderGitHubCheck
	}
	return CheckProviderUnknown
}

func parseRequired(m map[string]any) string {
	for _, key := range []string{"required", "isRequired"} {
		if v, ok := m[key]; ok {
			if b, ok := v.(bool); ok {
				if b {
					return RequiredTrue
				}
				return RequiredFalse
			}
			if s, ok := v.(string); ok {
				switch strings.ToLower(strings.TrimSpace(s)) {
				case "true", "required":
					return RequiredTrue
				case "false", "optional":
					return RequiredFalse
				}
			}
		}
	}
	return RequiredUnknown
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch typed := v.(type) {
		case string:
			return typed
		case map[string]any:
			if s := firstString(typed, "name", "title", "id"); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstID(m map[string]any, keys ...string) string {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch typed := v.(type) {
		case string:
			return typed
		case float64:
			return strconv.FormatInt(int64(typed), 10)
		case int:
			return strconv.Itoa(typed)
		}
	}
	return ""
}

func normalizeStatus(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "unknown"
	}
	return s
}

func normalizeConclusion(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

var semanticIDCleanRe = regexp.MustCompile(`[^a-z0-9._/-]+`)

// SemanticCheckID returns a deterministic, agent-facing check identifier.
func SemanticCheckID(provider, workflow, suite, name string) string {
	provider = slug(provider)
	if provider == "" {
		provider = CheckProviderUnknown
	}
	var parts []string
	if workflow != "" {
		parts = append(parts, workflow)
	} else if suite != "" {
		parts = append(parts, suite)
	}
	if name != "" {
		parts = append(parts, name)
	}
	if len(parts) == 0 {
		return provider + ":unknown"
	}
	for i := range parts {
		parts[i] = slug(parts[i])
	}
	return provider + ":" + strings.Join(parts, ":")
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = semanticIDCleanRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

func summarizePRComments(comments []ghComment, reviews []ghReview) CommentSummary {
	summary := CommentSummary{
		ConversationCount: len(comments),
		ReviewCount:       len(reviews),
		InspectCommand:    "stack-pr comments",
	}
	for _, c := range comments {
		addSnippet(&summary, CommentKindConversation, login(c.Author), c.Body, c.URL)
	}
	for _, r := range reviews {
		if strings.EqualFold(r.State, "CHANGES_REQUESTED") {
			summary.RequestedChanges++
		}
		addSnippet(&summary, CommentKindReview, login(r.Author), r.Body, r.URL)
		summary.ReviewCommentCount += len(r.Comments)
		for _, c := range r.Comments {
			addSnippet(&summary, CommentKindReviewComment, login(c.Author), c.Body, c.URL)
		}
	}
	return summary
}

func addSnippet(summary *CommentSummary, kind, author, body, url string) {
	if len(summary.Snippets) >= 3 {
		return
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	if len(body) > 160 {
		body = strings.TrimSpace(body[:157]) + "..."
	}
	summary.Snippets = append(summary.Snippets, CommentSnippet{
		Kind:   kind,
		Author: author,
		Body:   body,
		URL:    url,
	})
}

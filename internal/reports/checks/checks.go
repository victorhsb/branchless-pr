package checks

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/invocation"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
	"github.com/victorhsb/branchless-pr/internal/stackstate"
)

type AppContext = invocation.AppContext
type CommonArgs = invocation.CommonArgs

type Options struct {
	Format       string
	FailedOnly   bool
	RequiredOnly bool
	PRNumber     int
	Commit       string
	Verbose      bool
}

type Fetcher func(prRef string) (*pr.PullRequestChecks, error)

func Run(app *AppContext, opts Options, w io.Writer) error {
	return RunWithFetcher(app, opts, w, pr.FetchChecks)
}

func RunWithFetcher(app *AppContext, opts Options, w io.Writer, fetch Fetcher) error {
	if opts.Format != "text" && opts.Format != "json" {
		return fmt.Errorf("unknown checks format %q: expected \"text\" or \"json\"", opts.Format)
	}
	if opts.PRNumber < 0 {
		return fmt.Errorf("--pr must be a positive pull request number")
	}
	report, err := Build(app, opts, fetch)
	if err != nil {
		return err
	}
	return Write(w, report, opts.Format, opts.Verbose)
}

type checksReport struct {
	SchemaVersion string                    `json:"schema_version"`
	Command       string                    `json:"command"`
	Repository    string                    `json:"repository"`
	Range         checksRange               `json:"range"`
	Stack         []checksStackEntry        `json:"stack"`
	PullRequests  []checksPullRequestReport `json:"pull_requests"`
	FailedChecks  []failedCheckSummary      `json:"failed_checks"`
}

type checksRange struct {
	Base   string `json:"base"`
	Head   string `json:"head"`
	Remote string `json:"remote"`
	Target string `json:"target"`
}

type checksStackEntry struct {
	Index      int    `json:"index"`
	Commit     string `json:"commit"`
	ShortSHA   string `json:"short_sha"`
	Title      string `json:"title"`
	HeadBranch string `json:"head_branch"`
	BaseBranch string `json:"base_branch"`
	PRURL      string `json:"pr_url,omitempty"`
	PRNumber   int    `json:"pr_number,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type checksPullRequestReport struct {
	Index          int               `json:"index"`
	Commit         string            `json:"commit"`
	ShortSHA       string            `json:"short_sha"`
	Title          string            `json:"title"`
	HeadBranch     string            `json:"head_branch"`
	BaseBranch     string            `json:"base_branch"`
	PRURL          string            `json:"pr_url,omitempty"`
	PRNumber       int               `json:"pr_number,omitempty"`
	Status         string            `json:"status"`
	Error          string            `json:"error,omitempty"`
	Warnings       []string          `json:"warnings,omitempty"`
	Checks         []pr.Check        `json:"checks"`
	CommentSummary pr.CommentSummary `json:"comment_summary,omitempty"`
}

type failedCheckSummary struct {
	ID         string `json:"id"`
	PRNumber   int    `json:"pr_number,omitempty"`
	PRURL      string `json:"pr_url,omitempty"`
	Commit     string `json:"commit"`
	ShortSHA   string `json:"short_sha"`
	Title      string `json:"title"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	Workflow   string `json:"workflow,omitempty"`
	Conclusion string `json:"conclusion"`
	URL        string `json:"url,omitempty"`
}

func Build(app *AppContext, opts Options, fetch Fetcher) (*checksReport, error) {
	st, err := stackstate.Load(stackstate.Args{
		Base:               app.Args.Base,
		Head:               app.Args.Head,
		Remote:             app.Args.Remote,
		Target:             app.Args.Target,
		BranchNameTemplate: app.Args.BranchNameTemplate,
		Username:           app.Username,
		OrigBranch:         app.OrigBranch,
	})
	if err != nil {
		return nil, err
	}

	filtered, err := filterChecksStack(st, opts)
	if err != nil {
		return nil, err
	}

	report := &checksReport{
		SchemaVersion: "1",
		Command:       "stack-pr checks",
		Repository:    app.RepoRoot,
		Range: checksRange{
			Base:   app.Args.Base,
			Head:   app.Args.Head,
			Remote: app.Args.Remote,
			Target: app.Args.Target,
		},
	}

	for i, e := range filtered {
		entry := newChecksPullRequestReport(stackstate.Index(st, e), e)
		if entry.Index == 0 {
			entry.Index = i + 1
		}
		if !e.HasPR() {
			entry.Status = "missing"
			entry.Error = "missing PR metadata"
			report.Stack = append(report.Stack, checksStackEntryFromPR(entry))
			report.PullRequests = append(report.PullRequests, entry)
			continue
		}

		fetched, err := fetch(e.PR())
		if err != nil {
			if pr.IsAuthError(err) {
				return nil, err
			}
			entry.Status = "failed"
			entry.Error = err.Error()
			report.Stack = append(report.Stack, checksStackEntryFromPR(entry))
			report.PullRequests = append(report.PullRequests, entry)
			continue
		}

		entry.PRNumber = fetched.Number
		if fetched.URL != "" {
			entry.PRURL = fetched.URL
		}
		if fetched.HeadRefName != "" {
			entry.HeadBranch = fetched.HeadRefName
		}
		if fetched.BaseRefName != "" {
			entry.BaseBranch = fetched.BaseRefName
		}
		entry.Warnings = append(entry.Warnings, fetched.Warnings...)
		entry.CommentSummary = fetched.CommentSummary
		entry.Checks = filterChecks(fetched.Checks, opts)
		if opts.FailedOnly && len(entry.Checks) == 0 {
			continue
		}
		if len(entry.Checks) == 0 {
			entry.Status = "empty"
		} else {
			entry.Status = "fetched"
		}
		report.Stack = append(report.Stack, checksStackEntryFromPR(entry))
		report.PullRequests = append(report.PullRequests, entry)
		for _, check := range entry.Checks {
			if check.Failed() {
				report.FailedChecks = append(report.FailedChecks, newFailedCheckSummary(entry, check))
			}
		}
	}

	return report, nil
}

func filterChecksStack(st stack.Stack, opts Options) (stack.Stack, error) {
	var out stack.Stack
	commitFilter := strings.TrimSpace(opts.Commit)
	for _, e := range st {
		if opts.PRNumber != 0 {
			n, err := e.PRNumber()
			if err != nil || n != opts.PRNumber {
				continue
			}
		}
		if commitFilter != "" && !strings.HasPrefix(e.Commit.SHA, commitFilter) {
			continue
		}
		out = append(out, e)
	}
	if opts.PRNumber != 0 && len(out) == 0 {
		return nil, fmt.Errorf("no stack entry is associated with pull request #%d", opts.PRNumber)
	}
	if commitFilter != "" {
		if len(out) == 0 {
			return nil, fmt.Errorf("no stack entry matches commit %q", commitFilter)
		}
		if len(out) > 1 {
			return nil, fmt.Errorf("commit %q matches multiple stack entries; use a longer SHA", commitFilter)
		}
	}
	return out, nil
}

func filterChecks(checks []pr.Check, opts Options) []pr.Check {
	out := make([]pr.Check, 0, len(checks))
	for _, check := range checks {
		if opts.RequiredOnly && check.Required != pr.RequiredTrue {
			continue
		}
		if opts.FailedOnly && !check.Failed() {
			continue
		}
		out = append(out, check)
	}
	return out
}

func newChecksPullRequestReport(index int, e *stack.Entry) checksPullRequestReport {
	entry := checksPullRequestReport{
		Index:      index,
		Commit:     e.Commit.SHA,
		ShortSHA:   e.Commit.ShortSHA(),
		Title:      e.Commit.Title,
		HeadBranch: stackstate.SafeHead(e),
		BaseBranch: e.Base(),
		Checks:     []pr.Check{},
	}
	if e.HasPR() {
		entry.PRURL = e.PR()
		if n, err := e.PRNumber(); err == nil {
			entry.PRNumber = n
		}
	}
	return entry
}

func checksStackEntryFromPR(entry checksPullRequestReport) checksStackEntry {
	return checksStackEntry{
		Index:      entry.Index,
		Commit:     entry.Commit,
		ShortSHA:   entry.ShortSHA,
		Title:      entry.Title,
		HeadBranch: entry.HeadBranch,
		BaseBranch: entry.BaseBranch,
		PRURL:      entry.PRURL,
		PRNumber:   entry.PRNumber,
		Status:     entry.Status,
		Error:      entry.Error,
	}
}

func newFailedCheckSummary(entry checksPullRequestReport, check pr.Check) failedCheckSummary {
	return failedCheckSummary{
		ID:         check.ID,
		PRNumber:   entry.PRNumber,
		PRURL:      entry.PRURL,
		Commit:     entry.Commit,
		ShortSHA:   entry.ShortSHA,
		Title:      entry.Title,
		Name:       check.Name,
		Provider:   check.Provider,
		Workflow:   check.Workflow,
		Conclusion: check.Conclusion,
		URL:        check.URL,
	}
}

func Write(w io.Writer, report *checksReport, format string, verbose bool) error {
	switch format {
	case "text":
		writeChecksText(w, report, verbose)
		return nil
	case "json":
		payload, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(payload))
		return err
	default:
		return fmt.Errorf("unknown checks format %q: expected \"text\" or \"json\"", format)
	}
}

// -- summary-first text output -------------------------------------------------

type checkBucket string

const (
	bucketPassing    checkBucket = "passing"
	bucketFailing    checkBucket = "failing"
	bucketInProgress checkBucket = "in-progress"
	bucketPending    checkBucket = "pending"
	bucketSkipped    checkBucket = "skipped"
	bucketUnknown    checkBucket = "unknown"
)

// bucketPriority returns a numeric priority where lower = more actionable.
func bucketPriority(b checkBucket) int {
	switch b {
	case bucketFailing:
		return 0
	case bucketInProgress:
		return 1
	case bucketPending:
		return 2
	case bucketUnknown:
		return 3
	case bucketPassing:
		return 4
	case bucketSkipped:
		return 5
	default:
		return 6
	}
}

func classifyCheck(c pr.Check) checkBucket {
	if c.Failed() {
		return bucketFailing
	}
	switch strings.ToLower(c.Conclusion) {
	case "success":
		return bucketPassing
	case "skipped", "neutral", "cancelled":
		return bucketSkipped
	case "action_required":
		return bucketUnknown
	}
	switch strings.ToLower(c.Status) {
	case "completed":
		if strings.ToLower(c.Conclusion) == "success" {
			return bucketPassing
		}
		return bucketUnknown
	case "in_progress":
		return bucketInProgress
	case "queued", "waiting", "pending":
		return bucketPending
	default:
		return bucketUnknown
	}
}

// visibleCheckIdentity returns the key used for deduplication in summary text.
func visibleCheckIdentity(c pr.Check) string {
	if c.ID != "" {
		return c.ID
	}
	if c.Name != "" {
		return c.Name
	}
	return c.Provider + ":unknown"
}

// collapsedCheck represents a deduplicated visible check for default text.
type collapsedCheck struct {
	Identity string
	Name     string
	Bucket   checkBucket
	Count    int
	URL      string
}

// collapseVisibleChecks groups checks by visible identity and keeps the
// most actionable bucket per group. It preserves deterministic order.
func collapseVisibleChecks(checks []pr.Check) []collapsedCheck {
	if len(checks) == 0 {
		return nil
	}
	// Order of first appearance.
	order := make([]string, 0, len(checks))
	seen := make(map[string]bool)
	groups := make(map[string]*collapsedCheck)
	for _, c := range checks {
		id := visibleCheckIdentity(c)
		if !seen[id] {
			seen[id] = true
			order = append(order, id)
			groups[id] = &collapsedCheck{
				Identity: id,
				Name:     c.Name,
				Bucket:   classifyCheck(c),
				Count:    0,
				URL:      c.URL,
			}
		}
		g := groups[id]
		g.Count++
		b := classifyCheck(c)
		if bucketPriority(b) < bucketPriority(g.Bucket) {
			g.Bucket = b
			g.URL = c.URL
		}
	}
	out := make([]collapsedCheck, 0, len(order))
	for _, id := range order {
		out = append(out, *groups[id])
	}
	return out
}

// prCheckSummary computes counts per bucket and returns the dominant state.
type prCheckSummary struct {
	Total       int
	Passing     int
	Failing     int
	InProgress  int
	Pending     int
	Skipped     int
	Unknown     int
	FailedIDs   []string
	FailedNames []string
}

func summarizePRChecks(checks []pr.Check) prCheckSummary {
	var s prCheckSummary
	s.Total = len(checks)
	seenFailed := make(map[string]bool)
	for _, c := range checks {
		switch classifyCheck(c) {
		case bucketPassing:
			s.Passing++
		case bucketFailing:
			s.Failing++
			fid := visibleCheckIdentity(c)
			if !seenFailed[fid] {
				seenFailed[fid] = true
				s.FailedIDs = append(s.FailedIDs, fid)
				if c.Name != "" {
					s.FailedNames = append(s.FailedNames, c.Name)
				} else {
					s.FailedNames = append(s.FailedNames, fid)
				}
			}
		case bucketInProgress:
			s.InProgress++
		case bucketPending:
			s.Pending++
		case bucketSkipped:
			s.Skipped++
		default:
			s.Unknown++
		}
	}
	return s
}

// coverageStats aggregates stack-level coverage.
type coverageStats struct {
	StackSize          int
	WithPRMetadata     int
	MissingMetadata    int
	UnreadablePRs      int
	ActivePRFilter     int
	ActiveCommitFilter int
}

func computeCoverage(entries []checksPullRequestReport, stackSize int, prFilter int, commitFilter string) coverageStats {
	var s coverageStats
	s.StackSize = stackSize
	s.ActivePRFilter = prFilter
	if commitFilter != "" {
		s.ActiveCommitFilter = 1
	}
	for _, e := range entries {
		switch e.Status {
		case "missing":
			s.MissingMetadata++
		case "failed":
			s.UnreadablePRs++
		default:
			if e.PRNumber != 0 || e.PRURL != "" {
				s.WithPRMetadata++
			}
		}
	}
	return s
}

func writeChecksText(w io.Writer, report *checksReport, verbose bool) {
	fmt.Fprintf(w, "# stack-pr checks\n\n")
	fmt.Fprintf(w, "Range: `%s..%s` (%s/%s)\n\n", report.Range.Base, report.Range.Head, report.Range.Remote, report.Range.Target)

	if len(report.PullRequests) == 0 {
		fmt.Fprintln(w, "No stack entries found.")
		return
	}

	// Stack coverage summary
	cov := computeCoverage(report.PullRequests, len(report.Stack), 0, "")
	fmt.Fprintf(w, "Stack: %d entr", cov.StackSize)
	if cov.StackSize == 1 {
		fmt.Fprint(w, "y")
	} else {
		fmt.Fprint(w, "ies")
	}
	fmt.Fprintf(w, ", %d with PR metadata", cov.WithPRMetadata)
	if cov.MissingMetadata > 0 {
		fmt.Fprintf(w, ", %d missing metadata", cov.MissingMetadata)
	}
	if cov.UnreadablePRs > 0 {
		fmt.Fprintf(w, ", %d unreadable", cov.UnreadablePRs)
	}
	if cov.ActivePRFilter != 0 || cov.ActiveCommitFilter != 0 {
		fmt.Fprint(w, " (filtered")
		if cov.ActivePRFilter != 0 {
			fmt.Fprintf(w, " --pr=%d", cov.ActivePRFilter)
		}
		if cov.ActiveCommitFilter != 0 {
			fmt.Fprint(w, " --commit")
		}
		fmt.Fprint(w, ")")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)

	// Failed checks section (prominent)
	if len(report.FailedChecks) > 0 {
		fmt.Fprintln(w, "## Failed checks")
		fmt.Fprintln(w)
		for _, failed := range report.FailedChecks {
			prLabel := formatPRLabel(failed.PRNumber, failed.PRURL)
			fmt.Fprintf(w, "- `%s` on %s `%s`: %s", failed.ID, prLabel, failed.ShortSHA, failed.Conclusion)
			if failed.URL != "" {
				fmt.Fprintf(w, " - %s", failed.URL)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}

	totalChecks := 0
	for _, entry := range report.PullRequests {
		totalChecks += len(entry.Checks)
	}
	if totalChecks == 0 {
		fmt.Fprintf(w, "No checks were found across %d stack entr", len(report.Stack))
		if len(report.Stack) == 1 {
			fmt.Fprint(w, "y")
		} else {
			fmt.Fprint(w, "ies")
		}
		fmt.Fprintf(w, " and %d pull request", countKnownCheckPRs(report.PullRequests))
		if countKnownCheckPRs(report.PullRequests) == 1 {
			fmt.Fprint(w, ".\n\n")
		} else {
			fmt.Fprint(w, "s.\n\n")
		}
	}

	for _, entry := range report.PullRequests {
		writeChecksEntryText(w, entry, verbose)
	}
}

func countKnownCheckPRs(entries []checksPullRequestReport) int {
	count := 0
	for _, entry := range entries {
		if entry.PRNumber != 0 || entry.PRURL != "" {
			count++
		}
	}
	return count
}

func writeChecksEntryText(w io.Writer, entry checksPullRequestReport, verbose bool) {
	prLabel := formatPRLabel(entry.PRNumber, entry.PRURL)
	fmt.Fprintf(w, "## %d. %s `%s` (%s)\n\n", entry.Index, entry.Title, entry.ShortSHA, prLabel)
	fmt.Fprintf(w, "- Head: `%s`\n- Base: `%s`\n", entry.HeadBranch, entry.BaseBranch)
	if entry.PRURL != "" {
		fmt.Fprintf(w, "- PR: %s\n", entry.PRURL)
	}
	if entry.Error != "" {
		fmt.Fprintf(w, "- Warning: %s\n", entry.Error)
	}
	for _, warning := range entry.Warnings {
		fmt.Fprintf(w, "- Warning: %s\n", warning)
	}
	writeCommentSummaryText(w, entry.CommentSummary)

	if entry.Status == "missing" || entry.Status == "failed" {
		fmt.Fprintln(w)
		return
	}

	if len(entry.Checks) == 0 {
		fmt.Fprintln(w, "\nNo checks.")
		fmt.Fprintln(w)
		return
	}

	// Per-PR roll-up
	summary := summarizePRChecks(entry.Checks)
	writePRCheckRollUp(w, summary)

	// In verbose mode, also render every raw check.
	if verbose {
		fmt.Fprintln(w, "\nChecks:")
		for _, check := range entry.Checks {
			writeCheckLine(w, check, true)
		}
		fmt.Fprintln(w)
		return
	}

	// Default mode: collapsed summary of visible checks.
	collapsed := collapseVisibleChecks(entry.Checks)
	if len(collapsed) > 0 {
		fmt.Fprintln(w, "\nChecks:")
		for _, c := range collapsed {
			marker := string(c.Bucket)
			if c.Count > 1 {
				fmt.Fprintf(w, "- %s `%s` %s (%d)\n", marker, c.Identity, c.Name, c.Count)
			} else {
				fmt.Fprintf(w, "- %s `%s` %s\n", marker, c.Identity, c.Name)
			}
			if c.URL != "" {
				fmt.Fprintf(w, "  - %s\n", c.URL)
			}
		}
		fmt.Fprintln(w)
	}
}

func writePRCheckRollUp(w io.Writer, s prCheckSummary) {
	if s.Total == 0 {
		return
	}
	parts := make([]string, 0, 6)
	if s.Failing > 0 {
		parts = append(parts, fmt.Sprintf("%d failing", s.Failing))
	}
	if s.InProgress > 0 {
		parts = append(parts, fmt.Sprintf("%d in-progress", s.InProgress))
	}
	if s.Pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", s.Pending))
	}
	if s.Passing > 0 {
		parts = append(parts, fmt.Sprintf("%d passing", s.Passing))
	}
	if s.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", s.Skipped))
	}
	if s.Unknown > 0 {
		parts = append(parts, fmt.Sprintf("%d unknown", s.Unknown))
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, "\nRoll-up: %s / %d checks\n", strings.Join(parts, ", "), s.Total)
	}
	if len(s.FailedNames) > 0 {
		fmt.Fprintf(w, "Failed: %s\n", strings.Join(s.FailedNames, ", "))
	}
}

func writeCheckLine(w io.Writer, check pr.Check, verbose bool) {
	marker := "ok"
	if check.Failed() {
		marker = "failed"
	} else if check.Conclusion != "" {
		marker = check.Conclusion
	} else if check.Status != "" {
		marker = check.Status
	}
	fmt.Fprintf(w, "- %s `%s` %s", marker, check.ID, check.Name)
	if verbose || (check.Required != "" && check.Required != pr.RequiredUnknown) {
		fmt.Fprintf(w, " (required: %s)", check.Required)
	}
	if check.URL != "" {
		fmt.Fprintf(w, " - %s", check.URL)
	}
	fmt.Fprintln(w)
}

func writeCommentSummaryText(w io.Writer, summary pr.CommentSummary) {
	total := summary.ConversationCount + summary.ReviewCount + summary.ReviewCommentCount + summary.ReviewThreadCount
	if total == 0 && summary.RequestedChanges == 0 {
		return
	}
	fmt.Fprintf(w, "- Comments: %d conversation, %d reviews, %d review comments", summary.ConversationCount, summary.ReviewCount, summary.ReviewCommentCount)
	if summary.ReviewThreadCount != 0 {
		fmt.Fprintf(w, ", %d review threads", summary.ReviewThreadCount)
	}
	if summary.RequestedChanges != 0 {
		fmt.Fprintf(w, ", %d requested changes", summary.RequestedChanges)
	}
	if summary.InspectCommand != "" {
		fmt.Fprintf(w, " (full: `%s`)", summary.InspectCommand)
	}
	fmt.Fprintln(w)
	for _, snippet := range summary.Snippets {
		author := snippet.Author
		if author == "" {
			author = "unknown"
		}
		fmt.Fprintf(w, "  - %s by `%s`: %s\n", snippet.Kind, author, snippet.Body)
	}
}

func formatPRLabel(number int, url string) string {
	if number != 0 {
		return "#" + strconv.Itoa(number)
	}
	if url != "" {
		return url
	}
	return "no PR"
}

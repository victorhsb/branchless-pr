package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

type checksOptions struct {
	format       string
	failedOnly   bool
	requiredOnly bool
	prNumber     int
	commit       string
}

type checksFetcher func(prRef string) (*pr.PullRequestChecks, error)

func checksCmd() *cobra.Command {
	opts := checksOptions{format: "text"}

	cmd := &cobra.Command{
		Use:   "checks",
		Short: "Report CI and review-attention state across the current stack.",
		Long:  `Read-only report of all GitHub checks for the current stack, with stable failed-check IDs and brief comment summaries. Use stack-pr comments for full comment inspection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			return runChecks(app, opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.format, "format", "text", `Output format: "text" or "json"`)
	cmd.Flags().BoolVar(&opts.failedOnly, "failed-only", false, "Show only failed checks with their stack context")
	cmd.Flags().BoolVar(&opts.requiredOnly, "required-only", false, "Show only checks known to be required")
	cmd.Flags().IntVar(&opts.prNumber, "pr", 0, "Only show the stack entry associated with this pull request number")
	cmd.Flags().StringVar(&opts.commit, "commit", "", "Only show the stack entry matching this full or unambiguous abbreviated commit SHA")
	return cmd
}

func runChecks(app *AppContext, opts checksOptions, w io.Writer) error {
	return runChecksWithFetcher(app, opts, w, pr.FetchChecks)
}

func runChecksWithFetcher(app *AppContext, opts checksOptions, w io.Writer, fetch checksFetcher) error {
	if opts.format != "text" && opts.format != "json" {
		return fmt.Errorf("unknown checks format %q: expected \"text\" or \"json\"", opts.format)
	}
	if opts.prNumber < 0 {
		return fmt.Errorf("--pr must be a positive pull request number")
	}
	report, err := buildChecksReport(app, opts, fetch)
	if err != nil {
		return err
	}
	return writeChecksReport(w, report, opts.format)
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

func buildChecksReport(app *AppContext, opts checksOptions, fetch checksFetcher) (*checksReport, error) {
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
	if err != nil {
		return nil, err
	}
	for _, e := range st {
		e.ReadMetadata()
	}
	if !st.IsEmpty() {
		tmpl := stack.ParseTemplate(app.Args.BranchNameTemplate)
		if err := st.AssignHeads(tmpl, app.Username, app.OrigBranch, app.Args.Remote); err != nil {
			return nil, err
		}
		st.AssignBases(app.Args.Target)
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
		entry := newChecksPullRequestReport(stackIndex(st, e), e)
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
		if opts.failedOnly && len(entry.Checks) == 0 {
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

func filterChecksStack(st stack.Stack, opts checksOptions) (stack.Stack, error) {
	var out stack.Stack
	commitFilter := strings.TrimSpace(opts.commit)
	for _, e := range st {
		if opts.prNumber != 0 {
			n, err := e.PRNumber()
			if err != nil || n != opts.prNumber {
				continue
			}
		}
		if commitFilter != "" && !strings.HasPrefix(e.Commit.SHA, commitFilter) {
			continue
		}
		out = append(out, e)
	}
	if opts.prNumber != 0 && len(out) == 0 {
		return nil, fmt.Errorf("no stack entry is associated with pull request #%d", opts.prNumber)
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

func filterChecks(checks []pr.Check, opts checksOptions) []pr.Check {
	out := make([]pr.Check, 0, len(checks))
	for _, check := range checks {
		if opts.requiredOnly && check.Required != pr.RequiredTrue {
			continue
		}
		if opts.failedOnly && !check.Failed() {
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
		HeadBranch: safeHead(e),
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

func stackIndex(st stack.Stack, entry *stack.Entry) int {
	for i, e := range st {
		if e == entry {
			return i + 1
		}
	}
	return 0
}

func writeChecksReport(w io.Writer, report *checksReport, format string) error {
	switch format {
	case "text":
		writeChecksText(w, report)
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

func writeChecksText(w io.Writer, report *checksReport) {
	fmt.Fprintf(w, "# stack-pr checks\n\n")
	fmt.Fprintf(w, "Range: `%s..%s` (%s/%s)\n\n", report.Range.Base, report.Range.Head, report.Range.Remote, report.Range.Target)
	if len(report.PullRequests) == 0 {
		fmt.Fprintln(w, "No stack entries found.")
		return
	}

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
		fmt.Fprintf(w, "No checks were found across %d stack entries and %d pull requests.\n\n", len(report.Stack), countKnownCheckPRs(report.PullRequests))
	}

	for _, entry := range report.PullRequests {
		writeChecksEntryText(w, entry)
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

func writeChecksEntryText(w io.Writer, entry checksPullRequestReport) {
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
	if len(entry.Checks) == 0 {
		fmt.Fprintln(w, "\nNo checks.")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w, "\nChecks:")
	for _, check := range entry.Checks {
		marker := "ok"
		if check.Failed() {
			marker = "failed"
		} else if check.Conclusion != "" {
			marker = check.Conclusion
		} else if check.Status != "" {
			marker = check.Status
		}
		fmt.Fprintf(w, "- %s `%s` %s", marker, check.ID, check.Name)
		if check.Required != "" {
			fmt.Fprintf(w, " (required: %s)", check.Required)
		}
		if check.URL != "" {
			fmt.Fprintf(w, " - %s", check.URL)
		}
		fmt.Fprintln(w)
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

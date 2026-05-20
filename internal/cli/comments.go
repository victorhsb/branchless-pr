package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

type commentsOptions struct {
	format         string
	unresolvedOnly bool
	kinds          string
	author         string
	ignoredAuthors []string
}

type commentsFetcher func(prRef string) (*pr.PullRequestComments, error)

func commentsCmd() *cobra.Command {
	opts := commentsOptions{format: "text"}

	cmd := &cobra.Command{
		Use:   "comments",
		Short: "Collect review comments across the current stack.",
		Long:  `Read-only report of pull request conversation comments, reviews, review comments, and review threads for the current stack.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			return runComments(app, opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.format, "format", "text", `Output format: "text" or "json"`)
	cmd.Flags().BoolVar(&opts.unresolvedOnly, "unresolved-only", false, "Show only unresolved or attention-required comments")
	cmd.Flags().StringVar(&opts.kinds, "kind", "", "Comma-separated comment kinds: conversation, review, review_comment, review_thread")
	cmd.Flags().StringVar(&opts.author, "author", "", "Only show comments authored by this GitHub login")
	return cmd
}

func runComments(app *AppContext, opts commentsOptions, w io.Writer) error {
	return runCommentsWithFetcher(app, opts, w, pr.FetchComments)
}

func runCommentsWithFetcher(app *AppContext, opts commentsOptions, w io.Writer, fetch commentsFetcher) error {
	kinds, err := parseCommentKinds(opts.kinds)
	if err != nil {
		return err
	}
	if opts.format != "text" && opts.format != "json" {
		return fmt.Errorf("unknown comments format %q: expected \"text\" or \"json\"", opts.format)
	}
	opts.ignoredAuthors = resolveCommentIgnoredAuthors(app)

	report, err := buildCommentsReport(app, opts, kinds, fetch)
	if err != nil {
		return err
	}
	return writeCommentsReport(w, report, opts.format)
}

type commentsReport struct {
	SchemaVersion string                      `json:"schema_version"`
	Command       string                      `json:"command"`
	Repository    string                      `json:"repository"`
	Range         commentsRange               `json:"range"`
	Stack         []commentsStackEntry        `json:"stack"`
	PullRequests  []commentsPullRequestReport `json:"pull_requests"`
}

type commentsRange struct {
	Base   string `json:"base"`
	Head   string `json:"head"`
	Remote string `json:"remote"`
	Target string `json:"target"`
}

type commentsStackEntry struct {
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

type commentsPullRequestReport struct {
	Index      int              `json:"index"`
	Commit     string           `json:"commit"`
	ShortSHA   string           `json:"short_sha"`
	Title      string           `json:"title"`
	HeadBranch string           `json:"head_branch"`
	BaseBranch string           `json:"base_branch"`
	PRURL      string           `json:"pr_url,omitempty"`
	PRNumber   int              `json:"pr_number,omitempty"`
	Status     string           `json:"status"`
	Error      string           `json:"error,omitempty"`
	Warnings   []string         `json:"warnings,omitempty"`
	Comments   []pr.CommentItem `json:"comments"`
}

func buildCommentsReport(app *AppContext, opts commentsOptions, kinds map[string]bool, fetch commentsFetcher) (*commentsReport, error) {
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

	report := &commentsReport{
		SchemaVersion: "1",
		Command:       "stack-pr comments",
		Repository:    app.RepoRoot,
		Range: commentsRange{
			Base:   app.Args.Base,
			Head:   app.Args.Head,
			Remote: app.Args.Remote,
			Target: app.Args.Target,
		},
	}

	for i, e := range st {
		entry := newCommentsPullRequestReport(i+1, e)
		if !e.HasPR() {
			entry.Status = "missing"
			entry.Error = "missing PR metadata"
			report.Stack = append(report.Stack, stackEntryFromPR(entry))
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
			report.Stack = append(report.Stack, stackEntryFromPR(entry))
			report.PullRequests = append(report.PullRequests, entry)
			continue
		}

		entry.PRNumber = fetched.Number
		if fetched.URL != "" {
			entry.PRURL = fetched.URL
		}
		entry.Warnings = append(entry.Warnings, fetched.Warnings...)
		entry.Comments = filterCommentItems(fetched.Items, opts, kinds)
		if len(entry.Comments) == 0 {
			entry.Status = "empty"
		} else {
			entry.Status = "fetched"
		}
		report.Stack = append(report.Stack, stackEntryFromPR(entry))
		report.PullRequests = append(report.PullRequests, entry)
	}

	return report, nil
}

func newCommentsPullRequestReport(index int, e *stack.Entry) commentsPullRequestReport {
	entry := commentsPullRequestReport{
		Index:      index,
		Commit:     e.Commit.SHA,
		ShortSHA:   e.Commit.ShortSHA(),
		Title:      e.Commit.Title,
		HeadBranch: safeHead(e),
		BaseBranch: e.Base(),
		Comments:   []pr.CommentItem{},
	}
	if e.HasPR() {
		entry.PRURL = e.PR()
		if n, err := e.PRNumber(); err == nil {
			entry.PRNumber = n
		}
	}
	return entry
}

func stackEntryFromPR(entry commentsPullRequestReport) commentsStackEntry {
	return commentsStackEntry{
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

func safeHead(e *stack.Entry) string {
	if e.HasHead() {
		return e.Head()
	}
	return ""
}

func parseCommentKinds(raw string) (map[string]bool, error) {
	allowed := map[string]bool{
		pr.CommentKindConversation:  true,
		pr.CommentKindReview:        true,
		pr.CommentKindReviewComment: true,
		pr.CommentKindReviewThread:  true,
	}
	if strings.TrimSpace(raw) == "" {
		return allowed, nil
	}

	selected := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		kind := strings.TrimSpace(part)
		if kind == "" {
			continue
		}
		if !allowed[kind] {
			return nil, fmt.Errorf("unknown comments kind %q: expected conversation, review, review_comment, or review_thread", kind)
		}
		selected[kind] = true
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("comments kind filter cannot be empty")
	}
	return selected, nil
}

func parseCommentAuthorList(raw string) []string {
	var authors []string
	for _, part := range strings.Split(raw, ",") {
		author := strings.TrimSpace(part)
		if author == "" {
			continue
		}
		authors = append(authors, author)
	}
	return authors
}

func resolveCommentIgnoredAuthors(app *AppContext) []string {
	if app == nil || app.Config == nil {
		return nil
	}
	return parseCommentAuthorList(app.Config.Get("comments", "ignore_authors"))
}

func filterCommentItems(items []pr.CommentItem, opts commentsOptions, kinds map[string]bool) []pr.CommentItem {
	var out []pr.CommentItem
	ignored := ignoredAuthorSet(opts.ignoredAuthors)
	for _, item := range items {
		filtered, ok := filterCommentItem(item, opts, kinds, ignored)
		if ok {
			out = append(out, filtered)
		}
	}
	return out
}

func ignoredAuthorSet(authors []string) map[string]bool {
	if len(authors) == 0 {
		return nil
	}
	ignored := make(map[string]bool, len(authors))
	for _, author := range authors {
		author = strings.TrimSpace(author)
		if author == "" {
			continue
		}
		ignored[strings.ToLower(author)] = true
	}
	return ignored
}

func filterCommentItem(item pr.CommentItem, opts commentsOptions, kinds map[string]bool, ignored map[string]bool) (pr.CommentItem, bool) {
	if !kinds[item.Kind] {
		return pr.CommentItem{}, false
	}
	if opts.unresolvedOnly && !isUnresolvedOrAttentionRequired(item) {
		return pr.CommentItem{}, false
	}
	item, ok := filterIgnoredAuthors(item, ignored)
	if !ok {
		return pr.CommentItem{}, false
	}
	author := strings.TrimSpace(opts.author)
	if author == "" {
		return item, true
	}
	if strings.EqualFold(item.Author, author) {
		item.Replies = filterRepliesByAuthor(item.Replies, author)
		return item, true
	}
	item.Replies = filterRepliesByAuthor(item.Replies, author)
	if len(item.Replies) > 0 {
		return item, true
	}
	return pr.CommentItem{}, false
}

func filterIgnoredAuthors(item pr.CommentItem, ignored map[string]bool) (pr.CommentItem, bool) {
	if len(ignored) == 0 {
		return item, true
	}
	item.Replies = filterRepliesByIgnoredAuthors(item.Replies, ignored)
	if !isIgnoredAuthor(item.Author, ignored) {
		return item, true
	}
	if item.Kind == pr.CommentKindReviewThread && len(item.Replies) > 0 {
		first := item.Replies[0]
		item.Author = first.Author
		item.Body = first.Body
		item.URL = first.URL
		item.CreatedAt = first.CreatedAt
		item.UpdatedAt = first.UpdatedAt
		item.SubmittedAt = first.SubmittedAt
		return item, true
	}
	return pr.CommentItem{}, false
}

func filterRepliesByIgnoredAuthors(replies []pr.CommentItem, ignored map[string]bool) []pr.CommentItem {
	var out []pr.CommentItem
	for _, reply := range replies {
		if !isIgnoredAuthor(reply.Author, ignored) {
			out = append(out, reply)
		}
	}
	return out
}

func isIgnoredAuthor(author string, ignored map[string]bool) bool {
	return ignored[strings.ToLower(strings.TrimSpace(author))]
}

func filterRepliesByAuthor(replies []pr.CommentItem, author string) []pr.CommentItem {
	var out []pr.CommentItem
	for _, reply := range replies {
		if strings.EqualFold(reply.Author, author) {
			out = append(out, reply)
		}
	}
	return out
}

func isUnresolvedOrAttentionRequired(item pr.CommentItem) bool {
	if item.Resolved != nil {
		return !*item.Resolved
	}
	return strings.EqualFold(item.State, "CHANGES_REQUESTED")
}

func writeCommentsReport(w io.Writer, report *commentsReport, format string) error {
	switch format {
	case "text":
		writeCommentsText(w, report)
		return nil
	case "json":
		payload, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(payload))
		return err
	default:
		return fmt.Errorf("unknown comments format %q: expected \"text\" or \"json\"", format)
	}
}

func writeCommentsText(w io.Writer, report *commentsReport) {
	fmt.Fprintf(w, "# stack-pr comments\n\n")
	fmt.Fprintf(w, "Range: `%s..%s` (%s/%s)\n\n", report.Range.Base, report.Range.Head, report.Range.Remote, report.Range.Target)
	if len(report.PullRequests) == 0 {
		fmt.Fprintln(w, "No stack entries found.")
		return
	}

	total := 0
	for _, entry := range report.PullRequests {
		total += len(entry.Comments)
	}
	if total == 0 {
		fmt.Fprintf(w, "No matching comments were found across %d stack entries and %d pull requests.\n\n", len(report.Stack), countKnownPRs(report.PullRequests))
	}

	for _, entry := range report.PullRequests {
		writeCommentsEntryText(w, entry)
	}
}

func countKnownPRs(entries []commentsPullRequestReport) int {
	count := 0
	for _, entry := range entries {
		if entry.PRNumber != 0 || entry.PRURL != "" {
			count++
		}
	}
	return count
}

func writeCommentsEntryText(w io.Writer, entry commentsPullRequestReport) {
	prLabel := "no PR"
	if entry.PRNumber != 0 {
		prLabel = fmt.Sprintf("#%d", entry.PRNumber)
	} else if entry.PRURL != "" {
		prLabel = entry.PRURL
	}
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
	if len(entry.Comments) == 0 {
		fmt.Fprintln(w, "\nNo matching comments.")
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w)
	for _, item := range entry.Comments {
		writeCommentItemText(w, item, "")
	}
}

func writeCommentItemText(w io.Writer, item pr.CommentItem, indent string) {
	label := item.Kind
	if item.Resolved != nil {
		if *item.Resolved {
			label += " resolved"
		} else {
			label += " unresolved"
		}
	}
	if item.State != "" {
		label += " " + item.State
	}
	author := item.Author
	if author == "" {
		author = "unknown"
	}
	fmt.Fprintf(w, "%s- **%s** by `%s`", indent, label, author)
	if item.URL != "" {
		fmt.Fprintf(w, " - %s", item.URL)
	}
	fmt.Fprintln(w)
	if item.Location != nil {
		fmt.Fprintf(w, "%s  - Location: %s\n", indent, formatCommentLocation(item.Location))
	}
	if item.Body != "" {
		for _, line := range strings.Split(item.Body, "\n") {
			fmt.Fprintf(w, "%s  > %s\n", indent, line)
		}
	}
	for _, reply := range item.Replies {
		writeCommentItemText(w, reply, indent+"  ")
	}
}

func formatCommentLocation(loc *pr.CommentLocation) string {
	if loc == nil {
		return ""
	}
	var parts []string
	if loc.Path != "" {
		parts = append(parts, loc.Path)
	}
	if loc.StartLine != 0 && loc.Line != 0 && loc.StartLine != loc.Line {
		parts = append(parts, fmt.Sprintf("lines %d-%d", loc.StartLine, loc.Line))
	} else if loc.Line != 0 {
		parts = append(parts, fmt.Sprintf("line %d", loc.Line))
	} else if loc.OriginalLine != 0 {
		parts = append(parts, fmt.Sprintf("original line %d", loc.OriginalLine))
	}
	if loc.Outdated {
		parts = append(parts, "outdated")
	}
	return strings.Join(parts, ", ")
}

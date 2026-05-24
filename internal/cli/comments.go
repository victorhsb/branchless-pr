package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	commentsreport "github.com/victorhsb/branchless-pr/internal/reports/comments"
)

type commentsOptions = commentsreport.Options
type commentsFetcher = commentsreport.Fetcher

func commentsCmd() *cobra.Command {
	opts := commentsOptions{Format: "text"}

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
	cmd.Flags().StringVar(&opts.Format, "format", "text", `Output format: "text" or "json"`)
	cmd.Flags().BoolVar(&opts.UnresolvedOnly, "unresolved-only", false, "Show only unresolved or attention-required comments")
	cmd.Flags().StringVar(&opts.Kinds, "kind", "", "Comma-separated comment kinds: conversation, review, review_comment, review_thread")
	cmd.Flags().StringVar(&opts.Author, "author", "", "Only show comments authored by this GitHub login")
	return cmd
}

func runComments(app *AppContext, opts commentsOptions, w io.Writer) error {
	return commentsreport.Run(app, opts, w)
}

func runCommentsWithFetcher(app *AppContext, opts commentsOptions, w io.Writer, fetch commentsFetcher) error {
	return commentsreport.RunWithFetcher(app, opts, w, fetch)
}

package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	checksreport "github.com/victorhsb/branchless-pr/internal/reports/checks"
)

type checksOptions = checksreport.Options
type checksFetcher = checksreport.Fetcher

func checksCmd() *cobra.Command {
	opts := checksOptions{Format: "text"}

	cmd := &cobra.Command{
		Use:   "checks",
		Short: "Report CI and review-attention state across the current stack.",
		Long: `Read-only report of GitHub checks across the current stack.

By default the output is summary-first: a compact roll-up per pull request with check counts, failed-check names, and lightweight comment/review counts. Use --verbose to include full per-check detail in text output. Use stack-pr comments for full comment inspection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			return runChecks(app, opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.Format, "format", "text", `Output format: "text" or "json"`)
	cmd.Flags().BoolVar(&opts.FailedOnly, "failed-only", false, "Show only failed checks with their stack context")
	cmd.Flags().BoolVar(&opts.RequiredOnly, "required-only", false, "Show only checks known to be required")
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", false, "Include full per-check detail in text output")
	cmd.Flags().IntVar(&opts.PRNumber, "pr", 0, "Only show the stack entry associated with this pull request number")
	cmd.Flags().StringVar(&opts.Commit, "commit", "", "Only show the stack entry matching this full or unambiguous abbreviated commit SHA")
	return cmd
}

func runChecks(app *AppContext, opts checksOptions, w io.Writer) error {
	return checksreport.Run(app, opts, w)
}

func runChecksWithFetcher(app *AppContext, opts checksOptions, w io.Writer, fetch checksFetcher) error {
	return checksreport.RunWithFetcher(app, opts, w, fetch)
}

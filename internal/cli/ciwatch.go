package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/victorhsb/branchless-pr/internal/reports/ciwatch"
)

func ciWatchCmd() *cobra.Command {
	opts := ciwatch.Options{Interval: time.Minute, Timeout: 15 * time.Minute, Live: true}
	var noLive bool

	cmd := &cobra.Command{
		Use:   "ci-watch",
		Short: "Watch CI checks across the stack until they complete or need action.",
		Long: `Poll GitHub CI for every pull request in the current stack and return when
the agent needs to act or the stack's checks complete.

Live mode (default) streams one complete JSON event per stdout line, flushed
immediately: an initial state record, observed check transitions, revision
changes, and a final event containing the full report. Lines are emitted only
when something changed, which suits Monitor-style background execution that
wakes the agent per stdout line.

Buffered mode (--no-live) emits a single aggregated JSON report when the watch
ends, including on timeout or error. Both modes share the same final
assessment, field names, and exit codes: 0 on success, 1 otherwise.

Every CI check counts, not only branch-protection-required checks. The watch
follows the latest remote head of each PR: a new revision emits a
revision-change notification, replaces the watched checks, and resets the
rolling --timeout deadline. Ordinary check transitions do not reset it.

A revision that reports no checks is accepted only after successful empty
observations span at least two minutes, which lets workflows register after a
push. Pending or running checks keep the watch alive; the first observed
failure, cancellation, or approval requirement returns immediately.

The timeout is bpr's own rolling deadline. A harness that runs bpr in the
background may impose its own independent process timeout.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			if noLive {
				opts.Live = false
			}
			return ciwatch.Run(app, opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().DurationVar(&opts.Interval, "interval", time.Minute, "Polling interval; the first poll runs immediately")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 15*time.Minute, "Rolling stack-wide timeout; resets when any watched PR changes revision")
	cmd.Flags().BoolVar(&opts.Live, "live", true, "Stream one JSON event per stdout line")
	cmd.Flags().BoolVar(&noLive, "no-live", false, "Emit a single aggregated JSON report when the watch ends")
	cmd.MarkFlagsMutuallyExclusive("live", "no-live")
	return cmd
}

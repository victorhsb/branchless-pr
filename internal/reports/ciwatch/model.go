// Package ciwatch implements the bpr ci-watch polling loop and output contract.
//
// The watcher polls GitHub check state for every pull request in the current
// stack and returns when checks complete acceptably, a check fails or needs
// attention, the rolling timeout expires, or observation fails. Live mode
// streams one JSON event per stdout line; buffered mode emits a single final
// JSON report. Both modes share the same final assessment.
package ciwatch

import (
	"time"

	"github.com/victorhsb/branchless-pr/internal/invocation"
	"github.com/victorhsb/branchless-pr/internal/pr"
)

type AppContext = invocation.AppContext
type CommonArgs = invocation.CommonArgs

const (
	schemaVersion = "1"
	commandName   = "bpr ci-watch"

	// noChecksWindow is how long successful empty observations must span
	// before a revision is classified as having no checks.
	noChecksWindow = 2 * time.Minute
)

// Outcomes reported in the final report.
const (
	OutcomeSuccess   = "success"
	OutcomeFailure   = "failure"
	OutcomeAttention = "attention"
	OutcomeTimeout   = "timeout"
	OutcomeError     = "error"
)

// Check-level states.
const (
	StatePassing   = "passing"
	StateFailing   = "failing"
	StateAttention = "attention"
	StatePending   = "pending"
)

// PR-level state used when a revision produced no checks.
const StateNoChecks = "no_checks"

// Live event types.
const (
	eventStart    = "watch_start"
	eventCheck    = "check_transition"
	eventRevision = "revision_change"
	eventNoChecks = "no_checks"
	eventFinal    = "final"
)

// Options controls a watch run.
type Options struct {
	Interval time.Duration
	Timeout  time.Duration
	Live     bool
}

// Fetcher retrieves the current head SHA and checks for one pull request.
type Fetcher func(prRef string) (*pr.PullRequestChecks, error)

// Clock abstracts time so tests can run without real waits.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time        { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }

// ---------------------------------------------------------------------------
// Final report (shared by live and buffered modes)
// ---------------------------------------------------------------------------

// Report is the self-contained final assessment.
type Report struct {
	SchemaVersion   string     `json:"schema_version"`
	Command         string     `json:"command"`
	Repository      string     `json:"repository"`
	Outcome         string     `json:"outcome"`
	Reason          string     `json:"reason"`
	Live            bool       `json:"live"`
	IntervalSeconds int64      `json:"interval_seconds"`
	TimeoutSeconds  int64      `json:"timeout_seconds"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         time.Time  `json:"ended_at"`
	ElapsedSeconds  int64      `json:"elapsed_seconds"`
	Deadline        time.Time  `json:"deadline"`
	PullRequests    []PRReport `json:"pull_requests"`
	NextSteps       []string   `json:"next_steps"`
}

// PRReport is the latest known state of one watched pull request.
type PRReport struct {
	Index      int         `json:"index"`
	PRNumber   int         `json:"pr_number"`
	PRURL      string      `json:"pr_url"`
	Commit     string      `json:"commit"`
	ShortSHA   string      `json:"short_sha"`
	Title      string      `json:"title"`
	HeadBranch string      `json:"head_branch"`
	BaseBranch string      `json:"base_branch"`
	Revision   string      `json:"revision"`
	State      string      `json:"state"`
	Note       string      `json:"note,omitempty"`
	ObservedAt time.Time   `json:"observed_at"`
	Checks     []CheckJSON `json:"checks"`
}

// CheckJSON is the agent-facing view of one observed check execution.
type CheckJSON struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Provider   string    `json:"provider"`
	Workflow   string    `json:"workflow,omitempty"`
	RunID      string    `json:"run_id,omitempty"`
	CheckRunID string    `json:"check_run_id,omitempty"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion,omitempty"`
	State      string    `json:"state"`
	URL        string    `json:"url,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

// ---------------------------------------------------------------------------
// Live events
// ---------------------------------------------------------------------------

type startEvent struct {
	Type            string     `json:"type"`
	SchemaVersion   string     `json:"schema_version"`
	Command         string     `json:"command"`
	Repository      string     `json:"repository"`
	IntervalSeconds int64      `json:"interval_seconds"`
	TimeoutSeconds  int64      `json:"timeout_seconds"`
	Deadline        time.Time  `json:"deadline"`
	StackSize       int        `json:"stack_size"`
	PullRequests    []PRReport `json:"pull_requests"`
	ObservedAt      time.Time  `json:"observed_at"`
}

type stateRef struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion,omitempty"`
	State      string `json:"state"`
}

type checkEvent struct {
	Type       string    `json:"type"`
	Transition string    `json:"transition"` // appeared, changed, rerun
	PRNumber   int       `json:"pr_number"`
	PRURL      string    `json:"pr_url,omitempty"`
	Revision   string    `json:"revision"`
	Check      CheckJSON `json:"check"`
	Previous   *stateRef `json:"previous,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type revisionEvent struct {
	Type             string    `json:"type"`
	PRNumber         int       `json:"pr_number"`
	PRURL            string    `json:"pr_url,omitempty"`
	PreviousRevision string    `json:"previous_revision,omitempty"`
	Revision         string    `json:"revision"`
	Note             string    `json:"note"`
	ObservedAt       time.Time `json:"observed_at"`
}

type noChecksEvent struct {
	Type       string    `json:"type"`
	PRNumber   int       `json:"pr_number"`
	PRURL      string    `json:"pr_url,omitempty"`
	Revision   string    `json:"revision"`
	Note       string    `json:"note"`
	ObservedAt time.Time `json:"observed_at"`
}

type finalEvent struct {
	Type   string  `json:"type"`
	Report *Report `json:"report"`
}

const noChecksNote = "No checks observed for this revision."
const revisionNote = "New revision detected; watching checks for the new revision."

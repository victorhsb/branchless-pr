package ciwatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/shell"
)

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time        { return c.now }
func (c *fakeClock) Sleep(d time.Duration) { c.now = c.now.Add(d) }

type fetchStep struct {
	result *pr.PullRequestChecks
	err    error
}

// scriptFetcher returns scripted responses per PR ref and repeats the last
// step when the script is exhausted.
type scriptFetcher struct {
	t     *testing.T
	steps map[string][]fetchStep
	calls map[string]int
}

func newScriptFetcher(t *testing.T) *scriptFetcher {
	return &scriptFetcher{t: t, steps: map[string][]fetchStep{}, calls: map[string]int{}}
}

func (f *scriptFetcher) on(ref string, steps ...fetchStep) {
	f.steps[ref] = steps
}

func (f *scriptFetcher) fetch(ref string) (*pr.PullRequestChecks, error) {
	steps, ok := f.steps[ref]
	if !ok || len(steps) == 0 {
		f.t.Fatalf("unexpected fetch for %s", ref)
	}
	i := f.calls[ref]
	f.calls[ref]++
	if i >= len(steps) {
		i = len(steps) - 1
	}
	return steps[i].result, steps[i].err
}

const (
	ref7 = "https://github.com/acme/widgets/pull/7"
	ref8 = "https://github.com/acme/widgets/pull/8"
)

func prResult(number int, head string, checks ...pr.Check) *pr.PullRequestChecks {
	return &pr.PullRequestChecks{
		Number:      number,
		URL:         fmt.Sprintf("https://github.com/acme/widgets/pull/%d", number),
		HeadRefName: "alice/stack/" + fmt.Sprint(number),
		BaseRefName: "main",
		HeadSHA:     head,
		Checks:      checks,
	}
}

func mkCheck(id, checkRunID, runID, status, conclusion string) pr.Check {
	return pr.Check{
		ID:         id,
		Name:       id,
		Provider:   pr.CheckProviderGitHubActions,
		CheckRunID: checkRunID,
		RunID:      runID,
		Status:     status,
		Conclusion: conclusion,
		URL:        "https://example.test/checks/" + id,
	}
}

func target(index int, ref string) Target {
	return Target{
		Index:      index,
		Commit:     strings.Repeat(fmt.Sprint(index), 40),
		ShortSHA:   strings.Repeat(fmt.Sprint(index), 12),
		Title:      fmt.Sprintf("Commit %d", index),
		HeadBranch: "alice/stack/" + fmt.Sprint(index),
		BaseBranch: "main",
		PRRef:      ref,
	}
}

func runWatch(opts Options, targets []Target, fetch Fetcher, clock Clock) (*Report, *bytes.Buffer) {
	var out bytes.Buffer
	w := newWatcher(opts, fetch, clock, &out, "/repo")
	return w.Watch(targets), &out
}

func defaultOpts() Options {
	return Options{Interval: time.Minute, Timeout: 15 * time.Minute, Live: true}
}

type parsedEvent struct {
	Type       string
	Transition string
	Raw        map[string]any
}

func parseLiveEvents(t *testing.T, out *bytes.Buffer) []parsedEvent {
	t.Helper()
	text := strings.TrimSpace(out.String())
	if text == "" {
		return nil
	}
	var events []parsedEvent
	for _, line := range strings.Split(text, "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("live output line is not one JSON object: %v\nline: %s", err, line)
		}
		ev := parsedEvent{Raw: m}
		ev.Type, _ = m["type"].(string)
		ev.Transition, _ = m["transition"].(string)
		events = append(events, ev)
	}
	return events
}

func eventTypes(events []parsedEvent) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Type)
	}
	return out
}

func finalFromEvents(t *testing.T, events []parsedEvent) *Report {
	t.Helper()
	if len(events) == 0 || events[len(events)-1].Type != eventFinal {
		t.Fatalf("last event is not final: %v", eventTypes(events))
	}
	data, err := json.Marshal(events[len(events)-1].Raw["report"])
	if err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("final report does not round-trip: %v", err)
	}
	return &report
}

// ---------------------------------------------------------------------------
// Polling transitions
// ---------------------------------------------------------------------------

func TestWatchSuccessAfterPendingTransition(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "in_progress", ""))},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "completed", "success"))},
	)

	report, out := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q, reason %q", report.Outcome, report.Reason)
	}
	if report.ElapsedSeconds != 60 {
		t.Fatalf("elapsed = %d, want 60", report.ElapsedSeconds)
	}
	events := parseLiveEvents(t, out)
	wantTypes := []string{eventStart, eventCheck, eventFinal}
	if !reflect.DeepEqual(eventTypes(events), wantTypes) {
		t.Fatalf("events = %v, want %v", eventTypes(events), wantTypes)
	}
	transition := events[1]
	if transition.Transition != "changed" {
		t.Fatalf("transition = %q, want changed", transition.Transition)
	}
	prev, ok := transition.Raw["previous"].(map[string]any)
	if !ok || prev["status"] != "in_progress" || prev["state"] != StatePending {
		t.Fatalf("previous = %#v", transition.Raw["previous"])
	}
	check := transition.Raw["check"].(map[string]any)
	if _, ok := check["revision"]; ok {
		t.Fatalf("check event should not carry revision inside check: %#v", check)
	}
	if transition.Raw["revision"] != "aaa" || transition.Raw["pr_number"].(float64) != 7 {
		t.Fatalf("event context = %#v", transition.Raw)
	}
	final := finalFromEvents(t, events)
	if final.Outcome != OutcomeSuccess || final.PullRequests[0].Revision != "aaa" || final.PullRequests[0].State != StatePassing {
		t.Fatalf("final = %+v", final.PullRequests[0])
	}
	if final.PullRequests[0].ObservedAt.IsZero() {
		t.Fatal("observed_at missing from PR report")
	}
}

func TestWatchQueuedStraightToSuccessEmitsSingleObservedTransition(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "queued", ""))},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "completed", "success"))},
	)

	report, out := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q", report.Outcome)
	}
	events := parseLiveEvents(t, out)
	transitions := 0
	for _, e := range events {
		if e.Type == eventCheck {
			transitions++
			prev := e.Raw["previous"].(map[string]any)
			if prev["status"] != "queued" {
				t.Fatalf("previous status = %v, want queued (no invented running event)", prev["status"])
			}
		}
	}
	if transitions != 1 {
		t.Fatalf("check transitions = %d, want 1", transitions)
	}
}

func TestWatchFailureReturnsPromptlyWithOtherChecksIncluded(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	pendingBuild := mkCheck("build", "11", "101", "in_progress", "")
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", pendingBuild)},
	)
	fetch.on(ref8,
		fetchStep{result: prResult(8, "bbb", mkCheck("test", "21", "201", "in_progress", ""))},
		fetchStep{result: prResult(8, "bbb", mkCheck("test", "21", "201", "completed", "failure"))},
	)

	opts := defaultOpts()
	opts.Live = false
	report, out := runWatch(opts, []Target{target(1, ref7), target(2, ref8)}, fetch.fetch, clock)
	if report.Outcome != OutcomeFailure {
		t.Fatalf("outcome = %q, reason %q", report.Outcome, report.Reason)
	}
	if report.ElapsedSeconds != 60 {
		t.Fatalf("failure did not return promptly: elapsed = %d", report.ElapsedSeconds)
	}
	// The still-pending check from the other PR must be in the final state.
	if len(report.PullRequests) != 2 {
		t.Fatalf("pull_requests = %d", len(report.PullRequests))
	}
	other := report.PullRequests[0]
	if other.State != StatePending || len(other.Checks) != 1 || other.Checks[0].State != StatePending {
		t.Fatalf("unrelated pending check missing from final report: %+v", other)
	}
	var parsed map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &parsed); err != nil {
		t.Fatalf("buffered output is not one JSON object: %v\n%s", err, out.String())
	}
	joined := strings.Join(report.NextSteps, "\n")
	if !strings.Contains(joined, "Inspect failing check") || !strings.Contains(joined, "gh run view 201 --log-failed") {
		t.Fatalf("next_steps = %v", report.NextSteps)
	}
}

func TestWatchAttentionOnActionRequired(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("deploy", "11", "101", "completed", "action_required"))},
	)

	report, _ := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeAttention {
		t.Fatalf("outcome = %q, want attention (neither passing nor failure)", report.Outcome)
	}
	if report.Reason != "1 check needs attention" {
		t.Fatalf("reason = %q", report.Reason)
	}
	if !strings.Contains(strings.Join(report.NextSteps, "\n"), "action_required") {
		t.Fatalf("next_steps = %v", report.NextSteps)
	}
}

// ---------------------------------------------------------------------------
// Reruns and revision changes
// ---------------------------------------------------------------------------

func TestWatchRerunAppearsAsNewExecution(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "in_progress", ""))},
		// Same name, new check run ID: a rerun, not a new check.
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "12", "102", "queued", ""))},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "12", "102", "completed", "success"))},
	)

	report, out := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q", report.Outcome)
	}
	events := parseLiveEvents(t, out)
	var sawRerun bool
	for _, e := range events {
		if e.Type == eventCheck && e.Transition == "rerun" {
			sawRerun = true
		}
	}
	if !sawRerun {
		t.Fatalf("no rerun transition emitted: %v", eventTypes(events))
	}
	checks := report.PullRequests[0].Checks
	if len(checks) != 1 || checks[0].CheckRunID != "12" {
		t.Fatalf("superseded run reported as current: %+v", checks)
	}
}

func TestWatchRevisionChangeResetsDeadlineAndDropsSupersededChecks(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "in_progress", ""))},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "in_progress", ""))},
		fetchStep{result: prResult(7, "bbb", mkCheck("build", "21", "201", "queued", ""))},
		fetchStep{result: prResult(7, "bbb", mkCheck("build", "21", "201", "in_progress", ""))},
		fetchStep{result: prResult(7, "bbb", mkCheck("build", "21", "201", "completed", "success"))},
	)

	// 3-minute timeout would fire at t=3m without the revision reset at t=2m.
	opts := defaultOpts()
	opts.Timeout = 3 * time.Minute
	report, out := runWatch(opts, []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q, reason %q (rolling deadline should have reset)", report.Outcome, report.Reason)
	}
	if report.ElapsedSeconds != 240 {
		t.Fatalf("elapsed = %d, want 240", report.ElapsedSeconds)
	}
	start := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	if !report.Deadline.Equal(start.Add(2*time.Minute + 3*time.Minute)) {
		t.Fatalf("deadline = %v, want start+5m", report.Deadline)
	}
	events := parseLiveEvents(t, out)
	var revision *parsedEvent
	for i, e := range events {
		if e.Type == eventRevision {
			revision = &events[i]
		}
	}
	if revision == nil {
		t.Fatalf("no revision_change event: %v", eventTypes(events))
	}
	if revision.Raw["previous_revision"] != "aaa" || revision.Raw["revision"] != "bbb" {
		t.Fatalf("revision event = %#v", revision.Raw)
	}
	note, _ := revision.Raw["note"].(string)
	if !strings.Contains(note, "New revision detected") || strings.Contains(note, "rerun triggered") || strings.Contains(note, "triggered") {
		t.Fatalf("revision note = %q", note)
	}
	for _, c := range report.PullRequests[0].Checks {
		if c.CheckRunID == "11" {
			t.Fatalf("superseded revision check reported as current: %+v", c)
		}
	}
	if report.PullRequests[0].Revision != "bbb" {
		t.Fatalf("revision = %q", report.PullRequests[0].Revision)
	}
}

// ---------------------------------------------------------------------------
// No-checks discovery
// ---------------------------------------------------------------------------

func TestWatchNoChecksTwoMinuteRuleAtOneMinuteInterval(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa")},
		fetchStep{result: prResult(7, "aaa")},
		fetchStep{result: prResult(7, "aaa")},
	)

	report, out := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q", report.Outcome)
	}
	if report.ElapsedSeconds != 120 {
		t.Fatalf("elapsed = %d, want the two-minute threshold (120)", report.ElapsedSeconds)
	}
	entry := report.PullRequests[0]
	if entry.State != StateNoChecks || entry.Note == "" {
		t.Fatalf("entry = %+v", entry)
	}
	if !strings.Contains(entry.Note, "No checks observed for this revision") {
		t.Fatalf("note = %q", entry.Note)
	}
	events := parseLiveEvents(t, out)
	if !reflect.DeepEqual(eventTypes(events), []string{eventStart, eventNoChecks, eventFinal}) {
		t.Fatalf("events = %v", eventTypes(events))
	}
}

func TestWatchNoChecksDecisionAtFiveMinuteInterval(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa")},
		fetchStep{result: prResult(7, "aaa")},
	)

	opts := defaultOpts()
	opts.Interval = 5 * time.Minute
	report, _ := runWatch(opts, []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q", report.Outcome)
	}
	if report.ElapsedSeconds != 300 {
		t.Fatalf("elapsed = %d, want first scheduled poll at/after threshold (300)", report.ElapsedSeconds)
	}
}

func TestWatchEmptyThenChecksAppearKeepsWatching(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa")},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "queued", ""))},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "completed", "success"))},
	)

	report, out := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q", report.Outcome)
	}
	for _, e := range parseLiveEvents(t, out) {
		if e.Type == eventNoChecks {
			t.Fatal("no_checks emitted although checks appeared")
		}
	}
	if report.PullRequests[0].State != StatePassing {
		t.Fatalf("state = %q", report.PullRequests[0].State)
	}
}

func TestWatchNoChecksKeepsPollingWhileOtherPRsFinish(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa")},
		fetchStep{result: prResult(7, "aaa")},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "completed", "success"))},
	)
	// PR 8 stays pending through the no_checks classification at t=2m, so the
	// watch must keep polling PR 7 and pick up the check that appears at t=2m.
	fetch.on(ref8,
		fetchStep{result: prResult(8, "bbb", mkCheck("test", "21", "201", "in_progress", ""))},
		fetchStep{result: prResult(8, "bbb", mkCheck("test", "21", "201", "in_progress", ""))},
		fetchStep{result: prResult(8, "bbb", mkCheck("test", "21", "201", "in_progress", ""))},
		fetchStep{result: prResult(8, "bbb", mkCheck("test", "21", "201", "completed", "success"))},
	)

	report, _ := runWatch(defaultOpts(), []Target{target(1, ref7), target(2, ref8)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q, reason %q", report.Outcome, report.Reason)
	}
	entry := report.PullRequests[0]
	if entry.State != StatePassing || len(entry.Checks) != 1 {
		t.Fatalf("check that appeared after empty polls was not tracked: %+v", entry)
	}
}

// ---------------------------------------------------------------------------
// Completion and timeout
// ---------------------------------------------------------------------------

func TestWatchAcceptsSkippedAndNeutral(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa",
			mkCheck("lint", "11", "101", "completed", "skipped"),
			mkCheck("build", "12", "101", "completed", "neutral"),
		)},
	)

	report, _ := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeSuccess {
		t.Fatalf("outcome = %q", report.Outcome)
	}
}

func TestWatchTimeoutReportsOutstandingChecks(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "in_progress", ""))},
	)

	opts := defaultOpts()
	opts.Timeout = 3 * time.Minute
	report, out := runWatch(opts, []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeTimeout {
		t.Fatalf("outcome = %q", report.Outcome)
	}
	if report.ElapsedSeconds != 180 {
		t.Fatalf("elapsed = %d, want 180", report.ElapsedSeconds)
	}
	if !strings.Contains(report.Reason, "timeout") || !strings.Contains(report.Reason, "pending") {
		t.Fatalf("reason = %q", report.Reason)
	}
	joined := strings.Join(report.NextSteps, "\n")
	if !strings.Contains(joined, "Still pending") || !strings.Contains(joined, "bpr ci-watch") {
		t.Fatalf("next_steps = %v", report.NextSteps)
	}
	// Unchanged polls stay quiet: only the initial record and the final event.
	events := parseLiveEvents(t, out)
	if !reflect.DeepEqual(eventTypes(events), []string{eventStart, eventFinal}) {
		t.Fatalf("events = %v", eventTypes(events))
	}
	// Timeout itself is decided by the rolling deadline, so the watch must not
	// have slept past it by a full interval.
	if fetch.calls[ref7] != 3 {
		t.Fatalf("fetches = %d, want 3 (t=0,1m,2m)", fetch.calls[ref7])
	}
}

// ---------------------------------------------------------------------------
// Observation errors
// ---------------------------------------------------------------------------

func TestWatchObservationErrorIsNotNoChecks(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{err: errors.New("gh: api rate limit exceeded")},
	)

	opts := defaultOpts()
	opts.Live = false
	report, out := runWatch(opts, []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeError {
		t.Fatalf("outcome = %q, want error", report.Outcome)
	}
	if strings.Contains(report.Reason, "No checks") {
		t.Fatalf("API error masqueraded as empty checks: %q", report.Reason)
	}
	if !strings.Contains(report.Reason, "rate limit") {
		t.Fatalf("reason = %q", report.Reason)
	}
	var parsed map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &parsed); err != nil {
		t.Fatalf("buffered error output is not one JSON object: %v", err)
	}
}

func TestWatchErrorMidWatchkeepsLastKnownStates(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "in_progress", ""))},
		fetchStep{err: errors.New("gh: network unreachable")},
	)

	opts := defaultOpts()
	opts.Live = false
	report, _ := runWatch(opts, []Target{target(1, ref7)}, fetch.fetch, clock)
	if report.Outcome != OutcomeError {
		t.Fatalf("outcome = %q", report.Outcome)
	}
	entry := report.PullRequests[0]
	if len(entry.Checks) != 1 || entry.Checks[0].State != StatePending {
		t.Fatalf("last known state missing from error report: %+v", entry)
	}
	if entry.ObservedAt.IsZero() {
		t.Fatal("observed_at missing; stale data must be identifiable")
	}
}

// ---------------------------------------------------------------------------
// Buffered vs live agreement
// ---------------------------------------------------------------------------

func TestLiveAndBufferedModesAgreeOnFinalAssessment(t *testing.T) {
	script := func() *scriptFetcher {
		fetch := newScriptFetcher(t)
		fetch.on(ref7,
			fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "in_progress", ""))},
			fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "completed", "failure"))},
		)
		return fetch
	}

	liveReport, liveOut := runWatch(defaultOpts(), []Target{target(1, ref7)}, script().fetch, &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)})
	bufferedOpts := defaultOpts()
	bufferedOpts.Live = false
	bufferedReport, bufferedOut := runWatch(bufferedOpts, []Target{target(1, ref7)}, script().fetch, &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)})

	finalLive := finalFromEvents(t, parseLiveEvents(t, liveOut))
	if !reflect.DeepEqual(finalLive, liveReport) {
		t.Fatal("live final event is not the same report object")
	}
	liveCopy, bufferedCopy := *liveReport, *bufferedReport
	liveCopy.Live, bufferedCopy.Live = false, false
	if !reflect.DeepEqual(liveCopy, bufferedCopy) {
		t.Fatalf("final assessments differ:\nlive: %+v\nbuffered: %+v", liveCopy, bufferedCopy)
	}

	var one map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(bufferedOut.Bytes()), &one); err != nil {
		t.Fatalf("buffered output is not a single JSON object: %v\n%s", err, bufferedOut.String())
	}
	if _, ok := one["outcome"]; !ok {
		t.Fatal("buffered output missing outcome")
	}
}

// ---------------------------------------------------------------------------
// Classification and identity units
// ---------------------------------------------------------------------------

func TestClassifyCheck(t *testing.T) {
	cases := []struct {
		status, conclusion, want string
	}{
		{"completed", "success", StatePassing},
		{"completed", "skipped", StatePassing},
		{"completed", "neutral", StatePassing},
		{"completed", "failure", StateFailing},
		{"completed", "error", StateFailing},
		{"completed", "timed_out", StateFailing},
		{"completed", "startup_failure", StateFailing},
		{"completed", "action_required", StateAttention},
		{"completed", "cancelled", StateAttention},
		{"completed", "stale", StateAttention},
		{"completed", "", StateAttention},
		{"in_progress", "", StatePending},
		{"queued", "", StatePending},
		{"waiting", "", StatePending},
		{"", "", StatePending},
	}
	for _, tc := range cases {
		got := classifyCheck(pr.Check{Status: tc.status, Conclusion: tc.conclusion})
		if got != tc.want {
			t.Errorf("classifyCheck(%q, %q) = %q, want %q", tc.status, tc.conclusion, got, tc.want)
		}
	}
}

func TestExecKeyPrefersExecutionIdentity(t *testing.T) {
	withRun := execKey(pr.Check{ID: "a", RunID: "9"})
	if withRun != "a@9" {
		t.Fatalf("execKey = %q", withRun)
	}
	withCheckRun := execKey(pr.Check{ID: "a", RunID: "9", CheckRunID: "42"})
	if withCheckRun != "a@42" {
		t.Fatalf("execKey = %q", withCheckRun)
	}
	if got := execKey(pr.Check{ID: "a", ProviderID: "7", CheckRunID: "42"}); got != "a@42" {
		t.Fatalf("execKey = %q", got)
	}
	nameOnly := execKey(pr.Check{ID: "a"})
	if nameOnly != "a" {
		t.Fatalf("execKey = %q", nameOnly)
	}
}

// ---------------------------------------------------------------------------
// Stack integration through RunWithFetcher
// ---------------------------------------------------------------------------

func makeWatchTestRepo(t *testing.T, withMetadata []bool) (repoDir, base, head string) {
	t.Helper()
	repoDir = t.TempDir()
	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	writeFile(t, repoDir, "file.txt", "base\n")
	runGit(t, repoDir, "add", "file.txt")
	runGit(t, repoDir, "commit", "-m", "base")
	base = gitOut(t, repoDir, "rev-parse", "HEAD")

	for i, meta := range withMetadata {
		writeFile(t, repoDir, "file.txt", fmt.Sprintf("base\n%d\n", i))
		runGit(t, repoDir, "add", "file.txt")
		if meta {
			runGit(t, repoDir, "commit", "-m", fmt.Sprintf("commit %d", i), "-m",
				fmt.Sprintf("stack-info: PR: https://github.com/acme/widgets/pull/%d, branch: alice/stack/%d", 7+i, 7+i))
		} else {
			runGit(t, repoDir, "commit", "-m", fmt.Sprintf("commit %d", i))
		}
	}
	head = gitOut(t, repoDir, "rev-parse", "HEAD")

	bare := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, "", "init", "--bare", bare)
	runGit(t, repoDir, "remote", "add", "origin", bare)
	return repoDir, base, head
}

func watchTestApp(repoDir, base, head string) *AppContext {
	return &AppContext{
		Args: CommonArgs{
			Base:               base,
			Head:               head,
			Remote:             "origin",
			Target:             "main",
			BranchNameTemplate: "$USERNAME/stack",
		},
		Git:        git.New(repoDir, shell.Default{}),
		RepoRoot:   repoDir,
		Username:   "alice",
		OrigBranch: "main",
	}
}

func TestRunWithFetcherWatchesStackAndReturnsNilOnSuccess(t *testing.T) {
	repoDir, base, head := makeWatchTestRepo(t, []bool{true, true})
	chdir(t, repoDir)
	app := watchTestApp(repoDir, base, head)

	fetch := newScriptFetcher(t)
	fetch.on(ref7, fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "completed", "success"))})
	fetch.on(ref8, fetchStep{result: prResult(8, "bbb", mkCheck("build", "21", "201", "completed", "success"))})

	var out bytes.Buffer
	err := RunWithFetcher(app, defaultOpts(), &out, fetch.fetch, &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("RunWithFetcher: %v", err)
	}
	events := parseLiveEvents(t, &out)
	if eventTypes(events)[0] != eventStart || eventTypes(events)[len(events)-1] != eventFinal {
		t.Fatalf("events = %v", eventTypes(events))
	}
	final := finalFromEvents(t, events)
	if len(final.PullRequests) != 2 {
		t.Fatalf("watched PRs = %d, want 2", len(final.PullRequests))
	}
}

func TestRunWithFetcherRejectsMissingPRMetadata(t *testing.T) {
	repoDir, base, head := makeWatchTestRepo(t, []bool{false, true})
	chdir(t, repoDir)
	app := watchTestApp(repoDir, base, head)

	fetch := newScriptFetcher(t)
	var out bytes.Buffer
	err := RunWithFetcher(app, defaultOpts(), &out, fetch.fetch, &fakeClock{now: time.Now()})
	if err == nil || !strings.Contains(err.Error(), "missing PR metadata") {
		t.Fatalf("err = %v", err)
	}
	if out.Len() != 0 {
		t.Fatal("incomplete coverage must not produce a successful report")
	}
}

func TestRunWithFetcherValidatesOptionsAndPropagatesOutcomes(t *testing.T) {
	repoDir, base, head := makeWatchTestRepo(t, []bool{true})
	chdir(t, repoDir)
	app := watchTestApp(repoDir, base, head)

	var out bytes.Buffer
	if err := RunWithFetcher(app, Options{Interval: 0, Timeout: time.Minute}, &out, nil, &fakeClock{}); err == nil ||
		!strings.Contains(err.Error(), "--interval must be positive") {
		t.Fatalf("interval validation err = %v", err)
	}

	fetch := newScriptFetcher(t)
	fetch.on(ref7, fetchStep{result: prResult(7, "aaa", mkCheck("build", "11", "101", "completed", "failure"))})
	err := RunWithFetcher(app, defaultOpts(), &out, fetch.fetch, &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)})
	if err == nil || !strings.Contains(err.Error(), "failing") {
		t.Fatalf("failure outcome should surface as error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Small git helpers (mirrors the checks report test helpers)
// ---------------------------------------------------------------------------

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWatchSupersededExecutionsOnSameRevision(t *testing.T) {
	for _, live := range []bool{false, true} {
		for _, conclusion := range []string{"cancelled", "failure"} {
			for _, reverse := range []bool{false, true} {
				t.Run(fmt.Sprintf("live=%t/old=%s/reverse=%t", live, conclusion, reverse), func(t *testing.T) {
					old := mkCheck("build", "9", "99", "completed", conclusion)
					newer := mkCheck("build", "10", "100", "queued", "")
					checks := []pr.Check{old, newer}
					if reverse {
						checks[0], checks[1] = checks[1], checks[0]
					}
					fetch := newScriptFetcher(t)
					fetch.on(ref7,
						fetchStep{result: prResult(7, "aaa", checks...)},
						fetchStep{result: prResult(7, "aaa", old, mkCheck("build", "10", "100", "completed", "success"))},
					)
					opts := defaultOpts()
					opts.Live = live
					report, out := runWatch(opts, []Target{target(1, ref7)}, fetch.fetch, &fakeClock{now: time.Now()})
					if report.Outcome != OutcomeSuccess || fetch.calls[ref7] != 2 {
						t.Fatalf("outcome = %s, calls = %d", report.Outcome, fetch.calls[ref7])
					}
					if checks := report.PullRequests[0].Checks; len(checks) != 1 || checks[0].CheckRunID != "10" {
						t.Fatalf("current checks = %+v", checks)
					}
					if strings.Contains(out.String(), conclusion) {
						t.Fatalf("superseded conclusion leaked into output: %s", out)
					}
				})
			}
		}
	}
}

func TestWatchCancellationWithoutReplacementNeedsAttention(t *testing.T) {
	fetch := newScriptFetcher(t)
	fetch.on(ref7, fetchStep{result: prResult(7, "aaa", mkCheck("build", "9", "99", "completed", "cancelled"))})
	report, _ := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, &fakeClock{now: time.Now()})
	if report.Outcome != OutcomeAttention {
		t.Fatalf("outcome = %s", report.Outcome)
	}
}

func TestNewerExecution(t *testing.T) {
	for _, tt := range []struct {
		name         string
		older, newer pr.Check
	}{
		{"timestamps", pr.Check{StartedAt: "2026-09-28T10:00:00Z"}, pr.Check{StartedAt: "2026-09-28T08:00:01-02:00"}},
		{"run IDs", pr.Check{RunID: "9"}, pr.Check{RunID: "10"}},
		{"check run IDs", pr.Check{RunID: "1", CheckRunID: "9"}, pr.Check{RunID: "1", CheckRunID: "10"}},
		{"provider IDs", pr.Check{ProviderID: "9"}, pr.Check{ProviderID: "10"}},
		{"equal timestamps", pr.Check{StartedAt: "2026-09-28T10:00:00Z", RunID: "9"}, pr.Check{StartedAt: "2026-09-28T10:00:00Z", RunID: "10"}},
		{"queued replacement", pr.Check{StartedAt: "2026-09-28T10:00:00Z", RunID: "9"}, pr.Check{StartedAt: "0001-01-01T00:00:00Z", RunID: "10"}},
		{"invalid timestamps", pr.Check{StartedAt: "invalid", RunID: "9"}, pr.Check{RunID: "10"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !newerExecution(tt.newer, tt.older) || newerExecution(tt.older, tt.newer) {
				t.Fatal("execution ordering is incorrect")
			}
		})
	}
	if newerExecution(pr.Check{ProviderID: "opaque-B"}, pr.Check{ProviderID: "opaque-A"}) {
		t.Fatal("opaque IDs must not establish ordering")
	}
}

func TestWatchSupersessionKeepsDifferentWorkflowsAndJobs(t *testing.T) {
	checks := []pr.Check{
		mkCheck("github-actions:first:build", "9", "99", "completed", "cancelled"),
		mkCheck("github-actions:second:build", "10", "100", "completed", "success"),
		mkCheck("github-actions:first:test", "11", "100", "completed", "success"),
	}
	fetch := newScriptFetcher(t)
	fetch.on(ref7, fetchStep{result: prResult(7, "aaa", checks...)})
	report, _ := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, &fakeClock{now: time.Now()})
	if report.Outcome != OutcomeAttention || len(report.PullRequests[0].Checks) != 3 {
		t.Fatalf("unrelated checks superseded: %+v", report)
	}
}

func TestWatchReplacementSuppressesOldTransition(t *testing.T) {
	fetch := newScriptFetcher(t)
	fetch.on(ref7,
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "9", "99", "in_progress", ""))},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "9", "99", "completed", "cancelled"), mkCheck("build", "10", "100", "queued", ""))},
		fetchStep{result: prResult(7, "aaa", mkCheck("build", "9", "99", "completed", "cancelled"), mkCheck("build", "10", "100", "completed", "success"))},
	)
	report, out := runWatch(defaultOpts(), []Target{target(1, ref7)}, fetch.fetch, &fakeClock{now: time.Now()})
	if report.Outcome != OutcomeSuccess || strings.Contains(out.String(), "cancelled") {
		t.Fatalf("superseded transition reported: %s", out)
	}
	var transitions []string
	for _, event := range parseLiveEvents(t, out) {
		if event.Type == eventCheck {
			transitions = append(transitions, event.Transition)
		}
	}
	if !reflect.DeepEqual(transitions, []string{"rerun", "changed"}) {
		t.Fatalf("transitions = %v", transitions)
	}
}

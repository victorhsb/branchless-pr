package ciwatch

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stackstate"
)

// Target identifies one watched pull request.
type Target struct {
	Index      int
	Commit     string
	ShortSHA   string
	Title      string
	HeadBranch string
	BaseBranch string
	PRRef      string
}

// Run watches the current stack with the GitHub-backed fetcher and a real clock.
func Run(app *AppContext, opts Options, w io.Writer) error {
	return RunWithFetcher(app, opts, w, app.PR.FetchChecksForWatch, realClock{})
}

// RunWithFetcher loads the stack and runs the watch loop with injected
// dependencies. It returns nil only when the outcome is success; for every
// other outcome the final report has already been written and the returned
// error carries the reason.
func RunWithFetcher(app *AppContext, opts Options, w io.Writer, fetch Fetcher, clock Clock) error {
	if opts.Interval <= 0 {
		return fmt.Errorf("bpr ci-watch: --interval must be positive")
	}
	if opts.Timeout <= 0 {
		return fmt.Errorf("bpr ci-watch: --timeout must be positive")
	}
	st, err := stackstate.Load(stackstate.Args{
		Repo:               app.Git,
		Base:               app.Args.Base,
		Head:               app.Args.Head,
		Remote:             app.Args.Remote,
		Target:             app.Args.Target,
		BranchNameTemplate: app.Args.BranchNameTemplate,
		Username:           app.Username,
		OrigBranch:         app.OrigBranch,
	})
	if err != nil {
		return err
	}

	var targets []Target
	var missing []string
	for _, e := range st {
		if !e.HasPR() {
			missing = append(missing, fmt.Sprintf("%d (%s %q)", stackstate.Index(st, e), e.Commit.ShortSHA(), e.Commit.Title))
			continue
		}
		targets = append(targets, Target{
			Index:      stackstate.Index(st, e),
			Commit:     e.Commit.SHA,
			ShortSHA:   e.Commit.ShortSHA(),
			Title:      e.Commit.Title,
			HeadBranch: stackstate.SafeHead(e),
			BaseBranch: e.Base(),
			PRRef:      e.PR(),
		})
	}
	if len(missing) > 0 {
		return fmt.Errorf("bpr ci-watch: stack entries missing PR metadata: %s; watch coverage would be incomplete", strings.Join(missing, ", "))
	}
	if len(targets) == 0 {
		return fmt.Errorf("bpr ci-watch: no pull requests found in the current stack")
	}

	watcher := newWatcher(opts, fetch, clock, w, app.RepoRoot)
	report := watcher.Watch(targets)
	if report.Outcome == OutcomeSuccess {
		return nil
	}
	return fmt.Errorf("bpr ci-watch: %s", report.Reason)
}

// ---------------------------------------------------------------------------
// Watcher
// ---------------------------------------------------------------------------

type watcher struct {
	opts     Options
	fetch    Fetcher
	clock    Clock
	out      io.Writer
	enc      *json.Encoder
	repoRoot string

	start    time.Time
	deadline time.Time
}

func newWatcher(opts Options, fetch Fetcher, clock Clock, out io.Writer, repoRoot string) *watcher {
	w := &watcher{opts: opts, fetch: fetch, clock: clock, out: out, repoRoot: repoRoot}
	if opts.Live {
		w.enc = json.NewEncoder(out)
	}
	return w
}

type checkObs struct {
	check      pr.Check
	state      string
	observedAt time.Time
}

type prWatch struct {
	target Target

	prNumber int
	prURL    string

	revision   string
	checks     map[string]*checkObs
	order      []string
	emptySince time.Time
	noChecks   bool
	observedAt time.Time
}

// Watch runs the polling loop and returns the final report. The report is
// written to the configured output in both live and buffered modes.
func (w *watcher) Watch(targets []Target) *Report {
	w.start = w.clock.Now()
	w.deadline = w.start.Add(w.opts.Timeout)

	prs := make([]*prWatch, 0, len(targets))
	for _, t := range targets {
		prs = append(prs, &prWatch{target: t, prURL: t.PRRef, checks: map[string]*checkObs{}})
	}

	// Poll immediately, before the initial state record.
	if err := w.round(prs, true); err != nil {
		return w.final(prs, OutcomeError, err.Error())
	}
	w.emitStart(prs)
	if outcome, reason, done := evaluate(prs); done {
		return w.final(prs, outcome, reason)
	}

	for {
		if !w.clock.Now().Before(w.deadline) {
			return w.final(prs, OutcomeTimeout, timeoutReason(prs))
		}
		w.clock.Sleep(w.opts.Interval)
		if !w.clock.Now().Before(w.deadline) {
			return w.final(prs, OutcomeTimeout, timeoutReason(prs))
		}
		if err := w.round(prs, false); err != nil {
			return w.final(prs, OutcomeError, err.Error())
		}
		if outcome, reason, done := evaluate(prs); done {
			return w.final(prs, outcome, reason)
		}
	}
}

// round fetches every watched PR once. Any observation error aborts the
// watch; an API error is never an empty check list.
func (w *watcher) round(prs []*prWatch, initial bool) error {
	for _, pw := range prs {
		fetched, err := w.fetch(pw.target.PRRef)
		if err != nil {
			return fmt.Errorf("observe %s: %w", pw.target.PRRef, err)
		}
		now := w.clock.Now()
		pw.observedAt = now
		if fetched.Number != 0 {
			pw.prNumber = fetched.Number
		}
		if fetched.URL != "" {
			pw.prURL = fetched.URL
		}
		if fetched.HeadRefName != "" {
			pw.target.HeadBranch = fetched.HeadRefName
		}
		if fetched.BaseRefName != "" {
			pw.target.BaseBranch = fetched.BaseRefName
		}

		revision := fetched.HeadSHA
		if revision != "" && revision != pw.revision {
			prev := pw.revision
			pw.revision = revision
			// Superseded results are dropped, never reported as current.
			pw.checks = map[string]*checkObs{}
			pw.order = nil
			pw.emptySince = time.Time{}
			pw.noChecks = false
			if !initial {
				// A new revision resets the rolling, stack-wide deadline.
				w.deadline = now.Add(w.opts.Timeout)
				w.emit(revisionEvent{
					Type:             eventRevision,
					PRNumber:         pw.prNumber,
					PRURL:            pw.prURL,
					PreviousRevision: prev,
					Revision:         revision,
					Note:             revisionNote,
					ObservedAt:       now,
				})
			}
		}

		if len(fetched.Checks) == 0 {
			if pw.emptySince.IsZero() {
				pw.emptySince = now
			}
			if !pw.noChecks && now.Sub(pw.emptySince) >= noChecksWindow {
				pw.noChecks = true
				if !initial {
					w.emit(noChecksEvent{
						Type:       eventNoChecks,
						PRNumber:   pw.prNumber,
						PRURL:      pw.prURL,
						Revision:   pw.revision,
						Note:       noChecksNote,
						ObservedAt: now,
					})
				}
			}
			continue
		}
		pw.emptySince = time.Time{}
		pw.noChecks = false
		w.applyChecks(pw, fetched.Checks, now, initial)
	}
	return nil
}

// applyChecks reconciles the latest poll with tracked state and emits one
// event per observed transition. Unchanged checks stay quiet.
func (w *watcher) applyChecks(pw *prWatch, checks []pr.Check, now time.Time, initial bool) {
	next := make(map[string]*checkObs, len(checks))
	order := make([]string, 0, len(checks))
	for _, c := range checks {
		key := execKey(c)
		obs := &checkObs{check: c, state: classifyCheck(c), observedAt: now}
		next[key] = obs
		order = append(order, key)
		if initial {
			continue
		}
		prev, tracked := pw.checks[key]
		switch {
		case !tracked:
			transition := "appeared"
			if supersedesRun(pw, c) {
				transition = "rerun"
			}
			w.emitCheck(pw, obs, transition, nil)
		case prev.check.Status != c.Status || prev.check.Conclusion != c.Conclusion:
			w.emitCheck(pw, obs, "changed", &stateRef{
				Status:     prev.check.Status,
				Conclusion: prev.check.Conclusion,
				State:      prev.state,
			})
		default:
			obs.observedAt = prev.observedAt
		}
	}
	sort.Slice(order, func(i, j int) bool {
		a, b := next[order[i]], next[order[j]]
		if a.check.ID != b.check.ID {
			return a.check.ID < b.check.ID
		}
		return order[i] < order[j]
	})
	pw.checks = next
	pw.order = order
}

// supersedesRun reports whether c is a new execution of an already tracked
// check, which marks a rerun rather than a newly appearing check.
func supersedesRun(pw *prWatch, c pr.Check) bool {
	for _, key := range pw.order {
		if prev := pw.checks[key]; prev != nil && prev.check.ID == c.ID {
			return true
		}
	}
	return false
}

func (w *watcher) emitCheck(pw *prWatch, obs *checkObs, transition string, prev *stateRef) {
	w.emit(checkEvent{
		Type:       eventCheck,
		Transition: transition,
		PRNumber:   pw.prNumber,
		PRURL:      pw.prURL,
		Revision:   pw.revision,
		Check:      checkToJSON(obs),
		Previous:   prev,
		ObservedAt: w.clock.Now(),
	})
}

func (w *watcher) emitStart(prs []*prWatch) {
	w.emit(startEvent{
		Type:            eventStart,
		SchemaVersion:   schemaVersion,
		Command:         commandName,
		Repository:      w.repoRoot,
		IntervalSeconds: int64(w.opts.Interval / time.Second),
		TimeoutSeconds:  int64(w.opts.Timeout / time.Second),
		Deadline:        w.deadline,
		StackSize:       len(prs),
		PullRequests:    snapshots(prs),
		ObservedAt:      w.clock.Now(),
	})
}

func (w *watcher) emit(v any) {
	if w.enc == nil {
		return
	}
	// json.Encoder writes one complete JSON value per line; the underlying
	// writer is unbuffered os.Stdout in production, so lines flush immediately.
	_ = w.enc.Encode(v)
}

// final builds, writes, and returns the terminal report.
func (w *watcher) final(prs []*prWatch, outcome, reason string) *Report {
	now := w.clock.Now()
	report := &Report{
		SchemaVersion:   schemaVersion,
		Command:         commandName,
		Repository:      w.repoRoot,
		Outcome:         outcome,
		Reason:          reason,
		Live:            w.opts.Live,
		IntervalSeconds: int64(w.opts.Interval / time.Second),
		TimeoutSeconds:  int64(w.opts.Timeout / time.Second),
		StartedAt:       w.start,
		EndedAt:         now,
		ElapsedSeconds:  int64(now.Sub(w.start) / time.Second),
		Deadline:        w.deadline,
		PullRequests:    snapshots(prs),
		NextSteps:       nextSteps(outcome, prs),
	}
	if w.enc != nil {
		w.emit(finalEvent{Type: eventFinal, Report: report})
	} else {
		payload, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			_, _ = fmt.Fprintln(w.out, string(payload))
		}
	}
	return report
}

// ---------------------------------------------------------------------------
// Evaluation
// ---------------------------------------------------------------------------

// evaluate decides whether the watch can end after a polling round. Failures
// and attention needs return promptly; success requires every PR to have
// acceptable completed checks or an accepted no_checks result.
func evaluate(prs []*prWatch) (outcome, reason string, done bool) {
	failures := collect(prs, StateFailing)
	if len(failures) > 0 {
		return OutcomeFailure, fmt.Sprintf("%d failing %s observed", len(failures), plural(len(failures), "check", "checks")), true
	}
	attentions := collect(prs, StateAttention)
	if len(attentions) > 0 {
		if len(attentions) == 1 {
			return OutcomeAttention, "1 check needs attention", true
		}
		return OutcomeAttention, fmt.Sprintf("%d checks need attention", len(attentions)), true
	}
	for _, pw := range prs {
		if len(pw.order) == 0 {
			if !pw.noChecks {
				return "", "", false
			}
			continue
		}
		for _, key := range pw.order {
			if pw.checks[key].state == StatePending {
				return "", "", false
			}
		}
	}
	return OutcomeSuccess, "all watched checks completed acceptably", true
}

func collect(prs []*prWatch, state string) []*checkObs {
	var out []*checkObs
	for _, pw := range prs {
		for _, key := range pw.order {
			if pw.checks[key].state == state {
				out = append(out, pw.checks[key])
			}
		}
	}
	return out
}

func timeoutReason(prs []*prWatch) string {
	pending := 0
	discovery := 0
	for _, pw := range prs {
		if len(pw.order) == 0 {
			if !pw.noChecks {
				discovery++
			}
			continue
		}
		pending += len(collect([]*prWatch{pw}, StatePending))
	}
	parts := []string{}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending %s", pending, plural(pending, "check", "checks")))
	}
	if discovery > 0 {
		parts = append(parts, fmt.Sprintf("%d %s awaiting check discovery", discovery, plural(discovery, "PR", "PRs")))
	}
	if len(parts) == 0 {
		return "timeout reached"
	}
	return "timeout reached with " + strings.Join(parts, " and ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// ---------------------------------------------------------------------------
// Classification and identity
// ---------------------------------------------------------------------------

// classifyCheck maps a check to passing, failing, attention, or pending.
// Everything completed that is not clearly acceptable or a code failure is
// surfaced as attention rather than passing.
func classifyCheck(c pr.Check) string {
	switch strings.ToLower(strings.TrimSpace(c.Conclusion)) {
	case "success", "skipped", "neutral":
		return StatePassing
	case "failure", "error", "timed_out", "startup_failure":
		return StateFailing
	case "action_required", "cancelled":
		return StateAttention
	case "":
		if strings.EqualFold(strings.TrimSpace(c.Status), "completed") {
			return StateAttention
		}
		return StatePending
	default:
		return StateAttention
	}
}

// execKey identifies one execution of a check, preferring execution identity
// over the check name so reruns and superseded runs are distinguishable.
func execKey(c pr.Check) string {
	switch {
	case c.CheckRunID != "":
		return c.ID + "@" + c.CheckRunID
	case c.ProviderID != "":
		return c.ID + "@" + c.ProviderID
	case c.RunID != "":
		return c.ID + "@" + c.RunID
	default:
		return c.ID
	}
}

// ---------------------------------------------------------------------------
// Report assembly
// ---------------------------------------------------------------------------

func snapshots(prs []*prWatch) []PRReport {
	out := make([]PRReport, 0, len(prs))
	for _, pw := range prs {
		entry := PRReport{
			Index:      pw.target.Index,
			PRNumber:   pw.prNumber,
			PRURL:      pw.prURL,
			Commit:     pw.target.Commit,
			ShortSHA:   pw.target.ShortSHA,
			Title:      pw.target.Title,
			HeadBranch: pw.target.HeadBranch,
			BaseBranch: pw.target.BaseBranch,
			Revision:   pw.revision,
			State:      prState(pw),
			ObservedAt: pw.observedAt,
			Checks:     []CheckJSON{},
		}
		if entry.State == StateNoChecks {
			entry.Note = noChecksNote
		}
		for _, key := range pw.order {
			entry.Checks = append(entry.Checks, checkToJSON(pw.checks[key]))
		}
		out = append(out, entry)
	}
	return out
}

// prState rolls a PR's checks up to one state: failing > attention > pending
// > passing, with no_checks only once absence is confirmed.
func prState(pw *prWatch) string {
	if len(pw.order) == 0 {
		if pw.noChecks {
			return StateNoChecks
		}
		return StatePending
	}
	state := StatePassing
	for _, key := range pw.order {
		switch pw.checks[key].state {
		case StateFailing:
			return StateFailing
		case StateAttention:
			state = StateAttention
		case StatePending:
			if state == StatePassing {
				state = StatePending
			}
		}
	}
	return state
}

func checkToJSON(obs *checkObs) CheckJSON {
	return CheckJSON{
		ID:         obs.check.ID,
		Name:       obs.check.Name,
		Provider:   obs.check.Provider,
		Workflow:   obs.check.Workflow,
		RunID:      obs.check.RunID,
		CheckRunID: obs.check.CheckRunID,
		Status:     obs.check.Status,
		Conclusion: obs.check.Conclusion,
		State:      obs.state,
		URL:        obs.check.URL,
		ObservedAt: obs.observedAt,
	}
}

// nextSteps points the agent at failures, inspection URLs, or verified log
// commands. It never prescribes edits, reruns, submissions, or merges.
func nextSteps(outcome string, prs []*prWatch) []string {
	var steps []string
	seenRuns := map[string]bool{}
	addInspection := func(prefix string, pw *prWatch, obs *checkObs) {
		label := fmt.Sprintf("%s %q on PR #%d", prefix, obs.check.Name, pw.prNumber)
		if obs.check.URL != "" {
			label += ": " + obs.check.URL
		}
		steps = append(steps, label)
		if obs.check.RunID != "" && !seenRuns[obs.check.RunID] {
			seenRuns[obs.check.RunID] = true
			steps = append(steps, fmt.Sprintf("View failed logs: gh run view %s --log-failed", obs.check.RunID))
		}
	}

	switch outcome {
	case OutcomeFailure:
		for _, pw := range prs {
			for _, key := range pw.order {
				if obs := pw.checks[key]; obs.state == StateFailing {
					addInspection("Inspect failing check", pw, obs)
				}
			}
		}
	case OutcomeAttention:
		for _, pw := range prs {
			for _, key := range pw.order {
				if obs := pw.checks[key]; obs.state == StateAttention {
					addInspection(fmt.Sprintf("Check ended with %q and needs attention:", obs.check.Conclusion), pw, obs)
				}
			}
		}
	case OutcomeTimeout:
		for _, pw := range prs {
			if len(pw.order) == 0 && !pw.noChecks {
				steps = append(steps, fmt.Sprintf("PR #%d has no checks yet; discovery window still open", pw.prNumber))
			}
			for _, key := range pw.order {
				if obs := pw.checks[key]; obs.state == StatePending {
					label := fmt.Sprintf("Still pending: %q on PR #%d", obs.check.Name, pw.prNumber)
					if obs.check.URL != "" {
						label += ": " + obs.check.URL
					}
					steps = append(steps, label)
				}
			}
		}
		steps = append(steps, "Inspect the URLs above or run `bpr ci-watch` again to keep watching.")
	case OutcomeError:
		steps = append(steps, "Resolve the observation error and run `bpr ci-watch` again.")
	}
	return steps
}

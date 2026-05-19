## Context

`stack-pr agent diagnose` is already a read-only, degraded-mode command with optional online checks. In online mode it currently checks `gh auth status` and then queries PR state with `gh pr view`, but a GitHub outage can still be reported as an ambiguous PR-state failure. That ambiguity is risky for agents: when remote state cannot be trusted, the CLI should clearly say so and avoid steering callers toward operations that can rewrite commit metadata, update PRs, or merge.

The existing `internal/diagnose` package is a good fit for this change because it already owns check ordering, the JSON/text report model, the recommendation engine, and an injectable `Runner` for tests. The implementation must continue to use `internal/shell` through the runner abstraction and must not introduce a GitHub SDK.

## Goals / Non-Goals

**Goals:**

- Add a stable `github_availability` check in `agent diagnose --online`.
- Distinguish likely GitHub service unavailability from missing `gh`, failed authentication, missing PR metadata, malformed local metadata, and individual PR lookup problems.
- Make outage findings visible in both JSON and Markdown output.
- Mark likely outage findings as blocking for mutation-oriented commands (`submit`, `land`, `abandon`) and any command path that depends on live GitHub PR state.
- Gate recommendation selection so a detected outage recommends waiting/retrying or using local-only inspection, not changing commit metadata or remote PR state.
- Preserve the existing default offline contract: no network I/O without `--online`.

**Non-Goals:**

- Adding a GitHub status-page integration or web scraping.
- Changing the behavior of `submit`, `land`, `abandon`, `view`, or non-diagnose root preflight checks.
- Guaranteeing perfect outage detection for every network, DNS, proxy, or GitHub incident mode.
- Mutating local Git, commit messages, branches, the index, working tree, remotes, or GitHub resources from diagnose.

## Decisions

1. **Add a first-class `github_availability` check instead of overloading `online_pr_state`.**
   - The new check runs only when `--online` is true and reports `unknown` in offline mode.
   - It executes before `online_pr_state` so later online checks can avoid repeated, noisy PR calls when GitHub is unavailable.
   - Alternative considered: classify outages inside each `gh pr view` failure. Rejected because it repeats classification logic and hides the most important diagnosis behind per-PR state.

2. **Use a lightweight read-only `gh api` probe through the existing runner.**
   - The probe should use a read-only endpoint such as `gh api /rate_limit` or another stable GitHub REST endpoint that requires no repository-specific state.
   - Missing `gh` remains the responsibility of `gh_installed`; failed login remains the responsibility of `github_authentication`.
   - Alternative considered: query the public GitHub status API or status page. Rejected because it adds a second external service surface and is not necessary for answering whether `gh` can currently reach GitHub.

3. **Classify only strong outage signals as blocking.**
   - Strong signals include 5xx responses, service-unavailable messages, timeout/connection failures that affect the availability probe, and similar transport-level failures from `gh`.
   - Authentication/authorization failures are not outages and should stay under `github_authentication`.
   - Repository- or PR-specific 404/permission errors are not global outages and should stay under the relevant PR-state check.
   - Alternative considered: treat every `gh` failure as an outage. Rejected because it would mask local setup problems and reduce the diagnostic value of the report.

4. **Use blocking status for likely GitHub outage.**
   - A detected outage blocks `submit`, `land`, and `abandon` because they either write GitHub state, rewrite commit metadata, delete branches, or otherwise depend on trustworthy remote state.
   - `view` can remain a possible local/metadata inspection action, but online PR-state-dependent conclusions must be marked unavailable.
   - Alternative considered: report outages as `warning`. Rejected because agents need a clear stop signal before choosing mutating actions.

5. **Update recommendation priority.**
   - After local blockers that must be resolved regardless of GitHub availability (not a repo, rebase in progress, empty stack, dirty tree), a blocking `github_availability` check takes priority over missing-PR and fully-submitted recommendations.
   - Rationale: once the local repository is coherent enough to inspect, the next critical fact is whether remote GitHub state can be trusted before suggesting PR or commit-metadata workflows.
   - Alternative considered: put outage ahead of dirty tree. Rejected because a dirty tree is a local mutation blocker even if GitHub recovers.

6. **Keep output schema version stable unless the model shape changes incompatibly.**
   - Adding a new check entry is compatible with the existing `checks` array contract. No schema bump is required unless implementation adds required top-level fields or changes existing meanings.

## Risks / Trade-offs

- **Risk: False positives from local network or proxy failures.** -> Mitigate by wording the check as "GitHub appears unavailable" and keeping detailed command error text in the message.
- **Risk: False negatives during partial GitHub incidents.** -> Mitigate by still allowing `online_pr_state` to classify per-PR failures as `unknown` or `warning`; the availability probe is a quick global signal, not the only signal.
- **Risk: Probe endpoint behavior changes.** -> Mitigate by isolating the probe command and classifier in small functions with tests using fake runner outputs.
- **Risk: Recommendation logic becomes too conservative.** -> Accepted; under suspected outage the safer agent behavior is to wait or inspect local state rather than mutate stack state.

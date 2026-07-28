---
title: agent diagnose
status: stable
---

# Agent Diagnose

## Overview

Provide a read-only, best-effort diagnostic command for agents and humans to inspect repository, stack, and remote-readiness state before choosing a `stack-pr` action.

## Behavior

### Subcommand and read-only guarantee

The `stack-pr` CLI provides a `diagnose` subcommand under the `agent` command group that inspects the current repository and emits a structured report describing repository, stack, and check state.

- `stack-pr agent diagnose` is invoked from a shell → execute and produce a report on standard output.
- Invoked under any conditions → never perform any local Git mutation, including but not limited to checkouts, rebases, commit amendments, branch creation, branch deletion, stash save, stash pop, index modification, or working-tree modification; never perform any remote push, fetch-write, or GitHub write operation, including but not limited to creating, editing, closing, merging, or changing the draft state of pull requests.

### Output format selection

`--format` accepts `text` and `json` and defaults to `text`.

| `--format` | Behavior |
|------------|----------|
| omitted or `text` | emit a human-readable Markdown report on standard output |
| `json` | emit a single JSON document on standard output conforming to the diagnosis JSON schema |
| any other value | report an invalid-flag error and exit with the non-zero exit code reserved for invocation errors |

### Online mode flag

`--online` defaults to false.

- Invoked without `--online` → perform no network I/O; never contact GitHub or any other remote service.
- `--online` and the stack contains entries with PR metadata → may consult GitHub (for example via `gh`) for live PR state; the result of any such query is reflected in the report.
- `--online` and a GitHub query fails → record the failure as a check entry with status `unknown` or `warning`, continue running remaining checks, and exit with code `0`.

### GitHub availability check

A stable `github_availability` check entry reports whether GitHub appears reachable to the configured `gh` CLI when online mode is enabled. The check is read-only and never performs a GitHub write operation.

- Invoked without `--online` → report a `github_availability` check with status `unknown` and a message indicating that `--online` was not specified; do not contact GitHub or any other remote service for the availability check.
- `--online` and a read-only GitHub availability probe succeeds → report `github_availability` with status `ok` and a message indicating that GitHub appears reachable.
- `--online` and the probe fails with a likely service outage or transport-level availability failure → report `github_availability` with status `blocking`, `blocks` listing at least `submit`, `land`, and `abandon`, and a `suggested_fix` directing the user or agent to wait and retry after GitHub availability recovers; continue evaluating remaining checks where they can be evaluated without relying on live GitHub state; exit with code `0`.
- `--online` and `gh` reports an authentication or authorization failure → `github_availability` does not classify that failure as a GitHub outage; authentication state is surfaced by the `github_authentication` check.
- `--online` can reach GitHub but an individual PR lookup fails because the PR is missing, inaccessible, or repository-specific → `github_availability` does not classify that failure as a GitHub outage; the individual PR lookup result is surfaced by the relevant online PR-state check.

### Outage-safe online PR checks

When `github_availability` has status `blocking`, online PR-state checks avoid making conclusions from live PR state.

- A blocking `github_availability` check is detected before evaluating live PR state → the `online_pr_state` check has status `unknown` or `blocking`, its message indicates that live PR state was not trusted because GitHub appears unavailable, and the report never claims that the stack is fully synchronized with live GitHub PR state.
- A likely GitHub outage is detected → local checks such as repository detection, working tree cleanliness, rebase state, base/head resolution, branch-name template validity, and stack discovery are still evaluated when possible.

### Exit code stability

- Any reportable outcome, including outcomes where one or more checks have status `blocking` → exit with code `0`; the report includes each blocking check with its status, message, `blocks`, and `suggested_fix`.
- Non-zero exit codes are reserved for catastrophic, unexpected failures (for example, the JSON encoder failing) and for invocation errors such as an invalid flag value.
- Invoked outside any Git repository → emit a report indicating that the working directory is not in a Git repository and exit with code `0`.

### Degraded-mode check behavior

Each check reports a `status` rather than aborting the command.

- The underlying logic for any single check returns an error → report that check entry with status `unknown` and a message describing why it could not be evaluated, still evaluate remaining checks, and exit with code `0`.
- Tracked files have staged or unstaged changes → do not abort; report a `working_tree_clean` check entry with status `blocking`, `blocks` listing commands that require a clean working tree (for example `submit`, `land`, `abandon`), and a `suggested_fix` describing how to clean the working tree.

### Check entry schema

- Any check entry in the JSON report → includes `id` (a stable string identifier), `status` (one of `ok`, `warning`, `blocking`, `unknown`), and `message` (a human-readable description).
- Check entry has status `blocking` → additionally includes `blocks` (a list of the command names this issue prevents) and `suggested_fix` (a human-readable remediation hint describing how a user or agent can resolve the blocker).

### Required checks

Each of the following best-effort checks is surfaced as a check entry in the report; a check that cannot be evaluated is reported with status `unknown` rather than omitted.

| Check | Scope | Reports |
|-------|-------|---------|
| Git repository | always | whether the working directory is inside a Git repository |
| `gh` installed | always | whether the `gh` CLI is installed and discoverable |
| GitHub authentication | always | whether GitHub authentication is available to `gh` |
| Working tree clean | inside a Git repository | whether the working tree is clean |
| Rebase in progress | inside a Git repository | whether a rebase is currently in progress |
| Base and head resolution | inside a Git repository | whether the base and head revisions used for stack discovery can be resolved |
| Target branch existence | inside a Git repository | whether the configured target branch exists on the configured remote |
| Branch-name template | always | whether the configured branch-name template is valid |
| Stack size | base and head resolvable | number of commits in the stack (check or top-level stack summary) |
| Stack metadata coverage | stack discoverable | how many commits already carry `stack-info` metadata (check or top-level stack summary) |
| Missing PRs | stack discoverable | how many commits are missing a pull request (check or top-level stack summary) |
| PR base coherence | stack discoverable and at least one entry has PR metadata | whether the PR base relationships across the stack are coherent with bottom-to-top stacking |
| Local base behind remote target | base resolvable | whether the local base is behind the configured remote target branch |
| Online PR state | `--online` and one or more stack entries have PR metadata | live PR state retrieved from GitHub (check or per-entry annotation); in offline mode this check is not present, or is reported with status `unknown` and a message indicating that `--online` was not specified |

### JSON output envelope

`--format json` emits a single JSON object. The JSON output is stable across patch-level releases for a given `schema_version` value; incompatible changes require incrementing `schema_version`.

| Field | Content |
|-------|---------|
| `schema_version` | non-empty string identifying the diagnosis JSON schema version |
| `status` | overall result: `ok`, `warning`, `blocking`, or `unknown` |
| `repo` | repository context (such as root, current branch, remote, target, base, head) |
| `stack` | stack inspection summary (such as size, entries with PR metadata, entries missing PRs) |
| `checks` | array of check entries |
| `recommendation` | recommendation object as described in the recommendation contract |

- One or more check entries are present → the top-level `status` is at least as severe as the most severe check status, with severity ordered `ok` < `warning` < `blocking`, and `unknown` reported when overall severity cannot be determined.

### Text output information set

`--format text` emits a human-readable Markdown report whose information set is equivalent to the JSON output: repository context, stack summary, each check (with at minimum its identifier or short label, status, and message), any blocking check's `suggested_fix`, and the recommendation including its command, reason, and safety metadata. Exact headings, ordering, and prose are an implementation choice.

- Invoked without `--format json` → the output identifies the recommended command, the reason for the recommendation, and whether the recommended command has side effects and requires explicit confirmation.
- The report includes one or more blocking checks → the text output surfaces the suggested fix for each blocking check.

### Recommendation contract

- Invoked under any conditions → the report includes a recommendation, even when the repository is not a Git repository or no useful action is available.
- Any recommendation → includes `command`, `reason`, `side_effects` (boolean), and `requires_confirmation` (boolean).
- Recommendation has `side_effects` equal to true → `requires_confirmation` is also true.

### Recommendation decision tree

The recommendation is chosen by the following priority, evaluated top-down on the first matching condition:

1. If the working directory is not inside a Git repository, recommend changing into a Git repository.
2. Otherwise, if a rebase is in progress, recommend finishing or aborting the rebase.
3. Otherwise, if the stack is empty, recommend creating commits before using `stack-pr`.
4. Otherwise, if the working tree is dirty, recommend cleaning the working tree (commit, stash, or revert).
5. Otherwise, if online mode detected that GitHub appears unavailable, recommend waiting for GitHub availability to recover or using local-only inspection; the primary recommendation is never `stack-pr submit`, `stack-pr land`, or `stack-pr abandon`, and the recommendation states that live GitHub state cannot currently be trusted for mutating stack-pr operations.
6. Otherwise, if one or more commits are missing PR metadata, recommend `stack-pr submit --dry-run`; the recommendation states that no mutation has yet occurred and that the dry run can preview the create-or-update plan.
7. Otherwise (the stack appears fully submitted), recommend `stack-pr view --format json` as the primary recommendation `command`; the report may surface `stack-pr land` only as a conservative potential next action that requires confirmation.

### Conservative land recommendation

- `stack-pr land` is never the primary recommendation, including when the decision tree yields the fully-submitted state.
- `stack-pr land` is surfaced anywhere in the report as a potential next action → the entry includes `side_effects: true` and `requires_confirmation: true` and describes `land` as conservative guidance rather than an outright recommendation.

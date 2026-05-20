# stack-checks-report Specification

## Purpose

Provide a read-only stack-wide checks report that helps users and agents identify CI failures, optional and required check state, and brief review-attention signals across stacked pull requests.

## Requirements

### Requirement: Stack Checks Command

The `stack-pr` CLI SHALL provide a top-level `checks` command that reports GitHub check and lightweight review-attention state for pull requests represented by the current stack metadata.

#### Scenario: Checks command is available

- **WHEN** the user runs `stack-pr checks --help`
- **THEN** the CLI SHALL describe a read-only command for reporting check state across the current stack's pull requests
- **AND** the help SHALL include supported output format and filtering flags

#### Scenario: Stack PRs are discovered

- **WHEN** `stack-pr checks` is invoked inside a repository with a non-empty stack
- **THEN** the command SHALL discover stack entries using the same base, head, branch-template, and stack metadata rules as other stack inspection commands
- **AND** the command SHALL associate fetched check state with the stack entry and pull request it belongs to

#### Scenario: Missing PR metadata is reported

- **WHEN** a stack entry has no pull request metadata
- **THEN** the command SHALL report that entry as missing PR metadata
- **AND** the command SHALL continue collecting check state for other entries that have PR metadata

### Requirement: Read-Only Behavior

The `checks` command SHALL NOT mutate local Git state, remote branches, commit messages, or GitHub pull request state.

#### Scenario: Dirty worktree is allowed

- **WHEN** `stack-pr checks` is invoked while tracked files have staged or unstaged changes
- **THEN** the command SHALL still attempt to produce a checks report
- **AND** it SHALL NOT require the user to clean or stash the worktree first

#### Scenario: No GitHub writes

- **WHEN** `stack-pr checks` fetches check or review-attention information from GitHub
- **THEN** it SHALL use read-only GitHub operations
- **AND** it SHALL NOT create, edit, close, merge, approve, rerun, resolve, dismiss, or delete pull requests, checks, reviews, or comments

#### Scenario: No local stack mutation

- **WHEN** `stack-pr checks` runs successfully or with reportable per-PR failures
- **THEN** it SHALL NOT checkout branches, create branches, delete branches, amend commits, rebase, stash, push, or fetch in a way that mutates repository state

### Requirement: All Check Reporting

The checks report SHALL include all GitHub checks and status contexts available for each stack pull request head commit, not only required checks.

#### Scenario: Required and optional checks are included

- **WHEN** a stack pull request has both required and optional checks
- **THEN** the report SHALL include both required and optional checks
- **AND** each check SHALL indicate whether it is required when that information is available

#### Scenario: Required state unknown

- **WHEN** GitHub does not expose whether a check is required
- **THEN** the report SHALL include the check with required state `unknown`
- **AND** the command SHALL NOT infer required state from check name alone

#### Scenario: Status contexts are included

- **WHEN** a stack pull request head commit has legacy status contexts or non-Actions check providers
- **THEN** the report SHALL include those statuses in the same check collection as GitHub Actions check runs
- **AND** the report SHALL preserve provider and URL information when available

#### Scenario: Pending and skipped checks are included

- **WHEN** a check is queued, in progress, pending, skipped, cancelled, neutral, successful, or failed
- **THEN** the report SHALL include the check with its normalized status and conclusion

### Requirement: Stable Check Identity

Each reported check SHALL include an agent-usable stable check identifier and SHALL include exact GitHub provider identifiers when available.

#### Scenario: Semantic check ID is emitted

- **WHEN** a check is reported
- **THEN** the check SHALL include an `id` field derived from stable semantic fields such as provider, workflow or suite name, and job or check name
- **AND** the ID SHALL be deterministic for the same check source and name

#### Scenario: Provider IDs are emitted

- **WHEN** GitHub exposes exact identifiers for a check, run, suite, or workflow
- **THEN** the check SHALL include those identifiers in provider-specific fields such as `provider_id`, `run_id`, `check_run_id`, or `workflow`
- **AND** the semantic `id` SHALL remain present

#### Scenario: Failed check summary references identifiers

- **WHEN** one or more checks have failing conclusions
- **THEN** the report SHALL include a failed-check summary
- **AND** each failed-check summary entry SHALL include the semantic check ID, pull request number, stack entry commit SHA, check name, conclusion, and URL when available

### Requirement: Human-Readable Output

The `checks` command SHALL default to human-readable Markdown-compatible text output grouped by stack entry and pull request.

#### Scenario: Default output is text

- **WHEN** `stack-pr checks` is invoked without `--format`
- **THEN** the command SHALL produce text output
- **AND** the output SHALL group checks by stack entry in deterministic stack order
- **AND** each group SHALL identify the commit title, short SHA, pull request number, pull request URL, head branch, and base branch when known

#### Scenario: Failed checks are prominent in text

- **WHEN** any stack pull request has failed checks
- **THEN** text output SHALL visibly list the failed checks with their semantic check IDs and URLs when available before or within the relevant pull request group

#### Scenario: Empty checks text report

- **WHEN** all stack entries with PR metadata can be read and no checks are available
- **THEN** text output SHALL clearly state that no checks were found
- **AND** it SHALL still identify the inspected stack and PR count

#### Scenario: Per-PR failures in text report

- **WHEN** one or more pull requests cannot be read but at least one stack entry is reportable
- **THEN** text output SHALL include a warning for each unreadable pull request or stack entry
- **AND** it SHALL continue rendering available checks from other pull requests

### Requirement: JSON Output

The `checks` command SHALL support `--format json` and emit a single machine-readable JSON object suitable for agents.

#### Scenario: JSON output is structured

- **WHEN** `stack-pr checks --format json` is invoked
- **THEN** stdout SHALL contain exactly one JSON object
- **AND** the object SHALL include `schema_version`, `command`, `repository`, `range`, `stack`, `pull_requests`, and `failed_checks` fields
- **AND** it SHALL contain no ANSI escape sequences, terminal hyperlinks, or human progress logs

#### Scenario: Pull request JSON fields

- **WHEN** JSON output includes a pull request entry
- **THEN** the entry SHALL include pull request number, URL, head branch, base branch, stack index, commit SHA, short SHA, commit title, status, checks, and lightweight comment summary when available

#### Scenario: Check JSON fields

- **WHEN** JSON output includes a check entry
- **THEN** the entry SHALL include `id`, provider, name, status, conclusion, required state, and URL when available
- **AND** it SHALL include provider-specific identifiers when available

#### Scenario: Failed-check JSON summary

- **WHEN** JSON output includes failed checks
- **THEN** the top-level `failed_checks` array SHALL contain one entry per failed check in deterministic stack order
- **AND** each entry SHALL include enough identity to route follow-up work to the relevant pull request, commit, and check

#### Scenario: Unknown format is rejected

- **WHEN** `stack-pr checks --format <unknown>` is invoked with a value other than `text` or `json`
- **THEN** the command SHALL exit non-zero with a clear error message

### Requirement: Filtering

The `checks` command SHALL provide filtering flags that reduce output without changing the underlying stack or GitHub state.

#### Scenario: Failed-only filtering

- **WHEN** `stack-pr checks --failed-only` is invoked
- **THEN** the report SHALL include only failed checks and pull request groups needed to contextualize those failures
- **AND** it SHALL still report pull requests or stack entries whose check state could not be read

#### Scenario: Required-only filtering

- **WHEN** `stack-pr checks --required-only` is invoked
- **THEN** the report SHALL include only checks known to be required
- **AND** it SHALL NOT include checks whose required state is `false` or `unknown`

#### Scenario: Pull request filtering

- **WHEN** `stack-pr checks --pr <number>` is invoked
- **THEN** the report SHALL include only the stack entry associated with that pull request number
- **AND** the command SHALL report a clear invocation error if no stack entry is associated with that pull request number

#### Scenario: Commit filtering

- **WHEN** `stack-pr checks --commit <sha>` is invoked
- **THEN** the report SHALL include only the stack entry whose commit SHA matches the provided full or unambiguous abbreviated SHA
- **AND** the command SHALL report a clear invocation error if no stack entry matches

### Requirement: Lightweight Comment Summary

The checks report SHALL include a brief pull-request-level comment and review-attention summary when available, while leaving full comment inspection to the `comments` command.

#### Scenario: Comment counts are summarized

- **WHEN** a stack pull request has conversation comments, reviews, review comments, or review threads available from GitHub
- **THEN** the checks report SHALL include counts for those categories when available
- **AND** the report SHALL NOT require fetching or rendering full comment thread bodies

#### Scenario: Brief comment snippets are bounded

- **WHEN** the checks report includes comment snippets
- **THEN** each snippet SHALL be bounded in count and length
- **AND** each snippet SHALL include enough context to identify the pull request and source category

#### Scenario: Full comment inspection remains separate

- **WHEN** comment or review-attention summary indicates that detailed inspection may be useful
- **THEN** the checks report SHALL point the user or agent toward `stack-pr comments` for full comment details
- **AND** the checks report SHALL NOT attempt to render full review-thread trees

### Requirement: Error Handling

The `checks` command SHALL distinguish invocation errors from reportable per-stack-entry failures.

#### Scenario: Missing gh is an invocation error

- **WHEN** `stack-pr checks` is invoked and the GitHub CLI is not installed
- **THEN** the command SHALL exit non-zero with a clear message that `gh` is required

#### Scenario: GitHub authentication failure

- **WHEN** GitHub rejects all check queries because the user is not authenticated or authorized
- **THEN** the command SHALL exit non-zero with a clear authentication or authorization message

#### Scenario: Individual PR read failure

- **WHEN** one pull request cannot be read because it is missing, inaccessible, deleted, or otherwise fails while other pull requests can be read
- **THEN** the command SHALL include that failure in the report for the relevant stack entry
- **AND** the command SHALL continue reporting checks for readable pull requests

#### Scenario: Empty stack

- **WHEN** `stack-pr checks` is invoked and the stack is empty
- **THEN** the command SHALL produce an empty-stack report in the requested format
- **AND** it SHALL NOT query GitHub for pull request checks

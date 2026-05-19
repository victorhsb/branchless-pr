## ADDED Requirements

### Requirement: GitHub Availability Check
The `agent diagnose` command SHALL include a stable `github_availability` check entry that reports whether GitHub appears reachable to the configured `gh` CLI when online mode is enabled. The check SHALL be read-only and SHALL NOT perform any GitHub write operation.

#### Scenario: Offline mode does not probe GitHub availability
- **WHEN** `stack-pr agent diagnose` is invoked without `--online`
- **THEN** the report SHALL include a `github_availability` check with status `unknown` and a message indicating that `--online` was not specified
- **AND** the command SHALL NOT contact GitHub or any other remote service for the availability check

#### Scenario: Online mode reports GitHub available
- **WHEN** `stack-pr agent diagnose --online` is invoked and a read-only GitHub availability probe succeeds
- **THEN** the report SHALL include a `github_availability` check with status `ok`
- **AND** the check message SHALL indicate that GitHub appears reachable

#### Scenario: Online mode reports likely GitHub outage
- **WHEN** `stack-pr agent diagnose --online` is invoked and the GitHub availability probe fails with a likely service outage or transport-level availability failure
- **THEN** the report SHALL include a `github_availability` check with status `blocking`
- **AND** the check SHALL include `blocks` listing at least `submit`, `land`, and `abandon`
- **AND** the check SHALL include `suggested_fix` directing the user or agent to wait and retry after GitHub availability recovers
- **AND** the command SHALL continue evaluating remaining checks where they can be evaluated without relying on live GitHub state
- **AND** the command SHALL exit with code `0`

#### Scenario: Authentication failure is not classified as outage
- **WHEN** `stack-pr agent diagnose --online` is invoked and `gh` reports an authentication or authorization failure
- **THEN** the `github_availability` check SHALL NOT classify that failure as a GitHub outage
- **AND** authentication state SHALL be surfaced by the `github_authentication` check

#### Scenario: Repository-specific PR failure is not classified as outage
- **WHEN** `stack-pr agent diagnose --online` can reach GitHub but an individual PR lookup fails because the PR is missing, inaccessible, or repository-specific
- **THEN** the `github_availability` check SHALL NOT classify that failure as a GitHub outage
- **AND** the individual PR lookup result SHALL be surfaced by the relevant online PR-state check

### Requirement: Outage-Safe Online PR Checks
When `github_availability` has status `blocking`, online PR-state checks SHALL avoid making conclusions from live PR state and SHALL report that live PR state is unavailable because GitHub appears unavailable.

#### Scenario: PR state is skipped during likely outage
- **WHEN** `stack-pr agent diagnose --online` detects a blocking `github_availability` check before evaluating live PR state
- **THEN** the `online_pr_state` check SHALL have status `unknown` or `blocking`
- **AND** its message SHALL indicate that live PR state was not trusted because GitHub appears unavailable
- **AND** the report SHALL NOT claim that the stack is fully synchronized with live GitHub PR state

#### Scenario: Local checks still run during likely outage
- **WHEN** `stack-pr agent diagnose --online` detects a likely GitHub outage
- **THEN** local checks such as repository detection, working tree cleanliness, rebase state, base/head resolution, branch-name template validity, and stack discovery SHALL still be evaluated when possible

## MODIFIED Requirements

### Requirement: Recommendation Decision Tree

The `agent diagnose` command SHALL choose its recommendation according to the following priority, evaluated top-down on the first matching condition:

1. If the working directory is not inside a Git repository, recommend changing into a Git repository.
2. Otherwise, if a rebase is in progress, recommend finishing or aborting the rebase.
3. Otherwise, if the stack is empty, recommend creating commits before using `stack-pr`.
4. Otherwise, if the working tree is dirty, recommend cleaning the working tree (commit, stash, or revert).
5. Otherwise, if online mode detected that GitHub appears unavailable, recommend waiting for GitHub availability to recover or using local-only inspection.
6. Otherwise, if one or more commits are missing PR metadata, recommend `stack-pr submit --dry-run`.
7. Otherwise (the stack appears fully submitted), recommend `stack-pr view --format json` and surface `stack-pr land` only as a conservative potential next action that requires confirmation.

#### Scenario: Not a git repo

- **WHEN** the working directory is not inside a Git repository
- **THEN** the recommendation SHALL direct the user to change into a Git repository

#### Scenario: Rebase in progress

- **WHEN** a rebase is in progress
- **THEN** the recommendation SHALL direct the user to finish or abort the rebase

#### Scenario: Empty stack

- **WHEN** the stack is empty
- **THEN** the recommendation SHALL direct the user to create commits before using `stack-pr`

#### Scenario: Dirty working tree

- **WHEN** the working tree is dirty and no higher-priority condition matches
- **THEN** the recommendation SHALL direct the user to clean the working tree

#### Scenario: GitHub unavailable

- **WHEN** `stack-pr agent diagnose --online` detects a blocking `github_availability` check and no higher-priority condition matches
- **THEN** the recommendation SHALL direct the user or agent to wait for GitHub availability to recover or use local-only inspection
- **AND** the primary recommendation SHALL NOT be `stack-pr submit`, `stack-pr land`, or `stack-pr abandon`
- **AND** the recommendation SHALL state that live GitHub state cannot currently be trusted for mutating stack-pr operations

#### Scenario: Missing PR metadata

- **WHEN** one or more commits in the stack lack PR metadata and no higher-priority condition matches
- **THEN** the recommendation `command` SHALL be `stack-pr submit --dry-run`
- **AND** the recommendation SHALL state that no mutation has yet occurred and that the dry run can preview the create-or-update plan

#### Scenario: Fully submitted stack

- **WHEN** every commit in the stack has PR metadata, the working tree is clean, and no rebase is in progress
- **THEN** the primary recommendation `command` SHALL be `stack-pr view --format json`
- **AND** the report MAY surface `stack-pr land` as a separate potential next action

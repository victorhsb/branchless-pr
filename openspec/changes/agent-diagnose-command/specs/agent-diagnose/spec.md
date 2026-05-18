## ADDED Requirements

### Requirement: Agent Diagnose Subcommand

The `stack-pr` CLI SHALL provide a `diagnose` subcommand under the `agent` command group that inspects the current repository and emits a structured report describing repository, stack, and check state.

#### Scenario: Diagnose subcommand is invokable

- **WHEN** `stack-pr agent diagnose` is invoked from a shell
- **THEN** the command SHALL execute and produce a report on standard output

#### Scenario: Diagnose is read-only

- **WHEN** `stack-pr agent diagnose` is invoked under any conditions
- **THEN** the command SHALL NOT perform any local Git mutation, including but not limited to checkouts, rebases, commit amendments, branch creation, branch deletion, stash save, stash pop, index modification, or working-tree modification
- **AND** the command SHALL NOT perform any remote push, fetch-write, or GitHub write operation, including but not limited to creating, editing, closing, merging, or changing the draft state of pull requests

### Requirement: Output Format Selection

The `agent diagnose` command SHALL support a `--format` flag with values `text` and `json`. The default value SHALL be `text`. The `text` format SHALL render human-readable Markdown.

#### Scenario: Default format is text

- **WHEN** `stack-pr agent diagnose` is invoked without `--format`
- **THEN** the command SHALL emit a Markdown report on standard output

#### Scenario: JSON format is selectable

- **WHEN** `stack-pr agent diagnose --format json` is invoked
- **THEN** the command SHALL emit a single JSON document on standard output that conforms to the diagnosis JSON schema

#### Scenario: Unknown format value

- **WHEN** `stack-pr agent diagnose --format <unsupported>` is invoked with a value other than `text` or `json`
- **THEN** the command SHALL report an invalid-flag error
- **AND** SHALL exit with a non-zero exit code reserved for invocation errors

### Requirement: Online Mode Flag

The `agent diagnose` command SHALL support an `--online` flag that defaults to false. When the flag is false, the command SHALL NOT perform any network I/O. When the flag is true, the command MAY consult GitHub (for example via `gh`) to fetch live pull request state.

#### Scenario: Default mode is offline

- **WHEN** `stack-pr agent diagnose` is invoked without `--online`
- **THEN** the command SHALL NOT contact GitHub or any other remote service

#### Scenario: Online mode fetches PR state

- **WHEN** `stack-pr agent diagnose --online` is invoked and the stack contains entries with PR metadata
- **THEN** the command MAY query GitHub for live PR state
- **AND** the result of any such query SHALL be reflected in the report

#### Scenario: Online mode network failure is degraded, not fatal

- **WHEN** `stack-pr agent diagnose --online` is invoked and a GitHub query fails
- **THEN** the command SHALL record the failure as a check entry with status `unknown` or `warning`
- **AND** SHALL continue running remaining checks
- **AND** SHALL exit with code `0`

### Requirement: Exit Code Stability

The `agent diagnose` command SHALL exit with code `0` for any reportable outcome, including outcomes in which one or more checks have status `blocking`. Non-zero exit codes SHALL be reserved for catastrophic, unexpected failures (for example, the JSON encoder failing) and for invocation errors such as an invalid flag value.

#### Scenario: Clean repository

- **WHEN** `stack-pr agent diagnose` is invoked in a fully healthy repository
- **THEN** the command SHALL exit with code `0`

#### Scenario: Blocking checks present

- **WHEN** `stack-pr agent diagnose` is invoked and one or more checks have status `blocking`
- **THEN** the command SHALL still exit with code `0`
- **AND** the report SHALL include each blocking check with its status, message, `blocks`, and `suggested_fix`

#### Scenario: Not a git repository

- **WHEN** `stack-pr agent diagnose` is invoked outside any Git repository
- **THEN** the command SHALL emit a report indicating that the working directory is not in a Git repository
- **AND** SHALL exit with code `0`

### Requirement: Degraded-Mode Check Behavior

Each individual check performed by `agent diagnose` SHALL report a `status` rather than aborting the command. A check that cannot determine its result (for example, because an underlying helper returned an error) SHALL be reported with status `unknown` and a message describing why it could not be evaluated.

#### Scenario: Individual check failure is contained

- **WHEN** the underlying logic for any single check returns an error during `agent diagnose`
- **THEN** the corresponding check entry SHALL be reported with status `unknown`
- **AND** remaining checks SHALL still be evaluated
- **AND** the command SHALL exit with code `0`

#### Scenario: Working tree dirty is reported, not raised

- **WHEN** `stack-pr agent diagnose` is invoked while tracked files have staged or unstaged changes
- **THEN** the command SHALL NOT abort
- **AND** the report SHALL include a `working_tree_clean` check entry with status `blocking`
- **AND** that entry SHALL include `blocks` listing commands that require a clean working tree (for example `submit`, `land`, `abandon`)
- **AND** that entry SHALL include a `suggested_fix` describing how to clean the working tree

### Requirement: Check Entry Schema

Each check entry emitted in the JSON output SHALL include at minimum the fields `id`, `status`, and `message`. The `status` field SHALL take one of the values `ok`, `warning`, `blocking`, or `unknown`. A check entry with status `blocking` SHALL additionally include a `blocks` field (a list of command names) and a `suggested_fix` field (a human-readable remediation hint).

#### Scenario: Check entry minimum fields

- **WHEN** any check entry is included in the JSON report
- **THEN** that entry SHALL include `id` (a stable string identifier), `status` (one of `ok`, `warning`, `blocking`, `unknown`), and `message` (a human-readable description)

#### Scenario: Blocking check entry fields

- **WHEN** a check entry has status `blocking`
- **THEN** that entry SHALL include `blocks` listing the command names this issue prevents
- **AND** SHALL include `suggested_fix` describing how a user or agent can resolve the blocker

### Requirement: Required Checks

The `agent diagnose` command SHALL perform the following best-effort checks and surface each one as a check entry in the report. A check that cannot be evaluated SHALL be reported with status `unknown` rather than omitted.

#### Scenario: Git repository check

- **WHEN** `stack-pr agent diagnose` is invoked
- **THEN** the report SHALL include a check that reports whether the working directory is inside a Git repository

#### Scenario: gh installed check

- **WHEN** `stack-pr agent diagnose` is invoked
- **THEN** the report SHALL include a check that reports whether the `gh` CLI is installed and discoverable

#### Scenario: GitHub authentication check

- **WHEN** `stack-pr agent diagnose` is invoked
- **THEN** the report SHALL include a check that reports whether GitHub authentication is available to `gh`

#### Scenario: Working tree clean check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository
- **THEN** the report SHALL include a check that reports whether the working tree is clean

#### Scenario: Rebase in progress check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository
- **THEN** the report SHALL include a check that reports whether a rebase is currently in progress

#### Scenario: Base and head resolution check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository
- **THEN** the report SHALL include a check that reports whether the base and head revisions used for stack discovery can be resolved

#### Scenario: Target branch existence check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository
- **THEN** the report SHALL include a check that reports whether the configured target branch exists on the configured remote

#### Scenario: Branch name template check

- **WHEN** `stack-pr agent diagnose` is invoked
- **THEN** the report SHALL include a check that reports whether the configured branch-name template is valid

#### Scenario: Stack size check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository whose base and head can be resolved
- **THEN** the report SHALL include a check (or top-level stack summary) reporting the number of commits in the stack

#### Scenario: Stack metadata coverage check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository whose stack can be discovered
- **THEN** the report SHALL include a check (or top-level stack summary) reporting how many commits already carry `stack-info` metadata

#### Scenario: Missing PR check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository whose stack can be discovered
- **THEN** the report SHALL include a check (or top-level stack summary) reporting how many commits are missing a pull request

#### Scenario: PR base coherence check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository whose stack can be discovered and at least one entry has PR metadata
- **THEN** the report SHALL include a check that reports whether the PR base relationships across the stack are coherent with bottom-to-top stacking

#### Scenario: Local base behind remote target check

- **WHEN** `stack-pr agent diagnose` is invoked inside a Git repository whose base can be resolved
- **THEN** the report SHALL include a check that reports whether the local base is behind the configured remote target branch

#### Scenario: Online PR state check

- **WHEN** `stack-pr agent diagnose --online` is invoked and one or more stack entries have PR metadata
- **THEN** the report SHALL include a check (or per-entry annotation) reporting the live PR state retrieved from GitHub
- **AND** in offline mode this check SHALL NOT be present or SHALL be reported with status `unknown` and a message indicating that `--online` was not specified

### Requirement: JSON Output Envelope

In `--format json` mode the command SHALL emit a single JSON object containing at minimum:

- a `schema_version` field with a stable string value identifying the diagnosis JSON schema version,
- a top-level `status` field summarizing the overall result (`ok`, `warning`, `blocking`, or `unknown`),
- a `repo` object describing repository context (such as root, current branch, remote, target, base, head),
- a `stack` object summarizing stack inspection (such as size, entries with PR metadata, entries missing PRs),
- a `checks` array of check entries, and
- a `recommendation` object as described in the recommendation requirements.

The JSON output SHALL be stable across patch-level releases for a given `schema_version` value; incompatible changes SHALL require incrementing `schema_version`.

#### Scenario: Schema version present

- **WHEN** `stack-pr agent diagnose --format json` is invoked
- **THEN** the emitted JSON object SHALL include a `schema_version` field whose value is a non-empty string

#### Scenario: Required top-level fields present

- **WHEN** `stack-pr agent diagnose --format json` is invoked
- **THEN** the emitted JSON object SHALL include `status`, `repo`, `stack`, `checks`, and `recommendation` fields

#### Scenario: Top-level status reflects worst check

- **WHEN** the emitted JSON object includes one or more check entries
- **THEN** the top-level `status` SHALL be at least as severe as the most severe check status, with severity ordered `ok` < `warning` < `blocking`, and `unknown` reported when overall severity cannot be determined

### Requirement: Text Output Information Set

In `--format text` mode the command SHALL emit a human-readable Markdown report whose information set is equivalent to the JSON output. The report SHALL surface the repository context, the stack summary, each check (with at minimum its identifier or short label, status, and message), any blocking check's `suggested_fix`, and the recommendation including its command, reason, and safety metadata. Exact headings, ordering, and prose are an implementation choice.

#### Scenario: Text report includes recommendation

- **WHEN** `stack-pr agent diagnose` is invoked without `--format json`
- **THEN** the output SHALL identify the recommended command, the reason for the recommendation, and whether the recommended command has side effects and requires explicit confirmation

#### Scenario: Text report surfaces blocking guidance

- **WHEN** the emitted report includes one or more blocking checks
- **THEN** the text output SHALL surface the suggested fix for each blocking check

### Requirement: Recommendation Contract

The `agent diagnose` command SHALL include a recommendation in every report. The recommendation object SHALL contain at minimum the fields `command`, `reason`, `side_effects`, and `requires_confirmation`. When `side_effects` is true, `requires_confirmation` SHALL be true.

#### Scenario: Recommendation always present

- **WHEN** `stack-pr agent diagnose` is invoked
- **THEN** the report SHALL include a recommendation, even when the repository is not a Git repository or no useful action is available

#### Scenario: Safety metadata fields present

- **WHEN** any recommendation is emitted
- **THEN** that recommendation SHALL include `command`, `reason`, `side_effects` (boolean), and `requires_confirmation` (boolean)

#### Scenario: Side effects imply confirmation

- **WHEN** a recommendation has `side_effects` equal to true
- **THEN** that recommendation SHALL also have `requires_confirmation` equal to true

### Requirement: Recommendation Decision Tree

The `agent diagnose` command SHALL choose its recommendation according to the following priority, evaluated top-down on the first matching condition:

1. If the working directory is not inside a Git repository, recommend changing into a Git repository.
2. Otherwise, if a rebase is in progress, recommend finishing or aborting the rebase.
3. Otherwise, if the stack is empty, recommend creating commits before using `stack-pr`.
4. Otherwise, if the working tree is dirty, recommend cleaning the working tree (commit, stash, or revert).
5. Otherwise, if one or more commits are missing PR metadata, recommend `stack-pr submit --dry-run`.
6. Otherwise (the stack appears fully submitted), recommend `stack-pr view --format json` and surface `stack-pr land` only as a conservative potential next action that requires confirmation.

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

#### Scenario: Missing PR metadata

- **WHEN** one or more commits in the stack lack PR metadata and no higher-priority condition matches
- **THEN** the recommendation `command` SHALL be `stack-pr submit --dry-run`
- **AND** the recommendation SHALL state that no mutation has yet occurred and that the dry run can preview the create-or-update plan

#### Scenario: Fully submitted stack

- **WHEN** every commit in the stack has PR metadata, the working tree is clean, and no rebase is in progress
- **THEN** the primary recommendation `command` SHALL be `stack-pr view --format json`
- **AND** the report MAY surface `stack-pr land` as a separate potential next action

### Requirement: Conservative Land Recommendation

The `agent diagnose` command SHALL NOT recommend `stack-pr land` as its primary recommendation. Any reference to `stack-pr land` SHALL be surfaced only as a potential next action, and any such entry SHALL be marked with `side_effects: true` and `requires_confirmation: true`.

#### Scenario: Land never primary

- **WHEN** the recommendation decision tree yields the fully-submitted state
- **THEN** the primary `recommendation.command` SHALL NOT be `stack-pr land`

#### Scenario: Land marked conservative

- **WHEN** `stack-pr land` is surfaced anywhere in the report as a potential next action
- **THEN** that entry SHALL include `side_effects: true` and `requires_confirmation: true`
- **AND** SHALL describe `land` as conservative guidance rather than an outright recommendation

## MODIFIED Requirements

### Requirement: Stack Checks Command

The `stack-pr` CLI SHALL provide a top-level `checks` command that reports GitHub check and lightweight review-attention state for pull requests represented by the current stack metadata.

#### Scenario: Checks command is available

- **WHEN** the user runs `stack-pr checks --help`
- **THEN** the CLI SHALL describe a read-only command for reporting check state across the current stack's pull requests
- **AND** the help SHALL include supported output format, filtering, and verbosity flags

#### Scenario: Stack PRs are discovered

- **WHEN** `stack-pr checks` is invoked inside a repository with a non-empty stack
- **THEN** the command SHALL discover stack entries using the same base, head, branch-template, and stack metadata rules as other stack inspection commands
- **AND** the command SHALL associate fetched check state with the stack entry and pull request it belongs to

#### Scenario: Missing PR metadata is reported

- **WHEN** a stack entry has no pull request metadata
- **THEN** the command SHALL report that entry as missing PR metadata
- **AND** the command SHALL continue collecting check state for other entries that have PR metadata

### Requirement: Human-Readable Output

The `checks` command SHALL default to summary-first human-readable Markdown-compatible text output grouped by stack entry and pull request, with exhaustive per-check detail available through `--verbose`.

#### Scenario: Default output is text

- **WHEN** `stack-pr checks` is invoked without `--format`
- **THEN** the command SHALL produce text output
- **AND** the output SHALL group pull request summaries by stack entry in deterministic stack order
- **AND** each group SHALL identify the commit title, short SHA, pull request number, pull request URL, head branch, and base branch when known

#### Scenario: Stack coverage is summarized

- **WHEN** `stack-pr checks` renders text output
- **THEN** the output SHALL identify the inspected stack size and pull request coverage
- **AND** it SHALL make missing PR metadata, unreadable pull requests, and active `--pr` or `--commit` filters visible

#### Scenario: Pull request status is summarized

- **WHEN** a stack pull request has check data
- **THEN** default text output SHALL include a compact roll-up for that pull request
- **AND** the roll-up SHALL include useful check counts such as passing, failing, in-progress, pending, skipped, and unknown where present
- **AND** the roll-up SHALL include lightweight comment and review counts when available

#### Scenario: Duplicate checks are collapsed in default text

- **WHEN** default text output includes multiple checks with the same visible check identity
- **THEN** the output SHALL summarize them as one visible item or count instead of rendering every duplicate check line
- **AND** the visible state SHALL prefer the most actionable state, including failed before in-progress, pending, successful, skipped, or unknown states

#### Scenario: Unknown required state is omitted from default text

- **WHEN** GitHub does not expose whether a check is required
- **THEN** default text output SHALL NOT print `required: unknown` for that check
- **AND** the report SHALL preserve unknown required state in JSON output and verbose text detail

#### Scenario: Failed checks are prominent in text

- **WHEN** any stack pull request has failed checks
- **THEN** text output SHALL visibly list the failed checks with their semantic check IDs and URLs when available before or within the relevant pull request group

#### Scenario: Verbose text renders full check detail

- **WHEN** `stack-pr checks --verbose` is invoked with text output
- **THEN** the output SHALL include the summary-first content
- **AND** it SHALL render every retained check in deterministic order with semantic check ID, name, status or conclusion, required state when available, and URL when available

#### Scenario: Empty checks text report

- **WHEN** all stack entries with PR metadata can be read and no checks are available
- **THEN** text output SHALL clearly state that no checks were found
- **AND** it SHALL still identify the inspected stack and PR count

#### Scenario: Per-PR failures in text report

- **WHEN** one or more pull requests cannot be read but at least one stack entry is reportable
- **THEN** text output SHALL include a warning for each unreadable pull request or stack entry
- **AND** it SHALL continue rendering available checks from other pull requests

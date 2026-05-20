## ADDED Requirements

### Requirement: Stack Comments Command

The `stack-pr` CLI SHALL provide a top-level `comments` command that collects review and conversation comments for pull requests represented by the current stack metadata.

#### Scenario: Comments command is available

- **WHEN** the user runs `stack-pr comments --help`
- **THEN** the CLI SHALL describe a read-only command for collecting comments across the current stack's pull requests
- **AND** the help SHALL include the supported output format and filtering flags

#### Scenario: Stack PRs are discovered

- **WHEN** `stack-pr comments` is invoked inside a repository with a non-empty stack
- **THEN** the command SHALL discover stack entries using the same base, head, branch-template, and stack metadata rules as other stack inspection commands
- **AND** the command SHALL associate fetched comments with the stack entry and pull request they belong to

#### Scenario: Missing PR metadata is reported

- **WHEN** a stack entry has no pull request metadata
- **THEN** the command SHALL report that entry as missing PR metadata
- **AND** the command SHALL continue collecting comments for other entries that have PR metadata

### Requirement: Read-Only Behavior

The `comments` command SHALL NOT mutate local Git state, remote branches, commit messages, or GitHub pull request state.

#### Scenario: Dirty worktree is allowed

- **WHEN** `stack-pr comments` is invoked while tracked files have staged or unstaged changes
- **THEN** the command SHALL still attempt to produce a comments report
- **AND** it SHALL NOT require the user to clean or stash the worktree first

#### Scenario: No GitHub writes

- **WHEN** `stack-pr comments` fetches comment information from GitHub
- **THEN** it SHALL use read-only GitHub operations
- **AND** it SHALL NOT create, edit, close, merge, mark ready, resolve, or delete pull requests or comments

#### Scenario: No local stack mutation

- **WHEN** `stack-pr comments` runs successfully or with reportable per-PR failures
- **THEN** it SHALL NOT checkout branches, create branches, delete branches, amend commits, rebase, stash, push, or fetch in a way that mutates repository state

### Requirement: Comment Sources

The comments report SHALL include GitHub pull request feedback from conversation comments, reviews, review comments, and review threads when those sources are available through `gh`.

#### Scenario: Conversation comments are included

- **WHEN** a stack pull request has issue-style conversation comments
- **THEN** the report SHALL include those comments with kind `conversation`
- **AND** each comment SHALL include author, body, creation time, update time when available, URL when available, and the owning pull request

#### Scenario: Reviews are included

- **WHEN** a stack pull request has submitted reviews
- **THEN** the report SHALL include those reviews with kind `review`
- **AND** each review SHALL include author, body when present, submitted time when available, state when available, URL when available, and the owning pull request

#### Scenario: Review threads are included

- **WHEN** a stack pull request has review threads
- **THEN** the report SHALL include those threads with kind `review_thread`
- **AND** each thread SHALL include resolution state when available, file path when available, line or range context when available, URL when available, and comments or replies in chronological order when available

#### Scenario: Review comments are included

- **WHEN** GitHub exposes review comments separately from review threads
- **THEN** the report SHALL include those comments with kind `review_comment`
- **AND** each comment SHALL include author, body, creation time, update time when available, URL when available, path when available, and line context when available

### Requirement: Human-Readable Output

The `comments` command SHALL default to a human-readable Markdown-compatible text output grouped by stack entry and pull request.

#### Scenario: Default output is text

- **WHEN** `stack-pr comments` is invoked without `--format`
- **THEN** the command SHALL produce text output
- **AND** the output SHALL group comments by stack entry in deterministic stack order
- **AND** each group SHALL identify the commit title, short SHA, pull request number, pull request URL, head branch, and base branch when known

#### Scenario: Empty comments text report

- **WHEN** all stack entries with PR metadata can be read and no matching comments exist
- **THEN** text output SHALL clearly state that no matching comments were found
- **AND** it SHALL still identify the inspected stack and PR count

#### Scenario: Per-PR failures in text report

- **WHEN** one or more pull requests cannot be read but at least one stack entry is reportable
- **THEN** text output SHALL include a warning for each unreadable pull request or stack entry
- **AND** it SHALL continue rendering available comments from other pull requests

### Requirement: JSON Output

The `comments` command SHALL support `--format json` and emit a single machine-readable JSON object suitable for agents.

#### Scenario: JSON output is structured

- **WHEN** `stack-pr comments --format json` is invoked
- **THEN** stdout SHALL contain exactly one JSON object
- **AND** the object SHALL include `schema_version`, `command`, `repository`, `range`, `stack`, and `pull_requests` fields
- **AND** it SHALL contain no ANSI escape sequences, terminal hyperlinks, or human progress logs

#### Scenario: Stack entry JSON fields

- **WHEN** JSON output includes a stack entry
- **THEN** the entry SHALL include commit SHA, short SHA, title, stack index, head branch, base branch, PR URL when known, PR number when known, and a status indicating whether comments were fetched, missing, empty, or failed

#### Scenario: Comment JSON fields

- **WHEN** JSON output includes a comment, review, or thread item
- **THEN** the item SHALL include a stable `id` when GitHub provides one, `kind`, owning PR number, author, body, URL when available, timestamps when available, and optional location or resolution fields when available

#### Scenario: Unknown format is rejected

- **WHEN** `stack-pr comments --format <unknown>` is invoked with a value other than `text` or `json`
- **THEN** the command SHALL exit non-zero with a clear error message

### Requirement: Filtering

The `comments` command SHALL provide filtering flags that reduce output without changing the underlying stack.

#### Scenario: Unresolved-only filtering

- **WHEN** `stack-pr comments --unresolved-only` is invoked
- **THEN** the report SHALL include only comments or threads that GitHub identifies as unresolved or otherwise requiring attention
- **AND** the report SHALL NOT guess unresolved state for comment kinds that do not expose resolution status

#### Scenario: Comment kind filtering

- **WHEN** `stack-pr comments --kind <kinds>` is invoked
- **THEN** the report SHALL include only the requested comment kinds
- **AND** unsupported kind values SHALL be rejected with a clear error

#### Scenario: Author filtering

- **WHEN** `stack-pr comments --author <login>` is invoked
- **THEN** the report SHALL include only matching comments, reviews, or threads authored by that GitHub login
- **AND** pull request groups with no matching items SHALL be shown as empty rather than omitted in JSON output

### Requirement: Error Handling

The `comments` command SHALL distinguish invocation errors from reportable per-stack-entry failures.

#### Scenario: Missing gh is an invocation error

- **WHEN** `stack-pr comments` is invoked and the GitHub CLI is not installed
- **THEN** the command SHALL exit non-zero with a clear message that `gh` is required

#### Scenario: GitHub authentication failure

- **WHEN** GitHub rejects all comment queries because the user is not authenticated or authorized
- **THEN** the command SHALL exit non-zero with a clear authentication or authorization message

#### Scenario: Individual PR read failure

- **WHEN** one pull request cannot be read because it is missing, inaccessible, deleted, or otherwise fails while other pull requests can be read
- **THEN** the command SHALL include that failure in the report for the relevant stack entry
- **AND** the command SHALL continue reporting comments for readable pull requests

#### Scenario: Empty stack

- **WHEN** `stack-pr comments` is invoked and the stack is empty
- **THEN** the command SHALL produce an empty-stack report in the requested format
- **AND** it SHALL NOT query GitHub for pull request comments

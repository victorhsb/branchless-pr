# View Algorithm

## Purpose

Define the canonical behavior of `stack-pr view` for inspecting a stack of commits.

The view command is a read-only inspection command. It discovers the commit stack in the current repository, resolves PR metadata and branch information, and renders the stack either as ANSI-colored text (default) or as structured JSON (`--format json`). Unlike mutating commands, view does not require a clean working tree and does not modify local or remote state.

## Requirements

### Requirement: Base Fast-Forward Warning

Before loading the stack, the command SHALL detect when the local base is behind the remote target in an auto-updatable way and warn the user instead of modifying anything.

#### Scenario: Local base behind remote target

- **WHEN** the local base is an ancestor of `REMOTE/TARGET`
- **AND** `REMOTE/TARGET` is an ancestor of `HEAD`
- **AND** the base hash differs from `REMOTE/TARGET`
- **THEN** the command SHALL print a warning that the local base is behind the remote target
- **AND** the command SHALL suggest `git rebase REMOTE/TARGET base` and `git checkout <original_branch>` as follow-up commands
- **AND** the command SHALL NOT load or print the stack

#### Scenario: Local base up to date

- **WHEN** the local base is not behind `REMOTE/TARGET` in the auto-updatable way
- **THEN** the command SHALL proceed to load the stack normally

### Requirement: Stack Discovery

The command SHALL discover and load the stack from the configured commit range.

#### Scenario: Stack loaded from base..head

- **WHEN** view runs
- **THEN** the stack SHALL be loaded from commits in `base..head`
- **AND** stack entries SHALL be ordered oldest-to-newest internally

#### Scenario: Empty stack

- **WHEN** the discovered stack contains no commits
- **THEN** the command SHALL print `Empty stack!`
- **AND** the command SHALL return without further action

### Requirement: Head Branch Resolution

The command SHALL assign head branches to entries that are missing metadata heads by scanning remote refs, without creating branches or pushing.

#### Scenario: Missing head branch resolved from remote

- **WHEN** a stack entry lacks a head branch in its metadata
- **THEN** the command SHALL scan remote refs to find a matching branch
- **AND** the discovered head branch SHALL be assigned to the entry
- **AND** the command SHALL NOT create a new branch
- **AND** the command SHALL NOT push to the remote

#### Scenario: Existing head branch preserved

- **WHEN** a stack entry already has a head branch in its metadata
- **THEN** the command SHALL use that head branch without scanning the remote

### Requirement: Base Branch Assignment

The command SHALL compute the base branch for each stack entry.

#### Scenario: Bottom entry targets remote target

- **WHEN** base branches are computed
- **THEN** the first (bottom) entry's base branch SHALL be the remote target branch

#### Scenario: Higher entries target previous head

- **WHEN** base branches are computed for entries above the bottom
- **THEN** each subsequent entry's base branch SHALL be the previous entry's head branch

### Requirement: Text Output Format

By default, the command SHALL render the stack as ANSI-colored, Markdown-compatible text grouped by stack entry in newest-to-oldest order.

#### Scenario: Default text rendering

- **WHEN** `stack-pr view` is invoked without `--format`
- **THEN** output SHALL use ANSI-colored text with terminal hyperlinks
- **AND** each stack line SHALL follow the format:
  - `* <short-sha> (#<pr-number or no PR>, '<head>' -> '<base>'): <commit title>`
- **AND** output SHALL contain no command banners such as `VIEW` or generic success markers such as `SUCCESS!`

#### Scenario: Stack printing order

- **WHEN** the stack is rendered as text
- **THEN** entries SHALL be printed newest-to-oldest

### Requirement: JSON Output Format

The command SHALL support `--format json` to produce machine-readable JSON instead of the default text format.

#### Scenario: JSON format produces structured output

- **WHEN** `stack-pr view --format json` is invoked
- **THEN** output SHALL be a single JSON array ordered newest-to-oldest
- **AND** each array element SHALL be a flat object with the following fields:
  - `commit` — full commit hash
  - `short_sha` — abbreviated commit hash
  - `title` — first line of the commit message
  - `author` — full author string (name and email)
  - `author_name` — author name
  - `author_email` — author email
  - `pr_url` — pull request URL, or `""` if none
  - `pr_number` — pull request number, or `0` if none
  - `head_branch` — the branch name for this stack entry
  - `base_branch` — the base branch for this stack entry
- **AND** output SHALL contain no ANSI escape sequences, terminal hyperlinks, progress logs, or extra stdout text

#### Scenario: Missing PR fields in JSON

- **WHEN** a stack entry has no associated PR
- **THEN** `pr_url` SHALL be `""`
- **AND** `pr_number` SHALL be `0`

#### Scenario: Unknown format is rejected

- **WHEN** `stack-pr view --format <unknown>` is invoked with a value other than `text` or `json`
- **THEN** the command SHALL exit with an error returning a clear message

### Requirement: Post-View Tips

After printing the stack, the command SHALL provide guidance based on the completeness of PR metadata.

#### Scenario: Stack ready to land

- **WHEN** every entry has PR, head, and base metadata
- **THEN** the command SHALL indicate the stack is ready to land
- **AND** the command SHALL display update and land commands

#### Scenario: Stack not ready to land

- **WHEN** one or more entries lack PR, head, or base metadata
- **THEN** the command SHALL indicate the stack cannot be landed yet
- **AND** the command SHALL display the export (submit) command

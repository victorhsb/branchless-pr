## Purpose

Define behavior for the `bpr fix` command used to repair stack metadata on local HEAD.

## Requirements

### Requirement: Fix Command Registration

The CLI SHALL expose a first-class `fix` subcommand for local repair of stack metadata on `HEAD`.

#### Scenario: Fix command is registered

- **WHEN** the user runs `bpr fix --help`
- **THEN** the CLI SHALL exit successfully
- **AND** the help output SHALL describe `--pr`, `--replace`, and `--dry-run`

#### Scenario: PR flag is required

- **WHEN** the user runs `bpr fix` without `--pr`
- **THEN** the command SHALL exit non-zero with a clear validation error
- **AND** no local Git mutation SHALL occur

### Requirement: Fix Preflight

Before repairing metadata, `fix` SHALL validate repository state and block unsafe local rewrites.

#### Scenario: Dirty working tree blocks fix

- **WHEN** the working tree has staged or unstaged changes
- **THEN** `bpr fix --pr <number>` SHALL exit non-zero with a clean-tree error
- **AND** no local Git mutation SHALL occur

#### Scenario: In-progress Git operation blocks fix

- **WHEN** a rebase, merge, or cherry-pick operation is in progress
- **THEN** `bpr fix --pr <number>` SHALL exit non-zero with an actionable error
- **AND** no local Git mutation SHALL occur

#### Scenario: Stack discovery is not required before repair

- **WHEN** repository and PR preflight succeeds
- **THEN** `fix` SHALL inspect and amend `HEAD` directly
- **AND** it SHALL NOT require successful stack discovery before attempting the repair

### Requirement: Explicit PR Metadata Source

`fix` SHALL use the explicitly selected existing PR as the source for local `stack-info` metadata.

#### Scenario: Existing PR is loaded

- **WHEN** the user runs `bpr fix --pr <number>`
- **THEN** the command SHALL load PR `url`, `number`, `headRefName`, `baseRefName`, and `headRefOid` through `gh pr view`
- **AND** the command SHALL use the PR URL and head branch for the local metadata line

#### Scenario: PR head mismatch warns

- **WHEN** the selected PR's `headRefOid` differs from local `HEAD`
- **THEN** the command SHALL print a warning identifying the PR head SHA and local `HEAD` SHA
- **AND** the command SHALL continue with the local metadata repair

### Requirement: Local Metadata Repair

`fix` SHALL repair only the current local `HEAD` commit message.

#### Scenario: Missing metadata is appended

- **WHEN** `HEAD` has no `stack-info` metadata
- **AND** the user runs `bpr fix --pr <number>`
- **THEN** the command SHALL append `stack-info: PR: <pr-url>, branch: <head-branch>` to the current commit message
- **AND** the metadata line SHALL be separated from the commit title/body by at least one blank line
- **AND** the command SHALL amend `HEAD` with `git commit --amend -F -`

#### Scenario: Matching metadata is already fixed

- **WHEN** `HEAD` already has `stack-info` metadata for the selected PR URL and PR head branch
- **THEN** the command SHALL report that `HEAD` is already fixed
- **AND** the command SHALL NOT amend the commit

#### Scenario: Different metadata is refused by default

- **WHEN** `HEAD` already has `stack-info` metadata that differs from the selected PR
- **AND** `--replace` is not set
- **THEN** the command SHALL exit non-zero with an error explaining that existing metadata is present
- **AND** no local Git mutation SHALL occur

#### Scenario: Different metadata is replaced explicitly

- **WHEN** `HEAD` already has `stack-info` metadata that differs from the selected PR
- **AND** `--replace` is set
- **THEN** the command SHALL replace the existing metadata line with `stack-info: PR: <pr-url>, branch: <head-branch>`
- **AND** the command SHALL amend `HEAD` with `git commit --amend -F -`

### Requirement: Local-only Side Effects

`fix` SHALL be limited to local commit metadata repair.

#### Scenario: Fix does not publish

- **WHEN** the user runs `bpr fix --pr <number>`
- **THEN** the command SHALL NOT create or reset local generated branches
- **AND** the command SHALL NOT push to any remote
- **AND** the command SHALL NOT create, edit, retarget, mark draft, mark ready, merge, or close any PR

#### Scenario: Success hint points to submit

- **WHEN** `fix` completes successfully
- **THEN** the command SHALL print a hint that metadata was fixed locally
- **AND** the hint SHALL tell the user to run `bpr submit` to push the amended commit and update PRs

### Requirement: Fix Dry-run

`fix --dry-run` SHALL report the planned repair without mutating local Git or GitHub state.

#### Scenario: Dry-run reports planned metadata

- **WHEN** the user runs `bpr fix --pr <number> --dry-run`
- **THEN** the command SHALL load the selected PR and inspect `HEAD`
- **AND** the command SHALL print the PR URL, PR head branch, local `HEAD` SHA, existing metadata state, and the metadata line it would add or replace
- **AND** the command SHALL state that no commit was changed

#### Scenario: Dry-run does not amend

- **WHEN** the user runs `bpr fix --pr <number> --dry-run`
- **THEN** the command SHALL NOT amend `HEAD`
- **AND** the command SHALL NOT push to any remote
- **AND** the command SHALL NOT write to GitHub

### Requirement: Advisory Stack Readiness

After planning or applying a fix, the command SHALL provide advisory warnings about whether the stack appears ready for submit.

#### Scenario: Missing metadata warning

- **WHEN** the advisory stack inspection succeeds
- **AND** one or more discovered stack entries are missing PR metadata
- **THEN** `fix` SHALL print a warning that the stack is not fully ready to submit
- **AND** the warning SHALL include the count of entries missing PR metadata

#### Scenario: Malformed metadata warning

- **WHEN** the advisory stack inspection finds malformed PR metadata
- **THEN** `fix` SHALL print a warning that the stack has malformed PR metadata
- **AND** the warning SHALL NOT cause a successful local repair to fail

#### Scenario: Stack inspection failure warning

- **WHEN** advisory stack inspection fails
- **THEN** `fix` SHALL print a warning explaining that stack readiness could not be determined
- **AND** the warning SHALL NOT cause a successful local repair to fail

#### Scenario: Dry-run includes advisory warnings

- **WHEN** the user runs `bpr fix --pr <number> --dry-run`
- **THEN** the command SHALL run the same read-only advisory stack-readiness inspection
- **AND** any warnings SHALL be phrased as dry-run diagnostics

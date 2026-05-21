# Land Algorithm

## Purpose

Define the canonical behavior of `stack-pr land` for landing the bottom-most PR in a stack using GitHub squash merge, then rebasing the remaining stack branches onto the latest remote target.

The land command mutates local Git state (branch checkout, rebasing), updates GitHub PR state (base branch changes, squash merge), and force-pushes remaining stack branches. It is only available when `land.style` is `bottom-only` (the default). If `land.style` is `disable`, the command is not registered.

## Requirements

### Requirement: Command Registration

The `land` subcommand SHALL be registered only when the configured land style permits it.

#### Scenario: Default bottom-only registration

- **WHEN** `land.style` is `bottom-only` (or unset, defaulting to `bottom-only`)
- **THEN** the `stack-pr land` command SHALL be available in the CLI

#### Scenario: Disable hides the command

- **WHEN** `land.style` is `disable`
- **THEN** the `stack-pr land` command SHALL NOT be registered
- **AND** invoking it SHALL result in an unknown command or usage error

### Requirement: Pre-flight and Setup

Before landing, the command SHALL prepare the repository and discover the stack.

#### Scenario: Current branch recorded

- **WHEN** land begins
- **THEN** the current branch name SHALL be recorded for later restoration

#### Scenario: Optional base fast-forward

- **WHEN** the local base is an ancestor of `REMOTE/TARGET`
- **AND** `REMOTE/TARGET` is an ancestor of `HEAD`
- **AND** the base hash differs from `REMOTE/TARGET`
- **THEN** the command SHALL run `git rebase REMOTE/TARGET base`
- **AND** the command SHALL checkout the original branch afterward

#### Scenario: Stack loaded from base..head

- **WHEN** land runs
- **THEN** the stack SHALL be loaded from commits in `base..head`
- **AND** stack entries SHALL be ordered oldest-to-newest internally

#### Scenario: Empty stack rejected

- **WHEN** the discovered stack contains no commits
- **THEN** the command SHALL print `Empty stack!` and return without further action

### Requirement: Base Branches and Verification

The command SHALL compute base branches, print the stack, and verify GitHub state before merging.

#### Scenario: Base branches computed

- **WHEN** the stack is loaded and non-empty
- **THEN** base branches SHALL be computed for each entry
- **AND** the first (bottom) entry's base SHALL be the remote target branch
- **AND** each subsequent entry's base SHALL be the previous entry's head branch
- **AND** the stack SHALL be printed newest-to-oldest

#### Scenario: Bottom PR must be mergeable

- **WHEN** the stack is verified with `check_base=True`
- **THEN** the bottom PR SHALL have state `OPEN`
- **AND** the bottom PR's base, head, and number SHALL match GitHub state
- **AND** the bottom PR's `mergeStateStatus` SHALL be one of `CLEAN`, `UNKNOWN`, or `UNSTABLE`
- **AND** on failure the command SHALL print an error and exit without merging

### Requirement: Bottom-Only Merge

The command SHALL land only the bottom-most PR using GitHub squash merge.

#### Scenario: Fetch and prepare remote head

- **WHEN** the bottom PR is ready to land
- **THEN** the remote SHALL be fetched and pruned
- **AND** the command SHALL checkout the remote head branch locally with:
  - `git checkout REMOTE/<head> -B <head>`

#### Scenario: Set PR base to target before merge

- **WHEN** the bottom PR is prepared
- **THEN** the command SHALL set its base branch to the target branch with:
  - `gh pr edit <pr> -B <target>`

#### Scenario: Squash merge with title and body

- **WHEN** the bottom PR is ready to merge
- **THEN** the squash merge title SHALL be:
  - `<original first commit-message line> (#<pr-number>)`
- **AND** the squash merge body SHALL be the remaining commit message after stripping the `stack-info` metadata line
- **AND** if the resulting body is empty, the body SHALL be a single space
- **AND** the command SHALL run:
  - `gh pr merge <pr> --squash -t <title> -F -`

### Requirement: Rebase Remaining Stack

If additional PRs remain above the bottom one, the command SHALL rebase each remaining branch onto the latest remote target and update their PR bases.

#### Scenario: Rebase announcement

- **WHEN** one or more PRs remain after the bottom PR is merged
- **THEN** the command SHALL print `Rebasing the rest of the stack` and print those entries

#### Scenario: Rebase each remaining branch

- **WHEN** rebasing the remaining stack
- **THEN** for each remaining entry:
  - the remote SHALL be fetched and pruned
  - the command SHALL checkout `REMOTE/<head>` to local branch `<head>`
  - the command SHALL rebase the branch onto `REMOTE/TARGET` with `--committer-date-is-author-date`
  - the command SHALL force-push `<head>:<head>` to the remote

#### Scenario: New bottom PR base updated

- **WHEN** all remaining branches have been rebased and pushed
- **THEN** the new bottom PR's base SHALL be set to the target branch with:
  - `gh pr edit <pr> -B <target>`

### Requirement: Cleanup and Restoration

After landing, the command SHALL restore the local repository to a clean state.

#### Scenario: Original branch restored

- **WHEN** landing and rebasing complete
- **THEN** the command SHALL checkout the original recorded branch

#### Scenario: Local generated branches deleted

- **WHEN** the original branch is restored
- **THEN** all local stack generated branches SHALL be deleted

#### Scenario: Local target branch rebased

- **WHEN** a local branch exists whose name matches the remote target branch (e.g. `main`)
- **THEN** that branch SHALL be rebased onto `REMOTE/TARGET`

#### Scenario: Original branch rebased onto remote target

- **WHEN** cleanup completes
- **THEN** the original branch SHALL be rebased onto `REMOTE/TARGET`

### Requirement: Remote Branch Handling

The command SHALL not directly delete remote branches.

#### Scenario: Remote branches left to GitHub

- **WHEN** a PR is squash-merged
- **THEN** the command SHALL NOT run `git push` to delete the merged remote branch
- **AND** GitHub MAY delete the merged PR branch depending on repository settings

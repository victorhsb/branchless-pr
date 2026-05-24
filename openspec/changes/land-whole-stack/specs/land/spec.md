# Land Algorithm

## Purpose

Define the canonical behavior of `stack-pr land` for landing stacked pull requests. The command supports two styles:

- `bottom-only` (default): lands the bottom-most PR using GitHub squash merge, then rebases remaining stack branches onto the latest remote target.
- `whole-stack`: lands all PRs in the stack atomically by retargeting the tip PR to the target branch and performing a GitHub rebase merge.

The land command mutates local Git state (branch checkout, rebasing), updates GitHub PR state (base branch changes, merge), and force-pushes remaining stack branches (in `bottom-only` mode). It is only available when `land.style` is not `disable`. If `land.style` is `disable`, the command is not registered.

## Requirements

### Requirement: Command Registration

The `land` subcommand SHALL be registered only when the configured land style permits it.

#### Scenario: Default bottom-only registration

- **GIVEN** `land.style` is `bottom-only` (or unset, defaulting to `bottom-only`)
- **WHEN** the CLI is initialized
- **THEN** the `stack-pr land` command SHALL be available

#### Scenario: Whole-stack registration

- **GIVEN** `land.style` is `whole-stack`
- **WHEN** the CLI is initialized
- **THEN** the `stack-pr land` command SHALL be available

#### Scenario: Disable hides the command

- **GIVEN** `land.style` is `disable`
- **WHEN** the CLI is initialized
- **THEN** the `stack-pr land` command SHALL NOT be registered
- **AND** invoking it SHALL result in an unknown command or usage error

### Requirement: Style Selection

The effective land style SHALL be determined by merging config and CLI flag.

#### Scenario: Config bottom-only, no flag

- **GIVEN** `land.style` is `bottom-only`
- **AND** no `--whole-stack` flag is provided
- **WHEN** `stack-pr land` is invoked
- **THEN** the bottom-only algorithm SHALL execute

#### Scenario: Config whole-stack, no flag

- **GIVEN** `land.style` is `whole-stack`
- **AND** no `--whole-stack` flag is provided
- **WHEN** `stack-pr land` is invoked
- **THEN** the whole-stack algorithm SHALL execute

#### Scenario: --whole-stack flag overrides config

- **GIVEN** `land.style` is `bottom-only`
- **AND** the `--whole-stack` flag is provided
- **WHEN** `stack-pr land` is invoked
- **THEN** the whole-stack algorithm SHALL execute

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

#### Scenario: Bottom PR must be mergeable (bottom-only)

- **WHEN** the stack is verified with `check_base=True` under `bottom-only` style
- **THEN** the bottom PR SHALL have state `OPEN`
- **AND** the bottom PR's base, head, and number SHALL match GitHub state
- **AND** the bottom PR's `mergeStateStatus` SHALL be one of `CLEAN`, `UNKNOWN`, or `UNSTABLE`
- **AND** on failure the command SHALL print an error and exit without merging

#### Scenario: All PRs must be open (whole-stack)

- **WHEN** the stack is verified under `whole-stack` style
- **THEN** all PRs SHALL have state `OPEN`
- **AND** each PR's base, head, and number SHALL match GitHub state
- **AND** on failure the command SHALL print an error and exit without merging

### Requirement: Bottom-Only Merge

The command SHALL land only the bottom-most PR using GitHub squash merge when `bottom-only` style is active.

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

### Requirement: Rebase Remaining Stack (bottom-only)

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

### Requirement: Whole-Stack Merge

When `whole-stack` style is active, the command SHALL land all PRs in the stack by rebase-merging the tip PR directly into the target branch.

#### Scenario: Repository must allow rebase merges

- **WHEN** `whole-stack` style is active
- **THEN** the command SHALL query the repository's merge settings via the GitHub GraphQL API
- **AND** if `rebaseMergeAllowed` is false, the command SHALL print an error message explaining that rebase merges are disabled and exit without mutating state
- **AND** if the API call fails, the command SHALL propagate the error

#### Scenario: Retarget tip PR to target

- **WHEN** the repository allows rebase merges
- **THEN** the command SHALL set the tip PR's base branch to the target branch with:
  - `gh pr edit <tip-pr> -B <target>`

#### Scenario: Rebase merge the tip PR

- **WHEN** the tip PR is retargeted to the target branch
- **THEN** the command SHALL run:
  - `gh pr merge <tip-pr> --rebase`

#### Scenario: No per-entry rebase or push needed

- **WHEN** the tip PR is rebase-merged into the target branch
- **THEN** the command SHALL NOT checkout, rebase, or force-push any remaining stack branches
- **AND** all commits from the stack SHALL appear linearly on the target branch

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

#### Scenario: Fetch after merge (whole-stack)

- **WHEN** the whole-stack merge completes
- **THEN** the remote SHALL be fetched and pruned before cleanup rebases

### Requirement: Remote Branch Handling

The command SHALL not directly delete remote branches.

#### Scenario: Remote branches left to GitHub

- **WHEN** a PR is merged (squash or rebase merge)
- **THEN** the command SHALL NOT run `git push` to delete the merged remote branch
- **AND** GitHub MAY delete the merged PR branch depending on repository settings

# Abandon Algorithm

## Purpose

Define the canonical behavior of `stack-pr abandon` for removing stack metadata from commits, deleting local generated branches, and deleting matching remote generated branches.

The abandon command is a destructive cleanup operation that:
1. Strips `stack-info` metadata lines from all commits in the stack
2. Rebases each commit onto a clean branch without stack tracking
3. Deletes local generated branches created by submit/export
4. Deletes matching remote generated branches from the repository

The current implementation does not call `gh pr close`; it only strips metadata and deletes branches. PRs remain open on GitHub unless manually closed.

## Requirements

### Requirement: Pre-flight Checks

Before abandoning, the command SHALL validate the repository state and discover the stack.

#### Scenario: Rebase in progress rejected

- **WHEN** a rebase is in progress (`.git/rebase-merge` or `.git/rebase-apply` exists)
- **THEN** the command SHALL print an error and exit with status 1

#### Scenario: Stack loaded from base..head

- **WHEN** abandon runs
- **THEN** the stack SHALL be loaded from commits in `base..head`
- **AND** stack entries SHALL be ordered oldest-to-newest internally

#### Scenario: Empty stack rejected

- **WHEN** the discovered stack contains no commits
- **THEN** the command SHALL print `Empty stack!` and return without further action

#### Scenario: Current branch recorded

- **WHEN** abandon begins
- **THEN** the current branch name SHALL be recorded for later restoration

### Requirement: Branch Initialization

The command SHALL initialize local branches for every stack commit before stripping metadata.

#### Scenario: Preserve existing head branches

- **WHEN** a stack entry already has a head branch from `stack-info` metadata
- **THEN** that branch SHALL be used as-is

#### Scenario: Assign new head branches for missing metadata

- **WHEN** a stack entry has no head branch in its metadata
- **THEN** a new generated branch SHALL be assigned using the branch name template
- **AND** the next available numeric ID SHALL be used

### Requirement: Base Branch Computation

The command SHALL compute base branches for all stack entries.

#### Scenario: Base branches set

- **WHEN** the stack is loaded and non-empty
- **THEN** base branches SHALL be computed for each entry
- **AND** the first (bottom) entry's base SHALL be the remote target branch
- **AND** each subsequent entry's base SHALL be the previous entry's head branch

#### Scenario: Stack printed before stripping

- **WHEN** base branches are computed
- **THEN** the stack SHALL be printed newest-to-oldest for user confirmation

### Requirement: Metadata Stripping

The command SHALL remove `stack-info` metadata from every commit in the stack while preserving the original commit message content.

#### Scenario: First entry checkout and amend

- **WHEN** stripping metadata for the first (bottom) stack entry
- **THEN** the command SHALL checkout that entry's head branch
- **AND** the `stack-info: PR: ..., branch: ...` line SHALL be removed from the commit message
- **AND** the amended commit message (without metadata) SHALL be applied with `git commit --amend -F -`
- **AND** the new commit hash SHALL be recorded from `git rev-parse <head>`

#### Scenario: Later entries rebase and amend

- **WHEN** stripping metadata for subsequent stack entries
- **THEN** the command SHALL rebase the entry's head branch onto its base branch with `--committer-date-is-author-date`
- **AND** the `stack-info: PR: ..., branch: ...` line SHALL be removed from the commit message
- **AND** the amended commit message (without metadata) SHALL be applied with `git commit --amend -F -`
- **AND** the new commit hash SHALL be recorded from `git rev-parse <head>`

#### Scenario: Commit message preservation

- **WHEN** metadata is stripped from a commit
- **THEN** all original commit message content except the `stack-info` line SHALL be preserved unchanged

### Requirement: Current Branch Rebase

After all metadata is stripped, the command SHALL rebase the user's current branch onto the new clean commits.

#### Scenario: Rebase onto final stripped commit

- **WHEN** all stack entries have had their metadata stripped
- **THEN** the current branch SHALL be rebased onto the final (top-most) stripped commit hash
- **AND** the user ends up on their original branch with clean commits

### Requirement: Local Branch Cleanup

The command SHALL delete all local generated branches associated with the stack.

#### Scenario: Delete local generated branches

- **WHEN** metadata stripping and rebasing are complete
- **THEN** all local branches that were heads for stack entries SHALL be deleted
- **AND** deletion SHALL use force if necessary

### Requirement: Remote Branch Cleanup

The command SHALL delete matching remote generated branches.

#### Scenario: Delete remote branches by prefix match

- **WHEN** local branches are deleted
- **THEN** remote branches SHALL be deleted that:
  - match the configured branch name base (the prefix before `$ID`), and
  - are heads for stack entries

#### Scenario: Remote deletion command format

- **WHEN** deleting remote branches
- **THEN** the command SHALL use:
  - `git push -f <remote> :<branch1> :<branch2> ...`
- **AND** all matching remote branches SHALL be deleted in a single push

#### Scenario: No remote branches for non-matching heads

- **WHEN** a stack entry's head branch does not match the configured branch name base
- **THEN** no remote branch deletion SHALL be attempted for that entry

### Requirement: Error Recovery

On failure, the command SHALL attempt to restore the repository to a known state.

#### Scenario: Restore original branch on failure

- **WHEN** the abandon command fails at any point
- **THEN** the command SHALL checkout the original branch recorded at the start
- **AND** the user SHALL be informed of the failure

#### Scenario: Partial metadata strip handling

- **WHEN** metadata stripping fails partway through the stack
- **THEN** already-stripped commits remain amended
- **AND** the original branch is restored
- **AND** the user may need to manually clean up the partially stripped stack

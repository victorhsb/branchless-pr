---
title: Abandon
status: stable
---

# Abandon

## Overview

`stack-pr abandon` is a destructive cleanup operation that removes stack metadata from commits, deletes local generated branches, and deletes matching remote generated branches. It strips `stack-info` metadata lines from all commits in the stack, rebases each commit onto a clean branch without stack tracking, deletes local generated branches created by submit/export, and deletes matching remote generated branches from the repository.

The current implementation does not call `gh pr close`; it only strips metadata and deletes branches. PRs remain open on GitHub unless manually closed.

## Behavior

### Pre-flight checks

- Rebase in progress (`.git/rebase-merge` or `.git/rebase-apply` exists) → print an error and exit 1.
- The stack is loaded from commits in `base..head`, ordered oldest-to-newest internally.
- Discovered stack contains no commits → print `Empty stack!` and return without further action.
- The current branch name is recorded at the start for later restoration.

### Branch initialization

Local branches are initialized for every stack commit before stripping metadata.

- Stack entry already has a head branch from `stack-info` metadata → use that branch as-is.
- Stack entry has no head branch in its metadata → assign a new generated branch using the branch name template and the next available numeric ID.

### Base branch computation

- Stack loaded and non-empty → compute base branches for each entry: the first (bottom) entry's base is the remote target branch; each subsequent entry's base is the previous entry's head branch.
- Base branches computed → print the stack newest-to-oldest for user confirmation.

### Metadata stripping

#### Scenario: Strip metadata from every commit

- **WHEN** stripping metadata for the first (bottom) stack entry
- **THEN** checkout that entry's head branch, remove the `stack-info: PR: ..., branch: ...` line from the commit message, apply the amended commit message (without metadata) with `git commit --amend -F -`, and record the new commit hash from `git rev-parse <head>`
- **WHEN** stripping metadata for subsequent stack entries
- **THEN** rebase the entry's head branch onto its base branch with `--committer-date-is-author-date`, remove the `stack-info: PR: ..., branch: ...` line from the commit message, apply the amended commit message (without metadata) with `git commit --amend -F -`, and record the new commit hash from `git rev-parse <head>`
- All original commit message content except the `stack-info` line is preserved unchanged.

### Current branch rebase

- All stack entries have had their metadata stripped → rebase the current branch onto the final (top-most) stripped commit hash; the user ends up on their original branch with clean commits.

### Local branch cleanup

- Metadata stripping and rebasing complete → delete all local branches that were heads for stack entries, using force if necessary.

### Remote branch cleanup

- Local branches deleted → delete remote branches that match the configured branch name base (the prefix before `$ID`) and are heads for stack entries.
- Remote deletion uses `git push -f <remote> :<branch1> :<branch2> ...`; all matching remote branches are deleted in a single push.
- Stack entry's head branch does not match the configured branch name base → no remote branch deletion is attempted for that entry.

### Error recovery

- Abandon fails at any point → checkout the original branch recorded at the start and inform the user of the failure.
- Metadata stripping fails partway through the stack → already-stripped commits remain amended, the original branch is restored, and the user may need to manually clean up the partially stripped stack.

### Native stack abandon preflight

When native integration is enabled, abandon inspects and safely dissolves matching native membership through the REST API before deleting generated remote branches.

- Local PR sequence exactly matches one GitHub native Stack → POST with no body to `repos/{owner}/{repo}/stacks/{stack_number}/unstack` before stripping local metadata or deleting generated remote branches, and verify that no affected unmerged local PR remains unexpectedly stacked before remote branch deletion.
- Native membership conflicts with the local PR sequence → fail before commit amendment, local branch mutation, remote branch deletion, or native Stack mutation; provide manual unstack guidance.

| Mode and repository state | Behavior |
|---------------------------|----------|
| `github.native_stacks = auto`, every local PR unstacked | skip the native unstack operation; continue with the existing metadata and branch cleanup algorithm |
| `github.native_stacks = auto`, native Stacks unavailable | warn once; continue with legacy abandon behavior |
| `github.native_stacks = required`, native Stacks unavailable | fail before local or remote mutation |

| Unstack result | Behavior |
|----------------|----------|
| `200` with one or more surviving members | preserve and report the surviving Stack; stop before deleting a generated remote branch for any affected unmerged PR that remains stacked |
| `204` with no body | treat the native Stack as dissolved; may continue with the existing metadata and branch cleanup algorithm |
| request fails with an uncertain server outcome | re-read the numbered Stack before deciding whether cleanup is safe; never blindly repeat the unstack request; fail before branch deletion when the result cannot be proven safe |
| partial unstack returns a local PR with `state: "closed"` and `merged_at: null` | treat that PR as unmerged; stop before deleting the branch |

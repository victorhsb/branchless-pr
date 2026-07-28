---
title: Land
status: stable
---

# Land

## Overview

`stack-pr land` lands stacked pull requests in one of two styles:

- `bottom-only` (default): lands the bottom-most PR using GitHub squash merge, then rebases remaining stack branches onto the latest remote target.
- `whole-stack`: lands all PRs in the stack atomically by retargeting the tip PR to the target branch and queuing a GitHub rebase auto-merge; requires the repository target branch to have GitHub merge queue enabled.

Land mutates local Git state (branch checkout, rebasing), updates GitHub PR state (base branch changes, merge queue scheduling), and force-pushes remaining stack branches (in `bottom-only` mode). It is only available when `land.style` is not `disable`.

## Behavior

### Command registration

| `land.style` | `stack-pr land` |
|--------------|-----------------|
| `bottom-only` (or unset, defaulting to `bottom-only`) | available |
| `whole-stack` | available |
| `disable` | not registered; invoking it results in an unknown command or usage error |

### Style selection

| Config | Flag | Effective style |
|--------|------|-----------------|
| `land.style = bottom-only` | no `--whole-stack` | bottom-only |
| `land.style = whole-stack` | no `--whole-stack` | whole-stack |
| `land.style = bottom-only` | `--whole-stack` | whole-stack |

### Pre-flight and setup

- Land begins → record the current branch name for later restoration.
- Local base is an ancestor of `REMOTE/TARGET` and `REMOTE/TARGET` is an ancestor of `HEAD` and the base hash differs from `REMOTE/TARGET` → run `git rebase REMOTE/TARGET base`, then check out the original branch afterward.
- The stack is loaded from commits in `base..head`, ordered oldest-to-newest internally.
- Discovered stack contains no commits → print `Empty stack!` and return without further action.

### Base branches and verification

- Stack loaded and non-empty → compute base branches for each entry: the first (bottom) entry's base is the remote target branch, and each subsequent entry's base is the previous entry's head branch; print the stack newest-to-oldest.
- Stack verified with `check_base=True` under `bottom-only` → the bottom PR must have state `OPEN`; its base, head, and number must match GitHub state; its `mergeStateStatus` must be one of `CLEAN`, `UNKNOWN`, or `UNSTABLE`; on failure print an error and exit without merging.
- Stack verified under `whole-stack` → all PRs must have state `OPEN` and each PR's base, head, and number must match GitHub state; on failure print an error and exit without merging.

### Bottom-only merge

#### Scenario: Squash-merge the bottom PR

- **WHEN** the bottom PR is ready to land
- **THEN** fetch and prune the remote, and check out the remote head branch locally with `git checkout REMOTE/<head> -B <head>`
- **AND** set the PR's base branch to the target branch with `gh pr edit <pr> -B <target>`
- **AND** run `gh pr merge <pr> --squash -t <title> -F -`, where the squash merge title is `<original first commit-message line> (#<pr-number>)` and the body is the remaining commit message after stripping the `stack-info` metadata line (a single space if the resulting body is empty)

#### Scenario: Rebase the remaining stack after the bottom merge

- **WHEN** one or more PRs remain after the bottom PR is merged
- **THEN** print `Rebasing the rest of the stack` and print those entries
- **AND** for each remaining entry: fetch and prune the remote; check out `REMOTE/<head>` to local branch `<head>`; rebase the branch onto `REMOTE/TARGET` with `--committer-date-is-author-date`; force-push `<head>:<head>` to the remote
- **AND** after all remaining branches have been rebased and pushed, set the new bottom PR's base to the target branch with `gh pr edit <pr> -B <target>`

### Whole-stack merge

- `whole-stack` style active → query the repository's merge settings via the GitHub API.
- `rebaseMergeAllowed` is false → print an error message explaining that rebase merges are disabled and exit without mutating state.
- Merge-settings API call fails → propagate the error.
- Merge queue is not enabled for the repository target branch → print `ERROR: --whole-stack only works for repositories with merge queue enabled` and exit without editing PR bases, merging PRs, fetching, checking out branches, deleting local branches, rebasing local branches, or pushing branches; merge queue is verified before retargeting the tip PR.

#### Scenario: Queue the tip PR for whole-stack landing

- **WHEN** the repository allows rebase merges and the repository target branch has merge queue enabled
- **THEN** set the tip PR's base branch to the target branch with `gh pr edit <tip-pr> -B <target>`, then run `gh pr merge <tip-pr> --rebase --auto`
- **AND** GitHub owns waiting for required checks, approvals, merge-queue grouping, and final merge; do not poll GitHub for CI or merge completion
- **AND** do not check out, rebase, or force-push any remaining stack branches; all commits from the stack are expected to appear linearly on the target branch when GitHub completes the queued merge

### Cleanup and restoration

| Aspect | bottom-only, landing and rebasing complete | whole-stack, merge-queue scheduling succeeds |
|--------|--------------------------------------------|----------------------------------------------|
| Original recorded branch | checked out; rebased onto `REMOTE/TARGET` | checked out; not rebased onto `REMOTE/TARGET` |
| Queued-landing message | — | print a message that whole-stack landing has been queued for the tip PR |
| Local stack generated branches | all deleted | not deleted |
| Local branch whose name matches the remote target branch (e.g. `main`) | rebased onto `REMOTE/TARGET` | not rebased onto `REMOTE/TARGET` |
| Remote fetch after landing | — | no fetch after scheduling the queued merge |

### Remote branch handling

- A PR is merged (squash or rebase merge) → never run `git push` to delete the merged remote branch; GitHub may delete the merged PR branch depending on repository settings.

### Native landing safety gate

When native integration is enabled, land inspects GitHub membership before executing any bottom-only or whole-stack mutation and refuses unsupported landing for a matching native Stack.

| Native preflight state | Behavior |
|------------------------|----------|
| Native integration enabled, every local PR belongs to one native Stack in the same complete bottom-to-top sequence, effective style `bottom-only` | return an actionable unsupported error before editing or merging a PR or mutating a branch; the error identifies the bottom PR URL for initiating the merge in GitHub's UI and explains that branchless-pr does not yet synchronize GitHub's server-rebased remaining branches locally |
| Native integration enabled, every local PR belongs to one native Stack in the same complete bottom-to-top sequence, effective style `whole-stack` | return an actionable unsupported error before retargeting or merging a PR or mutating a branch; the error identifies the top PR URL for initiating whole-stack landing in GitHub's UI and explains that branchless-pr does not yet synchronize GitHub's server-side landing locally |
| `github.native_stacks = auto`, every local PR is unstacked | execute the existing land algorithm for the effective style |
| `github.native_stacks = auto`, native Stack membership is unavailable for the repository | warn once and execute the existing land algorithm for the effective style |
| `github.native_stacks = required`, an eligible multi-PR local stack is unstacked | fail before PR, branch, or merge mutation |
| `github.native_stacks` is `auto` or `required`, native membership conflicts with the local sequence | fail before PR, branch, or merge mutation; never fall back to legacy landing or automatically unstack the remote Stack |
| `github.native_stacks = off` | use the existing land algorithm; never query or mutate native Stack membership |

- Land refuses native landing for a matching native Stack → never call the REST unstack endpoint, never run `gh pr merge`, and never edit PR bases, check out generated heads, rebase, push, fetch for cleanup, or delete branches.

### Native landing deferred

Supported native landing and remote-to-local synchronization are treated as follow-up behavior because the documented Stacks REST API provides no merge or rebase endpoint.

- Stacks REST API exposes no merge operation → never assume `gh pr merge` provides a stable stacked-PR landing contract when handling a matching native Stack.
- A user rebases or merges a native Stack through GitHub's UI and GitHub rewrites generated remote branches → never claim that the local commit stack has been synchronized; documentation describes the limitation and manual recovery requirement.

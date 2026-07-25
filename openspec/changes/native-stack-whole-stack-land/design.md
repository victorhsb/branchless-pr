## Context

The `land --whole-stack` command currently retargets the tip PR's base to `main` via `gh pr edit -B main` and then queues a rebase auto-merge via `gh pr merge --rebase --auto`. For stacks linked to a GitHub native Stack, GitHub rejects all base edits with `"Cannot change the base branch because the pull request is part of a stack."` This makes the current whole-stack flow impossible for native-stacked PRs, so the `nativeLandPreflight` refuses landing entirely and points the user to the GitHub UI.

GitHub's native Stack mechanism manages the base chain server-side: each PR's base is the branch of the PR below it, and when the bottom PR merges, GitHub automatically retargets the next PR to the stack's base (e.g. `main`). This means the merge queue can cascade merges from bottom to top without any client-side base manipulation.

## Goals / Non-Goals

**Goals:**
- Allow `land --whole-stack` to land a matching native Stack by queuing the tip PR for rebase auto-merge, relying on GitHub's merge queue to cascade the chain.
- Eliminate the failed `gh pr edit -B` call and the refusal error for whole-stack native landing.
- Keep `bottom-only` land refused for native stacks (squash-merge + rebase-rest is incompatible with server-side base management).

**Non-Goals:**
- Supporting `bottom-only` landing for native stacks.
- Polling GitHub for merge completion or CI status.
- Synchronizing local branches after GitHub completes the queued merge (deferred to a future change).
- Adding a new `gh stack` subcommand or REST endpoint for stack-level merge (none exists).

## Decisions

### Decision 1: Queue the tip PR without base retargeting

**Choice:** Skip `pr.EditBase(tip.PR(), target)` for native-stacked PRs and call `pr.MergeRebaseAuto(tip.PR())` directly.

**Rationale:** GitHub's native Stack already sets each PR's base to the branch below it. The merge queue processes dependencies in order: the bottom PR (base=main) merges first, GitHub auto-retargets the next PR to main, it merges next, and so on up to the tip. Retargeting the tip PR to main is both unnecessary and rejected by the API.

**Alternative considered:** Merge the bottom PR first with `gh pr merge --squash`, wait for GitHub to retarget the next PR, then merge it, and so on. Rejected because: (a) it requires polling GitHub for merge completion, (b) it's sequential and slow, (c) the merge queue already handles this cascade automatically when the tip PR is queued.

### Decision 2: Reuse the existing merge-queue preflight

**Choice:** Keep the existing `RebaseMergeAllowed` and `MergeQueueEnabled` checks from `landWholeStackImpl` as preconditions for native whole-stack landing.

**Rationale:** The merge queue is what makes the cascade work. Without it, queuing the tip PR for auto-merge would not cascade the chain. The rebase-merge-allowed check ensures the queued merge uses the correct merge method.

### Decision 3: New `landWholeStackNative` function

**Choice:** Add a separate `landWholeStackNative` function rather than adding conditionals inside `landWholeStackImpl`.

**Rationale:** The legacy and native whole-stack flows differ in a single critical step (skip `EditBase`), but separating them makes the code easier to test, reason about, and extend. The native path is strictly simpler: check merge settings → fetch → queue tip PR → checkout original branch → print message. The legacy path retains the `EditBase` + auto-merge sequence.

### Decision 4: `nativeLandPreflight` allows whole-stack for `ActionNoop`

**Choice:** When `result.Kind == ActionNoop` and `style == "whole-stack"`, return `nil` (allow landing) instead of calling `nativeLandRefusal`.

**Rationale:** `ActionNoop` means the local stack exactly matches a native Stack — this is the case where native whole-stack landing is safe. `ActionAppend` (partial match) and `ActionConflict` (drift) remain refused for both styles.

### Decision 5: Bottom-only remains refused for native stacks

**Choice:** `ActionNoop` with `bottom-only` still calls `nativeLandRefusal`.

**Rationale:** Bottom-only requires squash-merging the bottom PR, then rebasing remaining branches onto the new target and updating their PR bases. For native stacks, GitHub manages bases server-side and rejects `EditBase` calls. The rebase + base-update flow cannot work without client-side base edits.

## Risks / Trade-offs

- [Merge queue cascade assumption] → The design relies on GitHub's merge queue cascading native Stack PRs from bottom to top when the tip PR is queued. If GitHub does not support this, the tip PR would stay queued indefinitely waiting on its dependencies. **Mitigation:** The merge-queue preflight (`MergeQueueEnabled`) confirms the repo has merge queue enabled, which is the prerequisite for cascade behavior. If the cascade is not supported, the user can cancel the queued merge from the GitHub UI.

- [No local synchronization after merge] → After the merge queue completes, local branches and the original branch are not rebased onto the new target. **Mitigation:** The command prints a message explaining the landing was queued and the user should `git fetch` and rebase after GitHub completes the merge. A future change can add a `bpr sync` command.

- [Preflight runs `stack.Discover` twice] → `nativeLandPreflight` discovers the stack, then `landImpl` discovers it again. **Mitigation:** This is the existing behavior and the cost is negligible (local git rev-list). Refactoring to pass the discovered stack through the preflight is out of scope for this change.

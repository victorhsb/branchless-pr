## Context

The existing `whole-stack` land style retargets the tip PR to the target branch and immediately runs `gh pr merge <tip-pr> --rebase`. That works only when GitHub considers the PR mergeable at that moment. In branch-protected repositories, retargeting the tip PR commonly starts or restarts CI, so the immediate merge can fail even though the desired operation is "put this PR in the merge queue once required checks and approvals pass."

Polling GitHub until CI completes would make `stack-pr land` a long-running command whose duration depends on repository CI, GitHub queue state, and branch protection. GitHub already owns that state machine through merge queue. GitHub's docs say `gh pr merge` adds a PR to the queue when the target branch requires a merge queue; if required checks have not passed, it enables auto-merge for the PR so GitHub queues it when requirements are met.

## Goals / Non-Goals

**Goals:**

- Require merge queue for `whole-stack` landing.
- Queue the retargeted tip PR for GitHub-managed merge instead of attempting an immediate merge.
- Avoid polling CI or check status from `land`.
- Avoid local post-merge cleanup in queued mode, because the target branch has not advanced when the PR is only queued or waiting to be queued.
- Keep `bottom-only` and `disable` behavior unchanged.
- Keep all GitHub interaction behind `gh` and all subprocess execution behind `internal/shell`.

**Non-Goals:**

- Do not introduce a Go GitHub SDK.
- Do not add a long-running wait, watch, or polling mode.
- Do not decide readiness by querying every stack PR's checks before landing.
- Do not close intermediate PRs as part of this change.
- Do not change the `bottom-only` squash/rebase algorithm.

## Decisions

1. **Treat merge queue as a requirement for `whole-stack`.**

   `land.style = whole-stack` and `bpr land --whole-stack` should only work when the target repository/branch has merge queue enabled. If merge queue is not enabled or cannot be confirmed, the command should fail before retargeting the tip PR with:

   ```text
   ERROR: --whole-stack only works for repositories with merge queue enabled
   ```

   This keeps the feature narrowly tied to the GitHub workflow that handles long-running CI without local polling.

   Alternative considered: support both direct and queued whole-stack modes. Rejected because the old immediate behavior is exactly the fragile path this change is removing from the whole-stack contract. Users without merge queue can keep using `bottom-only`.

2. **Detect merge queue support before retargeting the tip PR.**

   The implementation should add an `internal/pr` helper that verifies merge queue is enabled for the target branch using the best available GitHub CLI/API surface. The preferred source is GitHub ruleset data because the REST rules API exposes a `merge_queue` rule type. If the rules API cannot provide a reliable answer in the current environment, the command should still normalize GitHub's `gh pr merge` disabled-queue failure into the required error string.

   The existing `rebaseMergeAllowed` check remains useful because whole-stack landing preserves commits through a rebase merge. Both merge queue support and rebase merge support must be satisfied.

   Alternative considered: skip preflight and let `gh pr merge` fail after retargeting. Rejected because the user-facing contract should say `--whole-stack` only works for merge-queue repositories, and avoid remote mutations when that prerequisite is absent.

3. **Queued whole-stack uses `gh pr merge <tip-pr> --rebase --auto`.**

   Add an `internal/pr` wrapper, following the existing `MergeRebase` pattern, that runs the GitHub CLI command through `internal/shell`. This keeps the implementation aligned with the project's no-SDK invariant and lets GitHub enforce branch protection, approval, merge queue, and check requirements.

   Under a merge-queue-protected target branch, GitHub CLI adds the PR to the queue when requirements have passed; when they have not, it enables auto-merge so GitHub can queue and merge it once requirements are met.

4. **Queued mode stops after scheduling the merge.**

   After `gh pr merge --rebase --auto` succeeds, the command should restore the original branch and print a message that whole-stack landing has been queued for the tip PR. It should not fetch, delete local stack branches, rebase local target, or rebase the original branch onto `REMOTE/TARGET`, because the target branch may not include the stack yet.

   Alternative considered: poll until the PR merges, then run normal cleanup. Rejected because CI can take many minutes and polling would turn `land` into a watch command with uncertain runtime.

5. **Keep intermediate PR handling unchanged.**

   Whole-stack land still relies on GitHub's auto-close detection or manual cleanup for intermediate PRs. Merge queue does not change that trade-off; it only changes when the tip PR is merged.

## Risks / Trade-offs

- **Merge queue detection may be incomplete across branch protection and ruleset implementations** -> Prefer documented GitHub rules API data where available, and normalize GitHub CLI failures to the required error message as a fallback.
- **Queue scheduling succeeds but CI later fails** -> GitHub removes or blocks the PR according to merge queue behavior; users can inspect with `bpr checks` or GitHub.
- **Local branches remain after queued mode** -> This is intentional because the stack has not landed yet. A future follow-up could add a cleanup command for already-merged stacks.
- **Tip PR base is retargeted before queue scheduling fails** -> Preflight merge queue and rebase merge settings reduce predictable failure causes, but later GitHub failures can still happen. `WithRecovery` should restore local branch state; PR base retargeting is a remote mutation and should be reflected in error output.

## Migration Plan

1. Add merge queue support detection and the queued merge PR wrapper.
2. Update whole-stack land to preflight merge queue and rebase merge support, retarget the tip PR, queue the merge, and restore the original branch.
3. Remove direct whole-stack cleanup assumptions from the queued path.
4. Update specs, `SPEC.md`, README/help text, and tests.
5. Leave `bottom-only` as the path for repositories without merge queue.

## Open Questions

None.

## Why

When `github.native_stacks` is enabled and a stack matches a GitHub native Stack, `land --whole-stack` currently refuses with an actionable error pointing the user to the GitHub UI. This is unnecessarily restrictive for whole-stack landing: GitHub's native Stack mechanism already manages the base chain server-side, so queuing the tip PR for rebase auto-merge lets GitHub's merge queue cascade the merges from bottom to top without any local base retargeting. Supporting this turns `land --whole-stack` into a one-command landing flow for native-stacked PRs.

## What Changes

- `land --whole-stack` for matching native Stacks SHALL queue the tip PR for rebase auto-merge via `gh pr merge --rebase --auto` **without** first retargeting the tip PR's base to the target branch (GitHub rejects base edits on stacked PRs and manages the chain server-side).
- The native landing safety gate SHALL allow whole-stack landing for matching native Stacks instead of refusing.
- `bottom-only` land for matching native Stacks SHALL remain refused, since it requires squash-merging one PR and rebasing the rest — operations that conflict with GitHub's server-side base management.
- The "Native Landing Is Deferred Pending Synchronization" requirement is relaxed for whole-stack: `gh pr merge --rebase --auto` on the tip PR is now the supported native landing contract for whole-stack, relying on GitHub's merge queue to cascade the chain.
- After queuing a native whole-stack merge, the command SHALL NOT delete local branches, rebase the local target, or rebase the original branch — the stack has not landed yet.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `land`: Whole-stack landing for matching native Stacks transitions from refused to supported via tip-PR auto-merge queueing without base retargeting. The native landing safety gate now allows whole-stack for matching stacks while still refusing bottom-only.

## Impact

- `internal/cli/land.go` — `nativeLandPreflight` stops refusing whole-stack for matching native Stacks; new `landWholeStackNative` function queues the tip PR without `EditBase`; `nativeLandRefusal` is only used for bottom-only.
- `internal/pr/pr.go` — no changes (existing `MergeRebaseAuto` is reused).
- `internal/nativestacks/` — no changes (existing `Classify`/`LoadMembership` are reused).
- `SPEC.md` §6 (land) — update whole-stack native behavior description.
- `openspec/specs/land/spec.md` — delta spec updates the "Native Landing Safety Gate" and "Whole-Stack Merge" requirements.

### Port compatibility

The Python `stack-pr` does not support GitHub native Stacks. This is a Go-port-only extension. No divergence from Python behavior is introduced because native Stacks are an opt-in feature (`github.native_stacks = off` by default).

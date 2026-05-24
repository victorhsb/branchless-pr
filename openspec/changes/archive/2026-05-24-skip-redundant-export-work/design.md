## Context

`submit` and the `export` alias share `submitImpl`. The current implementation intentionally follows the Python algorithm closely, but it repeats several expensive operations during normal re-export:

- Existing PRs are queried before draft/base reset, then queried again during stack verification, and queried again under `--keep-body`.
- Existing PRs are edited even when their draft state, base branch, title, or rendered body is already correct.
- A second batch force-push runs even when all commits already contain stack metadata and no commit was amended.

The final state is correct, but unchanged stacks still pay multiple network round trips and at least one avoidable push.

Commit `c387468` refactored command-wide execution boundaries: shared invocation state moved to `internal/invocation`, report-oriented stack loading moved toward `internal/stackstate`, and `internal/cli/submit.go` was split into explicit submit phases (`validateSubmitPreconditions`, `discoverAndPrepareStack`, `applyMutations`, `tempDraftAndResetBases`, `amendCommitMetadata`). This change should build on those boundaries instead of adding new cross-package coupling back into `cli`.

## Goals / Non-Goals

**Goals:**

- Gate the optimized submit/export engine behind an explicit env or repo config opt-in while the implementation is experimental.
- Reduce repeated `gh` calls during one submit/export run by reusing PR state fetched for the stack.
- Avoid GitHub write calls when the requested PR state already matches the current state.
- Avoid the second batch force-push when no metadata amendment changed commit SHAs.
- Preserve submit/export's final local Git, remote branch, and GitHub PR state.
- Keep dry-run behavior side-effect free and aligned with the real execution plan.

**Non-Goals:**

- Do not add a Go GitHub SDK dependency.
- Do not change the generated branch naming scheme or stack metadata format.
- Do not change PR body rendering semantics.
- Do not parallelize mutating GitHub operations in this change.
- Do not change command-line flags or user-visible default output except for avoiding progress from skipped no-op operations.

## Decisions

0. **Gate the optimized submit/export engine with explicit opt-ins.**
   - The current submit/export path remains the default behavior.
   - Setting `STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE=1` selects the optimized engine for both `submit` and the `export` alias for that invocation.
   - Setting `submit.experimental_engine = true` in `.stack-pr.cfg` selects the optimized engine for the repository.
   - The env flag is useful for one-off trials; the config setting is useful when a repository wants persistent opt-in while the engine is experimental.
   - Dry-run uses the same engine selection as the corresponding real submit/export invocation so previewed work matches the selected execution path.
   - The gate should be resolved near submit/export dispatch, after `AppContext` and command-specific `submitOptions` are available. Do not overload `internal/invocation.PolicyFor`; that package models generic command pre-run policy, not feature selection.
   - Alternative considered: add a CLI flag. Rejected for now because this is an experimental rollout valve, not a user-facing command mode.

1. **Introduce a per-run PR state cache.**
   - Fetch each existing PR's submit-relevant fields at most once for a given phase and reuse the result for draft/base reset, verification, and `--keep-body` content.
   - The cache should not be exposed as a `cli` type to lower-level packages. Either keep it private to the optimized submit engine and pass lower-level packages a provider function, or define a small non-CLI model in `internal/pr` or a submit-specific helper package.
   - Preferred implementation: add a `pr.ViewMany`, `pr.LoadForSubmit`, or equivalent provider that still shells out through `gh`, either via `gh api graphql` for batching or a small internal abstraction that can later switch to GraphQL.
   - `stack.Verify` should not import submit/CLI state. If verification reuses cached data, extend the stack package with an API that accepts a PR info provider or `pr.Info` map while preserving the current `stack.Verify(st, checkBase)` behavior as the uncached default.
   - Alternative considered: keep independent `pr.View` calls and only skip writes. Rejected because repeated reads are a major cost on existing stacks and make no-op exports feel slow.

2. **Skip no-op PR writes by comparing desired state to fetched state.**
   - Existing ready PRs still need temporary draft protection, but PRs already draft do not need `ready --undo`.
   - Existing PRs whose base is already the temporary target do not need an intermediate base edit.
   - Final PR update should compare desired title, base, and body against fetched/current state and skip `gh pr edit` when all three already match.
   - Cache freshness is part of the write contract: after `ReadyUndo`, `EditBase`, `Create`, `Edit`, or `Ready`, update or invalidate the cached fields affected by that operation before later verification or final edit comparison uses them.
   - Alternative considered: always write to keep the path simple. Rejected because it is the main source of redundant GitHub mutation and notification risk.

3. **Guard the second force-push with metadata mutation state.**
   - The first push still publishes generated heads before PR creation/update.
   - The second push is only required when metadata amendments or subsequent rebases changed branch tips.
   - `amendCommitMetadata` should return an explicit `changedTips` result, or equivalent phase state, instead of requiring callers to infer changes from `needsMeta` after the fact. The result must become true for both direct metadata amendments and cascade rebases triggered by an earlier metadata amendment.
   - Alternative considered: compare local and remote branch SHAs before pushing. Rejected for this change because the implementation already knows whether metadata changed, and remote SHA comparison can be added later if needed.

4. **Keep verification semantics, allow reuse of fetched PR state.**
   - Verification must still reject missing/malformed metadata and mismatched PR state.
   - It can use already-fetched PR data as long as newly created PRs are included and stale data is refreshed after writes that change the checked fields.
   - Verification reuse should be wired at the phase boundary after missing PRs are created and before metadata is amended, matching the current `applyMutations` order.

5. **Respect the refactored stack-loading boundary.**
   - `c387468` introduced `internal/stackstate.Load` as shared stack discovery/metadata/head/base preparation for report paths. Submit currently still uses `discoverAndPrepareStack` because it also computes `needsMeta` and draft planning state.
   - The optimized engine should not introduce a third independent stack-loading path. Either extend `stackstate.Load` with submit-compatible metadata reporting, or keep `discoverAndPrepareStack` as the single submit/export loader and document why submit needs a richer return shape.
   - If `stackstate.Load` is extended, keep it side-effect free: no checkout, rebase, push, PR calls, or stash changes.

## Risks / Trade-offs

- **Stale cached PR state** -> Refresh or update the cache after PR create/edit/draft operations so later verification and final comparisons observe the intended state.
- **Import cycles from cached verification** -> Keep submit/CLI cache types out of `internal/stack`; pass a provider or `pr.Info` data structure instead.
- **Skipped edit misses a required update due to body normalization differences** -> Compare exact rendered body bytes/string against the GitHub body returned by `gh`; keep existing edit behavior if the body cannot be confidently compared.
- **GraphQL batching adds query complexity** -> Keep the public wrapper small and testable, and retain `gh pr view` fallback if batching is awkward.
- **Receipt operation coverage may change once receipts are implemented** -> Record only operations that actually occur; skipped no-op operations can be omitted or represented as non-side-effect planning details in a later receipt design update.

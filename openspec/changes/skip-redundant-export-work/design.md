## Context

`submit` and the `export` alias share `submitImpl`. The current implementation intentionally follows the Python algorithm closely, but it repeats several expensive operations during normal re-export:

- Existing PRs are queried before draft/base reset, then queried again during stack verification, and queried again under `--keep-body`.
- Existing PRs are edited even when their draft state, base branch, title, or rendered body is already correct.
- A second batch force-push runs even when all commits already contain stack metadata and no commit was amended.

The final state is correct, but unchanged stacks still pay multiple network round trips and at least one avoidable push.

## Goals / Non-Goals

**Goals:**

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

1. **Introduce a per-run PR state cache.**
   - Fetch each existing PR's submit-relevant fields at most once for a given phase and reuse the result for draft/base reset, verification, and `--keep-body` content.
   - Preferred implementation: add a `pr.ViewMany` or `pr.LoadForSubmit` wrapper that still shells out through `gh`, either via `gh api graphql` for batching or a small internal abstraction that can later switch to GraphQL.
   - Alternative considered: keep independent `pr.View` calls and only skip writes. Rejected because repeated reads are a major cost on existing stacks and make no-op exports feel slow.

2. **Skip no-op PR writes by comparing desired state to fetched state.**
   - Existing ready PRs still need temporary draft protection, but PRs already draft do not need `ready --undo`.
   - Existing PRs whose base is already the temporary target do not need an intermediate base edit.
   - Final PR update should compare desired title, base, and body against fetched/current state and skip `gh pr edit` when all three already match.
   - Alternative considered: always write to keep the path simple. Rejected because it is the main source of redundant GitHub mutation and notification risk.

3. **Guard the second force-push with metadata mutation state.**
   - The first push still publishes generated heads before PR creation/update.
   - The second push is only required when metadata amendments or subsequent rebases changed branch tips.
   - Alternative considered: compare local and remote branch SHAs before pushing. Rejected for this change because the implementation already knows whether metadata changed, and remote SHA comparison can be added later if needed.

4. **Keep verification semantics, allow reuse of fetched PR state.**
   - Verification must still reject missing/malformed metadata and mismatched PR state.
   - It can use already-fetched PR data as long as newly created PRs are included and stale data is refreshed after writes that change the checked fields.

## Risks / Trade-offs

- **Stale cached PR state** -> Refresh or update the cache after PR create/edit/draft operations so later verification and final comparisons observe the intended state.
- **Skipped edit misses a required update due to body normalization differences** -> Compare exact rendered body bytes/string against the GitHub body returned by `gh`; keep existing edit behavior if the body cannot be confidently compared.
- **GraphQL batching adds query complexity** -> Keep the public wrapper small and testable, and retain `gh pr view` fallback if batching is awkward.
- **Receipt operation coverage may change once receipts are implemented** -> Record only operations that actually occur; skipped no-op operations can be omitted or represented as non-side-effect planning details in a later receipt design update.

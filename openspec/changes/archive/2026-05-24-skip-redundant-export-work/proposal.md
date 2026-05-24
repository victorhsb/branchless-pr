## Why

Repeated `stack-pr submit` / `stack-pr export` runs currently redo several expensive GitHub and Git operations even when the stack is already up to date. This makes normal iteration slower than necessary, especially for existing multi-PR stacks where most PR state and metadata are unchanged.

## What Changes

- Reuse fetched PR state during a submit/export execution instead of querying the same PR repeatedly for draft state, verification data, and keep-body content.
- Skip temporary draft/base-reset operations when the existing PR already has the desired state.
- Skip final PR title/body/base edits when the rendered update would not change the PR.
- Skip the second batch force-push when no commit metadata was amended.
- Enable the optimized submit/export path only when the experimental environment feature flag or `.stack-pr.cfg` setting opts in; the default path preserves the current submit/export algorithm.
- Preserve existing command output and mutating behavior when work is actually required.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `submit-export`: Submit/export should avoid redundant no-op work while preserving the same final local Git, remote branch, and GitHub PR state.
- `submit-export`: The optimized submit/export engine should be gated by an environment feature flag or repo config setting until it is promoted to the default path.

## Impact

- `internal/cli/submit.go`: Introduce submit planning/state reuse and no-op guards around PR and push steps.
- `internal/cli/submit.go`: Dispatch between the current submit/export path and the optimized engine based on the experimental feature gate.
- `internal/config/config.go`: Add `.stack-pr.cfg` support for opting into the experimental submit/export engine.
- `internal/pr/pr.go`: Add an efficient way to fetch the PR fields submit/export needs without repeated per-phase queries.
- `internal/stack/verify.go`: Allow verification to reuse already-fetched PR state where practical.
- Tests: Add focused coverage for feature flag selection, skipping no-op PR edits, draft/base resets, repeated PR views, and the second push when metadata is unchanged.

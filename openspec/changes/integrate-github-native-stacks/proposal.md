## Why

GitHub now provides a native Stacked PR object with stack-aware review, rules, CI, rebasing, and landing, while branchless-pr currently represents the same PR chain only through generated branches, commit metadata, and PR-body cross-links. Publishing branchless-pr stacks to GitHub's native model gives reviewers and repository automation first-class stack awareness without replacing branchless-pr's commit-oriented local workflow.

## What Changes

- Add opt-in native-stack modes (`off`, `auto`, and `required`), defaulting to `off` while GitHub Stacked PRs remains in private preview.
- Keep `BASE..HEAD`, one commit per PR, generated branches, and `stack-info` commit metadata as branchless-pr's local source of truth.
- Reconcile the submitted PR chain after PR heads, bases, metadata, and descriptions reach their final state, using read-only REST membership data and the supported `gh stack link` external-tool workflow.
- Create a native stack for an unstacked multi-PR chain, append a local suffix to a matching native stack, and treat exact membership as a no-op.
- Detect conflicting or reordered native membership and fail without automatically dissolving or replacing the remote stack.
- Extend dry-run output to report the native-stack action that a real submit/export would attempt without performing GitHub writes.
- Surface native stack number, position, size, and ultimate base in stack inspection output and report local/remote membership drift.
- Make `land` native-aware by refusing to run the legacy landing algorithms against a matching native stack and directing the user to the relevant GitHub PR until supported non-interactive landing and local synchronization contracts exist.
- Make `abandon` dissolve matching native membership through `gh stack unstack <stack-number>` before deleting generated remote branches.
- Retain generated PR-body cross-links during the initial integration for compatibility and non-native readers.
- Update `SPEC.md`, user documentation, generated configuration, and operation receipts for the new behavior.

## Capabilities

### New Capabilities

- `github-native-stacks`: Configuration, optional gh-stack extension requirements, GitHub REST representation, reconciliation rules, feature availability handling, membership validation, and shared native-stack data model.

### Modified Capabilities

- `submit-export`: Reconcile eligible submitted PR chains with GitHub native stacks after existing PR mutations complete.
- `submit-operation-receipts`: Record native-stack create, append, no-op, fallback, and failure outcomes.
- `export-dry-run`: Preview native-stack reconciliation while preserving the no-write guarantee.
- `land`: Refuse unsupported CLI landing for matching native stacks while preserving legacy landing for non-native stacks.
- `abandon`: Unstack matching native PRs before deleting generated remote branches.
- `view`: Inspect and validate native membership without mutating local or remote state.
- `view-json-output`: Add optional native-stack metadata to machine-readable stack entries.
- `config-init-command`: Include documented native-stack mode defaults in generated configuration.

## Impact

- GitHub integration: Uses `gh api` for read-only Stack membership and verification, `gh stack link` for create/append writes, and `gh stack unstack` for dissolution; no Go GitHub SDK is introduced.
- Optional runtime dependency: Native mutating behavior requires the `github/gh-stack` extension at a supported version, while `github.native_stacks = off` retains the existing `git` and base `gh` requirements only.
- `internal/pr` or a focused internal package: Adds typed read/verification wrappers and non-interactive gh-stack command wrappers through `internal/shell`.
- `internal/cli/submit.go`: Adds post-submit reconciliation, dry-run planning, failure-mode handling, and receipt entries.
- `internal/cli/land.go`: Adds native membership verification and an unsupported-native-landing guard; the existing `land.style = disable` command gate remains unchanged.
- `internal/cli/abandon.go`: Adds native unstack preflight and remote cleanup ordering.
- `internal/cli/view.go` and stack JSON rendering: Add optional native metadata and drift reporting.
- `internal/config/config.go`: Adds the new default mode and generated-file comments.
- Tests: Add table-driven API/reconciliation tests and command tests for disabled, unavailable, missing-extension, exact, appendable, divergent, dry-run, native landing refusal, and native abandon cases.
- Compatibility constraints: GitHub native stacks require 2-100 same-repository PRs with linear history and do not support cross-fork stacks.

## Port Compatibility

This deliberately diverges from the Python `stack-pr` behavior described in `SPEC.md` sections 5, 6.2, 7, 13, 16, and 17 because the upstream tool predates GitHub's native stack object. Legacy behavior remains the default under `github.native_stacks = off`; the Go port and `SPEC.md` will define the opt-in native behavior together.

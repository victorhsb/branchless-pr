## Why

Existing PRs can become disconnected from local commits when a commit was pushed or a PR was opened before the commit message received `stack-info` metadata. Users need a narrow recovery command that repairs the current local commit without running the full submit/export workflow or pushing anything.

## What Changes

- Add a first-class `bpr fix --pr <number> [--replace] [--dry-run]` command.
- `fix` repairs only `HEAD` by appending local `stack-info` metadata from the explicitly named existing PR.
- `fix` is local-only: it does not create branches, push branches, create PRs, edit PRs, or retarget PRs.
- `fix` warns, but does not fail, when the PR head SHA differs from local `HEAD`.
- `fix` refuses to rewrite `HEAD` if stack metadata already exists, unless `--replace` is supplied; matching existing metadata is treated as already fixed.
- `fix --dry-run` performs read-only inspection and reports what would change without amending the commit.
- `fix` prints a post-command hint to run `bpr submit` and includes advisory stack-readiness warnings when the stack cannot be inspected, has missing PR metadata, or has malformed PR metadata.
- Add agent prompt guidance for `fix` as a recovery command after the core command semantics are implemented.

## Capabilities

### New Capabilities

- `fix-command`: Defines local-only repair of `HEAD` stack metadata from an explicitly selected existing PR.

### Modified Capabilities

- `agent-prompt-command`: Adds `fix` recovery guidance to the deterministic agent prompt content.

## Impact

- Affected CLI surface: new `fix` subcommand on the `bpr` root command.
- Affected internals: CLI command wiring, GitHub PR inspection through the existing `gh` wrapper path, commit message metadata handling, clean/sequencer preflight checks, advisory stack inspection, dry-run rendering, and tests.
- Affected agent content: static prompt metadata should describe `bpr fix --pr <number>` as local metadata repair and tell agents to run `bpr submit` afterward for push/update behavior.
- No new external dependencies. The command must continue to shell out through `internal/shell`; no Go GitHub SDK is introduced.
- Root `SPEC.md` is deprecated and is not updated by this change; OpenSpec specs are the behavioral contract.
- Land behavior is not affected.

## Port Compatibility

This is a branchless-pr recovery command, not a known Python `stack-pr` compatibility requirement. It intentionally diverges by adding a local-only repair flow for already-open PRs whose local commit metadata is missing or wrong.

## Why

Landing a stack of N PRs with the current `bottom-only` strategy requires N sequential `land` invocations. Each invocation performs a squash-merge of the bottom PR, fetches, checks out, rebases, and force-pushes every remaining branch. For a 5-PR stack this totals ~21 network round trips and N local rebase+push cycles. The per-PR overhead makes landing even moderately sized stacks slow.

## What Changes

- Add a `whole-stack` land style that lands the entire stack in one operation by retargeting the tip PR to the target branch and performing a GitHub rebase merge on the tip PR.
- The `--whole-stack` CLI flag selects this mode for a single invocation; `land.style = whole-stack` in `.stack-pr.cfg` makes it the default.
- When `whole-stack` is selected, the command checks that the repository allows rebase merges via the GitHub GraphQL API. If rebase merges are disabled, the command errors out with a clear message.
- Commits land individually and linearly on the target branch via the rebase merge, preserving each commit's original message (which already contains PR number references in stack metadata).
- After the tip PR merges, the command cleans up local state (restores original branch, deletes local stack branches, rebases onto new target) without needing to rebase or push remaining stack branches individually.
- The existing `bottom-only` style remains the default; `whole-stack` is opt-in.

## Capabilities

### Modified Capabilities

- `land`: The land command gains a new `whole-stack` style that atomically lands all PRs in the stack by rebase-merging the tip PR directly into the target branch.
- `land`: The `land.style` config key gains a new value `whole-stack` alongside `bottom-only` and `disable`.
- `land`: The `--whole-stack` CLI flag overrides the configured style for a single invocation.

## Impact

- `internal/cli/land.go`: Dispatch between `bottom-only` and `whole-stack` implementations; add `--whole-stack` flag.
- `internal/cli/root.go`: Register the `land` subcommand when `land.style` is `whole-stack` (in addition to `bottom-only`).
- `internal/pr/pr.go`: Add a function to query repository merge settings (`rebaseMergeAllowed`) via the GitHub GraphQL API.
- `internal/config/config.go`: Support the `whole-stack` value for the `land.style` config key.
- `openspec/specs/land/spec.md`: Add scenarios for `whole-stack` land style.
- Tests: Add coverage for the new style dispatch, merge-settings check, and cleanup flow.

## Port Compatibility

The Python `stack-pr` tool only supports `bottom-only` landing. The `whole-stack` style is a Go-port addition with no Python equivalent. It does not alter the behavior of `bottom-only` mode.

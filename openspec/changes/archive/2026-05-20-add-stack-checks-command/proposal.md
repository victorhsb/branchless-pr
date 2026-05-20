## Why

Stacked pull requests are hard to keep review-ready because CI and check failures are spread across multiple GitHub pull requests. Users and agents need one read-only command that reports all check state across the stack and identifies failed checks precisely enough to decide what to fix next.

## What Changes

- Add a top-level `stack-pr checks` command that discovers the current stack and reads check status for every pull request with PR metadata.
- Report all checks, not only branch-protection-required checks, while identifying whether each check is required when that information is available.
- Provide text and JSON output formats with deterministic stack order and stable check grouping.
- Include a failed-check summary whose entries carry stable, agent-usable check IDs plus GitHub provider IDs when available.
- Support focused read-only filters such as failed-only, PR number, commit, and required-only without mutating local Git or GitHub state.

## Capabilities

### New Capabilities

- `stack-checks-report`: Read-only stack-wide GitHub check reporting for all pull requests in the current stack, including stable failed-check identifiers for agent workflows.

### Modified Capabilities

None.

## Impact

- Affected CLI surface: `internal/cli` command registration and output handling.
- Affected GitHub integration: `internal/pr` read-only `gh` wrappers or adjacent helpers for check runs/status contexts.
- Affected stack flow: reuse current stack discovery and PR metadata mapping from `view` and other inspection commands.
- Documentation impact: README/help text and `SPEC.md` should describe `stack-pr checks`, formats, filters, all-check behavior, and read-only guarantees.
- No new GitHub SDK dependency; implementation should continue shelling out through `internal/shell`.

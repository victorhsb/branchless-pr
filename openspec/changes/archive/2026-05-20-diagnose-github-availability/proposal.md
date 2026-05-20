## Why

GitHub outages can make `stack-pr` operations fail in ways that look like local stack or commit-state problems. `stack-pr agent diagnose --online` should identify likely GitHub service unavailability early and steer agents away from mutating or commit-state-dependent actions until remote state can be trusted.

## What Changes

- Add an explicit GitHub availability check to `stack-pr agent diagnose --online`.
- Classify likely GitHub outages separately from authentication, missing `gh`, malformed local metadata, and ordinary PR lookup failures.
- Surface outage findings in both text and JSON output with stable check IDs and actionable guidance.
- Ensure outage findings block mutation-oriented recommendations, including `submit`, `land`, and `abandon`, while preserving the command's read-only and exit-code-0 diagnosis contract.
- Keep offline diagnose behavior network-free; GitHub availability is only evaluated when `--online` is provided.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-diagnose`: Add online GitHub availability detection and recommendation gating when GitHub appears unavailable.

## Impact

- Affected code: `internal/diagnose`, `internal/cli` diagnose wiring if flags/help need clarification, and tests for online check behavior and recommendation selection.
- Affected external contract: `stack-pr agent diagnose --online --format json` gains a stable check entry for GitHub availability and may emit a blocking recommendation when GitHub is unavailable.
- Dependencies: continue using existing `gh`/shell abstractions; no Go GitHub SDK dependency.

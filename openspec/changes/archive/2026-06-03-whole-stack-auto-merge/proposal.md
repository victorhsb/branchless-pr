## Why

The current `whole-stack` land style attempts an immediate GitHub rebase merge of the tip PR after retargeting it to the target branch. In repositories with branch protection, CI often starts only after that retargeting and can take many minutes, so an immediate merge commonly fails even though the desired action is to add the tip PR to GitHub's merge queue once requirements are satisfied.

## What Changes

- Require GitHub merge queue for `whole-stack` landing.
- Change `whole-stack` landing to schedule the retargeted tip PR through GitHub's merge queue instead of polling CI or attempting an immediate merge.
- Return a clear error when the repository or target branch does not have merge queue enabled: `--whole-stack only works for repositories with merge queue enabled`.
- Preserve `bottom-only` and `disable` land styles unchanged.
- Update `SPEC.md` and the land spec so the behavioral source of truth distinguishes queued whole-stack landing from completed bottom-only landing.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `land`: The `whole-stack` style requires merge queue and queues the retargeted tip PR for GitHub-managed merge once requirements pass.

## Impact

- `openspec/specs/land/spec.md`: Add merge-queue requirements, queued whole-stack behavior, and cleanup differences.
- `SPEC.md`: Update command and algorithm sections for the whole-stack merge queue semantics.
- `internal/cli/land.go`: Check merge queue support before mutating, then queue the retargeted tip PR.
- `internal/pr/pr.go`: Add a wrapper for `gh pr merge <tip-pr> --rebase --auto`.
- Tests: Cover merge queue disabled errors, GitHub command arguments, and the fact that queued whole-stack mode does not perform post-merge cleanup that assumes the target branch already advanced.

## Port Compatibility

The Python `stack-pr` tool does not support `whole-stack` landing. This change only affects the Go port's existing `whole-stack` extension and leaves Python-compatible `bottom-only` behavior unchanged.

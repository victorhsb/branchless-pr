## Why

`submit/export --stash` creates its automatic stash during persistent pre-run,
but cleanup currently begins only after command dispatch. Any later pre-run
failure strands the user's changes in the stash instead of restoring the
original working tree.

## What Changes

- Establish automatic-stash cleanup immediately after a stash is created.
- Restore the stash when post-stash clean validation, target validation, or
  merge-base deduction fails before command dispatch.
- Preserve restoration after successful command execution and command-body
  failure, covering the complete post-stash invocation lifecycle.
- Add integration tests that prove the original working tree is restored for
  successful and failed invocations.
- Clarify the lifecycle guarantee in `SPEC.md` sections 8 and 20.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `submit-export`: Adds an explicit automatic-stash lifecycle requirement that
  covers failures before and after command dispatch.

## Impact

The change affects root command pre-run cleanup in `internal/cli`, the existing
submit recovery boundary, integration tests, and the stash lifecycle text in
`SPEC.md`. It does not change `--dry-run` behavior, the land command, or any
Git/GitHub mutation algorithm. No dependency is added.

## Port compatibility

This brings the Go port into alignment with Python `stack-pr`'s `finally`
semantics documented in `SPEC.md` section 8: once an automatic stash is created,
it is restored regardless of whether failure occurs before or during command
dispatch.

## Context

Persistent pre-run creates the optional submit/export stash after repository and
identity discovery but before clean checking, target validation, and merge-base
deduction. The `AppContext` is installed on the Cobra command only after those
steps. Consequently, `WithRecovery` and `submitImpl` never run when a later
pre-run check fails, and neither can restore the stash.

Three boundaries currently own portions of recovery: root pre-run, the mutating
command wrapper, and submit's successful-return cleanup. The fix must cover the
gap without changing dry-run or the Git stash identity semantics tracked by the
next TODO item.

## Goals / Non-Goals

**Goals:**

- Restore an automatic stash after every error that occurs after stash creation.
- Preserve restoration after successful submit/export execution.
- Keep the original pre-run error visible if stash restoration also fails.
- Make stash restoration idempotent at the invocation-state level.
- Cover clean-check, target-validation, merge-base, command-success, and
  command-failure paths with real working-tree tests.

**Non-Goals:**

- Replacing output-based stash-creation detection.
- Identifying or applying a stash by stable object identity.
- Redesigning restoration-conflict handling beyond preserving both errors.
- Enabling stash mutation during dry-run.

## Decisions

### Give `AppContext` a single restoration operation

`AppContext` will expose a method that restores its recorded automatic stash and
clears the recorded state after a successful pop. Root pre-run, `WithRecovery`,
and submit success cleanup will all call this operation instead of invoking
`git.StashPop` directly.

Keeping a shared operation prevents successful cleanup from being repeated if
ownership boundaries evolve. Leaving direct pop calls in all three locations
was rejected because the boolean state would not be consistently consumed.

### Establish an error defer immediately after stash creation

Persistent pre-run will use a named error result and install a defer before any
post-stash validation. If pre-run returns an error and the invocation recorded a
stash, the defer restores it. On successful pre-run, ownership transfers to the
command lifecycle: `WithRecovery` handles command errors and submit cleanup
handles successful returns.

Moving stash creation after all validation was rejected because the clean check
is intentionally performed after stashing. Wrapping only command `RunE` was
rejected because Cobra does not call it after a pre-run failure.

### Preserve both initialization and cleanup failures

If pre-run fails and stash restoration also fails, the returned error will join
the original initialization error with an actionable restoration error. The
original cause must not be hidden, and a restoration failure must not be silent.

Logging only a warning was rejected because a failed restore means the promised
working-tree recovery did not happen.

## Risks / Trade-offs

- **Cleanup remains coordinated across Cobra lifecycle boundaries** → All
  boundaries call one state-consuming `AppContext` method, with integration
  tests for success and failure.
- **Current stash pop still targets the top stash entry** → This is explicitly
  deferred to the next TODO item, which introduces stable stash identity.
- **A pop conflict leaves recovery state unresolved** → The pre-run error
  preserves the restoration failure; dedicated conflict behavior is covered by
  the subsequent stash-identity change.

## Migration Plan

Update `SPEC.md`, introduce the shared restoration operation, add the pre-run
error defer, and migrate existing cleanup callers. Run focused lifecycle tests
and all CI gates. No data migration is needed; rollback restores the prior
cleanup calls and specification text.

## Open Questions

None.

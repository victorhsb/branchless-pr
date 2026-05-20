## Context

`stack-pr submit` and its `export` alias are the highest-risk commands for agents because they perform many side effects in one flow: branch checkout/creation, force-push, PR create/edit, draft-state changes, metadata amendments, rebases, local branch cleanup, and stash recovery. The command currently communicates progress through human stdout/stderr and returns an error when a step fails, but there is no structured record of which side effects completed before the failure.

The CLI already has stable JSON for read-only surfaces (`view`, `agent prompt`, and `agent diagnose`). Submit operation receipts extend that pattern to mutating command outcomes without changing default human behavior.

## Goals / Non-Goals

**Goals:**

- Provide an opt-in JSON receipt for `stack-pr submit` and the `export` alias.
- Record successful side effects in execution order.
- Record handled failures, warnings, and best-effort recovery attempts.
- Support receipt defaults from `.stack-pr.cfg`.
- Keep default submit/export output and behavior unchanged when receipts are disabled.
- Keep the receipt schema stable and versioned.

**Non-Goals:**

- Do not add machine-readable dry-run plans in this change.
- Do not add receipts for `land` or `abandon` in this change.
- Do not introduce a GitHub SDK or replace existing `git`/`gh` shell behavior.
- Do not make receipts a transactional rollback mechanism; they report observed outcomes.

## Decisions

1. **Use a dedicated receipt destination flag instead of overloading `--format`.**
   - Add `--receipt <destination>` to `submit` and its `export` alias.
   - Supported values:
     - `off`: disable receipt emission.
     - `-`: emit the receipt JSON to stdout.
     - any other value: write the receipt JSON to that file path.
   - Rationale: mutating commands already print human progress directly. A receipt destination lets users keep human stdout while also writing structured JSON to a file. `-` still supports fully machine-oriented callers.
   - Alternative considered: `--format json`. Rejected for the first slice because it requires broader stdout/stderr cleanup across submit before the receipt model itself is proven.

2. **Configure submit receipts under a receipt section.**
   - Add `.stack-pr.cfg` key `receipt.submit`.
   - Default is `off`.
   - CLI `--receipt` overrides `receipt.submit`.
   - Rationale: a `[receipt]` section scales to future command receipts (`land`, `abandon`) without creating command-specific config shapes that need to be migrated later.
   - Alternative considered: `[submit] receipt = ...`. Rejected because the concept is cross-cutting even though the first implementation target is submit.

3. **Emit receipts for real submit/export executions only.**
   - `--dry-run --receipt <destination>` is rejected with a clear error.
   - Rationale: a receipt records what was attempted and observed. Dry-run needs a plan schema, not an operation receipt schema.
   - Alternative considered: emit a receipt-like dry-run object with no operations. Rejected because it would blur the distinction between plans and receipts.

4. **Record operations at the CLI orchestration boundary.**
   - Add a small receipt recorder in `internal/cli` or `internal/receipt`.
   - Submit appends an operation only after a side effect succeeds.
   - When a step fails, submit records a failed operation with a stable operation type and error message before returning the existing error.
   - Rationale: the CLI layer knows command intent, stack entries, and recovery boundaries; lower-level `git`/`pr` packages should remain typed shell wrappers.

5. **Make recovery observable.**
   - Extend or wrap `WithRecovery` so commands using receipts can record recovery checkout and stash-pop attempts.
   - Rationale: after failure, the most important question is whether the working copy returned to the original branch and whether an auto-stash was restored.
   - Alternative considered: leave recovery out of the receipt. Rejected because it weakens the receipt precisely when agents need it most.

6. **Preserve existing exit behavior.**
   - Receipts do not turn failed commands into successful exits.
   - If submit fails, the command still returns the same error class/message, and the receipt captures the failure when possible.
   - Receipt file write failures are command failures because the user explicitly requested an artifact.

## Risks / Trade-offs

- **Risk: Receipt output is corrupted by human stdout when destination is `-`.** -> In `--receipt -` mode, route submit's human progress output to stderr or suppress it so stdout contains one JSON document.
- **Risk: Receipts imply transactional guarantees.** -> The schema and docs must describe receipts as observed audit records, not rollback or consistency guarantees.
- **Risk: Failed pre-run checks do not produce receipts.** -> Define the boundary clearly: receipts cover submit/export execution after receipt configuration is resolved and the command body starts.
- **Risk: Operation recording becomes noisy or brittle.** -> Start with stable, high-value side-effect operations rather than every helper call.
- **Risk: Config paths differ by invocation directory.** -> Resolve relative receipt file paths against the current working directory, matching normal CLI file path expectations, and document this behavior.

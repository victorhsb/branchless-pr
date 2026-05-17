## Context

`submit` and its `export` alias share the same Cobra command and execute `submitImpl`, which currently interleaves planning with mutating operations: rebasing the base, checking out generated branches, force-pushing heads, creating/updating PRs through `gh`, amending commit metadata, restoring draft state, rebasing/checking out the original branch, and deleting generated local branches.

A dry run should exercise the same stack discovery and validation path that determines what would happen, but it must not perform any local Git mutation, remote push, or GitHub write. The implementation should preserve the existing behavior and output of normal submit/export.

## Goals / Non-Goals

**Goals:**

- Add `--dry-run` to `submit` and the `export` alias.
- Show a clear plan for each stack entry: generated head branch, base branch, whether a PR would be created or updated, draft status for new PRs, and whether metadata would be added.
- Avoid all mutating operations in dry-run mode, including stash save/pop, rebases, checkouts, branch creation/deletion, commit amendments, pushes, and PR create/edit/draft-state calls.
- Reuse existing stack discovery, metadata parsing, branch-name assignment, base assignment, draft bitmask validation, and stack printing behavior where safe.
- Keep non-dry-run behavior unchanged.

**Non-Goals:**

- Guarantee that a later non-dry-run export cannot fail due to remote races or changed repository state.
- Contact GitHub to simulate PR creation beyond read-only validation of existing PR metadata already required by stack verification.
- Add machine-readable dry-run output in this change.
- Change the submit/export algorithm for real execution.

## Decisions

1. **Thread a `dryRun` boolean through submit command execution.**
   - Add a `--dry-run` flag in `submitCmd`, pass it to `runSubmit`, and then to `submitImpl`.
   - Rationale: this is the smallest API change and keeps submit/export behavior centralized.
   - Alternative considered: create a separate `export dry-run` subcommand. Rejected because Cobra aliases make `submit --dry-run` and `export --dry-run` simpler and consistent with common CLI conventions.

2. **Branch early into a dedicated dry-run implementation after non-mutating setup.**
   - `submitImpl` should validate rebase-in-progress, discover the stack, read metadata, validate draft settings, fetch as needed, assign heads/bases, and then call a dry-run planner before any checkout/rebase/push/PR-write/amend operation.
   - Rationale: dry-run output stays aligned with real submit inputs while enforcing a clear mutation boundary.
   - Alternative considered: guard every mutating statement inline. Rejected because it is easier to accidentally miss a mutating path and harder to test.

3. **Do not auto-stash in dry-run mode.**
   - Update the persistent pre-run so `--stash` only runs for submit/export when dry-run is false.
   - Rationale: saving and popping a stash is itself a local mutation and violates dry-run expectations.
   - Alternative considered: allow `--dry-run --stash` to verify that a real export with stash would pass. Rejected because dry-run should be side-effect free.

4. **Keep existing cleanliness and target/base validation unless they would require mutation.**
   - Dry-run should still require a clean repository like real submit/export, except `--stash` must not clean it implicitly.
   - It should still verify `gh` availability, remote target existence, and deduce base.
   - Rationale: users want to know whether the current invocation would be accepted.
   - Alternative considered: skip all external checks. Rejected because it would make dry-run less predictive.

5. **Use human-readable plan output for this change.**
   - Print a dry-run header and a per-entry plan with action (`create PR` or `update PR`), title, head, base, existing PR URL if any, draft state for new PRs, and metadata-amend indication.
   - Finish with an explicit note that no changes were made.
   - Rationale: meets the immediate safety need without introducing another output schema.
   - Alternative considered: JSON output. Rejected as out of scope; it can be proposed separately if automation needs emerge.

## Risks / Trade-offs

- **Risk: Dry-run accidentally performs a mutation.** → Mitigate by branching to a dedicated dry-run path before checkouts, pushes, PR writes, metadata amendments, branch cleanup, and stash handling; add tests around mutating collaborators where practical.
- **Risk: Dry-run diverges from real submit behavior over time.** → Mitigate by sharing pre-planning logic and keeping dry-run as close as possible to the real algorithm's decision points.
- **Risk: Fetch is surprising in dry-run.** → `git fetch` updates remote-tracking refs, but existing submit relies on fresh remote state for correctness. Document dry-run as avoiding branch/commit/PR mutations; if desired later, a stricter `--offline` mode can be proposed.
- **Risk: Existing PR verification may require GitHub reads.** → Reads are acceptable for predictive validation; writes remain forbidden.

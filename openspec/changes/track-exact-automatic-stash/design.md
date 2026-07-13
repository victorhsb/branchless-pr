## Context

`git.StashSave` currently returns a boolean by searching stdout for the English
text `No local changes to save`. `AppContext` records that boolean and recovery
runs `git stash pop`, which always targets the top stash entry. This makes
creation detection locale-dependent and loses ownership if another stash entry
appears before recovery.

The preceding lifecycle change established one state-consuming restore method on
`AppContext`. This change replaces its boolean with a stable stash identity and
strengthens the Git-layer restore primitive.

## Goals / Non-Goals

**Goals:**

- Detect creation without reading human-facing stash output.
- Preserve the exact automatic stash commit OID in invocation state.
- Apply and remove only the automatic stash, even when older or newer user
  stashes exist.
- Keep the stash reflog entry when application conflicts.
- Return actionable errors for missing, conflicting, or non-removable automatic
  stashes.
- Preserve dry-run and full post-stash lifecycle behavior.

**Non-Goals:**

- Coordinating stash changes made concurrently between individual Git commands.
- Automatically resolving stash application conflicts.
- Changing which tracked changes `git stash push` includes.
- Adding untracked files to the automatic stash.

## Decisions

### Detect creation by comparing `refs/stash`

Before and after `git stash push -m <message>`, the Git layer will resolve
`refs/stash^{commit}` with quiet `git rev-parse`. An unchanged ref means the
working tree was clean and no automatic stash was created; a changed ref yields
the new commit OID. Stash command stdout is ignored entirely.

Parsing `git stash` prose was rejected because it is localized and not a stable
interface. Comparing stash-list length was rejected because the OID is both a
creation signal and the identity recovery needs.

### Store a typed stash reference

The Git package will return a `StashRef` containing the commit OID, and
`AppContext` will hold that value instead of `StashCreated bool`. A zero value
means no automatic stash. Successful restoration clears the value; failed
restoration retains it.

Keeping a boolean plus a separate string was rejected because those fields can
become inconsistent.

### Apply the exact OID, then drop its matching reflog selector

Restoration first locates the OID in structured `git stash list` output, applies
the exact commit with `git stash apply <oid>`, locates the OID again, and drops
only the matching `stash@{n}` entry. Re-resolving after apply tolerates another
entry being added above it during normal invocation flow.

`git stash pop` was rejected because it targets a position rather than the
stored object. Applying without dropping was rejected because successful
automatic recovery should consume only its own stash entry.

### Preserve the stash when application fails

The reflog entry is dropped only after a successful apply. A conflict returns an
error that identifies the automatic stash and explains it remains available for
manual recovery. If dropping fails after apply, return an error explaining that
the working-tree changes were restored but the stash entry remains.

Dropping before apply was rejected because it could destroy the recovery copy
on conflict.

## Risks / Trade-offs

- **Apply and drop are separate Git operations** → Resolve and validate the
  matching reflog selector immediately before drop; never fall back to the top
  entry.
- **Duplicate reflog entries could point at one OID** → Drop the first exact
  match reported by Git; stash commits created by separate pushes normally have
  distinct identities.
- **Drop failure can leave an already-applied stash entry** → Return explicit
  manual cleanup guidance and retain invocation identity.
- **SHA format can vary by repository object format** → Treat the OID as an
  opaque non-empty string rather than enforcing a 40-character SHA-1 shape.

## Migration Plan

Update `SPEC.md`, introduce `StashRef`, migrate root and `AppContext`, replace
top-pop restoration, and extend Git/CLI integration tests. Run all CI gates. No
stored data migration is required.

## Open Questions

None.

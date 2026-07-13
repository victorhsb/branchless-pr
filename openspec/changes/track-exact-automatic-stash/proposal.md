## Why

Automatic stash creation is currently inferred from an English Git message, and
restoration blindly pops the top stash entry. Localized or unexpected output can
misclassify a clean tree, while another stash entry can cause recovery to restore
or remove user state that the command did not create.

## What Changes

- Detect stash creation by comparing the structured `refs/stash` object identity
  before and after `git stash push`, without parsing human output.
- Store the automatic stash's commit OID in invocation state.
- Restore by applying that exact OID and dropping only its matching stash reflog
  entry, preserving older and newer user stashes.
- Leave the automatic stash available when apply conflicts or cleanup fails, and
  return actionable recovery errors.
- Add tests for clean trees, pre-existing and newer user stashes, localized or
  unexpected output, successful restoration, and restoration conflicts.
- Update `SPEC.md` sections 8, 11, and 20 with the stable-identity contract.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `submit-export`: Strengthens automatic-stash creation detection and recovery
  so only the stash created by the invocation is restored and removed.

## Impact

The change replaces the `internal/git` boolean stash API with stable stash
identity, updates `AppContext` and root pre-run state, and extends Git and CLI
integration coverage. The land command and dry-run behavior are unchanged. No
dependency is introduced, and all Git calls continue through `internal/shell`.

## Port compatibility

Python `stack-pr` remembers whether `git stash save` appeared to create a stash
and later pops the top entry. The Go port intentionally strengthens this safety
boundary by tracking the exact stash object, avoiding localized-output parsing
and preventing unrelated stash removal.

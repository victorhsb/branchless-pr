## Why

Git operation detection currently assumes the repository metadata is a `.git`
directory at the invocation path. That produces false negatives from
subdirectories and in valid layouts such as linked worktrees, submodules, and
repositories with a separate Git directory, allowing commands to proceed while
a rebase, merge, or cherry-pick is active.

## What Changes

- Resolve the repository's actual Git metadata paths through Git instead of
  constructing `.git/...` paths.
- Detect rebase, merge, cherry-pick, and sequencer state from repository
  subdirectories, linked worktrees, submodules, and separate-Git-dir layouts.
- Add focused tests for every supported layout and operation state.
- Update `SPEC.md` sections 11 and 20 to make repository-layout-aware detection
  the port's required behavior.

## Capabilities

### New Capabilities

- `git-operation-safety`: Defines layout-aware discovery and detection of active
  Git operations before commands mutate repository state.

### Modified Capabilities

None.

## Impact

The change affects `internal/git` operation-state helpers and their callers in
the CLI and diagnostic paths, plus `SPEC.md` and focused Git integration tests.
All Git subprocesses continue to use `internal/shell`; no new dependency is
introduced. The land command is protected by the shared operation check but its
config gating and landing algorithm are unchanged.

## Port compatibility

This intentionally diverges from Python `stack-pr`, whose behavior documented in
`SPEC.md` section 11 checks a literal `.git` path. The Go port will use Git's
repository layout resolution so the same safety check works for all supported
Git repository layouts.

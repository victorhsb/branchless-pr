## Context

The operation-state helpers in `internal/git` currently construct paths beneath
`.git`. That only works when the caller points at a worktree root whose metadata
is a directory. Git itself already knows which paths belong to the current
worktree and repository, including per-worktree state in linked worktrees.

The public helpers return booleans and are used as safety gates. This change must
preserve that API while routing all subprocess execution through
`internal/shell`.

## Goals / Non-Goals

**Goals:**

- Resolve every operation marker through Git for the invocation context.
- Work from repository roots and subdirectories in ordinary repositories,
  linked worktrees, submodules, and separate-Git-dir repositories.
- Preserve detection of rebase, merge, cherry-pick, and sequencer markers.
- Keep detection read-only and retain the existing boolean helper API.

**Non-Goals:**

- Changing how callers recover from or abort an active Git operation.
- Detecting additional Git operations beyond the currently modeled states.
- Changing command availability, land configuration, or mutation algorithms.

## Decisions

### Resolve each marker with `git rev-parse --git-path`

Each marker path will be obtained with `git rev-parse --git-path <marker>` in
the requested repository directory, then checked with `os.Stat`. Git's
`--git-path` accounts for `$GIT_DIR`, `$GIT_COMMON_DIR`, linked-worktree
metadata, `.git` files, and invocation from subdirectories.

Resolving only `--git-dir` and joining marker names was rejected because Git can
place worktree-specific and shared files differently. Parsing `.git` files was
also rejected because it would duplicate Git's repository-layout rules.

### Resolve relative paths in the command's working directory

`git rev-parse --git-path` can return a relative path. When a repository
directory is supplied, relative output is joined to that directory; otherwise
it remains relative to the process working directory from which Git was run.
Absolute output is used unchanged.

Requiring `--path-format=absolute` was rejected to avoid unnecessarily raising
the minimum supported Git version.

### Preserve best-effort boolean semantics

Failure to resolve a marker path or a missing marker continues to mean that the
specific operation is not reported as active. This preserves the existing API
and caller behavior. Repository validity is established separately by command
pre-run and diagnostic checks.

Changing the helpers to return `(bool, error)` was rejected as broader than this
fix and would require redesigning safety-gate error handling.

## Risks / Trade-offs

- **Git resolution failure can still produce a false negative** → Callers
  continue to validate repository state separately; focused tests cover every
  supported layout.
- **Relative path interpretation could drift from Git's working directory** →
  The helper uses the same `RunOpts.Dir` as the path base and tests invocation
  from nested directories.
- **Manually created test markers may not model all Git internals** → Tests use
  Git to create repository layouts and Git to resolve the marker location before
  placing the state marker.

## Migration Plan

Update `SPEC.md` and the helper implementation together, then run focused Git
tests and the full Go verification suite. The change is internal and needs no
data migration. Reverting the code and matching `SPEC.md` text restores the old
behavior.

## Open Questions

None.

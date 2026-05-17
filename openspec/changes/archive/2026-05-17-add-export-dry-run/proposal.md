## Why

`stack-pr export` currently performs Git branch creation, force-pushes, PR creation/updates, metadata amendments, and branch cleanup immediately. Users need a safe way to preview what an export would do before mutating local Git state or GitHub PRs.

## What Changes

- Add a `--dry-run` flag to `stack-pr submit` and its `export` alias.
- In dry-run mode, discover and validate the stack, compute generated head/base branches, PR creation/update intent, draft settings, metadata needs, and cleanup steps, then print a human-readable plan.
- Prevent local Git mutations, remote pushes, and GitHub PR create/edit/draft-state operations while dry-run is enabled.
- Keep existing non-dry-run `submit`/`export` behavior unchanged.

## Capabilities

### New Capabilities

- `export-dry-run`: Preview submit/export actions without changing local Git state or GitHub state

### Modified Capabilities

<!-- No existing specs are present; this change introduces a new capability while preserving current submit/export behavior. -->

## Impact

- `internal/cli/submit.go`: Add `--dry-run` flag, pass it into submit execution, and implement dry-run planning/output.
- `internal/cli/root.go`: Ensure pre-run safety behavior remains appropriate for dry-run, including no automatic stash/pop side effects.
- `internal/stack`: Reuse stack discovery, metadata, head/base assignment, printing, and PR body planning helpers as needed.
- `README.md` and command help: Document dry-run usage for `submit`/`export`.
- Tests: Add coverage for flag parsing, dry-run behavior boundaries, and plan formatting where practical.

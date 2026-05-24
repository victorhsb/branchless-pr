## Why

Submit/export initializes one generated local branch per stack entry by checking out each commit with `git checkout <commit-id> -B <entry.head>`. For larger stacks this repeatedly switches the worktree even though the required outcome is only that the generated branch refs point at the right commits before pushing.

## What Changes

- Change local branch initialization so submit/export ensures each generated head branch points at the intended commit without checking out every stack entry.
- Preserve existing generated branch names, base computation, force-push behavior, metadata amendment behavior, original branch restoration, and cleanup.
- Keep checkout/rebase operations where they are still needed later for metadata amendment or original-branch restoration.
- Update `SPEC.md` and OpenSpec requirements to describe the behavioral contract as ref initialization rather than per-entry worktree checkout.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `submit-export`: Local branch initialization should be specified by the branch refs it creates/resets, not by mandatory per-entry checkout commands.

## Impact

- `SPEC.md`: Update the submit/export algorithm step that currently mandates `git checkout <commit-id> -B <entry.head>`.
- `internal/cli/submit.go`: Replace the initial checkout loop with a non-checkout ref update path.
- `internal/git/git.go`: Add or reuse a wrapper for force-updating local branches without switching the worktree.
- Tests: Add coverage that submit/export initializes generated branches while preserving the current branch until later metadata/cleanup steps.

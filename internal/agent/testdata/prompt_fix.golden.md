# stack-pr agent prompt: Fix

Guidance for local metadata repair on HEAD.

Use `bpr fix --pr <number>` to attach an existing PR to local HEAD metadata when the commit message is missing or incorrect stack-info.

This command only amends the local HEAD commit. It does not push branches or write PR changes.

After fixing metadata, run `bpr submit` to push the amended commit and update PRs.

## Commands

- `stack-pr fix --dry-run` — Preview metadata repair on HEAD without amending the commit or writing to GitHub. Side effects: no.
- `stack-pr fix` — Repair stack-info metadata on HEAD from an existing PR. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Amends HEAD to add or replace stack-info metadata.
  - Does not create branches, push branches, or modify PRs on GitHub.

## Rules

- Use fix when the user has an existing PR whose local commit is missing stack-info metadata.
- Prefer `bpr fix --pr <number> --dry-run` first to preview the planned metadata change.
- Always tell the user to run `bpr submit` after a successful fix to publish the amended commit.
- Do not use fix as a substitute for submit when the user wants to publish or update PRs.

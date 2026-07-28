---
title: fix
status: stable
---

# Fix

## Overview

`bpr fix` repairs stack metadata on the local `HEAD` commit. It is a local-only repair operation: it amends only the current commit message and never touches remotes or GitHub.

## Behavior

### Command registration

- `bpr fix --help` → exit successfully; help output describes `--pr`, `--replace`, and `--dry-run`.
- `bpr fix` without `--pr` → exit non-zero with a clear validation error; no local Git mutation occurs.

### Preflight

Repository state is validated before repairing metadata; unsafe local rewrites are blocked.

- Working tree has staged or unstaged changes → exit non-zero with a clean-tree error; no local Git mutation occurs.
- Rebase, merge, or cherry-pick operation is in progress → exit non-zero with an actionable error; no local Git mutation occurs.
- Repository and PR preflight succeeds → inspect and amend `HEAD` directly; successful stack discovery is not required before attempting the repair.

### Explicit PR metadata source

The explicitly selected existing PR is the source for local `stack-info` metadata.

- `bpr fix --pr <number>` → load PR `url`, `number`, `headRefName`, `baseRefName`, and `headRefOid` through `gh pr view`; use the PR URL and head branch for the local metadata line.
- Selected PR's `headRefOid` differs from local `HEAD` → print a warning identifying the PR head SHA and local `HEAD` SHA, and continue with the local metadata repair.

### Local metadata repair

Only the current local `HEAD` commit message is repaired.

- `HEAD` has no `stack-info` metadata → append `stack-info: PR: <pr-url>, branch: <head-branch>` to the current commit message, separated from the commit title/body by at least one blank line, and amend `HEAD` with `git commit --amend -F -`.
- `HEAD` already has `stack-info` metadata for the selected PR URL and PR head branch → report that `HEAD` is already fixed; do not amend the commit.
- `HEAD` already has `stack-info` metadata that differs from the selected PR and `--replace` is not set → exit non-zero with an error explaining that existing metadata is present; no local Git mutation occurs.
- `HEAD` already has `stack-info` metadata that differs from the selected PR and `--replace` is set → replace the existing metadata line with `stack-info: PR: <pr-url>, branch: <head-branch>` and amend `HEAD` with `git commit --amend -F -`.

### Local-only side effects

- `bpr fix --pr <number>` → does not create or reset local generated branches, does not push to any remote, and does not create, edit, retarget, mark draft, mark ready, merge, or close any PR.
- Fix completes successfully → print a hint that metadata was fixed locally, telling the user to run `bpr submit` to push the amended commit and update PRs.

### Dry-run

`--dry-run` reports the planned repair without mutating local Git or GitHub state.

- `bpr fix --pr <number> --dry-run` → load the selected PR and inspect `HEAD`; print the PR URL, PR head branch, local `HEAD` SHA, existing metadata state, and the metadata line it would add or replace; state that no commit was changed.
- `bpr fix --pr <number> --dry-run` → does not amend `HEAD`, does not push to any remote, and does not write to GitHub.

### Advisory stack readiness

After planning or applying a fix, advisory warnings report whether the stack appears ready for submit.

- Advisory stack inspection succeeds and one or more discovered stack entries are missing PR metadata → print a warning that the stack is not fully ready to submit, including the count of entries missing PR metadata.
- Advisory stack inspection finds malformed PR metadata → print a warning that the stack has malformed PR metadata; the warning does not cause a successful local repair to fail.
- Advisory stack inspection fails → print a warning explaining that stack readiness could not be determined; the warning does not cause a successful local repair to fail.
- `bpr fix --pr <number> --dry-run` → run the same read-only advisory stack-readiness inspection; warnings are phrased as dry-run diagnostics.

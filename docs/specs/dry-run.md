---
title: Dry-run
status: stable
---

# Dry-run

## Overview

Several `stack-pr` commands support a `--dry-run` flag that previews the actions the command would perform without applying them. Dry-run is currently supported by `submit` (and its `export` alias) and by `fix`. Commands without a documented dry-run mode (such as `land` and `abandon`) do not accept `--dry-run`.

## Shared dry-run contract

These guarantees apply to every command's dry-run mode.

### Mutation safety

Dry-run performs no local Git mutations, no remote pushes, and no GitHub write operations.

- Dry-run invoked → does not checkout generated branches, rebase branches, amend commits, create or delete local generated branches, save a stash, or pop a stash.
- Dry-run invoked → does not push any branch to any remote.
- Dry-run invoked → does not create PRs, edit PR title/body/base fields, change PR draft/ready state, merge, or close PRs.

### Plan output

- Dry-run completes successfully → output clearly states that no local Git changes, remote pushes, or GitHub changes were made.

### Validation

Dry-run validates the same inputs and planning decisions that can be checked without mutation, and fails the same validation errors as the corresponding real execution.

### Non-dry-run behavior preservation

- Invoked without `--dry-run` → perform the command's full documented mutations as specified in the command's own spec.

## Submit/export dry-run

### Flag acceptance

- `stack-pr submit --dry-run` with otherwise valid options → execute dry-run behavior instead of real submit/export behavior.
- `stack-pr export --dry-run` with otherwise valid options → execute the same dry-run behavior as `stack-pr submit --dry-run`.

### Plan output

Dry-run prints a human-readable plan describing the submit/export actions that would be performed for the current stack.

- Non-empty stack → output includes each stack entry in stack order; each entry shows the commit title, generated head branch, computed base branch, and whether the associated PR would be created or updated; entries for new PRs show the draft state that would be used; entries requiring stack metadata indicate that metadata would be added during a real submit/export.
- Stack entry already has PR metadata → output identifies the existing PR and indicates that it would be updated rather than created.
- Empty stack → output reports that the stack is empty and reports success without attempting any mutation.
- Dry-run completes successfully → output clearly states that no local Git changes, remote pushes, or GitHub PR changes were made.

### Validation

- `--draft-bitmask` length does not match the stack length, or its characters are not `0` or `1` → report the same validation error as real submit/export.
- Non-empty stack → generated head branches are computed from the configured branch-name template; base branches are computed using the same bottom-to-top stacking rules as real submit/export.
- Tracked files have staged or unstaged changes → fail the existing clean-repository check; changes are not stashed automatically.
- Dry-run planning uses the same submit/export engine selection rule as the corresponding non-dry-run command (see the experimental submit engine gate in [Submit/Export](submit-export.md)); dry-run remains free of local Git mutations, remote pushes, and GitHub PR writes.

### Receipts unavailable

- `--dry-run` and `--receipt <destination>` (other than `off`) are both provided → report an invocation error explaining that operation receipts are only available for real submit/export executions, and perform no mutations. (See [Submit/Export](submit-export.md), "Operation receipts".)

### Native stack dry-run plan

Dry-run describes native Stack reconciliation without performing a GitHub write.

- Native integration enabled and every stack entry already has a PR number → plan reports the classified native action as `create`, `append`, `noop`, `conflict`, `ineligible`, or `unavailable fallback`.
- Native integration enabled and one or more stack entries do not yet have PR numbers → plan reports whether a real submit would prospectively create a Stack or append the new suffix after PR creation, distinguishing that prospective result from an exact membership classification.
- `github.native_stacks = off` → plan reports native integration as disabled or omits the native action consistently; no native Stacks endpoint is called.
- Any native mode → does not invoke REST create, append, or unstack operations, and preserves all existing local Git, remote push, and PR no-mutation guarantees.
- Native integration enabled and dry-run needs existing membership to classify a plan → may perform read-only GitHub API calls; those calls do not modify PR or Stack state.

## Fix dry-run

`--dry-run` reports the planned repair without mutating local Git or GitHub state.

- `bpr fix --pr <number> --dry-run` → load the selected PR and inspect `HEAD`; print the PR URL, PR head branch, local `HEAD` SHA, existing metadata state, and the metadata line it would add or replace; state that no commit was changed.
- `bpr fix --pr <number> --dry-run` → does not amend `HEAD`, does not push to any remote, and does not write to GitHub.
- `bpr fix --pr <number> --dry-run` → run the same read-only advisory stack-readiness inspection as a real fix (see [Fix](fix-command.md), "Advisory stack readiness"); warnings are phrased as dry-run diagnostics.

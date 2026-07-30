---
title: Git operation safety
status: stable
---

# Git operation safety

## Overview

Commands that mutate repository state detect active Git operations before acting. Operation-state paths are resolved by asking Git for the repository context rather than assuming metadata lives in a `.git` directory, so rebase, merge, cherry-pick, and sequencer state is detected correctly from repository subdirectories, linked worktrees, submodules, and repositories with a separate Git directory.

This is an explicit Go-port safety decision that differs from the literal `.git` path checks documented for Python `stack-pr` in `SPEC.md` section 11; the rationale is recorded in [ADR-0005](../adr/0005-resolve-git-operation-state-through-repository-context.md).

## Behavior

### Repository-layout-aware operation paths

- Current directory nested below a repository worktree root, checked without an explicit repository directory → inspect the operation path belonging to that repository.
- Invocation context is a linked Git worktree → inspect the operation path belonging to that worktree.
- Invocation context is a checked-out Git submodule → inspect the submodule's operation path.
- Worktree stores its repository metadata in a separate Git directory → inspect the operation path resolved by Git for that repository.

### Active operation detection

An active operation is reported when Git's resolved metadata contains a rebase, merge, cherry-pick, or sequencer marker already recognized by the port; no active operation is reported when none of those markers exists.

| Marker resolved by Git | Check | Reported state |
|------------------------|-------|----------------|
| `rebase-merge` or `rebase-apply` exists | rebase state | rebase active |
| `MERGE_HEAD` exists | merge state | merge active |
| `CHERRY_PICK_HEAD` exists | cherry-pick state | cherry-pick active |
| `sequencer/todo` exists | cherry-pick or aggregate sequencer state | operation active |
| none of the recognized paths exists | aggregate sequencer state | no operation active |

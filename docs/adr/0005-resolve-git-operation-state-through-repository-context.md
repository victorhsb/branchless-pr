---
kind: adr
id: ADR-0005
title: Resolve Git operation state through repository context, not .git paths
status: accepted
date: 2026-07-29
tags: git, safety, port
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0005: Resolve Git operation state through repository context, not .git paths

## Status

accepted

## Context

Commands that mutate repository state must detect active Git operations (rebase, merge, cherry-pick, sequencer) before acting. Python `stack-pr` detects these by checking literal `.git` paths (SPEC.md section 11), which assumes repository metadata always lives in a `.git` directory at the worktree root. That assumption fails when running from a repository subdirectory, inside a linked worktree, inside a submodule, or in a repository with a separate Git directory — causing mutating commands to miss an in-progress operation and proceed unsafely.

## Decision

Operation-state paths are resolved by asking Git for the repository context (for example via `git rev-parse`) rather than assuming metadata lives in a `.git` directory. The set of recognized markers (`rebase-merge`, `rebase-apply`, `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `sequencer/todo`) and the reported states are unchanged; only path resolution differs. This is an explicit Go-port safety decision that intentionally differs from the literal `.git` path checks documented for Python `stack-pr` in SPEC.md section 11.

## Consequences

Rebase, merge, cherry-pick, and sequencer state is detected correctly from repository subdirectories, linked worktrees, submodules, and repositories with a separate Git directory. The behavioral detection contract is enforced by `internal/git` and its tests; this ADR records why the port diverges from the Python reference. Mutating commands must not reintroduce literal `.git` path checks.

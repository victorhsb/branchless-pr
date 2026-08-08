---
kind: adr
id: ADR-0002
title: internal/shell is the only subprocess wrapper
status: accepted
date: 2026-07-28
tags: architecture, subprocess
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0002: internal/shell is the only subprocess wrapper

## Status

accepted

## Context

Ad-hoc os/exec call sites scattered across packages would make subprocess behavior hard to audit, mock in tests, and keep consistent (error wrapping, dry-run safety).

## Decision

internal/shell is the only package permitted to spawn subprocesses. No production code outside internal/shell may call exec.Command or equivalents; all git and gh invocations flow through typed wrappers in internal/git and internal/pr. Non-spawning uses of os/exec (such as exec.LookPath for PATH checks) are exempt, and _test.go helpers may invoke git directly to build fixture repositories.

## Consequences

There is a single boundary for stubbing subprocesses in tests and for auditing side effects. CONTRIBUTING.md codifies the rule with a documented-exception clause.

## Appendix: 2026-08-08 correction

Date: 2026-08-08

Decision reworded to match the enforced rule: the ban covers spawning subprocesses (exec.Command) in production code; exec.LookPath and _test.go fixture helpers are exempt. The previous absolute phrasing (no os/exec use at all) did not match the codebase.

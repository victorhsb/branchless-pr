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

internal/shell is the only package permitted to invoke subprocesses. No package outside internal/shell may call os/exec directly; all git and gh invocations flow through typed wrappers in internal/git and internal/pr.

## Consequences

There is a single boundary for stubbing subprocesses in tests and for auditing side effects. CONTRIBUTING.md codifies the rule.

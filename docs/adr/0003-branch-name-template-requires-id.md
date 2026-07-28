---
kind: adr
id: ADR-0003
title: Branch name template requires $ID
status: accepted
date: 2026-07-28
tags: branching, config
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0003: Branch name template requires $ID

## Status

accepted

## Context

Generated branch names must map uniquely to stack entries. A template without the $ID variable can produce colliding branch names for different commits in the stack.

## Decision

The branch name template must contain $ID (or gain it implicitly via a /$ID suffix). Validation happens in cli/root.go PersistentPreRunE before any mutation.

## Consequences

Invalid templates fail fast at pre-run with a clear error instead of colliding mid-stack. Template variables are documented in the config spec.

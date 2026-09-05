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

The branch name template must always yield an $ID component. A template that omits $ID gains an implicit /$ID suffix in stack.ParseTemplate; there is no rejection path, so any non-empty template is usable.

## Consequences

Collisions are prevented by construction rather than by pre-run validation errors. The empty template is the only invalid case, and it is reported by `agent diagnose` rather than enforced at pre-run. Template variables are documented in the README configuration section and the generated config template.

## Appendix: 2026-08-08 correction

Date: 2026-08-08

Decision and Consequences rewritten to match the implementation: stack.ParseTemplate appends /$ID implicitly and there is no pre-run validation or rejection path. The previous text described fail-fast validation in PersistentPreRunE, which does not exist.

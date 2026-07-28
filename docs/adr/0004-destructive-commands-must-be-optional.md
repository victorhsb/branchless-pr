---
kind: adr
id: ADR-0004
title: Destructive commands must be optional
status: accepted
date: 2026-07-28
tags: config, safety
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0004: Destructive commands must be optional

## Status

accepted

## Context

Some subcommands perform destructive operations — for example, `land` squash-merges the bottom PR and rebases the rest of the stack. Some teams prohibit such client-side operations entirely and want a guarantee the command cannot be invoked, not merely discouraged.

## Decision

Any subcommand that performs destructive operations must be removable via configuration. When disabled in `.stack-pr.cfg`, the subcommand is not registered at all: it does not appear in help output and cannot be invoked. This currently applies to `land` (`land.style = disable`); any future destructive command must follow the same pattern.

## Consequences

Teams can enforce policies such as "no client-side landing" via repo config. The gating lives in the Cobra command wiring in cli/root.go. Adding a destructive command without a disable switch violates this ADR.

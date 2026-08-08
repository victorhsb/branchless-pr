---
kind: adr
id: ADR-0007
title: The agent subtree is repo-free and side-effect-free by construction
status: accepted
date: 2026-08-08
tags: agent, architecture, llm
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0007: The agent subtree is repo-free and side-effect-free by construction

## Status

accepted

## Context

stack-pr agent commands are consumed by LLM coding agents that may run outside a git repository, without gh installed, and must never mutate user state. Routing them through the normal PersistentPreRunE (repo discovery, gh installation check, config resolution, stashing, clean-tree enforcement) would make them fail or refuse to run in exactly the environments where agents need them most.

## Decision

The agent command subtree short-circuits PersistentPreRunE: no repo discovery, no gh check, no config-path resolution, no stashing, no clean-tree enforcement. agent prompt is fully static and deterministic; agent diagnose is read-only and best-effort, reporting failure modes in a JSON envelope with status ok|warning|blocking|unknown instead of exiting non-zero, with gh-backed checks gated behind an opt-in --online flag.

## Consequences

Agent commands are safe to run unconditionally in any directory, but the pre-run pipeline is bypassed rather than shared: changes to PersistentPreRunE must explicitly consider whether they apply to the agent path. Because diagnose reports failures in-band instead of via exit codes, consumers must parse the JSON envelope rather than rely on process status. Behavioral contracts live in docs/specs/agent-prompt-command.md and docs/specs/agent-diagnose.md.

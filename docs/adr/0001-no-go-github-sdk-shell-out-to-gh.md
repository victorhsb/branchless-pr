---
kind: adr
id: ADR-0001
title: No Go GitHub SDK — shell out to gh
status: accepted
date: 2026-07-28
tags: dependencies, github
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0001: No Go GitHub SDK — shell out to gh

## Status

accepted

## Context

bpr is a Go port of the Python stack-pr CLI, which performs all GitHub operations by shelling out to the gh CLI. Introducing a Go GitHub SDK would add a large dependency, duplicate gh's authentication handling, and diverge from the reference implementation.

## Decision

All GitHub operations are performed by shelling out to the gh CLI, wrapped in internal/pr. No Go GitHub SDK dependency may be added.

## Consequences

gh must be installed on the user's machine (checked in PersistentPreRunE). GitHub API responses are consumed via gh's JSON output flags. AGENTS.md and CONTRIBUTING.md forbid adding a Go GitHub SDK.

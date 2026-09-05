---
kind: adr
id: ADR-0006
title: Stack metadata is encoded in commit messages
status: accepted
date: 2026-08-08
tags: metadata, port, stack
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0006: Stack metadata is encoded in commit messages

## Status

accepted

## Context

The tool must reconstruct the mapping between local commits and GitHub PRs across invocations, machines, and history rewrites, without relying on local state files. Alternatives considered: local state files (lost on clone, stale after rebase), git notes (not pushed or fetched by default), and PR-body-only metadata (cannot drive local discovery before PRs exist and is invisible to git log). The Python reference tool embeds a stack-info: line in each commit message, and the port must remain interoperable with it.

## Decision

Each commit in a submitted stack carries a 'stack-info: PR: <url>, branch: <head>' line in its commit message, and that line is the source of truth linking the commit to exactly one PR and its generated branch. Submit/export appends the line, abandon and land strip it, and fix repairs it. Stack discovery otherwise operates on the commit list in BASE..HEAD; the metadata line associates entries with existing PRs.

## Consequences

Amending metadata rewrites commit hashes, so submit/export rebase downstream stack entries onto amended bases. Commit messages become API surface: the stack-info line format is a compatibility contract with the Python tool and must not change without a migration path. The line grammar is defined in `internal/stack/entry.go`; per-command behavior lives in the submit/export, abandon, land, and fix implementations and their tests.

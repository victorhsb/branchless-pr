---
title: checks report
status: stable
---

# Checks Report

## Overview

`stack-pr checks` is a read-only stack-wide checks report that helps users and agents identify CI failures, optional and required check state, and brief review-attention signals across stacked pull requests. It reports GitHub check and lightweight review-attention state for pull requests represented by the current stack metadata.

## Behavior

### Command and stack discovery

- `stack-pr checks --help` → describe a read-only command for reporting check state across the current stack's pull requests; help includes supported output format and filtering flags.
- Invoked inside a repository with a non-empty stack → discover stack entries using the same base, head, branch-template, and stack metadata rules as other stack inspection commands, and associate fetched check state with the stack entry and pull request it belongs to.
- Stack entry has no pull request metadata → report that entry as missing PR metadata and continue collecting check state for other entries that have PR metadata.

### Read-only behavior

Checks never mutates local Git state, remote branches, commit messages, or GitHub pull request state.

- Invoked while tracked files have staged or unstaged changes → still attempt to produce a checks report; never require the user to clean or stash the worktree first.
- Fetching check or review-attention information from GitHub → use read-only GitHub operations; never create, edit, close, merge, approve, rerun, resolve, dismiss, or delete pull requests, checks, reviews, or comments.
- Runs successfully or with reportable per-PR failures → never check out branches, create branches, delete branches, amend commits, rebase, stash, push, or fetch in a way that mutates repository state.

### Check coverage

The report includes all GitHub checks and status contexts available for each stack pull request head commit, not only required checks.

- Pull request has both required and optional checks → include both; each check indicates whether it is required when that information is available.
- GitHub does not expose whether a check is required → include the check with required state `unknown`; never infer required state from check name alone.
- Head commit has legacy status contexts or non-Actions check providers → include those statuses in the same check collection as GitHub Actions check runs; preserve provider and URL information when available.
- Check is queued, in progress, pending, skipped, cancelled, neutral, successful, or failed → include it with its normalized status and conclusion.

### Stable check identity

- Check is reported → include an `id` field derived from stable semantic fields such as provider, workflow or suite name, and job or check name; the ID is deterministic for the same check source and name.
- GitHub exposes exact identifiers for a check, run, suite, or workflow → include those identifiers in provider-specific fields such as `provider_id`, `run_id`, `check_run_id`, or `workflow`; the semantic `id` remains present.
- One or more checks have failing conclusions → include a failed-check summary; each failed-check summary entry includes the semantic check ID, pull request number, stack entry commit SHA, check name, conclusion, and URL when available.

### Text output (default)

Default output is human-readable Markdown-compatible text grouped by stack entry and pull request.

- Invoked without `--format` → produce text output grouping pull request summaries by stack entry in deterministic stack order; each group identifies the commit title, short SHA, pull request number, pull request URL, head branch, and base branch when known.
- Text output identifies the inspected stack size and pull request coverage, and makes missing PR metadata, unreadable pull requests, and active `--pr` or `--commit` filters visible.
- Pull request has check data → include a compact roll-up with useful check counts such as passing, failing, in-progress, pending, skipped, and unknown where present, plus lightweight comment and review counts when available.
- Multiple checks share the same visible check identity → summarize them as one visible item or count instead of rendering every duplicate check line; the visible state prefers the most actionable state: failed before in-progress, pending, successful, skipped, or unknown.
- Any stack pull request has failed checks → visibly list the failed checks with their semantic check IDs and URLs when available before or within the relevant pull request group.
- `--verbose` with text output → include the summary-first content and render every retained check in deterministic order with semantic check ID, name, status or conclusion, required state when available, and URL when available.
- All stack entries with PR metadata can be read and no checks are available → clearly state that no checks were found; still identify the inspected stack and PR count.
- One or more pull requests cannot be read but at least one stack entry is reportable → include a warning for each unreadable pull request or stack entry and continue rendering available checks from other pull requests.

| Output | Unknown required state |
|--------|------------------------|
| default text | never print `required: unknown` |
| JSON | preserved |
| verbose text detail | preserved |

### JSON output

- `--format json` → stdout contains exactly one JSON object including `schema_version`, `command`, `repository`, `range`, `stack`, `pull_requests`, and `failed_checks` fields, with no ANSI escape sequences, terminal hyperlinks, or human progress logs.
- Pull request entry → include pull request number, URL, head branch, base branch, stack index, commit SHA, short SHA, commit title, status, checks, and lightweight comment summary when available.
- Check entry → include `id`, provider, name, status, conclusion, required state, and URL when available, plus provider-specific identifiers when available.
- Failed checks present → the top-level `failed_checks` array contains one entry per failed check in deterministic stack order; each entry includes enough identity to route follow-up work to the relevant pull request, commit, and check.
- `--format` value other than `text` or `json` → exit non-zero with a clear error message.

### Filtering

Filtering flags reduce output without changing the underlying stack or GitHub state.

| Flag | Behavior |
|------|----------|
| `--failed-only` | include only failed checks and the pull request groups needed to contextualize those failures; still report pull requests or stack entries whose check state could not be read |
| `--required-only` | include only checks known to be required; never include checks whose required state is `false` or `unknown` |
| `--pr <number>` | include only the stack entry associated with that pull request number; report a clear invocation error if no stack entry is associated with it |
| `--commit <sha>` | include only the stack entry whose commit SHA matches the provided full or unambiguous abbreviated SHA; report a clear invocation error if no stack entry matches |

### Lightweight comment summary

The report includes a brief pull-request-level comment and review-attention summary when available, leaving full comment inspection to the `comments` command.

- Pull request has conversation comments, reviews, review comments, or review threads available from GitHub → include counts for those categories when available; never require fetching or rendering full comment thread bodies.
- Comment snippets are included → bound each snippet in count and length; each snippet includes enough context to identify the pull request and source category.
- Comment or review-attention summary indicates detailed inspection may be useful → point the user or agent toward `stack-pr comments` for full comment details; never attempt to render full review-thread trees.

### Error handling

Invocation errors are distinguished from reportable per-stack-entry failures.

| Condition | Behavior |
|-----------|----------|
| GitHub CLI is not installed | exit non-zero with a clear message that `gh` is required |
| GitHub rejects all check queries because the user is not authenticated or authorized | exit non-zero with a clear authentication or authorization message |
| One pull request cannot be read (missing, inaccessible, deleted, or otherwise failing) while other pull requests can be read | include that failure in the report for the relevant stack entry; continue reporting checks for readable pull requests |
| Stack is empty | produce an empty-stack report in the requested format; do not query GitHub for pull request checks |

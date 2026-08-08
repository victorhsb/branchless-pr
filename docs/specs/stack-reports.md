---
title: Stack reports
status: stable
---

# Stack Reports

## Overview

`stack-pr checks` and `stack-pr comments` are read-only, stack-wide reports over the pull requests represented by the current stack metadata:

- `stack-pr checks` reports GitHub check and lightweight review-attention state, helping users and agents identify CI failures, optional and required check state, and brief review-attention signals across stacked pull requests.
- `stack-pr comments` collects review and conversation comments, helping users and agents inspect GitHub pull request feedback across the current stack.

Both commands share a common report contract (stack discovery, read-only guarantees, output formats, filtering, and error handling), specified once below; command-specific behavior follows in per-command sections.

## Shared report contract

### Command and stack discovery

- `stack-pr checks --help` → describe a read-only command for reporting check state across the current stack's pull requests; help includes supported output format and filtering flags.
- `stack-pr comments --help` → describe a read-only command for collecting comments across the current stack's pull requests; help includes the supported output format and filtering flags.
- Invoked inside a repository with a non-empty stack → discover stack entries using the same base, head, branch-template, and stack metadata rules as other stack inspection commands, and associate fetched report data with the stack entry and pull request it belongs to.
- Stack entry has no pull request metadata → report that entry as missing PR metadata and continue collecting report data for other entries that have PR metadata.

### Read-only behavior

Neither report command ever mutates local Git state, remote branches, commit messages, or GitHub pull request state.

- Invoked while tracked files have staged or unstaged changes → still attempt to produce the report; never require the user to clean or stash the worktree first.
- Fetching check, comment, or review-attention information from GitHub → use read-only GitHub operations; never create, edit, close, merge, approve, rerun, resolve, dismiss, or delete pull requests, checks, reviews, or comments.
- Runs successfully or with reportable per-PR failures → never check out branches, create branches, delete branches, amend commits, rebase, stash, push, or fetch in a way that mutates repository state.

### Text output (default)

Default output is human-readable Markdown-compatible text grouped by stack entry and pull request.

- Invoked without `--format` → produce text output grouping per-PR report data by stack entry in deterministic stack order; each group identifies the commit title, short SHA, pull request number, pull request URL, head branch, and base branch when known.
- All stack entries with PR metadata can be read and no matching report data exists → clearly state that nothing was found (no checks for `checks`, no matching comments for `comments`); still identify the inspected stack and PR count.
- One or more pull requests cannot be read but at least one stack entry is reportable → include a warning for each unreadable pull request or stack entry and continue rendering available data from other pull requests.
- Report data authored on GitHub (comment and review bodies, check names, pull request titles, author logins) contains terminal control characters → strip them from text output before printing, so remote-authored content cannot emit escape sequences that repaint the terminal, move the cursor, retitle the window, or forge tool output. Tab and line feed are preserved; all other C0 controls, DEL, and C1 controls are removed. Carriage return is removed rather than preserved, because a bare carriage return returns the cursor to the start of the line and lets remote text overwrite already-printed output; CRLF input is thereby normalized to line feed. The legible content of the text is otherwise unchanged.

### JSON output

- `--format json` → stdout contains exactly one JSON object including `schema_version`, `command`, `repository`, `range`, `stack`, and `pull_requests` fields, with no ANSI escape sequences, terminal hyperlinks, or human progress logs.
- `--format` value other than `text` or `json` → exit non-zero with a clear error message.

### Error handling

Invocation errors are distinguished from reportable per-stack-entry failures.

| Condition | Behavior |
|-----------|----------|
| GitHub CLI is not installed | exit non-zero with a clear message that `gh` is required |
| GitHub rejects all queries because the user is not authenticated or authorized | exit non-zero with a clear authentication or authorization message |
| One pull request cannot be read (missing, inaccessible, deleted, or otherwise failing) while other pull requests can be read | include that failure in the report for the relevant stack entry; continue reporting data for readable pull requests |
| Stack is empty | produce an empty-stack report in the requested format; do not query GitHub |

## Checks report (`stack-pr checks`)

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

### Checks text output

- Text output identifies the inspected stack size and pull request coverage, and makes missing PR metadata, unreadable pull requests, and active `--pr` or `--commit` filters visible.
- Pull request has check data → include a compact roll-up with useful check counts such as passing, failing, in-progress, pending, skipped, and unknown where present, plus lightweight comment and review counts when available.
- Multiple checks share the same visible check identity → summarize them as one visible item or count instead of rendering every duplicate check line; the visible state prefers the most actionable state: failed before in-progress, pending, successful, skipped, or unknown.
- Any stack pull request has failed checks → visibly list the failed checks with their semantic check IDs and URLs when available before or within the relevant pull request group.
- `--verbose` with text output → include the summary-first content and render every retained check in deterministic order with semantic check ID, name, status or conclusion, required state when available, and URL when available.

| Output | Unknown required state |
|--------|------------------------|
| default text | never print `required: unknown` |
| JSON | preserved |
| verbose text detail | preserved |

### Checks JSON output

- `--format json` → the JSON object additionally includes a `failed_checks` field.
- Pull request entry → include pull request number, URL, head branch, base branch, stack index, commit SHA, short SHA, commit title, status, checks, and lightweight comment summary when available.
- Check entry → include `id`, provider, name, status, conclusion, required state, and URL when available, plus provider-specific identifiers when available.
- Failed checks present → the top-level `failed_checks` array contains one entry per failed check in deterministic stack order; each entry includes enough identity to route follow-up work to the relevant pull request, commit, and check.

### Checks filtering

Filtering flags reduce output without changing the underlying stack or GitHub state.

| Flag | Behavior |
|------|----------|
| `--failed-only` | include only failed checks and the pull request groups needed to contextualize those failures; still report pull requests or stack entries whose check state could not be read |
| `--required-only` | include only checks known to be required; never include checks whose required state is `false` or `unknown` |
| `--pr <number>` | include only the stack entry associated with that pull request number; report a clear invocation error if no stack entry is associated with it |
| `--commit <sha>` | include only the stack entry whose commit SHA matches the provided full or unambiguous abbreviated SHA; report a clear invocation error if no stack entry matches |

### Lightweight comment summary

The checks report includes a brief pull-request-level comment and review-attention summary when available, leaving full comment inspection to the `comments` command.

- Pull request has conversation comments, reviews, review comments, or review threads available from GitHub → include counts for those categories when available; never require fetching or rendering full comment thread bodies.
- Comment snippets are included → bound each snippet in count and length; each snippet includes enough context to identify the pull request and source category.
- Comment or review-attention summary indicates detailed inspection may be useful → point the user or agent toward `stack-pr comments` for full comment details; never attempt to render full review-thread trees.

## Comments report (`stack-pr comments`)

### Comment sources

The report includes GitHub pull request feedback from all sources available through `gh`:

| Kind | Source | Fields |
|------|--------|--------|
| `conversation` | issue-style conversation comments | author, body, creation time, update time when available, URL when available, owning pull request |
| `review` | submitted reviews | author, body when present, submitted time when available, state when available, URL when available, owning pull request |
| `review_thread` | review threads | resolution state when available, file path when available, line or range context when available, URL when available, comments or replies in chronological order when available |
| `review_comment` | review comments exposed separately from review threads | author, body, creation time, update time when available, URL when available, path when available, line context when available |

### Comments JSON output

- Stack entry → include commit SHA, short SHA, title, stack index, head branch, base branch, PR URL when known, PR number when known, and a status indicating whether comments were fetched, missing, empty, or failed.
- Comment, review, or thread item → include a stable `id` when GitHub provides one, `kind`, owning PR number, author, body, URL when available, timestamps when available, and optional location or resolution fields when available.

### Comments filtering

Filtering flags reduce output without changing the underlying stack.

| Flag | Behavior |
|------|----------|
| `--unresolved-only` | include only comments or threads that GitHub identifies as unresolved or otherwise requiring attention; never guess unresolved state for comment kinds that do not expose resolution status |
| `--kind <kinds>` | include only the requested comment kinds; reject unsupported kind values with a clear error |
| `--author <login>` | include only matching comments, reviews, or threads authored by that GitHub login; pull request groups with no matching items are shown as empty rather than omitted in JSON output |

### Ignored comment authors

`.stack-pr.cfg` can exclude feedback authored by configured GitHub logins from comments report output.

| `comments.ignore_authors` | Behavior |
|---------------------------|----------|
| non-empty list (e.g. `ci-bot,release-bot`), no author override | exclude comments, reviews, review comments, review-thread items, and review-thread replies authored by the listed logins |
| key omitted | exclude no feedback because of ignored-author configuration |
| empty value (`comments.ignore_authors =`) | exclude no feedback because of ignored-author configuration |

- Matching is case-insensitive: `comments.ignore_authors = CI-Bot` treats feedback authored by `ci-bot` as authored by an ignored login.
- Ignored-author filtering is applied before positive author filtering: `comments.ignore_authors = ci-bot` with `--author ci-bot` → the report contains no `ci-bot` feedback.
- Review thread contains replies from both an ignored and a non-ignored author → exclude the ignored-author replies and retain the thread with the remaining non-ignored replies when the thread still has reportable feedback.

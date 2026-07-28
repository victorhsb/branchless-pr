---
title: comments report
status: stable
---

# Comments Report

## Overview

`stack-pr comments` is a read-only report that collects review and conversation comments for pull requests represented by the current stack metadata, helping users and agents inspect GitHub pull request feedback across the current stack.

## Behavior

### Command and stack discovery

- `stack-pr comments --help` → describe a read-only command for collecting comments across the current stack's pull requests; help includes the supported output format and filtering flags.
- Invoked inside a repository with a non-empty stack → discover stack entries using the same base, head, branch-template, and stack metadata rules as other stack inspection commands, and associate fetched comments with the stack entry and pull request they belong to.
- Stack entry has no pull request metadata → report that entry as missing PR metadata and continue collecting comments for other entries that have PR metadata.

### Read-only behavior

Comments never mutates local Git state, remote branches, commit messages, or GitHub pull request state.

- Invoked while tracked files have staged or unstaged changes → still attempt to produce a comments report; never require the user to clean or stash the worktree first.
- Fetching comment information from GitHub → use read-only GitHub operations; never create, edit, close, merge, mark ready, resolve, or delete pull requests or comments.
- Runs successfully or with reportable per-PR failures → never check out branches, create branches, delete branches, amend commits, rebase, stash, push, or fetch in a way that mutates repository state.

### Comment sources

The report includes GitHub pull request feedback from all sources available through `gh`:

| Kind | Source | Fields |
|------|--------|--------|
| `conversation` | issue-style conversation comments | author, body, creation time, update time when available, URL when available, owning pull request |
| `review` | submitted reviews | author, body when present, submitted time when available, state when available, URL when available, owning pull request |
| `review_thread` | review threads | resolution state when available, file path when available, line or range context when available, URL when available, comments or replies in chronological order when available |
| `review_comment` | review comments exposed separately from review threads | author, body, creation time, update time when available, URL when available, path when available, line context when available |

### Text output (default)

Default output is human-readable Markdown-compatible text grouped by stack entry and pull request.

- Invoked without `--format` → produce text output grouping comments by stack entry in deterministic stack order; each group identifies the commit title, short SHA, pull request number, pull request URL, head branch, and base branch when known.
- All stack entries with PR metadata can be read and no matching comments exist → clearly state that no matching comments were found; still identify the inspected stack and PR count.
- One or more pull requests cannot be read but at least one stack entry is reportable → include a warning for each unreadable pull request or stack entry and continue rendering available comments from other pull requests.

### JSON output

- `--format json` → stdout contains exactly one JSON object including `schema_version`, `command`, `repository`, `range`, `stack`, and `pull_requests` fields, with no ANSI escape sequences, terminal hyperlinks, or human progress logs.
- Stack entry → include commit SHA, short SHA, title, stack index, head branch, base branch, PR URL when known, PR number when known, and a status indicating whether comments were fetched, missing, empty, or failed.
- Comment, review, or thread item → include a stable `id` when GitHub provides one, `kind`, owning PR number, author, body, URL when available, timestamps when available, and optional location or resolution fields when available.
- `--format` value other than `text` or `json` → exit non-zero with a clear error message.

### Filtering

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

### Error handling

Invocation errors are distinguished from reportable per-stack-entry failures.

| Condition | Behavior |
|-----------|----------|
| GitHub CLI is not installed | exit non-zero with a clear message that `gh` is required |
| GitHub rejects all comment queries because the user is not authenticated or authorized | exit non-zero with a clear authentication or authorization message |
| One pull request cannot be read (missing, inaccessible, deleted, or otherwise failing) while other pull requests can be read | include that failure in the report for the relevant stack entry; continue reporting comments for readable pull requests |
| Stack is empty | produce an empty-stack report in the requested format; do not query GitHub for pull request comments |

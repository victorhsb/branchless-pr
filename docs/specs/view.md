---
title: View
status: stable
---

# View

## Overview

`stack-pr view` inspects the current stack of commits. It is read-only: it does not require a clean working tree and does not modify local or remote state. It discovers the stack, resolves PR metadata and branch information, and renders the stack as ANSI-colored text (default) or structured JSON (`--format json`).

## Behavior

### Base fast-forward warning

View never modifies state. When the base could be fast-forwarded automatically, it warns and stops instead of loading the stack.

- Local base is an ancestor of `REMOTE/TARGET`, `REMOTE/TARGET` is an ancestor of `HEAD`, and the hashes differ → print a warning that the local base is behind, suggest `git rebase REMOTE/TARGET base` and `git checkout <original_branch>` as follow-ups, and do not load or print the stack.
- Base not behind in this auto-updatable way → load the stack normally.

### Stack discovery

- The stack is loaded from commits in `base..head`, ordered oldest-to-newest internally.
- Discovered stack is empty → print `Empty stack!` and return without further action.

### Head branch resolution

Entries missing a head branch in their metadata get one by scanning remote refs — never by creating branches or pushing.

- Entry lacks a metadata head branch → scan remote refs for a matching branch and assign it to the entry; do not create a branch or push.
- Entry already has a metadata head branch → use it without scanning the remote.

### Base branch assignment

- Bottom entry → base branch is the remote target branch.
- Every entry above the bottom → base branch is the previous entry's head branch.

### Text output (default)

Without `--format`, output is ANSI-colored, Markdown-compatible text with terminal hyperlinks, grouped by stack entry newest-to-oldest.

- Each stack line: `* <short-sha> (#<pr-number or no PR>, '<head>' -> '<base>'): <commit title>`
- Output contains no command banners such as `VIEW` and no generic success markers such as `SUCCESS!`.

### JSON output

`--format json` produces a single JSON array ordered newest-to-oldest, with no ANSI escapes, terminal hyperlinks, progress logs, or extra stdout text. Each element is a flat object:

| Field | Content |
|-------|---------|
| `commit` | full commit hash |
| `short_sha` | abbreviated commit hash |
| `title` | first line of the commit message |
| `author` | full author string (name and email) |
| `author_name` | author name |
| `author_email` | author email |
| `pr_url` | pull request URL, `""` if none |
| `pr_number` | pull request number, `0` if none |
| `head_branch` | branch name for this stack entry |
| `base_branch` | base branch for this stack entry |

- `--format` with a value other than `text` or `json` → exit with a clear error message.

### Post-view tips

After printing, guidance depends on metadata completeness:

- Every entry has PR, head, and base metadata → indicate the stack is ready to land; display update and land commands.
- Any entry lacks PR, head, or base metadata → indicate the stack cannot be landed yet; display the export (submit) command.

### Native stack inspection

When native integration is enabled, view also inspects GitHub native Stack membership — using read operations only (never create, append, unstack, edit, rebase, or push).

- Every submitted local PR belongs to one native Stack in the same bottom-to-top order → text output identifies the native Stack number, size, and ultimate base branch; stack-entry lines remain newest-to-oldest.
- Submitted local PRs are not native Stack members → state that membership is absent; in `auto` mode this is not drift; in `required` mode, an eligible multi-PR stack reports the required-membership error.
- Remote membership is reordered, split, mixed, or contains PRs outside the local sequence → report a native membership drift warning or error identifying enough local and remote PR ordering information for manual resolution.
- `github.native_stacks = off` → preserve existing text and JSON behavior; make no native membership queries.

## 1. CLI Surface

- [ ] 1.1 Add a top-level `comments` Cobra command and register it in `internal/cli/root.go`.
- [ ] 1.2 Add flags for `--format text|json`, `--unresolved-only`, `--kind`, and `--author`.
- [ ] 1.3 Exempt `comments` from the clean-worktree requirement while keeping existing repo, config, target, and `gh` preflight behavior.
- [ ] 1.4 Reject unsupported output formats and unsupported comment kinds with clear invocation errors.

## 2. Stack Comment Model

- [ ] 2.1 Define internal report types for the command metadata, repository/range context, stack entries, per-PR statuses, warnings/errors, and normalized comment items.
- [ ] 2.2 Include stack entry fields needed by both text and JSON output: stack index, commit SHA, short SHA, title, head branch, base branch, PR URL, and PR number.
- [ ] 2.3 Define normalized comment fields for kind, GitHub ID, author, body, URL, timestamps, PR number, optional location, optional resolution state, and replies.
- [ ] 2.4 Ensure missing PR metadata is represented as a reportable per-entry status rather than a global failure.

## 3. GitHub Read Helpers

- [ ] 3.1 Add read-only helpers that call `gh` through `internal/shell` to retrieve conversation comments and reviews.
- [ ] 3.2 Add a read-only GraphQL helper for review threads and resolution state where `gh pr view --json` is insufficient.
- [ ] 3.3 Normalize all fetched GitHub responses into the shared report model without leaking raw GitHub schemas to command output.
- [ ] 3.4 Distinguish global authentication/authorization failures from individual PR read failures.
- [ ] 3.5 Avoid all GitHub write commands and all local Git mutation commands in the comments path.

## 4. Report Assembly and Filtering

- [ ] 4.1 Discover the stack using the same metadata, head assignment, and base assignment flow as `view`.
- [ ] 4.2 Fetch comments for each stack entry with PR metadata and attach results to the corresponding stack entry.
- [ ] 4.3 Apply `--kind` filtering while preserving deterministic stack and comment ordering.
- [ ] 4.4 Apply `--author` filtering to comments, reviews, threads, and replies where author data is available.
- [ ] 4.5 Apply `--unresolved-only` only to items with known unresolved or attention-required state and avoid guessing for unsupported kinds.

## 5. Output Rendering

- [ ] 5.1 Implement Markdown-compatible text output grouped by stack entry and pull request.
- [ ] 5.2 Render empty-comment, empty-stack, missing-PR, and per-PR failure cases clearly in text output.
- [ ] 5.3 Implement JSON output as one parseable object with `schema_version`, `command`, `repository`, `range`, `stack`, and `pull_requests`.
- [ ] 5.4 Ensure JSON output contains no ANSI escapes, terminal hyperlinks, progress logs, or extra stdout text.

## 6. Tests and Documentation

- [ ] 6.1 Add CLI tests for command registration, flag parsing, unsupported formats, unsupported kinds, and dirty-worktree allowance.
- [ ] 6.2 Add unit tests for GitHub response parsing and normalization using fixtures for conversation comments, reviews, review comments, and review threads.
- [ ] 6.3 Add report assembly tests for missing PR metadata, empty stack, empty comments, partial per-PR failures, and global authentication failure.
- [ ] 6.4 Add text output golden or focused assertions for grouping, warnings, and unresolved markers.
- [ ] 6.5 Add JSON contract tests that parse stdout and assert stable fields and deterministic ordering.
- [ ] 6.6 Update `SPEC.md` and user-facing command documentation to describe `stack-pr comments`, output formats, filters, and read-only behavior.
- [ ] 6.7 Run `go test ./...`, `go vet ./...`, and gofmt/fmt-check equivalents before marking the change complete.

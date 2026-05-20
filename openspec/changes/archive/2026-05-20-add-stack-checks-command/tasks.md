## 1. Command Surface

- [x] 1.1 Add a top-level `checks` Cobra command and register it in `internal/cli/root.go`.
- [x] 1.2 Add flags for `--format text|json`, `--failed-only`, `--required-only`, `--pr`, and `--commit`.
- [x] 1.3 Exempt `checks` from the clean-worktree requirement while keeping repo, config, target, and `gh` preflight behavior.
- [x] 1.4 Reject unsupported formats and invalid or unmatched `--pr` / `--commit` filters with clear invocation errors.

## 2. Report Model

- [x] 2.1 Define versioned report types for command metadata, repository/range context, stack entries, pull requests, checks, failed-check summaries, comment summaries, warnings, and errors.
- [x] 2.2 Define normalized check fields including semantic ID, provider, provider IDs, workflow or suite name, check name, status, conclusion, required state, timestamps, and URL.
- [x] 2.3 Define failed-check summary fields including check ID, pull request number, commit SHA, check name, conclusion, and URL.
- [x] 2.4 Define lightweight comment summary fields for category counts, bounded snippets, and a command hint for full `stack-pr comments` inspection.
- [x] 2.5 Represent missing PR metadata and per-PR read failures as reportable entry statuses rather than losing the rest of the stack report.

## 3. GitHub Read Helpers

- [x] 3.1 Add read-only helpers that call `gh` through `internal/shell` to retrieve check runs and status contexts for a pull request head commit.
- [x] 3.2 Add GraphQL or `gh pr view --json` helpers as needed to retrieve workflow names, provider IDs, URLs, and required-check state when available.
- [x] 3.3 Normalize GitHub Actions check runs, legacy status contexts, and third-party checks into the shared report model.
- [x] 3.4 Build deterministic semantic check IDs from provider, workflow or suite, and job or check name.
- [x] 3.5 Preserve exact provider identifiers such as check run IDs, run IDs, workflow names, and URLs when GitHub exposes them.
- [x] 3.6 Add bounded read-only helpers for comment/review attention counts and optional short snippets without rendering full comment threads.
- [x] 3.7 Distinguish global authentication/authorization failures from individual PR or check read failures.

## 4. Report Assembly

- [x] 4.1 Discover the stack using the same metadata, head assignment, and base assignment flow as `view`.
- [x] 4.2 Apply `--pr` and `--commit` filters after stack discovery and before GitHub reads.
- [x] 4.3 Fetch checks for each stack entry with PR metadata and attach results to the corresponding pull request entry.
- [x] 4.4 Include all checks by default, including optional checks and checks with unknown required state.
- [x] 4.5 Apply `--required-only` only to checks known to be required.
- [x] 4.6 Apply `--failed-only` while retaining enough pull request and stack context for each failed check.
- [x] 4.7 Build the top-level failed-check summary in deterministic stack order.
- [x] 4.8 Attach lightweight comment summaries when available and include a full-inspection hint for `stack-pr comments`.

## 5. Output Rendering

- [x] 5.1 Implement Markdown-compatible text output grouped by stack entry and pull request.
- [x] 5.2 Render failed checks prominently in text output with semantic IDs and URLs when available.
- [x] 5.3 Render empty-stack, empty-checks, missing-PR, per-PR failure, and global authentication failure cases clearly in text output.
- [x] 5.4 Implement JSON output as one parseable object with `schema_version`, `command`, `repository`, `range`, `stack`, `pull_requests`, and `failed_checks`.
- [x] 5.5 Ensure JSON output contains no ANSI escapes, terminal hyperlinks, progress logs, or extra stdout text.

## 6. Tests

- [x] 6.1 Add CLI tests for command registration, flag parsing, unsupported formats, dirty-worktree allowance, and invalid filter errors.
- [x] 6.2 Add unit tests for GitHub check response parsing and normalization using fixtures for GitHub Actions check runs, legacy status contexts, optional checks, required checks, pending checks, skipped checks, and failed checks.
- [x] 6.3 Add tests proving semantic check IDs are deterministic and provider IDs are preserved when available.
- [x] 6.4 Add report assembly tests for missing PR metadata, empty stack, empty checks, partial per-PR failures, global authentication failure, all-check default behavior, `--required-only`, and `--failed-only`.
- [x] 6.5 Add tests for lightweight comment summary counts, bounded snippets, and full-inspection hints without requiring full comment rendering.
- [x] 6.6 Add text output assertions for failed-check prominence, warnings, and comment summary hints.
- [x] 6.7 Add JSON contract tests that parse stdout and assert stable fields, failed-check summary contents, and deterministic ordering.

## 7. Documentation and Verification

- [x] 7.1 Update README command documentation for `stack-pr checks`, output formats, filters, all-check default behavior, and lightweight comment summaries.
- [x] 7.2 Update `SPEC.md` so the port specification agrees with the implemented checks behavior.
- [x] 7.3 Update command help text for check identity, required-check filtering, and comment-summary boundaries.
- [x] 7.4 Run `make fmt-check`, `make vet`, `make test`, and `make build`.

## Why

CI bots and automation accounts can add high-volume pull request comments that obscure human review feedback in `stack-pr comments`. Projects need a repository-local way to hide known noisy authors by default while preserving the command's read-only behavior and explicit filtering controls.

## What Changes

- Add `.stack-pr.cfg` configuration for ignored comment authors used by `stack-pr comments` / `bpr comments`.
- Exclude comments, reviews, review comments, and review-thread replies authored by ignored logins from text and JSON comments reports.
- Keep explicit `--author <login>` filtering deterministic when ignore configuration is present.
- Document the effective filter behavior and default to no ignored authors when configuration is absent.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `stack-comments-report`: Adds configuration-driven author exclusion for comments report output.

## Impact

- Affected command: `stack-pr comments` and the equivalent `bpr comments` alias.
- Affected configuration: `.stack-pr.cfg`, likely under a comments-related key such as `comments.ignore_authors`.
- Affected packages: `internal/config` for parsing, `internal/cli` for resolving effective comments options, and comment filtering/rendering code in `internal/pr` or `internal/cli`.
- Affected docs/specs: `SPEC.md`, OpenSpec `stack-comments-report`, command help, and README/config examples if present.

# Changelog

## v1.3.2 – 2026-05-18

- Fixed `git rev-list --header` parsing to split commits on NUL bytes (`0x00`) instead of
  scanning for 40-character SHA lines. Multi-commit stacks were previously truncated to a
  single commit in `stack.Discover` and in `stack-pr agent diagnose`.
- Removed command banners, trailing `SUCCESS!` markers, and Cobra error/usage preambles from primary CLI command output.

## v1.3.0 - 2026-05-18

- Added `stack-pr agent diagnose`, a read-only, best-effort diagnostic command
  for agents and humans, with Markdown/JSON output, offline-by-default checks,
  and safe next-action recommendations.

## v1.2.0 - 2026-05-17

- Added `stack-pr agent prompt`, a side-effect-free command that emits static,
  versioned guidance for LLM-agent consumption in text or JSON format.

## v1.1.1 - 2026-05-17

- Fixed `stack-pr submit` / `export` aborting after creating a PR when `gh pr create` output was not captured, preventing commit metadata updates.

## v1.1.0 - 2026-05-17

- Added `--dry-run` support to `stack-pr submit` / `export`.
- Added machine-readable JSON output for `stack-pr view` via `--format json`.

## v1.0.2 – 2026-05-15

- Default `--head` to the top commit of the current git-branchless stack when
  available, so submitting from a middle commit includes upward descendants.

## v1.0.1 – 2026-05-14

- Fixed default `--head` resolution so base deduction uses `HEAD` when no head
  revision is supplied.

## v1.0.0 – 2026-05-14

- Initial Go port of the Python `stack-pr` tool.
- Implemented `submit` / `export`, `view`, `land`, `abandon`, and `config` commands.
- Replicated INI configuration, branch-name templating, ANSI/hyperlink output,
  PR cross-linking, draft bitmask, stash flow, and verification against
  `gh pr view`.
- Added `--version` flag with `git describe` build-time injection (`internal/cli/version.go`).

## Historical context (Python release notes)

For changes to the original Python tool prior to this port, see the
[modular/stack-pr CHANGELOG](https://github.com/modular/stack-pr/blob/main/CHANGELOG.md).

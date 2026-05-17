# Changelog

## v1.1.0 - 2026-05-17

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

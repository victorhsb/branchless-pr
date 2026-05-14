# Changelog

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

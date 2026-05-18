## Why

LLM agents and AI coding assistants increasingly drive `stack-pr` invocations, but they have no canonical, in-tree guidance on how to use it safely. Today, agents must guess command semantics from README snippets, often missing side-effect boundaries (e.g., the difference between `view`, `submit --dry-run`, `submit`, `land`, and `abandon`). Shipping a built-in `agent prompt` command gives agents a deterministic, versioned source of usage rules that ships with the binary itself.

## What Changes

- Add a new top-level `agent` command group whose purpose is producing agent-facing artifacts (no repo mutation, no GitHub calls).
- Add the `agent prompt [topic]` subcommand that prints static prompt fragments designed to teach an LLM agent how to use `stack-pr`.
- Support the following topics as positional arguments: `overview`, `view`, `submit`, `land`, `abandon`, `recovery`, `all`. Default (no argument) renders the `all` pack.
- Add a `--format` flag with values `text` (markdown, default) and `json` (machine-readable).
- Bypass the standard repo / `gh` preflight checks that other `stack-pr` subcommands perform — `agent prompt` must run anywhere, including outside a git repo and without `gh` authentication.
- JSON output carries a versioned `id` field (e.g., `stack-pr.prompt.submit.v1`) and structured `commands` entries with side-effect metadata, so agents can reason about safety programmatically.

## Capabilities

### New Capabilities

- `agent-prompt-command`: Provides a built-in, deterministic, repo-independent way to emit LLM-agent guidance for using `stack-pr`, in either markdown or structured JSON.

### Modified Capabilities

<!-- No existing spec behavior is changing; this is a pure addition. -->

## Impact

- `internal/cli/`: Add an `agent` parent command and an `agent prompt` subcommand. The new command group must opt out of any shared repo/gh preflight middleware that other subcommands rely on.
- `internal/cli/root.go`: Register the new `agent` command group on the root command.
- New shared static command-metadata layer (e.g., `internal/agent/` or `internal/cli/agent/`) describing each `stack-pr` command's purpose, side effects, and safety constraints, so future siblings (e.g., `agent diagnose`) reuse the same source of truth.
- Tests: unit tests for the prompt renderer (text and JSON) and CLI-level tests confirming the command works without a git repo and without `gh` configured.
- Documentation: README mention of the new command for human users.

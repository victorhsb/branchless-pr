## Why

Agents (and humans) driving `stack-pr` often need to inspect a repository's situation before deciding which command to run next. Normal `stack-pr` commands fail early on issues such as a dirty working tree, missing PR metadata, or an unresolved base/head, which makes them poor tools for orientation. A read-only, best-effort diagnostic surface is needed so that an agent can parse a stable description of repository, stack, and remote state and pick a safe next action — including knowing when *no* action is safe without explicit user confirmation.

## What Changes

- Introduce a new `stack-pr agent diagnose` subcommand that inspects the current repository and emits a structured report of repository, stack, and check state.
- Add a `--format` flag with values `text` (default; Markdown) and `json`.
- Add an `--online` flag (default false) that gates any GitHub network calls; without it, `diagnose` is local-only.
- Require `diagnose` to be read-only: no writes, no pushes, no commits, no stashes, no remote mutations.
- Require degraded-mode behavior: individual checks SHALL report a `status` (`ok` / `warning` / `blocking` / `unknown`) rather than aborting the whole command on failure.
- Require the command to always exit with code `0` for any reportable outcome so agents can reliably parse its output; the `status` field on checks (and the top-level `status`) carries severity.
- Define a stable, versioned JSON output schema (with an explicit schema version field).
- Define a recommendation contract: every recommendation includes `command`, `reason`, `side_effects`, and `requires_confirmation`. `land` is never an outright recommendation; it is only ever surfaced as a conservative "potential next action" requiring explicit confirmation.

## Capabilities

### New Capabilities

- `agent-diagnose`: Read-only, best-effort diagnosis of the current repository, stack, and remote state, with a stable JSON schema and an agent-oriented recommendation contract.

### Modified Capabilities

<!-- No existing specs are being modified. The `agent` parent command group is introduced by a sibling change and is treated as a prerequisite, not a modification owned here. -->

## Impact

- Depends on the `agent` Cobra command group introduced by the sibling `agent-prompt-command` change. This change adds the `diagnose` subcommand under that group but does not own the group itself.
- New CLI surface: `stack-pr agent diagnose [--format text|json] [--online]`.
- New internal package(s) for diagnosis: repository introspection, stack inspection (reusing existing stack discovery and metadata helpers in `internal/stack` where possible), check runners, and recommendation logic.
- Anticipated reuse of a shared static command-metadata layer with `agent prompt` so that recommendation safety metadata (`side_effects`, `requires_confirmation`) stays consistent between agent subcommands. Concrete shape is left to the implementation; the spec only constrains the externally observable JSON.
- `README.md` and command help text: document `agent diagnose`, its flags, and the JSON schema version.
- Tests: add coverage for flag parsing, degraded-mode behavior on each check, JSON schema stability, recommendation decision tree, and the "exit 0 even when blocking" invariant.

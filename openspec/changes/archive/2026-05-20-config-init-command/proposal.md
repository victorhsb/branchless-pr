## Why

The `stack-pr` tool requires a `.stack-pr.cfg` file to configure repository-specific defaults, but users must currently create this file by hand by reading documentation. A `config init` command would scaffold a starter config file with sensible defaults, significantly lowering onboarding friction.

## What Changes

- Add a new `stack-pr config init` subcommand that generates a `.stack-pr.cfg` file in the repository root.
- The generated file includes a commented example with default values drawn from the tool's built-in defaults (e.g., `[common]` branch-name-template, `[repo]` remote/target sections, `[land]` style options).
- If a `.stack-pr.cfg` already exists, the command warns and exits with a descriptive error.
- **Non-breaking**: purely additive; no existing commands or behavior change.

## Capabilities

### New Capabilities

- `config-init-command`: Generate a starter `.stack-pr.cfg` file via CLI, with commented defaults and guard against overwriting existing files.

### Modified Capabilities

<!-- None. This change is purely additive and does not alter existing spec-level behavior. -->

## Impact

- **Affected packages**: `internal/cli/config.go` (new subcommand), `internal/config/` (optional: helper for writing default INI structure), `SPEC.md` (document new command behavior).
- **No new dependencies**.

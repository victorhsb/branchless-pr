---
title: agent prompt
status: stable
---

# Agent prompt

## Overview

`stack-pr agent prompt` emits deterministic, static guidance for LLM agents on how to use `stack-pr`. The output is fully side-effect-free: it does not depend on the contents of any git repository, the current working directory, network availability, or `gh` authentication state.

## Behavior

### Agent command group

The top-level `agent` command group is reserved for commands producing agent-facing artifacts. Subcommands under `agent` never mutate the git repository, never contact GitHub, and never require a git repository or `gh` authentication to run.

- `stack-pr agent --help` → exit successfully and list `prompt` as an available subcommand.
- Any `stack-pr agent` subcommand run from a directory that is not inside a git repository → run normally without emitting a "not a git repository" error.
- Any `stack-pr agent` subcommand run on a system where `gh` is not installed or not authenticated → run normally without emitting a `gh`-auth error.

### Prompt subcommand

- `stack-pr agent prompt` with no positional topic argument → exit successfully; output contains guidance for all supported topics (`overview`, `view`, `submit`, `land`, `abandon`, `fix`, `recovery`).
- `stack-pr agent prompt <topic>` invoked twice with identical arguments on the same binary build → both invocations produce byte-identical output.
- Run from a directory that is not inside a git working tree → exit successfully and emit the prompt content.
- Run on a system where `gh` is not installed or not authenticated → exit successfully and emit the prompt content.

### Supported topics

The optional positional topic argument accepts exactly: `overview`, `view`, `submit`, `land`, `abandon`, `fix`, `recovery`, `all`.

- `stack-pr agent prompt fix` → output contains guidance scoped to the `fix` command flow and does not contain the guidance bodies of unrelated topics such as `submit` or `recovery`.
- `stack-pr agent prompt all` → output contains guidance for every other supported topic in a canonical, stable order: `overview`, `view`, `submit`, `land`, `abandon`, `fix`, `recovery`.
- `stack-pr agent prompt <unknown>` with a value not in the allowed list → exit non-zero with a clear error message naming the allowed topics.

### Output format flag

`--format` accepts `text` and `json`; the default is `text`. `text` produces human-readable markdown; `json` produces machine-readable JSON suitable for consumption by an LLM agent.

- No `--format` → output is markdown text that includes a heading for the topic.
- `--format json` for a single topic → output is valid JSON, not markdown, with a top-level `id` field whose value is a versioned identifier of the form `stack-pr.prompt.submit.v<N>` where `<N>` is a positive integer, and a top-level `commands` array.
- `--format json` for `all` (or `--format json` with no positional argument) → output is a valid JSON array containing one object per supported non-`all` topic in canonical order.
- `--format` with a value other than `text` or `json` → exit non-zero with a clear error message.

### Side-effect metadata in output

Both text and JSON outputs clearly communicate which `stack-pr` commands have side effects and which do not, so an LLM agent can decide whether to ask for user confirmation before invoking them.

- `--format json` → every element in the `commands` array contains a boolean `side_effects` field; read-only commands such as `stack-pr view` and any `--dry-run` invocation have `side_effects: false`; mutating commands such as `stack-pr submit` (without `--dry-run`), `stack-pr land`, and `stack-pr abandon` have `side_effects: true`.
- `land` or `abandon` in the default text format → output explicitly states that the command is destructive or has side effects, and that the agent should obtain explicit user confirmation before invoking it.

### JSON schema stability

The JSON output carries a stable, agent-consumable schema with a versioned `id` field per topic. The version suffix is incremented for any backwards-incompatible change to that topic's JSON schema or semantics; a previously published version number is never reused for a different schema.

- `--format json` for any single supported topic other than `all` → JSON object contains an `id` field matching the pattern `stack-pr.prompt.<topic>.v<positive-integer>`.
- `--format json` for any single supported topic other than `all` → JSON object contains an `audience` field with value `"llm-agent"`.

### Fix prompt guidance

- `stack-pr agent prompt fix` → output describes `bpr fix --pr <number>` as a command for attaching an existing PR to local `HEAD` metadata, states that the command does not push branches or write PR changes, and tells agents to use `bpr submit` afterward when the user wants to publish the amended commit and update PRs.
- `stack-pr agent prompt fix --format json` → JSON command guidance includes `bpr fix --pr <number> --dry-run` with `side_effects: false`.
- `stack-pr agent prompt fix --format json` → JSON command guidance includes `bpr fix --pr <number>` with `side_effects: true`.

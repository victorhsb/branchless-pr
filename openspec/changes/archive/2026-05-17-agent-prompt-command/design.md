## Context

All current `stack-pr` subcommands (`view`, `submit`, `land`, `abandon`) assume they are run inside a git repository and, in most cases, that `gh` is authenticated. They rely on shared preflight logic at command startup to fail fast if that environment is missing.

The new `agent prompt` command breaks that assumption: it is a pure documentation emitter for LLM agents and must work in any environment, including:

- Outside any git checkout (e.g., an agent shell that has not yet `cd`-ed into the repo).
- On a machine where `gh` is not installed or not authenticated.
- In a CI sandbox with no network.

A sibling command, `agent diagnose`, is being designed in parallel by another agent. To prevent the two commands from drifting out of sync — `prompt` claiming `submit` is safe while `diagnose` reports otherwise — we want a single static metadata layer that both consume.

## Goals / Non-Goals

**Goals:**

- Ship a deterministic, side-effect-free `agent prompt` command that emits agent guidance in either markdown or JSON.
- Support all listed topics (`overview`, `view`, `submit`, `land`, `abandon`, `recovery`, `all`) with predictable, stable output.
- Make the JSON envelope agent-consumable: stable schema, versioned `id`, explicit side-effect flags per command.
- Provide a shared command-metadata layer that future `agent` siblings (like `diagnose`) reuse, so guidance and reality stay aligned.
- Skip the standard repo/`gh` preflight for everything under the `agent` group.

**Non-Goals:**

- Dynamic, repo-aware prompts (e.g., "your stack currently has 3 PRs"). That belongs to `agent diagnose` or future commands.
- Network-sourced or remotely fetched prompts. All content is compiled into the binary.
- Locale/translation support. English only for now.
- A general templating engine. Topic content can be plain string constants.

## Decisions

### Decision 1: New `agent` parent command group that opts out of preflight

We add a new cobra command group `agent` registered under root. The preflight logic that other subcommands invoke (repo discovery, `gh` auth check) is either run in each subcommand's `RunE` or in a `PersistentPreRunE` hook above it. The `agent` group will explicitly NOT invoke that preflight.

Rationale: Keeping `agent` as a separate parent (rather than e.g. a flag on every command) makes the boundary obvious to readers and to the cobra command tree. It also gives `agent diagnose` (sibling, designed separately) a natural home.

Alternative considered: a global `--no-preflight` flag. Rejected because it would be too easy to misuse on commands that genuinely need a repo.

### Decision 2: Static, in-binary prompt content

Topic content is plain Go string constants (markdown) and matching Go structs (for JSON). No file I/O at runtime, no `embed.FS` complexity required (though `//go:embed` is acceptable if it keeps source readable).

Rationale: Determinism. The same binary version always emits the same prompt text. Agents can rely on `--version` + topic to identify exactly what guidance they consumed.

### Decision 3: JSON envelope with versioned `id`

Each topic's JSON output is a single object with:

- `id`: `stack-pr.prompt.<topic>.v<N>` — bumped only when the schema or semantics change in an agent-visible way.
- `audience`: `"llm-agent"` (fixed for now; reserved for future audiences).
- `summary`: one-line description of the topic.
- `commands`: array of objects, each with `command` (string), `side_effects` (bool), `purpose` (string), and optionally `effects` (array of strings describing what happens).
- `rules`: array of strings — imperative usage rules for the agent.

For the `all` topic, the JSON output is an array of per-topic objects in canonical order (`overview`, `view`, `submit`, `land`, `abandon`, `recovery`).

Rationale: Flat, predictable shape that an agent can parse with a small JSON path. The `id` lets agents cache or validate which version of guidance they ingested.

Alternative considered: emit one big nested object keyed by topic name. Rejected for `all` because a positional array preserves the canonical reading order without relying on map ordering.

### Decision 4: Shared `AgentCommandSpec` metadata layer

Introduce a new package (suggested: `internal/agent`) that defines a Go type roughly:

```go
type AgentCommandSpec struct {
    Name                         string
    Purpose                      string
    SideEffects                  bool
    RequiresExplicitConfirmation bool
    Effects                      []string
    SafeBefore                   []string
    Never                        []string
}
```

…and a registry: `var Commands = map[string]AgentCommandSpec{ "view": {...}, "submit": {...}, ... }`.

Both `agent prompt` (this change) and `agent diagnose` (separate change) read from this registry. Prompt text for each topic is composed from registry entries plus a small amount of topic-specific narrative.

Rationale: Single source of truth. Adding a new side-effect to `submit` updates one struct, and both `prompt` and `diagnose` reflect it.

Note: This change introduces the registry and uses it for `prompt`. The exact field shape is not normative — the spec only mandates that prompt output reflects side-effect metadata. Sibling work can refine the type.

### Decision 5: Default topic is `all`

When the user runs `stack-pr agent prompt` with no positional argument, output the full pack (`all`).

Rationale: Agents calling this command for the first time typically want everything. Topic-specific calls are an optimization, not the common case.

Alternative considered: default to `overview`. Rejected because an agent that fetches `overview` then has to make six more calls to get the rest; defaulting to `all` is a single-call discovery.

### Decision 6: `--format` accepts `text` and `json`, defaults to `text`

Matches the convention established by the existing `view --format` flag (see `view-json-output` spec). Unknown values are rejected with a clear error.

## Risks / Trade-offs

- **Risk**: Static prompt text drifts from actual command behavior over time. **Mitigation**: The shared `AgentCommandSpec` registry forces a code change to update side-effect metadata, and prompt text references those fields directly where practical. CI test asserts that every `stack-pr` user-facing command has an entry in the registry.
- **Risk**: Agents pin to `id` strings, then we change the schema. **Mitigation**: Bump the `vN` suffix on any breaking change; never reuse a version number.
- **Risk**: Prompt becomes too long for some agent context windows. **Mitigation**: Topic-specific subcommands let agents fetch only what they need; topic content is kept terse.
- **Trade-off**: Static content means we can't tailor guidance to the user's repo state. Accepted — that's `agent diagnose`'s job.

## Migration Plan

Pure addition. No existing commands change. No data migration needed. Rollback is removing the `agent` command tree.

## Open Questions

- Exact final field set on `AgentCommandSpec` — left to implementation, since the sibling `diagnose` change may want to extend it. Spec only requires that the prompt output expose per-command side-effect metadata.
- Whether to expose a `--topic` flag in addition to positional. Current decision: positional only, to keep the surface minimal.

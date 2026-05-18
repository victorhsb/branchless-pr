## ADDED Requirements

### Requirement: Agent command group

The `stack-pr` CLI MUST expose a top-level command group named `agent` that is reserved for commands producing agent-facing artifacts. Subcommands under `agent` MUST NOT mutate the git repository, MUST NOT contact GitHub, and MUST NOT require a git repository or `gh` authentication to run.

#### Scenario: Agent group is registered

- **WHEN** the user runs `stack-pr agent --help`
- **THEN** the CLI exits successfully and lists `prompt` as an available subcommand

#### Scenario: Agent group bypasses repo preflight

- **WHEN** the user runs any `stack-pr agent` subcommand from a directory that is not inside a git repository
- **THEN** the command runs normally without emitting a "not a git repository" error

#### Scenario: Agent group bypasses gh auth preflight

- **WHEN** the user runs any `stack-pr agent` subcommand on a system where `gh` is not installed or not authenticated
- **THEN** the command runs normally without emitting a `gh`-auth error

### Requirement: Agent prompt subcommand

The `stack-pr agent prompt` subcommand MUST emit deterministic, static guidance for LLM agents on how to use `stack-pr`. The output MUST NOT depend on the contents of any git repository, the current working directory, network availability, or `gh` authentication state.

#### Scenario: Default invocation prints the full prompt pack

- **WHEN** the user runs `stack-pr agent prompt` with no positional topic argument
- **THEN** the command exits successfully
- **AND** the output contains guidance for all supported topics (`overview`, `view`, `submit`, `land`, `abandon`, `recovery`)

#### Scenario: Determinism across runs

- **WHEN** `stack-pr agent prompt <topic>` is invoked twice with identical arguments on the same binary build
- **THEN** both invocations produce byte-identical output

#### Scenario: Runs outside any git repository

- **WHEN** the user runs `stack-pr agent prompt` from a directory that is not inside a git working tree
- **THEN** the command exits successfully and emits the prompt content

#### Scenario: Runs without gh authentication

- **WHEN** the user runs `stack-pr agent prompt` on a system where `gh` is not installed or not authenticated
- **THEN** the command exits successfully and emits the prompt content

### Requirement: Supported prompt topics

The `agent prompt` subcommand MUST accept an optional positional topic argument with exactly the following allowed values: `overview`, `view`, `submit`, `land`, `abandon`, `recovery`, `all`.

#### Scenario: Topic-specific output

- **WHEN** the user runs `stack-pr agent prompt submit`
- **THEN** the output contains guidance scoped to the `submit` command flow
- **AND** the output does NOT contain the guidance bodies of unrelated topics such as `abandon` or `recovery`

#### Scenario: `all` topic emits the full pack

- **WHEN** the user runs `stack-pr agent prompt all`
- **THEN** the output contains guidance for every other supported topic in a canonical, stable order: `overview`, `view`, `submit`, `land`, `abandon`, `recovery`

#### Scenario: Unknown topic is rejected

- **WHEN** the user runs `stack-pr agent prompt <unknown>` with a value not in the allowed list
- **THEN** the command exits with a non-zero status and a clear error message naming the allowed topics

### Requirement: Output format flag

The `agent prompt` subcommand MUST accept a `--format` flag with allowed values `text` and `json`. The default value MUST be `text`. The `text` format MUST produce human-readable markdown. The `json` format MUST produce machine-readable JSON suitable for consumption by an LLM agent.

#### Scenario: Default format is markdown text

- **WHEN** the user runs `stack-pr agent prompt submit` without `--format`
- **THEN** the output is markdown text that includes a heading for the topic

#### Scenario: JSON format produces structured output

- **WHEN** the user runs `stack-pr agent prompt submit --format json`
- **THEN** the output is valid JSON
- **AND** the output is not markdown
- **AND** the output contains a top-level `id` field whose value is a versioned identifier of the form `stack-pr.prompt.submit.v<N>` where `<N>` is a positive integer
- **AND** the output contains a top-level `commands` array

#### Scenario: JSON format for `all` topic

- **WHEN** the user runs `stack-pr agent prompt all --format json` (or `stack-pr agent prompt --format json` with no positional argument)
- **THEN** the output is valid JSON
- **AND** the output is a JSON array containing one object per supported non-`all` topic in canonical order

#### Scenario: Unknown format is rejected

- **WHEN** the user runs `stack-pr agent prompt --format <unknown>` with a value other than `text` or `json`
- **THEN** the command exits with a non-zero status and a clear error message

### Requirement: Side-effect metadata in prompt output

Both the text and JSON outputs of `agent prompt` MUST clearly communicate which `stack-pr` commands have side effects and which do not, so that an LLM agent can decide whether to ask for user confirmation before invoking them.

#### Scenario: JSON commands carry side-effects flag

- **WHEN** the user runs `stack-pr agent prompt submit --format json`
- **THEN** every element in the `commands` array contains a boolean `side_effects` field
- **AND** read-only commands such as `stack-pr view` and any `--dry-run` invocation have `side_effects: false`
- **AND** mutating commands such as `stack-pr submit` (without `--dry-run`), `stack-pr land`, and `stack-pr abandon` have `side_effects: true`

#### Scenario: Destructive commands are flagged in text output

- **WHEN** the user runs `stack-pr agent prompt land` or `stack-pr agent prompt abandon` in the default text format
- **THEN** the output explicitly states that the command is destructive or has side effects
- **AND** the output states that the agent should obtain explicit user confirmation before invoking it

### Requirement: Stable JSON schema with versioned identifier

The JSON output of `agent prompt` MUST carry a stable, agent-consumable schema. Each topic object MUST include a versioned `id` field. The version suffix MUST be incremented for any backwards-incompatible change to that topic's JSON schema or semantics, and a previously published version number MUST NOT be reused for a different schema.

#### Scenario: id field is present and versioned

- **WHEN** the user runs `stack-pr agent prompt <topic> --format json` for any single supported topic other than `all`
- **THEN** the resulting JSON object contains an `id` field matching the pattern `stack-pr.prompt.<topic>.v<positive-integer>`

#### Scenario: audience field identifies the consumer

- **WHEN** the user runs `stack-pr agent prompt <topic> --format json` for any single supported topic other than `all`
- **THEN** the resulting JSON object contains an `audience` field with value `"llm-agent"`

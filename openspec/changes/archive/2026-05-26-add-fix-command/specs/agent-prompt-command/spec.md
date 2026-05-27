## MODIFIED Requirements

### Requirement: Agent prompt subcommand

The `stack-pr agent prompt` subcommand MUST emit deterministic, static guidance for LLM agents on how to use `stack-pr`. The output MUST NOT depend on the contents of any git repository, the current working directory, network availability, or `gh` authentication state.

#### Scenario: Default invocation prints the full prompt pack

- **WHEN** the user runs `stack-pr agent prompt` with no positional topic argument
- **THEN** the command exits successfully
- **AND** the output contains guidance for all supported topics (`overview`, `view`, `submit`, `land`, `abandon`, `fix`, `recovery`)

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

The `agent prompt` subcommand MUST accept an optional positional topic argument with exactly the following allowed values: `overview`, `view`, `submit`, `land`, `abandon`, `fix`, `recovery`, `all`.

#### Scenario: Topic-specific output

- **WHEN** the user runs `stack-pr agent prompt fix`
- **THEN** the output contains guidance scoped to the `fix` command flow
- **AND** the output does NOT contain the guidance bodies of unrelated topics such as `submit` or `recovery`

#### Scenario: `all` topic emits the full pack

- **WHEN** the user runs `stack-pr agent prompt all`
- **THEN** the output contains guidance for every other supported topic in a canonical, stable order: `overview`, `view`, `submit`, `land`, `abandon`, `fix`, `recovery`

#### Scenario: Unknown topic is rejected

- **WHEN** the user runs `stack-pr agent prompt <unknown>` with a value not in the allowed list
- **THEN** the command exits with a non-zero status and a clear error message naming the allowed topics

## ADDED Requirements

### Requirement: Fix Prompt Guidance

The agent prompt content SHALL describe `fix` as a local recovery command for repairing stack metadata on `HEAD`.

#### Scenario: Fix guidance explains local-only repair

- **WHEN** the user runs `stack-pr agent prompt fix`
- **THEN** the output SHALL describe `bpr fix --pr <number>` as a command for attaching an existing PR to local `HEAD` metadata
- **AND** the output SHALL state that the command does not push branches or write PR changes
- **AND** the output SHALL tell agents to use `bpr submit` afterward when the user wants to publish the amended commit and update PRs

#### Scenario: Fix dry-run is marked read-only

- **WHEN** the user runs `stack-pr agent prompt fix --format json`
- **THEN** the JSON command guidance SHALL include `bpr fix --pr <number> --dry-run`
- **AND** that dry-run command SHALL have `side_effects: false`

#### Scenario: Fix mutation is marked side-effecting

- **WHEN** the user runs `stack-pr agent prompt fix --format json`
- **THEN** the JSON command guidance SHALL include `bpr fix --pr <number>`
- **AND** that command SHALL have `side_effects: true`

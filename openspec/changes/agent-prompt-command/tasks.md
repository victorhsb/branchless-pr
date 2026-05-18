## 1. Shared agent command-metadata layer

- [ ] 1.1 Create `internal/agent/` package (or similar location) with `AgentCommandSpec` struct: `Name`, `Purpose`, `SideEffects`, `RequiresExplicitConfirmation`, `Effects`, `SafeBefore`, `Never`
- [ ] 1.2 Populate a registry (`var Commands = map[string]AgentCommandSpec{...}`) with entries for `view`, `submit`, `submit --dry-run`, `land`, `abandon`
- [ ] 1.3 Add unit test asserting every user-facing `stack-pr` command has a registry entry (or an explicit allow-list of exclusions)

## 2. Prompt content + renderer

- [ ] 2.1 Define topic constants (`overview`, `view`, `submit`, `land`, `abandon`, `recovery`, `all`) and a canonical topic order
- [ ] 2.2 Author markdown content for each topic (compiled-in strings or `//go:embed`), referencing the shared registry where practical
- [ ] 2.3 Implement `RenderText(topic string) (string, error)` that returns markdown for one topic or concatenated content for `all`
- [ ] 2.4 Implement `RenderJSON(topic string) ([]byte, error)` producing the documented JSON envelope (`id`, `audience`, `summary`, `commands`, `rules`); `all` returns a JSON array of per-topic objects in canonical order
- [ ] 2.5 Ensure JSON `id` follows the `stack-pr.prompt.<topic>.v1` pattern and `audience` is `"llm-agent"`
- [ ] 2.6 Ensure each `commands[]` entry includes `side_effects` (bool) and, for mutating commands, an `effects` array
- [ ] 2.7 Unit tests: golden-file comparison for text output of every topic
- [ ] 2.8 Unit tests: JSON output validates against expected schema for every topic (including `all` array shape)
- [ ] 2.9 Unit test: byte-identical output on repeated calls (determinism)

## 3. CLI wiring

- [ ] 3.1 Add `agentCmd` parent command in `internal/cli/agent.go` (or equivalent), registered on the root command
- [ ] 3.2 Add `agentPromptCmd` subcommand under `agentCmd` with optional positional topic argument (default `all`)
- [ ] 3.3 Add `--format` string flag with default `text`; validate against `{text, json}`
- [ ] 3.4 Validate topic argument against the allowed set; emit a clear error naming valid topics on mismatch
- [ ] 3.5 Ensure the `agent` command group does NOT run the standard repo/`gh` preflight (skip any `PersistentPreRunE` that performs it, or guard the preflight against the `agent` subtree)
- [ ] 3.6 Wire `RunE` to call `RenderText` or `RenderJSON` based on `--format` and print to stdout

## 4. CLI-level tests

- [ ] 4.1 Test `stack-pr agent prompt` (no args) emits the `all` pack in text
- [ ] 4.2 Test `stack-pr agent prompt submit` emits scoped markdown
- [ ] 4.3 Test `stack-pr agent prompt submit --format json` parses as JSON and contains expected `id`, `audience`, `commands[].side_effects`
- [ ] 4.4 Test `stack-pr agent prompt --format json` (no topic) returns a JSON array of all non-`all` topics in canonical order
- [ ] 4.5 Test unknown topic exits non-zero with a clear error
- [ ] 4.6 Test unknown `--format` value exits non-zero with a clear error
- [ ] 4.7 Test the command succeeds when CWD is outside any git repository
- [ ] 4.8 Test the command succeeds when `gh` is missing or unauthenticated (e.g., by stubbing or running in an env without `gh` on PATH)

## 5. Documentation

- [ ] 5.1 Update README with a short section on `stack-pr agent prompt` (usage, topics, formats)
- [ ] 5.2 Mention in release notes that the new command is intended for LLM-agent consumption and ships static, versioned guidance

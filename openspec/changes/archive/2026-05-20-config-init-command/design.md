## Context

The `stack-pr` tool reads repository-wide defaults from `.stack-pr.cfg`. Currently, users must consult documentation to understand available sections (`[common]`, `[repo]`, `[land]`, `[comments]`) and their keys. The existing `config` command only supports `config <section>.<key>=<value>` to mutate an already-existing file. There is no starter-generator, which causes friction for first-time adoption and increases support burden.

## Goals / Non-Goals

**Goals:**

- Provide a `stack-pr config init` CLI subcommand that generates a starter INI file with sensible defaults.
- Guard against accidental overwrite when `.stack-pr.cfg` already exists.
- Keep the generated configuration self-documenting via comments.
- Require no new external dependencies.

**Non-Goals:**

- Interactive prompts or wizard-style questionnaire during init.
- Guessing user-preferred values from git remotes (e.g., inferring target branch name).
- Backing up the old file automatically on overwrite attempt.

## Decisions

1. **File generation approach** — Write the default INI as a literal string template in the Go source rather than serialising `config.Config`. Rationale: Preserves comment lines and section ordering in a human-readable way. Serialising `Config.Save()` strips comments and deterministically sorts keys, producing bare machine output.
2. **Overwrite guard** — Use `os.Stat` before opening; return a clear error instead of prompting interactively. Rationale: Keeps the command non-interactive and safe for scripting.
3. **Where to place the command** — A new `init` subcommand under the existing `config` Cobra branch (`configCmd()`), not a top-level command. Rationale: Scopes naturally with existing config operations and avoids polluting the top-level command surface.
4. **No repo-root required?** — No, require repo-root via `config.FilePath()` same as before. Rationale: Consistent with how `config` already works; `config init` creates the file at the repo root.

## Risks / Trade-offs

- [Risk] Hard-coded default comments drift out of sync when new config keys are added.  
  → Mitigation: A CI check or unit test comparing generated output against `config.Defaults()` key set will flag omissions.
- [Risk] Users running `config init` outside a repo face a confusing error.  
  → Mitigation: Reuse `config.FilePath()` which already returns a meaningful error when repo root discovery fails.
- [Risk] Overwriting guard could frustrate users who genuinely want to reset.  
  → Mitigation: Error message explicitly says to delete the existing file first; non-interactive by design.

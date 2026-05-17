## Context

The `view` command (`internal/cli/view.go`) currently calls `st.PrintStack(links, color)` unconditionally. The `Stack.PrintStack` method in turn calls `Entry.PrettyLine` for each entry. To add `--format json` we need a second rendering path without disturbing the existing text output.

## Goals / Non-Goals

**Goals:**

- Add a `--format` flag with `text` (default) and `json` choices
- Produce clean flat JSON objects per stack entry
- Keep the JSON schema stable for tool consumers

**Non-Goals:**

- XML, YAML, or other output formats
- Streaming/chunked output (single JSON array is fine)
- Changing the text rendering path

## Decisions

### Decision 1: Flat JSON schema

Each entry becomes a flat JSON object with fields that a consumer would naturally query with `jq`:

- `commit`: full 40-char SHA
- `short_sha`: 8-char abbreviated SHA
- `title`: commit title
- `author`: raw `Name <email>` string
- `author_name`: parsed name
- `author_email`: parsed email
- `pr_url`: PR URL (empty if no PR)
- `pr_number`: parsed numeric PR number (0 if no PR)
- `head_branch`: generated head branch name
- `base_branch`: target/base branch name

Rationale: Flat fields are easier to `jq` than nested structures. The schema mirrors what `PrettyLine` already displays, so it feels natural.

### Decision 2: Add `ToJSON([]*Entry)` on `Stack`

We'll add a method `Stack.ToJSON() ([]byte, error)` in `internal/stack/stack.go` that returns a `json.Marshal`-able slice of anonymous structs with the flat fields above.

This keeps JSON serialization in the `stack` package where the data lives, rather than leaking field access into the CLI layer.

### Decision 3: Format validation in command layer

Format string validation (`text` | `json`) happens in `viewCmd` flag parsing via cobra's `ValidArgs` or a simple `switch` in `runView`. This keeps the stack package agnostic to CLI concerns.

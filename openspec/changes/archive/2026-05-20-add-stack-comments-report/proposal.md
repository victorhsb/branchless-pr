## Why

Stacked PR review context is currently split across several GitHub pages, which makes it slow for humans and agents to understand what feedback is still open across the whole stack. A read-only comments report should gather review threads, review comments, reviews, and PR conversation comments into one stable output that is easy to scan or feed into an agent.

## What Changes

- Add a read-only command for collecting comments from every PR represented by the current stack metadata.
- Support human-readable Markdown output by default and structured JSON output for agents and automation.
- Preserve stack order and attach each comment/thread to its stack entry, PR, author, timestamp, URL, state, file path, and line context when GitHub provides it.
- Provide filtering controls for common review workflows, including unresolved-only output and comment-type selection.
- Report missing PR metadata, inaccessible PRs, GitHub/`gh` failures, and empty-comment results without mutating local Git state or GitHub state.

## Capabilities

### New Capabilities

- `stack-comments-report`: Defines read-only collection and rendering of comments across all pull requests in a stack.

### Modified Capabilities

- None.

## Impact

- Affected commands: new top-level read-only CLI command, likely `stack-pr comments`, with `--format text|json` and filtering flags.
- Affected packages: `internal/cli` for command wiring/output, `internal/stack` for stack entry context reuse, and a GitHub/PR read helper that shells out through `internal/shell`.
- Affected external tools: requires `gh` and GitHub read access for the stack PRs; no GitHub SDK dependency is introduced.
- Affected docs/specs: `SPEC.md`, command help, README-style usage examples if this ships.

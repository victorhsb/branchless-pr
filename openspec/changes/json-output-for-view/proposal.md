## Why

The `stack-pr view` command only produces ANSI-colored terminal output. This makes it impossible for CI pipelines, editor plugins, or shell scripts to consume the stack state programmatically. Adding JSON output enables tool composition and automation.

## What Changes

- Add `--format` flag to `view` command with choices `text` (default) and `json`
- Extract stack entry serialization into a flat JSON schema
- Update `Stack` type with a `ToJSON()` method producing structured output

## Capabilities

### New Capabilities

- `view-json-output`: Produce machine-readable JSON representation of the stack

### Modified Capabilities

<!-- No existing spec behavior is changing; this is a pure addition. -->

## Impact

- `internal/cli/view.go`: Add `--format` flag, branch on format
- `internal/stack/stack.go`: Add `ToJSON()` method
- `internal/stack/entry.go`: Ensure all display fields are accessible

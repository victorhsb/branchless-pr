## Purpose

Define machine-readable output behavior for inspecting a stack.

## Requirements

### Requirement: View JSON Output

The `stack-pr view` command must support a `--format json` mode that produces machine-readable JSON instead of the default ANSI-colored text.

#### Scenario: Default format remains text

- **WHEN** `stack-pr view` is invoked without `--format`
- **THEN** output uses the existing ANSI-colored, hyperlink-enabled text format

#### Scenario: JSON format produces structured output

- **WHEN** `stack-pr view --format json` is invoked
- **THEN** output is a JSON array ordered newest-to-oldest
- **AND** each array element is a flat object with fields: `commit`, `short_sha`, `title`, `author`, `author_name`, `author_email`, `pr_url`, `pr_number`, `head_branch`, `base_branch`
- **AND** output contains no ANSI escape sequences or terminal hyperlinks

#### Scenario: Missing PR fields

- **WHEN** a stack entry has no associated PR
- **THEN** `pr_url` is `""` and `pr_number` is `0`

#### Scenario: Unknown format is rejected

- **WHEN** `stack-pr view --format <unknown>` is invoked with a value other than `text` or `json`
- **THEN** the command exits with an error returning a clear message

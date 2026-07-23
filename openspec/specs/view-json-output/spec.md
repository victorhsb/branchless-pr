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
- **AND** each array element is a flat object with fields: `commit`, `short_sha`, `title`, `author`, `author_name`, `author_email`, `pr_url`, `pr_number`, `head_branch`, `base_branch`, `github_stack_number`, `github_stack_position`, `github_stack_size`, `github_stack_base`
- **AND** output contains no ANSI escape sequences or terminal hyperlinks

#### Scenario: Missing PR fields

- **WHEN** a stack entry has no associated PR
- **THEN** `pr_url` is `""` and `pr_number` is `0`

#### Scenario: Unknown format is rejected

- **WHEN** `stack-pr view --format <unknown>` is invoked with a value other than `text` or `json`
- **THEN** the command exits with an error returning a clear message

#### Scenario: Entry is a native Stack member

- **GIVEN** native integration is enabled
- **AND** an entry belongs to a GitHub Stack
- **WHEN** JSON output is rendered
- **THEN** `github_stack_number` SHALL be the repository-scoped Stack number
- **AND** `github_stack_position` SHALL be the 1-based bottom-to-top position
- **AND** `github_stack_size` SHALL be the Stack size
- **AND** `github_stack_base` SHALL be the ultimate Stack base ref

#### Scenario: Entry is not a native Stack member

- **GIVEN** native integration is enabled
- **AND** an entry's PR is unstacked or membership is unavailable in auto mode
- **WHEN** JSON output is rendered
- **THEN** every `github_stack_*` field SHALL be `null`

#### Scenario: Native integration is disabled

- **GIVEN** `github.native_stacks = off`
- **WHEN** JSON output is rendered
- **THEN** every `github_stack_*` field SHALL be `null`
- **AND** no native membership query SHALL occur

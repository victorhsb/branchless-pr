## MODIFIED Requirements

### Requirement: Generated file mirrors current defaults

The generated configuration SHALL contain, at minimum, the same keys and values as the built-in `config.Defaults()` map, organised into sections `[common]`, `[repo]`, `[github]`, `[land]`, and `[comments]`.

#### Scenario: Defaults parity

- **WHEN** the user runs `stack-pr config init` successfully
- **THEN** parsing the generated file with `config.Load` and merging with `config.Defaults()` produces no new keys in either direction

#### Scenario: Native Stack default documented

- **WHEN** the user runs `stack-pr config init` successfully
- **THEN** the generated `[github]` section SHALL contain `native_stacks = off`
- **AND** inline comments SHALL document the `off`, `auto`, and `required` values
- **AND** the comments SHALL note that enabling native stacks changes GitHub CI, rules, review, and landing behavior
- **AND** the comments SHALL state that native Stack operations use the REST API through the base `gh` CLI
- **AND** the comments SHALL NOT require the `github/gh-stack` extension

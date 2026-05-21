## ADDED Requirements

### Requirement: Config init command generates starter config file

The system SHALL provide a `config init` subcommand that writes a `.stack-pr.cfg` file at the repository root with sensible defaults and inline documentation.

#### Scenario: Successful generation

- **WHEN** the user runs `stack-pr config init` inside a repository that has no `.stack-pr.cfg`
- **THEN** a `.stack-pr.cfg` file is created at `<repo-root>/.stack-pr.cfg` containing all default sections and keys, each with a descriptive comment

#### Scenario: Overwrite guard

- **WHEN** the user runs `stack-pr config init` inside a repository that already has `.stack-pr.cfg`
- **THEN** the command exits with a non-zero status and prints an error indicating the file already exists

### Requirement: Generated file mirrors current defaults

The generated configuration SHALL contain, at minimum, the same keys and values as the built-in `config.Defaults()` map, organised into sections `[common]`, `[repo]`, `[land]`, and `[comments]`.

#### Scenario: Defaults parity

- **WHEN** the user runs `stack-pr config init` successfully
- **THEN** parsing the generated file with `config.Load` and merging with `config.Defaults()` produces no new keys in either direction

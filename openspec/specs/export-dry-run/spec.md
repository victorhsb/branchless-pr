# Export Dry Run

Preview submit/export actions without changing local Git state or GitHub state.

## Requirements

### Requirement: Submit Export Dry Run Flag

The `stack-pr submit` command and its `export` alias SHALL support a `--dry-run` flag that previews submit/export actions without applying them.

#### Scenario: Dry-run flag is accepted on submit

- **WHEN** `stack-pr submit --dry-run` is invoked with otherwise valid options
- **THEN** the command SHALL execute dry-run behavior instead of real submit/export behavior

#### Scenario: Dry-run flag is accepted on export alias

- **WHEN** `stack-pr export --dry-run` is invoked with otherwise valid options
- **THEN** the command SHALL execute the same dry-run behavior as `stack-pr submit --dry-run`

### Requirement: Dry Run Plan Output

Dry-run mode SHALL print a human-readable plan describing the submit/export actions that would be performed for the current stack.

#### Scenario: Non-empty stack plan

- **WHEN** dry-run mode is invoked for a non-empty stack
- **THEN** output SHALL include each stack entry in stack order
- **AND** each entry SHALL show the commit title, generated head branch, computed base branch, and whether the associated PR would be created or updated
- **AND** entries for new PRs SHALL show the draft state that would be used
- **AND** entries requiring stack metadata SHALL indicate that metadata would be added during a real submit/export

#### Scenario: Existing PR plan

- **WHEN** a stack entry already has PR metadata
- **THEN** dry-run output SHALL identify the existing PR and indicate that it would be updated rather than created

#### Scenario: Empty stack

- **WHEN** dry-run mode is invoked for an empty stack
- **THEN** output SHALL report that the stack is empty
- **AND** output SHALL report success without attempting any mutation

#### Scenario: No changes note

- **WHEN** dry-run mode completes successfully
- **THEN** output SHALL clearly state that no local Git changes, remote pushes, or GitHub PR changes were made

### Requirement: Dry Run Mutation Safety

Dry-run mode SHALL NOT perform local Git mutations, remote pushes, or GitHub write operations.

#### Scenario: Local Git state is not mutated

- **WHEN** dry-run mode is invoked
- **THEN** the command SHALL NOT checkout generated branches, rebase branches, amend commits, create or delete local generated branches, save a stash, or pop a stash

#### Scenario: Remote branches are not pushed

- **WHEN** dry-run mode is invoked
- **THEN** the command SHALL NOT push generated head branches or amended branches to any remote

#### Scenario: GitHub PRs are not changed

- **WHEN** dry-run mode is invoked
- **THEN** the command SHALL NOT create PRs, edit PR title/body/base fields, or change PR draft/ready state

### Requirement: Dry Run Validation

Dry-run mode SHALL validate the same submit/export inputs and planning decisions that can be checked without mutation.

#### Scenario: Draft bitmask validation

- **WHEN** dry-run mode is invoked with `--draft-bitmask` whose length does not match the stack length or whose characters are not `0` or `1`
- **THEN** the command SHALL report the same validation error as real submit/export

#### Scenario: Branch and base planning

- **WHEN** dry-run mode is invoked for a non-empty stack
- **THEN** generated head branches SHALL be computed from the configured branch-name template
- **AND** base branches SHALL be computed using the same bottom-to-top stacking rules as real submit/export

#### Scenario: Clean repository remains required

- **WHEN** dry-run mode is invoked while tracked files have staged or unstaged changes
- **THEN** the command SHALL fail the existing clean-repository check
- **AND** the command SHALL NOT stash changes automatically

### Requirement: Non-Dry-Run Behavior Preservation

Normal `stack-pr submit` and `stack-pr export` behavior SHALL remain unchanged when `--dry-run` is not provided.

#### Scenario: Real submit remains mutating

- **WHEN** `stack-pr submit` is invoked without `--dry-run`
- **THEN** the command SHALL continue to create/update PRs, push branches, update metadata, and perform cleanup according to existing submit/export behavior

#### Scenario: Real export alias remains mutating

- **WHEN** `stack-pr export` is invoked without `--dry-run`
- **THEN** the command SHALL continue to behave as the submit alias according to existing submit/export behavior

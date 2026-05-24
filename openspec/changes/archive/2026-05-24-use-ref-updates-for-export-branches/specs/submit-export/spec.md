## MODIFIED Requirements

### Requirement: Local Branch Initialization
Submit/export SHALL create or ensure local generated branches for each stack entry before remote interaction without requiring a worktree checkout for every entry.

#### Scenario: Generated branch assignment

- **WHEN** local branches are initialized
- **THEN** the remote SHALL be fetched and pruned
- **AND** entries missing metadata heads SHALL receive generated head branches from the branch-name template
- **AND** for each entry, the command SHALL ensure the local branch `<entry.head>` points at `<commit-id>`
- **AND** this initialization SHALL NOT require checking out each stack entry

#### Scenario: Existing metadata head preserved

- **WHEN** a stack entry already has a head branch in its metadata
- **THEN** that head branch SHALL be reused
- **AND** the corresponding local branch SHALL be reset to the entry commit before the first batch force-push

#### Scenario: Current branch preserved during initialization

- **WHEN** local generated branches are initialized
- **THEN** the command SHALL preserve the current worktree branch unless a later submit/export step explicitly checks out or rebases a branch for metadata amendment, restoration, or cleanup

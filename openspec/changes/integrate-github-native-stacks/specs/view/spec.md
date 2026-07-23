## ADDED Requirements

### Requirement: Native Stack View Inspection
When native integration is enabled, view SHALL inspect GitHub native membership without modifying local Git, remote branches, PRs, or Stacks.

#### Scenario: Matching native Stack summary
- **GIVEN** native integration is enabled
- **AND** every submitted local PR belongs to one native Stack in the same bottom-to-top order
- **WHEN** `stack-pr view` renders text
- **THEN** output SHALL identify the native Stack number, size, and ultimate base branch
- **AND** existing stack-entry lines SHALL remain ordered newest-to-oldest

#### Scenario: Unstacked local PRs
- **GIVEN** native integration is enabled
- **AND** submitted local PRs are not native Stack members
- **WHEN** `stack-pr view` renders text
- **THEN** output SHALL state that native membership is absent
- **AND** it SHALL NOT treat absence as drift in `auto` mode
- **AND** it SHALL report the required-membership error in `required` mode for an eligible multi-PR stack

#### Scenario: Native membership drift
- **GIVEN** native integration is enabled
- **AND** remote membership is reordered, split, mixed, or contains PRs not in the local sequence
- **WHEN** `stack-pr view` runs
- **THEN** output SHALL report a native membership drift warning or error
- **AND** it SHALL identify enough local and remote PR ordering information for manual resolution

#### Scenario: Disabled mode preserves view behavior
- **GIVEN** `github.native_stacks = off`
- **WHEN** `stack-pr view` runs
- **THEN** it SHALL preserve the existing text and JSON behavior
- **AND** it SHALL NOT make native membership queries

#### Scenario: View remains read-only
- **GIVEN** native integration is enabled
- **WHEN** `stack-pr view` inspects membership
- **THEN** it SHALL use only read operations
- **AND** it SHALL NOT create, append, unstack, edit, rebase, or push anything

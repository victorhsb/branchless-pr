## MODIFIED Requirements

### Requirement: Native Stack Dry-Run Plan

Submit/export dry-run SHALL describe native Stack reconciliation without performing a GitHub write.

#### Scenario: Existing PRs produce exact native action

- **GIVEN** native integration is enabled
- **AND** every stack entry already has a PR number
- **WHEN** submit/export runs with `--dry-run`
- **THEN** the plan SHALL report the classified native action as `create`, `append`, `noop`, `conflict`, `ineligible`, or `unavailable fallback`

#### Scenario: New PRs produce prospective action

- **GIVEN** native integration is enabled
- **AND** one or more stack entries do not yet have PR numbers
- **WHEN** submit/export runs with `--dry-run`
- **THEN** the plan SHALL report whether a real submit would prospectively create a Stack or append the new suffix after PR creation
- **AND** it SHALL distinguish that prospective result from an exact membership classification

#### Scenario: Native integration disabled in dry-run

- **GIVEN** `github.native_stacks = off`
- **WHEN** submit/export runs with `--dry-run`
- **THEN** the plan SHALL report native integration as disabled or omit the native action consistently
- **AND** it SHALL NOT call a native Stacks endpoint

#### Scenario: Dry-run performs no native write

- **GIVEN** any native mode
- **WHEN** submit/export runs with `--dry-run`
- **THEN** it SHALL NOT invoke REST create, append, or unstack operations
- **AND** it SHALL preserve all existing local Git, remote push, and PR no-mutation guarantees

#### Scenario: Read-only membership query is allowed

- **GIVEN** native integration is enabled
- **WHEN** dry-run needs existing membership to classify a plan
- **THEN** it MAY perform read-only GitHub API calls
- **AND** those calls SHALL NOT modify PR or Stack state

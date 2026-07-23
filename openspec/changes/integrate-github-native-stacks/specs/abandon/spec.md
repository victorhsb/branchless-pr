## ADDED Requirements

### Requirement: Native Stack Abandon Preflight
When native integration is enabled, abandon SHALL inspect and safely dissolve matching native membership before deleting generated remote branches.

#### Scenario: Exact native Stack is unstacked first
- **GIVEN** native integration is enabled
- **AND** the local PR sequence exactly matches one GitHub native Stack
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL run `gh stack unstack <stack-number>` before stripping local metadata or deleting generated remote branches
- **AND** it SHALL verify that no unmerged local PR remains unexpectedly stacked before remote branch deletion

#### Scenario: Native conflict blocks abandon
- **GIVEN** native integration is enabled
- **AND** native membership conflicts with the local PR sequence
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL fail before commit amendment, local branch mutation, remote branch deletion, or native Stack mutation
- **AND** it SHALL provide manual unstack guidance

#### Scenario: Auto mode with unstacked PRs uses legacy cleanup
- **GIVEN** `github.native_stacks = auto`
- **AND** every local PR is unstacked
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL skip the native unstack operation
- **AND** it SHALL continue with the existing metadata and branch cleanup algorithm

#### Scenario: Auto mode unavailable uses legacy cleanup
- **GIVEN** `github.native_stacks = auto`
- **AND** native Stacks is unavailable for the repository
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL warn once
- **AND** it SHALL continue with legacy abandon behavior

#### Scenario: Auto mode missing extension blocks cleanup of native Stack
- **GIVEN** `github.native_stacks = auto`
- **AND** local PRs exactly match a native Stack
- **AND** the gh-stack extension required for unstack is missing or unsupported
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL fail before local metadata amendment or local or remote branch deletion
- **AND** it SHALL provide extension installation or upgrade guidance

#### Scenario: Required mode unavailable blocks abandon
- **GIVEN** `github.native_stacks = required`
- **AND** native Stacks is unavailable for the repository or the gh-stack extension is missing or unsupported
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL fail before local or remote mutation

#### Scenario: Unstack operation fails
- **GIVEN** exact native membership
- **AND** the native unstack operation fails or leaves an unmerged local PR stacked
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL fail before deleting generated remote branches
- **AND** existing PR and branch state SHALL be preserved except for effects already reported by GitHub

#### Scenario: Unstack success is verified through REST
- **GIVEN** `gh stack unstack <stack-number>` exits successfully
- **WHEN** abandon verifies the result
- **THEN** it SHALL read native membership through the REST API
- **AND** it SHALL stop before generated remote branch deletion if any unmerged local PR remains stacked

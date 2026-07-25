## MODIFIED Requirements

### Requirement: Native Stack Abandon Preflight

When native integration is enabled, abandon SHALL inspect and safely dissolve matching native membership through the REST API before deleting generated remote branches.

#### Scenario: Exact native Stack is unstacked first

- **GIVEN** native integration is enabled
- **AND** the local PR sequence exactly matches one GitHub native Stack
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL POST with no body to `repos/{owner}/{repo}/stacks/{stack_number}/unstack` before stripping local metadata or deleting generated remote branches
- **AND** it SHALL verify that no affected unmerged local PR remains unexpectedly stacked before remote branch deletion

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

#### Scenario: Required mode unavailable blocks abandon

- **GIVEN** `github.native_stacks = required`
- **AND** native Stacks is unavailable for the repository
- **WHEN** `stack-pr abandon` runs
- **THEN** it SHALL fail before local or remote mutation

#### Scenario: Partial unstack blocks unsafe cleanup

- **GIVEN** exact native membership
- **AND** the REST unstack operation returns `200` with one or more surviving members
- **WHEN** `stack-pr abandon` evaluates the result
- **THEN** it SHALL preserve and report the surviving Stack
- **AND** it SHALL stop before deleting a generated remote branch for any affected unmerged PR that remains stacked

#### Scenario: Dissolved unstack permits cleanup

- **GIVEN** exact native membership
- **AND** the REST unstack operation returns `204` with no body
- **WHEN** `stack-pr abandon` evaluates the result
- **THEN** it SHALL treat the native Stack as dissolved
- **AND** it MAY continue with the existing metadata and branch cleanup algorithm

#### Scenario: Uncertain unstack is reconciled

- **GIVEN** exact native membership
- **AND** the REST unstack request fails with an uncertain server outcome
- **WHEN** `stack-pr abandon` handles the failure
- **THEN** it SHALL re-read the numbered Stack before deciding whether cleanup is safe
- **AND** it SHALL NOT blindly repeat the unstack request
- **AND** it SHALL fail before branch deletion when the result cannot be proven safe

#### Scenario: Closed-unmerged survivor blocks cleanup

- **GIVEN** a partial unstack returns a local PR with `state: "closed"` and `merged_at: null`
- **WHEN** abandon evaluates whether its generated branch may be deleted
- **THEN** it SHALL treat that PR as unmerged
- **AND** it SHALL stop before deleting the branch

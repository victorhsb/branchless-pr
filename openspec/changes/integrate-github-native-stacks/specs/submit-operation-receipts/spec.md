## ADDED Requirements

### Requirement: Native Stack Receipt Operations
Submit/export operation receipts SHALL record the outcome of native Stack planning and reconciliation when native integration is enabled.

#### Scenario: Native Stack created
- **GIVEN** receipt emission is enabled
- **WHEN** submit/export creates a GitHub native Stack
- **THEN** the receipt SHALL include an `ok` operation containing the action `create`, Stack number, and ordered PR numbers

#### Scenario: Native Stack appended
- **GIVEN** receipt emission is enabled
- **WHEN** submit/export appends PRs to a GitHub native Stack
- **THEN** the receipt SHALL include an `ok` operation containing the action `append`, Stack number, and appended PR numbers

#### Scenario: Native Stack already exact
- **GIVEN** receipt emission is enabled
- **WHEN** native reconciliation is a no-op
- **THEN** the receipt SHALL include an `ok` operation containing the action `noop` and Stack number

#### Scenario: Auto fallback recorded
- **GIVEN** receipt emission is enabled
- **AND** auto mode falls back because native Stacks is unavailable or the stack is ineligible
- **WHEN** submit/export completes through the legacy path
- **THEN** the receipt SHALL record the fallback reason

#### Scenario: Native reconciliation failure recorded
- **GIVEN** receipt emission is enabled
- **WHEN** native classification or mutation fails after earlier submit operations succeeded
- **THEN** the receipt SHALL include a failed native Stack operation
- **AND** the top-level receipt status SHALL be `partial_failure`

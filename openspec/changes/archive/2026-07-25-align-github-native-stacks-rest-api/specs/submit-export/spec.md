## MODIFIED Requirements

### Requirement: Native Stack Submit Reconciliation

When native integration is enabled, submit/export SHALL reconcile the final submitted PR chain with GitHub after ordinary PR and branch publication succeeds.

#### Scenario: Reconciliation occurs after final PR state

- **GIVEN** native integration is enabled for an eligible stack
- **WHEN** submit/export has completed final branch pushes, commit metadata amendment, PR title/body/base updates, and temporary draft restoration
- **THEN** the command SHALL reload and validate candidate PR state and native membership
- **AND** it SHALL apply only a `create`, `append`, or `noop` result

#### Scenario: Both submit engines reconcile identically

- **GIVEN** native integration is enabled
- **WHEN** either the current submit engine or the optimized submit engine reaches its final remote phase
- **THEN** both engines SHALL use the same native reconciliation rules
- **AND** they SHALL produce the same final GitHub Stack membership

#### Scenario: New native Stack created

- **GIVEN** the final eligible PR chain is entirely unstacked
- **WHEN** submit/export reconciles native membership
- **THEN** it SHALL POST every existing PR number bottom-to-top to `repos/{owner}/{repo}/stacks`
- **AND** it SHALL require a `201` Stack response with the exact resulting complete sequence

#### Scenario: Existing native Stack extended

- **GIVEN** the remote native sequence is an exact prefix of the final local PR sequence
- **AND** the local suffix PRs are unstacked
- **WHEN** submit/export reconciles native membership
- **THEN** it SHALL POST only the suffix PR numbers to `repos/{owner}/{repo}/stacks/{stack_number}/add`
- **AND** it SHALL require a `200` Stack response with the exact resulting complete sequence

#### Scenario: Exact native Stack skips write

- **GIVEN** native membership exactly matches the final local PR sequence
- **WHEN** submit/export reconciles native membership
- **THEN** no native Stack write SHALL occur

#### Scenario: Native conflict fails after existing submit effects

- **GIVEN** ordinary submit/export effects have completed
- **AND** native membership is conflicting
- **WHEN** reconciliation runs
- **THEN** the command SHALL return an error without changing native membership
- **AND** the error SHALL state that earlier PR or branch updates may already have completed

#### Scenario: Uncertain create or append is reconciled

- **GIVEN** a native create or append request fails with an uncertain server outcome
- **WHEN** submit/export handles the failure
- **THEN** it SHALL read current native membership and the complete affected Stack
- **AND** it SHALL accept an exact intended sequence as success
- **AND** it SHALL report unchanged, divergent, or unverified state without blindly repeating the write

#### Scenario: Cross-links retained in native mode

- **GIVEN** a multi-PR stack is published as a GitHub native Stack
- **WHEN** PR bodies are finalized
- **THEN** the existing stacked-PR table of contents and delimiter SHALL remain present

### Requirement: Native Stack Availability Preflight

Required native integration SHALL fail before submit-specific mutation when repository support is unavailable, and native writes SHALL use the documented REST endpoints through the base GitHub CLI.

#### Scenario: Required mode unavailable before submit mutation

- **GIVEN** `github.native_stacks = required`
- **AND** GitHub native Stacks is unavailable for the repository
- **WHEN** submit/export begins
- **THEN** it SHALL fail before generated branch creation, commit amendment, remote push, PR mutation, or native Stack mutation

#### Scenario: Auto mode unavailable continues legacy submit

- **GIVEN** `github.native_stacks = auto`
- **AND** GitHub native Stacks is unavailable for the repository
- **WHEN** submit/export begins
- **THEN** it SHALL warn once
- **AND** it SHALL execute the legacy submit/export algorithm without native reconciliation

#### Scenario: Native writer requires no extension

- **GIVEN** GitHub native Stacks is available
- **AND** the final PR chain is eligible for create or append
- **WHEN** submit/export performs native preflight
- **THEN** it SHALL proceed using `gh api`
- **AND** it SHALL NOT inspect, require, install, or upgrade the `github/gh-stack` extension

## ADDED Requirements

### Requirement: Native Stack Submit Reconciliation
When native integration is enabled, submit/export SHALL reconcile the final submitted PR chain with GitHub after ordinary PR and branch publication succeeds.

#### Scenario: Reconciliation occurs after final PR state
- **GIVEN** native integration is enabled for an eligible stack
- **WHEN** submit/export has completed final branch pushes, commit metadata amendment, PR title/body/base updates, and temporary draft restoration
- **THEN** the command SHALL classify the local PR sequence against GitHub native membership
- **AND** it SHALL apply only a `create`, `append`, or `noop` result

#### Scenario: Both submit engines reconcile identically
- **GIVEN** native integration is enabled
- **WHEN** either the current submit engine or the optimized submit engine reaches its final remote phase
- **THEN** both engines SHALL use the same native reconciliation rules
- **AND** they SHALL produce the same final GitHub Stack membership

#### Scenario: New native Stack created
- **GIVEN** the final eligible PR chain is entirely unstacked
- **WHEN** submit/export reconciles native membership
- **THEN** it SHALL call `gh stack link` with every existing PR number bottom-to-top
- **AND** it SHALL verify the resulting complete sequence through the REST API

#### Scenario: Existing native Stack extended
- **GIVEN** the remote native sequence is an exact prefix of the final local PR sequence
- **AND** the local suffix PRs are unstacked
- **WHEN** submit/export reconciles native membership
- **THEN** it SHALL call `gh stack link` with the existing Stack number followed by only the suffix PR numbers
- **AND** it SHALL verify the resulting complete sequence through the REST API

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

#### Scenario: Cross-links retained in native mode
- **GIVEN** a multi-PR stack is published as a GitHub native Stack
- **WHEN** PR bodies are finalized
- **THEN** the existing stacked-PR table of contents and delimiter SHALL remain present

### Requirement: Native Stack Push Lease Safety
When native mode is active, submit/export SHALL avoid overwriting server-side Stack rebases or concurrent remote updates with an unconditional force push.

#### Scenario: Existing native branch uses observed lease
- **GIVEN** native mode is active
- **AND** a generated remote branch existed when submit preflight read its head OID
- **WHEN** submit/export force-updates that branch
- **THEN** the push SHALL require the remote branch still to equal the observed OID

#### Scenario: Lease mismatch stops submit
- **GIVEN** GitHub or another actor updates a generated remote branch after preflight
- **WHEN** submit/export pushes with its observed lease
- **THEN** the push SHALL fail without overwriting the newer remote head
- **AND** native reconciliation SHALL NOT run

#### Scenario: Newly created branch has absence expectation
- **GIVEN** native mode is active
- **AND** a generated remote branch did not exist at preflight
- **WHEN** submit/export first pushes that branch
- **THEN** the push SHALL require that the branch is still absent

### Requirement: Native Stack Availability Preflight
Required native integration SHALL fail before submit-specific mutation when repository support is unavailable.

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

#### Scenario: Auto mode missing extension skips unstacked create
- **GIVEN** `github.native_stacks = auto`
- **AND** every existing local PR is unstacked
- **AND** the gh-stack extension required for native creation is missing or unsupported
- **WHEN** submit/export begins
- **THEN** it SHALL warn that native Stack creation will be skipped
- **AND** it MAY execute the legacy submit/export algorithm with unstacked PRs

#### Scenario: Auto mode missing extension blocks native append
- **GIVEN** `github.native_stacks = auto`
- **AND** the remote native sequence is a prefix of the local sequence that requires append
- **AND** the gh-stack extension is missing or unsupported
- **WHEN** submit/export begins
- **THEN** it SHALL fail before generated branch creation, commit amendment, remote push, PR mutation, or native Stack mutation
- **AND** it SHALL NOT publish an unstacked suffix above the existing native Stack

#### Scenario: Required mode missing extension before submit mutation
- **GIVEN** `github.native_stacks = required`
- **AND** the gh-stack extension is missing or below the supported version
- **WHEN** submit/export begins
- **THEN** it SHALL fail before generated branch creation, commit amendment, remote push, PR mutation, or native Stack mutation
- **AND** the error SHALL provide an extension installation or upgrade command

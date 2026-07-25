## MODIFIED Requirements

### Requirement: Native Landing Safety Gate

When native integration is enabled, land SHALL inspect GitHub membership before executing any existing bottom-only or whole-stack mutation and SHALL refuse unsupported landing for a matching native Stack.

#### Scenario: Matching native bottom-only landing is refused

- **GIVEN** native integration is enabled
- **AND** every local PR belongs to one native Stack in the same complete bottom-to-top sequence
- **AND** the effective land style is `bottom-only`
- **WHEN** `stack-pr land` completes membership preflight
- **THEN** it SHALL return an actionable unsupported error before editing or merging a PR or mutating a branch
- **AND** the error SHALL identify the bottom PR URL for initiating the merge in GitHub's UI
- **AND** it SHALL explain that branchless-pr does not yet synchronize GitHub's server-rebased remaining branches locally

#### Scenario: Matching native whole-stack landing is refused

- **GIVEN** native integration is enabled
- **AND** every local PR belongs to one native Stack in the same complete bottom-to-top sequence
- **AND** the effective land style is `whole-stack`
- **WHEN** `stack-pr land` completes membership preflight
- **THEN** it SHALL return an actionable unsupported error before retargeting or merging a PR or mutating a branch
- **AND** the error SHALL identify the top PR URL for initiating whole-stack landing in GitHub's UI
- **AND** it SHALL explain that branchless-pr does not yet synchronize GitHub's server-side landing locally

#### Scenario: Native refusal does not dissolve membership

- **GIVEN** local membership exactly matches a native Stack
- **WHEN** `stack-pr land` refuses native landing
- **THEN** it SHALL NOT call the REST unstack endpoint
- **AND** it SHALL NOT run `gh pr merge`
- **AND** it SHALL NOT edit PR bases, checkout generated heads, rebase, push, fetch for cleanup, or delete branches

#### Scenario: Auto mode with unstacked PRs uses legacy landing

- **GIVEN** `github.native_stacks = auto`
- **AND** every local PR is unstacked
- **WHEN** `stack-pr land` completes preflight
- **THEN** it SHALL execute the existing land algorithm for the effective style

#### Scenario: Auto mode unavailable uses legacy landing

- **GIVEN** `github.native_stacks = auto`
- **AND** native Stack membership is unavailable for the repository
- **WHEN** `stack-pr land` completes preflight
- **THEN** it SHALL warn once
- **AND** it SHALL execute the existing land algorithm for the effective style

#### Scenario: Required mode rejects unstacked eligible stack

- **GIVEN** `github.native_stacks = required`
- **AND** an eligible multi-PR local stack is unstacked
- **WHEN** `stack-pr land` completes preflight
- **THEN** it SHALL fail before PR, branch, or merge mutation

#### Scenario: Drift blocks landing in every enabled mode

- **GIVEN** native integration is `auto` or `required`
- **AND** native membership conflicts with the local sequence
- **WHEN** `stack-pr land` completes preflight
- **THEN** it SHALL fail before PR, branch, or merge mutation
- **AND** it SHALL NOT fall back to legacy landing or automatically unstack the remote Stack

#### Scenario: Disabled mode preserves legacy landing

- **GIVEN** `github.native_stacks = off`
- **WHEN** `stack-pr land` runs
- **THEN** it SHALL use the existing land algorithm
- **AND** it SHALL NOT query or mutate native Stack membership

### Requirement: Native Landing Is Deferred Pending Synchronization

The system SHALL treat supported native landing and remote-to-local synchronization as follow-up behavior because the documented Stacks REST API provides no merge or rebase endpoint.

#### Scenario: Standard PR merge is not assumed stack-aware

- **GIVEN** the Stacks REST API exposes no merge operation
- **WHEN** branchless-pr handles a matching native Stack
- **THEN** it SHALL NOT assume `gh pr merge` provides a stable stacked-PR landing contract

#### Scenario: GitHub-side rebase or merge can stale local state

- **GIVEN** a user rebases or merges a native Stack through GitHub's UI
- **WHEN** GitHub rewrites generated remote branches
- **THEN** branchless-pr SHALL NOT claim that its local commit stack has been synchronized
- **AND** documentation SHALL describe the limitation and manual recovery requirement

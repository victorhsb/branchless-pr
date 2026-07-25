## ADDED Requirements

### Requirement: Native REST Representation

The system SHALL decode and validate the published pull-request membership and Stack resource schemas defined in `GITHUB_STACKS_REST_API.md` sections 5 and 6.

#### Scenario: Stacked pull request membership is decoded

- **GIVEN** a pull-request resource contains `stack.id`, `stack.number`, `stack.size`, `stack.position`, `stack.base.ref`, and `stack.base.sha`
- **WHEN** native membership is decoded
- **THEN** the system SHALL preserve the global ID and repository-scoped Stack number as distinct values
- **AND** it SHALL interpret position as 1-based bottom-to-top order
- **AND** it SHALL distinguish the PR's direct `base.ref` from the Stack's ultimate `stack.base.ref`

#### Scenario: Standalone pull request is decoded

- **GIVEN** a pull-request resource contains `stack: null`
- **WHEN** native membership is decoded
- **THEN** the system SHALL classify the PR as unstacked

#### Scenario: Complete Stack resource is decoded

- **GIVEN** a Stack resource contains `id`, `number`, `node_id`, `url`, `base.ref`, `open`, `created_at`, and `pull_requests`
- **WHEN** the Stack is decoded
- **THEN** the system SHALL preserve the `pull_requests` array in returned bottom-to-top order
- **AND** each member SHALL decode `number`, `state`, `draft`, nullable `merged_at`, `head.ref`, and `head.sha`
- **AND** the system SHALL NOT require `base.sha` on the Stack resource

#### Scenario: Unknown fields are tolerated

- **GIVEN** a pull-request membership or Stack response contains additional preview fields
- **WHEN** the response is decoded
- **THEN** the system SHALL ignore the unknown fields
- **AND** it SHALL continue to validate every documented required field

#### Scenario: Malformed or ambiguous resource fails closed

- **GIVEN** a response omits a required field, contains an impossible membership position, contains duplicate PR numbers, or cannot be decoded
- **WHEN** native state is processed
- **THEN** the system SHALL return a schema error
- **AND** it SHALL perform no native Stack write based on that response

#### Scenario: Merged and closed-unmerged members remain distinct

- **GIVEN** a Stack member has `state: "closed"`
- **WHEN** the member lifecycle is classified
- **THEN** `merged_at != null` SHALL classify the member as merged
- **AND** `merged_at == null` SHALL classify the member as closed but unmerged

### Requirement: Native REST Read Contract

The system SHALL use the canonical repository-scoped REST paths and complete remote resources needed for safe native reconciliation.

#### Scenario: Repository availability is probed safely

- **GIVEN** native integration is enabled
- **WHEN** repository support is probed
- **THEN** the system SHALL first confirm ordinary repository access
- **AND** it SHALL then query the repository-level Stack list endpoint
- **AND** only a Stack-list `404` after successful repository access SHALL be classified as feature unavailable

#### Scenario: Candidate pull requests are loaded

- **GIVEN** local entries already have PR numbers
- **WHEN** native reconciliation is planned
- **THEN** the system SHALL load each candidate PR from the repository
- **AND** it SHALL validate repository identity, state, head ref, direct base ref, and nullable Stack membership

#### Scenario: Complete referenced Stack is loaded

- **GIVEN** one or more candidate PRs contain Stack membership
- **WHEN** native reconciliation is planned
- **THEN** the system SHALL load every unique referenced Stack by its repository-scoped `stack.number`
- **AND** it SHALL NOT infer the complete ordered member list from membership `size` and `position`

#### Scenario: Filtered membership lookup

- **GIVEN** the system must rediscover a Stack containing a known PR
- **WHEN** it queries the Stack list endpoint with `pull_request={number}`
- **THEN** an empty array SHALL mean the PR is unstacked
- **AND** one result SHALL identify the Stack
- **AND** more than one result SHALL be treated as ambiguous membership

#### Scenario: Full Stack listing is paginated

- **GIVEN** an operation enumerates repository Stacks rather than using a filtered lookup
- **WHEN** the Stack list spans multiple pages
- **THEN** the system SHALL follow GitHub pagination until no next page remains
- **AND** it SHALL NOT treat one page as the complete repository set

### Requirement: Native REST Mutation Contract

The system SHALL create, append, and unstack native Stacks through the documented REST endpoints using JSON request bodies and repository-scoped identifiers.

#### Scenario: REST create request

- **GIVEN** reconciliation classifies an eligible sequence as `create`
- **WHEN** the native Stack is created
- **THEN** the system SHALL POST `{"pull_requests":[...]}` to `repos/{owner}/{repo}/stacks`
- **AND** the array SHALL contain 2 through 100 PR numbers in bottom-to-top order
- **AND** a successful response SHALL be `201` with the created Stack resource

#### Scenario: REST append request

- **GIVEN** reconciliation classifies an eligible suffix as `append`
- **WHEN** the native Stack is extended
- **THEN** the system SHALL POST `{"pull_requests":[...]}` to `repos/{owner}/{repo}/stacks/{stack_number}/add`
- **AND** the array SHALL contain only 1 through 100 new suffix PR numbers from the current top upward
- **AND** a successful response SHALL be `200` with the updated Stack resource

#### Scenario: REST partial unstack response

- **GIVEN** an unstack request leaves one or more server-locked members
- **WHEN** the system POSTs with no body to `repos/{owner}/{repo}/stacks/{stack_number}/unstack`
- **THEN** it SHALL accept `200`
- **AND** it SHALL decode and report the returned surviving Stack
- **AND** it SHALL NOT assume the Stack was dissolved

#### Scenario: REST dissolved unstack response

- **GIVEN** an unstack request removes every member
- **WHEN** the system POSTs with no body to `repos/{owner}/{repo}/stacks/{stack_number}/unstack`
- **THEN** it SHALL accept `204` with an empty body
- **AND** it SHALL classify the Stack as dissolved

#### Scenario: Unsupported structural edits are refused

- **GIVEN** the desired change requires insertion below the top, arbitrary removal, movement, reorder, replacement, rebase, or merge
- **WHEN** native reconciliation is planned
- **THEN** the system SHALL NOT emulate that operation with REST appends
- **AND** it SHALL report the operation as unsupported or conflicting

### Requirement: Native Write Concurrency and Recovery

The system SHALL treat native Stack mutations as non-idempotent read-modify-write operations and SHALL reconcile uncertain outcomes before any retry.

#### Scenario: Successful write response is verified

- **GIVEN** create or append returns a successful Stack resource
- **WHEN** the response is validated
- **THEN** its complete bottom-to-top PR sequence SHALL exactly equal the intended sequence
- **AND** the operation SHALL fail as conflicting if it does not

#### Scenario: Create outcome is uncertain

- **GIVEN** a create request fails after its server outcome becomes uncertain
- **WHEN** the system handles the failure
- **THEN** it SHALL query membership for a candidate PR and load any resulting complete Stack
- **AND** it SHALL accept an exact intended sequence as completed
- **AND** it SHALL NOT blindly repeat the create request

#### Scenario: Append outcome is uncertain

- **GIVEN** an append request fails after its server outcome becomes uncertain
- **WHEN** the system handles the failure
- **THEN** it SHALL reload the numbered Stack and recompute the ordered relationship
- **AND** it SHALL accept an exact intended sequence as completed
- **AND** it SHALL NOT blindly repeat the append request

#### Scenario: Unstack outcome is uncertain

- **GIVEN** an unstack request fails after its server outcome becomes uncertain
- **WHEN** the system handles the failure
- **THEN** it SHALL reload the numbered Stack
- **AND** it SHALL distinguish a dissolved Stack, a partial Stack, and an unverified result
- **AND** it SHALL NOT blindly repeat the unstack request

#### Scenario: Server-side branch rewrite is protected

- **GIVEN** GitHub rebases or retargets remaining Stack branches
- **WHEN** branchless-pr later updates a generated remote branch
- **THEN** it SHALL use the previously observed remote OID as a force-with-lease expectation
- **AND** it SHALL fail rather than overwrite an unexpected server-side rewrite

## MODIFIED Requirements

### Requirement: Native Stack Integration Mode

The system SHALL configure GitHub native Stack integration with `github.native_stacks` values `off`, `auto`, or `required`, and SHALL default to `off`.

#### Scenario: Integration disabled by default
- **GIVEN** `github.native_stacks` is unset or `off`
- **WHEN** a stack command runs
- **THEN** the command SHALL use legacy branchless-pr behavior
- **AND** it SHALL NOT call a GitHub native Stacks endpoint

#### Scenario: Auto mode uses supported native stacks
- **GIVEN** `github.native_stacks = auto`
- **AND** the repository supports GitHub native Stacks
- **WHEN** an eligible stack command runs
- **THEN** the command SHALL use the native behavior defined for that command

#### Scenario: Auto mode falls back when unavailable
- **GIVEN** `github.native_stacks = auto`
- **AND** ordinary repository access succeeds
- **AND** the repository-level GitHub Stacks endpoint reports that the feature is unavailable
- **WHEN** an eligible stack command runs
- **THEN** the command SHALL warn that native stacks are unavailable
- **AND** it SHALL use legacy behavior

#### Scenario: Required mode rejects unavailable feature
- **GIVEN** `github.native_stacks = required`
- **AND** the repository-level GitHub Stacks endpoint reports that the feature is unavailable
- **WHEN** an eligible stack command runs
- **THEN** the command SHALL fail before command-specific mutation

#### Scenario: Base GitHub CLI supplies REST transport
- **GIVEN** native integration is `auto` or `required`
- **WHEN** a native read or write is required
- **THEN** the system SHALL invoke the REST API through the base `gh api` command
- **AND** it SHALL NOT require the `github/gh-stack` extension

#### Scenario: Invalid mode rejected
- **GIVEN** `github.native_stacks` has a value other than `off`, `auto`, or `required`
- **WHEN** configuration is resolved
- **THEN** the command SHALL return a configuration error

### Requirement: Native Stack Eligibility

The system SHALL publish only branchless-pr stacks that satisfy the documented native Stack request, repository, ref-chain, and PR-state constraints while preserving the one-commit-per-PR invariant from `SPEC.md` section 5.

#### Scenario: Multi-PR same-repository stack is eligible
- **GIVEN** a branchless-pr stack contains between 2 and 100 PRs
- **AND** every PR head belongs to the repository in the endpoint path
- **AND** the bottom PR targets the ultimate Stack base
- **AND** every later PR directly targets the previous PR's head
- **AND** every PR to be added is open or draft, unmerged, not merge-queued, and not auto-merge-enabled
- **WHEN** native eligibility is evaluated
- **THEN** the stack SHALL be eligible for native reconciliation

#### Scenario: Single PR remains standalone
- **GIVEN** a branchless-pr stack contains exactly one PR
- **WHEN** native eligibility is evaluated in any mode
- **THEN** the PR SHALL remain a standalone PR
- **AND** the system SHALL NOT create a GitHub Stack

#### Scenario: Conservative aggregate limit in auto mode
- **GIVEN** `github.native_stacks = auto`
- **AND** a branchless-pr stack contains more than 100 PRs
- **WHEN** native eligibility is evaluated
- **THEN** the command SHALL warn that branchless-pr conservatively limits native Stacks to 100 total PRs
- **AND** it SHALL continue with legacy behavior

#### Scenario: Conservative aggregate limit in required mode
- **GIVEN** `github.native_stacks = required`
- **AND** a branchless-pr stack contains more than 100 PRs
- **WHEN** native eligibility is evaluated
- **THEN** the command SHALL fail before a native Stack write
- **AND** the error SHALL identify 100 as a branchless-pr client policy rather than a documented GitHub aggregate limit

#### Scenario: Cross-repository chain is ineligible
- **GIVEN** Stack PR heads do not all belong to the repository in the endpoint path
- **WHEN** native eligibility is evaluated
- **THEN** the system SHALL NOT attempt to create or extend a native Stack
- **AND** `auto` SHALL warn and use legacy behavior
- **AND** `required` SHALL fail

#### Scenario: Broken direct base chain is ineligible
- **GIVEN** the bottom PR does not target the ultimate base or a later PR does not target the preceding PR's head
- **WHEN** native eligibility is evaluated
- **THEN** the system SHALL fail before a native Stack write
- **AND** it SHALL report the mismatched PR and refs

#### Scenario: Ineligible PR lifecycle blocks addition
- **GIVEN** a PR to be created or appended is merged, closed-unmerged, merge-queued, or auto-merge-enabled
- **WHEN** native eligibility is evaluated
- **THEN** the system SHALL fail before a native Stack write
- **AND** it SHALL preserve the server response as authoritative if GitHub applies stricter validation

### Requirement: Non-Destructive Native Reconciliation

The system SHALL create or append native Stacks only for safe classifications, SHALL use the documented REST interface for writes, and SHALL NOT automatically replace a conflicting Stack.

#### Scenario: Create native Stack
- **GIVEN** reconciliation classifies an eligible stack as `create`
- **WHEN** reconciliation is applied
- **THEN** the system SHALL POST all local PR numbers bottom-to-top to `repos/{owner}/{repo}/stacks`
- **AND** it SHALL pass PR numbers rather than branch names
- **AND** it SHALL NOT modify PR lifecycle state as part of the Stack write

#### Scenario: Append native Stack
- **GIVEN** reconciliation classifies an eligible stack as `append`
- **WHEN** reconciliation is applied
- **THEN** the system SHALL POST only the unstacked local suffix to `repos/{owner}/{repo}/stacks/{stack_number}/add`
- **AND** it SHALL preserve suffix bottom-to-top order
- **AND** it SHALL NOT send the complete existing sequence

#### Scenario: REST write is verified
- **GIVEN** a create or append REST request returns successfully
- **WHEN** reconciliation validates the result
- **THEN** the complete returned or re-read remote PR sequence SHALL exactly equal the planned bottom-to-top sequence before the operation is considered successful

#### Scenario: Stack write does not own PR lifecycle
- **GIVEN** branchless-pr invokes a native Stack REST write
- **WHEN** the endpoint processes existing PR numbers
- **THEN** branchless-pr SHALL remain responsible for branch pushes, PR creation, PR title and body, reviewers, direct bases, and draft state

#### Scenario: Conflict is never auto-recreated
- **GIVEN** reconciliation classifies membership as `conflict`
- **WHEN** reconciliation is applied in `auto` or `required` mode
- **THEN** the command SHALL fail
- **AND** it SHALL NOT unstack, replace, reorder, or append the conflicting GitHub Stack

### Requirement: Native API Error Classification

The system SHALL preserve REST failure context and distinguish feature unavailability from authentication, authorization, missing resources, validation, rate limiting, transport uncertainty, and malformed responses.

#### Scenario: Documented endpoint unavailable
- **GIVEN** ordinary access to the repository succeeds
- **AND** the repository-level Stack list endpoint returns `404`
- **WHEN** availability is probed
- **THEN** the system SHALL classify native Stacks as unavailable

#### Scenario: Numbered Stack not found
- **GIVEN** a numbered Stack get, add, or unstack endpoint returns `404`
- **WHEN** the failure is classified
- **THEN** the system SHALL preserve the missing-resource context
- **AND** it SHALL NOT classify that response as repository-level feature unavailability without a separate availability probe

#### Scenario: Authentication or authorization error is not fallback
- **GIVEN** GitHub returns `401` or `403`
- **WHEN** native state is queried or mutated
- **THEN** the system SHALL return the underlying status and GitHub diagnostic
- **AND** it SHALL NOT silently classify the feature as unavailable

#### Scenario: Validation failure is not blindly retried
- **GIVEN** GitHub returns `422` for a create, append, or unstack request
- **WHEN** the failure is classified
- **THEN** the system SHALL preserve the status and GitHub message
- **AND** it SHALL re-read relevant PR and Stack state before any new plan
- **AND** it SHALL NOT repeat the same write blindly

#### Scenario: Rate limit is preserved
- **GIVEN** GitHub returns a rate-limit response
- **WHEN** the failure is classified
- **THEN** the system SHALL preserve the status and available retry diagnostics
- **AND** it SHALL NOT relabel the response as feature unavailable

#### Scenario: Transport or server failure after write is uncertain
- **GIVEN** a native write ends with a transport failure or `5xx` response
- **WHEN** the failure is classified
- **THEN** the system SHALL mark the write outcome as uncertain
- **AND** the next operation SHALL be read and reconciliation rather than an unconditional retry

#### Scenario: Malformed API response fails safely
- **GIVEN** a native Stack response cannot be decoded or violates required fields or ordering invariants
- **WHEN** the response is processed
- **THEN** the command SHALL return an error
- **AND** it SHALL perform no native Stack write based on that response

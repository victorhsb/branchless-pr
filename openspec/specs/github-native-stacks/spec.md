## ADDED Requirements

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
- **AND** the documented GitHub Stacks endpoint reports that the feature is unavailable
- **WHEN** an eligible stack command runs
- **THEN** the command SHALL warn that native stacks are unavailable
- **AND** it SHALL use legacy behavior

#### Scenario: Auto mode missing extension can skip prospective create
- **GIVEN** `github.native_stacks = auto`
- **AND** every local PR is unstacked or does not exist yet
- **AND** the gh-stack extension required to create native membership is missing or unsupported
- **WHEN** a command would otherwise create a native Stack
- **THEN** the command SHALL warn that native creation was skipped
- **AND** it MAY continue with legacy unstacked PR behavior

#### Scenario: Auto mode missing extension cannot skip append or unstack
- **GIVEN** `github.native_stacks = auto`
- **AND** one or more local PRs already belong to a native Stack
- **AND** the gh-stack extension required for append or unstack is missing or unsupported
- **WHEN** the command requires a native write to preserve consistency
- **THEN** the command SHALL fail before command-specific mutation
- **AND** it SHALL NOT fall back to behavior that leaves the existing native Stack inconsistent

#### Scenario: Required mode rejects unavailable feature
- **GIVEN** `github.native_stacks = required`
- **AND** the documented GitHub Stacks endpoint reports that the feature is unavailable or the gh-stack extension required for a native write is missing or unsupported
- **WHEN** an eligible stack command runs
- **THEN** the command SHALL fail before command-specific mutation

#### Scenario: Invalid mode rejected
- **GIVEN** `github.native_stacks` has a value other than `off`, `auto`, or `required`
- **WHEN** configuration is resolved
- **THEN** the command SHALL return a configuration error

### Requirement: Native Stack Eligibility
The system SHALL publish only branchless-pr stacks that satisfy GitHub's native Stack constraints while preserving the one-commit-per-PR invariant from `SPEC.md` section 5.

#### Scenario: Multi-PR same-repository stack is eligible
- **GIVEN** a branchless-pr stack contains between 2 and 100 PRs
- **AND** every PR belongs to the same repository
- **AND** the PR bases form the local bottom-to-top branch chain
- **WHEN** native eligibility is evaluated
- **THEN** the stack SHALL be eligible for native reconciliation

#### Scenario: Single PR remains standalone
- **GIVEN** a branchless-pr stack contains exactly one PR
- **WHEN** native eligibility is evaluated in any mode
- **THEN** the PR SHALL remain a standalone PR
- **AND** the system SHALL NOT create a GitHub Stack

#### Scenario: Auto mode leaves an oversized stack legacy
- **GIVEN** `github.native_stacks = auto`
- **AND** a branchless-pr stack contains more than 100 PRs
- **WHEN** native eligibility is evaluated
- **THEN** the command SHALL warn that the stack is ineligible
- **AND** it SHALL continue with legacy behavior

#### Scenario: Required mode rejects an oversized stack
- **GIVEN** `github.native_stacks = required`
- **AND** a branchless-pr stack contains more than 100 PRs
- **WHEN** native eligibility is evaluated
- **THEN** the command SHALL fail before a native Stack write

#### Scenario: Cross-repository chain is ineligible
- **GIVEN** stack PRs do not all belong to the same repository
- **WHEN** native eligibility is evaluated
- **THEN** the system SHALL NOT attempt to create or extend a native Stack
- **AND** `auto` SHALL warn and use legacy behavior
- **AND** `required` SHALL fail

### Requirement: Native Stack Reconciliation Classification
The system SHALL classify local and GitHub membership before writing a native Stack and SHALL preserve local PR order as authoritative.

#### Scenario: Unstacked PRs classify as create
- **GIVEN** every local PR is unstacked on GitHub
- **WHEN** an eligible local sequence is classified
- **THEN** the action SHALL be `create`
- **AND** the create payload SHALL list PR numbers bottom-to-top

#### Scenario: Exact sequence classifies as no-op
- **GIVEN** every local PR belongs to one GitHub Stack
- **AND** that Stack's complete PR sequence exactly equals the local bottom-to-top sequence
- **WHEN** membership is classified
- **THEN** the action SHALL be `noop`
- **AND** no Stack write SHALL occur

#### Scenario: Unstacked suffix classifies as append
- **GIVEN** one GitHub Stack's complete PR sequence is an exact proper prefix of the local sequence
- **AND** every local PR in the remaining suffix is unstacked
- **WHEN** membership is classified
- **THEN** the action SHALL be `append`
- **AND** only the suffix PR numbers SHALL be sent to the append endpoint in bottom-to-top order

#### Scenario: Remote sequence contains an extra PR
- **GIVEN** the local sequence is a proper prefix of the GitHub Stack sequence
- **WHEN** membership is classified
- **THEN** the action SHALL be `conflict`
- **AND** no Stack write SHALL occur

#### Scenario: Reordered or mixed membership conflicts
- **GIVEN** local PRs are reordered remotely, belong to multiple native Stacks, or contain a suffix PR already stacked elsewhere
- **WHEN** membership is classified
- **THEN** the action SHALL be `conflict`
- **AND** the error SHALL describe the local and remote membership sufficiently for manual resolution

### Requirement: Non-Destructive Native Reconciliation
The system SHALL create or append native Stacks only for safe classifications, SHALL use the gh-stack external-tool interface for writes, and SHALL NOT automatically replace a conflicting Stack.

#### Scenario: Create native Stack
- **GIVEN** reconciliation classifies an eligible stack as `create`
- **WHEN** reconciliation is applied
- **THEN** the system SHALL run `gh stack link <pr1> <pr2> ...` with all local PR numbers bottom-to-top
- **AND** it SHALL pass PR numbers rather than branch names
- **AND** it SHALL NOT pass `--open`

#### Scenario: Append native Stack
- **GIVEN** reconciliation classifies an eligible stack as `append`
- **WHEN** reconciliation is applied
- **THEN** the system SHALL run `gh stack link <stack-number> <suffix-pr1> ...`
- **AND** it SHALL pass only the unstacked local suffix as PR numbers
- **AND** it SHALL NOT pass `--open`

#### Scenario: Link write is verified
- **GIVEN** a create or append `gh stack link` command exits successfully
- **WHEN** reconciliation verifies the result
- **THEN** the system SHALL read the resulting Stack through the REST API
- **AND** the complete remote PR sequence SHALL exactly equal the planned bottom-to-top sequence before the operation is considered successful

#### Scenario: Link does not own PR lifecycle
- **GIVEN** branchless-pr invokes `gh stack link` for a native write
- **WHEN** the extension processes existing PR numbers
- **THEN** branchless-pr SHALL remain responsible for branch pushes, PR creation, PR title and body, reviewers, bases, and draft state

#### Scenario: Conflict is never auto-recreated
- **GIVEN** reconciliation classifies membership as `conflict`
- **WHEN** reconciliation is applied in `auto` or `required` mode
- **THEN** the command SHALL fail
- **AND** it SHALL NOT unstack, replace, reorder, or append the conflicting GitHub Stack

### Requirement: Native API Error Classification
The system SHALL distinguish feature unavailability from authentication, authorization, repository, transport, and malformed-response failures.

#### Scenario: Documented endpoint unavailable
- **GIVEN** ordinary access to the repository succeeds
- **AND** the documented Stacks endpoint returns its unsupported-feature response
- **WHEN** availability is probed
- **THEN** the system SHALL classify native Stacks as unavailable

#### Scenario: Authentication error is not fallback
- **GIVEN** GitHub authentication or authorization fails
- **WHEN** native state is queried
- **THEN** the system SHALL return the underlying error
- **AND** it SHALL NOT silently classify the feature as unavailable

#### Scenario: Extension reports native stacks unavailable
- **GIVEN** `gh stack link` or `gh stack unstack` exits with documented exit code 9
- **WHEN** native write failure is classified
- **THEN** the system SHALL classify GitHub native Stacks as unavailable
- **AND** `auto` SHALL warn and use its documented fallback
- **AND** `required` SHALL fail

#### Scenario: Extension API failure is not parsed from text
- **GIVEN** a gh-stack write command fails with an exit code other than the documented unavailable code
- **WHEN** the failure is classified
- **THEN** the system SHALL use the exit code and captured error as the failure contract
- **AND** it SHALL NOT depend on human-oriented status text for correctness

#### Scenario: Malformed API response fails safely
- **GIVEN** a native Stack response cannot be decoded or violates required ordering fields
- **WHEN** the response is processed
- **THEN** the command SHALL return an error
- **AND** it SHALL perform no native Stack write based on that response

### Requirement: Local Commit Metadata Remains Authoritative
Native integration SHALL NOT replace branchless-pr's commit-oriented local identity model.

#### Scenario: Successful native reconciliation preserves metadata
- **GIVEN** native reconciliation creates, appends, or confirms a GitHub Stack
- **WHEN** submit/export completes
- **THEN** every submitted commit SHALL retain its `stack-info` PR and generated-branch metadata
- **AND** subsequent local discovery SHALL continue to use `BASE..HEAD`

#### Scenario: Generic native Stack is not imported
- **GIVEN** a GitHub native Stack was not constructed from the current branchless-pr commit stack
- **WHEN** branchless-pr encounters it
- **THEN** branchless-pr SHALL NOT infer or rewrite local commits solely from that remote Stack

### Requirement: Explicit Go Port Behavior
GitHub native Stack integration SHALL be an explicit Go-port extension to the Python behavior documented in `SPEC.md`.

#### Scenario: Legacy compatibility when disabled
- **GIVEN** `github.native_stacks = off`
- **WHEN** submit, export, view, land, or abandon runs
- **THEN** behavior SHALL remain compatible with the non-native algorithms in `SPEC.md` sections 13 through 17

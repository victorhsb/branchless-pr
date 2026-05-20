## ADDED Requirements

### Requirement: Submit Receipt Request

The `stack-pr submit` command and its `export` alias SHALL support an opt-in receipt destination for real submit/export executions.

#### Scenario: Receipt flag is accepted on submit

- **WHEN** `stack-pr submit --receipt <destination>` is invoked without `--dry-run`
- **THEN** the command SHALL attempt to emit a submit operation receipt to `<destination>`

#### Scenario: Receipt flag is accepted on export alias

- **WHEN** `stack-pr export --receipt <destination>` is invoked without `--dry-run`
- **THEN** the command SHALL attempt to emit the same submit operation receipt as `stack-pr submit --receipt <destination>`

#### Scenario: Receipt disabled by default

- **WHEN** `stack-pr submit` is invoked without a receipt flag and without receipt configuration
- **THEN** the command SHALL NOT emit a receipt
- **AND** existing human output behavior SHALL remain unchanged

#### Scenario: Receipt destination values

- **WHEN** a receipt destination is provided
- **THEN** `off` SHALL disable receipt emission
- **AND** `-` SHALL emit one JSON receipt document on standard output
- **AND** any other value SHALL be interpreted as a filesystem path where the receipt JSON document is written

#### Scenario: Dry-run receipt rejected

- **WHEN** `stack-pr submit --dry-run --receipt <destination>` is invoked with a destination other than `off`
- **THEN** the command SHALL report a clear invocation error explaining that operation receipts are only available for real submit/export executions
- **AND** the command SHALL NOT perform submit/export mutations

### Requirement: Receipt Configuration

The CLI SHALL support `.stack-pr.cfg` configuration for default submit/export receipt behavior.

#### Scenario: Receipt config enables submit receipts

- **WHEN** `.stack-pr.cfg` contains `receipt.submit = <destination>`
- **AND** `stack-pr submit` is invoked without `--receipt`
- **THEN** the command SHALL use `<destination>` as the effective receipt destination

#### Scenario: Receipt config supports export alias

- **WHEN** `.stack-pr.cfg` contains `receipt.submit = <destination>`
- **AND** `stack-pr export` is invoked without `--receipt`
- **THEN** the command SHALL use `<destination>` as the effective receipt destination

#### Scenario: Receipt flag overrides config

- **WHEN** `.stack-pr.cfg` contains `receipt.submit = <configured-destination>`
- **AND** `stack-pr submit --receipt <flag-destination>` is invoked
- **THEN** the command SHALL use `<flag-destination>` as the effective receipt destination

#### Scenario: Receipt config default is off

- **WHEN** `.stack-pr.cfg` omits `receipt.submit`
- **THEN** the effective receipt destination SHALL be `off`

### Requirement: Receipt JSON Envelope

Each submit operation receipt SHALL be a single JSON object with a stable, versioned schema.

#### Scenario: Required receipt fields

- **WHEN** a submit operation receipt is emitted
- **THEN** the JSON object SHALL include `schema_version`, `command`, `status`, `side_effects`, `repo`, `stack`, and `operations`
- **AND** `schema_version` SHALL be a non-empty string
- **AND** `command` SHALL identify the invoked operation as `stack-pr submit` or `stack-pr export`
- **AND** `side_effects` SHALL be `true`

#### Scenario: Receipt status values

- **WHEN** a submit operation receipt is emitted
- **THEN** `status` SHALL be one of `ok`, `failed`, or `partial_failure`

#### Scenario: Repository context included

- **WHEN** a submit operation receipt is emitted
- **THEN** `repo` SHALL include the resolved repository root, original branch, remote, target, base, head, and branch-name template when those values are available

#### Scenario: Stack context included

- **WHEN** a submit operation receipt is emitted after stack discovery succeeds
- **THEN** `stack` SHALL include the stack size and per-entry commit SHA, title, head branch, base branch, and PR URL when known

#### Scenario: Stable JSON stdout mode

- **WHEN** the effective receipt destination is `-`
- **THEN** standard output SHALL contain exactly one valid JSON receipt document
- **AND** human progress output SHALL NOT be interleaved into standard output

### Requirement: Receipt Operation Entries

The receipt SHALL record high-value submit/export side effects in execution order.

#### Scenario: Successful side effects are recorded

- **WHEN** submit/export successfully completes a side-effecting operation
- **THEN** the receipt SHALL append an operation entry with `type`, `status`, and operation-specific details
- **AND** `status` SHALL be `ok`

#### Scenario: Failed operation is recorded

- **WHEN** submit/export fails during a side-effecting operation after receipt collection begins
- **THEN** the receipt SHALL append or update an operation entry for the failed operation
- **AND** that operation entry SHALL have `status` set to `failed`
- **AND** that operation entry SHALL include an error message

#### Scenario: Partial failure status

- **WHEN** a receipt contains at least one successful side-effect operation followed by a failed operation
- **THEN** the top-level receipt `status` SHALL be `partial_failure`

#### Scenario: Failed status without completed side effects

- **WHEN** submit/export fails before any side-effect operation succeeds and a receipt can be emitted
- **THEN** the top-level receipt `status` SHALL be `failed`

#### Scenario: Successful status

- **WHEN** submit/export completes successfully
- **THEN** the top-level receipt `status` SHALL be `ok`

### Requirement: Submit Operation Coverage

Submit/export receipts SHALL record the main categories of submit/export side effects.

#### Scenario: Branch operations recorded

- **WHEN** submit/export creates or checks out generated stack branches
- **THEN** the receipt SHALL record branch operation entries identifying the affected branch names and commits when available

#### Scenario: Push operations recorded

- **WHEN** submit/export force-pushes generated stack branches
- **THEN** the receipt SHALL record push operation entries identifying the remote and branch names

#### Scenario: Pull request operations recorded

- **WHEN** submit/export creates or updates a pull request
- **THEN** the receipt SHALL record pull request operation entries identifying the commit, head branch, base branch, title, and PR URL when available

#### Scenario: Metadata operations recorded

- **WHEN** submit/export amends commits to add `stack-info` metadata
- **THEN** the receipt SHALL record metadata operation entries identifying the affected head branch and commit when available

#### Scenario: Cleanup warnings recorded

- **WHEN** submit/export performs a best-effort cleanup operation that fails without failing the command
- **THEN** the receipt SHALL record a warning operation entry identifying the cleanup operation and error message

### Requirement: Recovery Recording

Submit/export receipts SHALL record best-effort recovery attempts made after handled errors.

#### Scenario: Original branch recovery recorded

- **WHEN** submit/export fails and recovery attempts to checkout the original branch
- **THEN** the receipt SHALL record a recovery operation entry with the target original branch and success or failure status

#### Scenario: Stash recovery recorded

- **WHEN** submit/export fails after an auto-stash was created and recovery attempts to pop the stash
- **THEN** the receipt SHALL record a recovery operation entry with success or failure status

### Requirement: Receipt Emission Failure

Receipt emission failures SHALL be visible to callers.

#### Scenario: Receipt file write fails

- **WHEN** the effective receipt destination is a filesystem path
- **AND** the command cannot write the receipt JSON document to that path
- **THEN** the command SHALL return a non-zero error explaining that receipt emission failed

#### Scenario: Receipt disabled suppresses receipt write failures

- **WHEN** the effective receipt destination is `off`
- **THEN** the command SHALL NOT attempt to write a receipt

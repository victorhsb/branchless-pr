## ADDED Requirements

### Requirement: Automatic stash lifecycle recovery

When non-dry-run submit/export creates an automatic stash, the command SHALL
attempt to restore that stash before returning from every subsequent success or
failure path. This requirement implements the Python-compatible `finally`
semantics documented in `SPEC.md` section 8.

#### Scenario: Clean validation fails after stashing

- **GIVEN** submit/export created an automatic stash
- **WHEN** the post-stash clean working-tree validation fails
- **THEN** the command SHALL attempt to restore the automatic stash before returning the validation error

#### Scenario: Target validation fails after stashing

- **GIVEN** submit/export created an automatic stash
- **WHEN** remote target validation fails before command dispatch
- **THEN** the command SHALL attempt to restore the automatic stash before returning the target error

#### Scenario: Merge-base deduction fails after stashing

- **GIVEN** submit/export created an automatic stash
- **WHEN** merge-base deduction fails before command dispatch
- **THEN** the command SHALL attempt to restore the automatic stash before returning the merge-base error

#### Scenario: Command execution succeeds

- **GIVEN** submit/export created an automatic stash and pre-run initialization succeeded
- **WHEN** command execution returns successfully
- **THEN** the command SHALL restore the automatic stash before the invocation returns

#### Scenario: Command execution fails

- **GIVEN** submit/export created an automatic stash and pre-run initialization succeeded
- **WHEN** command execution returns an error or panics
- **THEN** recovery SHALL attempt to restore the automatic stash before the invocation returns

#### Scenario: Pre-run restoration fails

- **GIVEN** post-stash pre-run initialization fails
- **WHEN** restoring the automatic stash also fails
- **THEN** the returned error SHALL preserve both the initialization failure and the restoration failure

#### Scenario: No automatic stash was created

- **GIVEN** the working tree was clean or dry-run prevented automatic stashing
- **WHEN** initialization or command execution returns
- **THEN** the command SHALL NOT attempt to pop a stash

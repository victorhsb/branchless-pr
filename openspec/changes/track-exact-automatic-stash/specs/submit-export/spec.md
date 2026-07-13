## ADDED Requirements

### Requirement: Exact automatic stash identity

Non-dry-run submit/export SHALL determine automatic stash creation from Git
reference state rather than human-facing command output, SHALL retain the exact
created stash identity, and SHALL restore and remove only that stash. This is an
explicit Go-port safety improvement over Python `stack-pr`'s boolean and top-pop
behavior documented in `SPEC.md` section 8.

#### Scenario: Clean working tree

- **GIVEN** no tracked working-tree changes exist
- **WHEN** automatic stash creation runs
- **THEN** the command SHALL record no automatic stash regardless of Git's human-facing output
- **AND** recovery SHALL NOT apply or drop any existing user stash

#### Scenario: Localized or unexpected stash output

- **GIVEN** `git stash push` emits localized, empty, or unexpected human-facing output
- **WHEN** Git changes `refs/stash` to a new stash commit
- **THEN** the command SHALL record the new commit as the automatic stash

#### Scenario: Pre-existing user stash

- **GIVEN** a user stash exists before automatic stash creation
- **WHEN** the automatic stash is successfully restored
- **THEN** the pre-existing user stash SHALL remain unchanged

#### Scenario: Newer user stash

- **GIVEN** another stash entry is added above the recorded automatic stash before recovery
- **WHEN** automatic stash restoration runs
- **THEN** the command SHALL apply and remove the recorded automatic stash
- **AND** the newer user stash SHALL remain unchanged

#### Scenario: Successful exact restoration

- **GIVEN** a recorded automatic stash exists and applies cleanly
- **WHEN** recovery runs
- **THEN** the exact automatic stash changes SHALL be restored to the working tree
- **AND** only its matching stash reflog entry SHALL be removed
- **AND** invocation state SHALL no longer record an automatic stash

#### Scenario: Restoration conflict

- **GIVEN** the recorded automatic stash conflicts with current working-tree state
- **WHEN** recovery attempts to apply it
- **THEN** recovery SHALL return an actionable error identifying the automatic stash
- **AND** the automatic stash entry SHALL remain available for manual recovery
- **AND** invocation state SHALL retain the automatic stash identity

#### Scenario: Automatic stash entry is missing

- **GIVEN** invocation state records an automatic stash whose reflog entry no longer exists
- **WHEN** recovery runs
- **THEN** recovery SHALL return an actionable error
- **AND** it SHALL NOT apply or remove a different stash entry

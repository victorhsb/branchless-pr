# git-operation-safety Specification

## Purpose

Define layout-aware discovery and detection of active Git operations before
commands mutate repository state, so that rebase, merge, cherry-pick, and
sequencer state is detected correctly from repository subdirectories, linked
worktrees, submodules, and repositories with a separate Git directory.

## Requirements

### Requirement: Repository-layout-aware operation paths

The system SHALL ask Git to resolve operation-state paths for the repository
context instead of assuming metadata exists in a `.git` directory. This is an
explicit Go-port safety decision that differs from the literal `.git` path
checks documented for Python `stack-pr` in `SPEC.md` section 11.

#### Scenario: Invocation from a repository subdirectory

- **GIVEN** the current directory is nested below a repository worktree root
- **WHEN** the system checks for an active Git operation without an explicit repository directory
- **THEN** it SHALL inspect the operation path belonging to that repository

#### Scenario: Linked worktree

- **GIVEN** the invocation context is a linked Git worktree
- **WHEN** the system checks for an active Git operation
- **THEN** it SHALL inspect the operation path belonging to that worktree

#### Scenario: Submodule worktree

- **GIVEN** the invocation context is a checked-out Git submodule
- **WHEN** the system checks for an active Git operation
- **THEN** it SHALL inspect the submodule's operation path

#### Scenario: Separate Git directory

- **GIVEN** a worktree stores its repository metadata in a separate Git directory
- **WHEN** the system checks for an active Git operation
- **THEN** it SHALL inspect the operation path resolved by Git for that repository

### Requirement: Active operation detection

The system SHALL report an active operation when Git's resolved metadata contains
a rebase, merge, cherry-pick, or sequencer marker already recognized by the
port, and SHALL report no active operation when none of those markers exists.

#### Scenario: Rebase marker

- **GIVEN** Git resolves an existing `rebase-merge` or `rebase-apply` path
- **WHEN** rebase state is checked
- **THEN** the system SHALL report that a rebase is active

#### Scenario: Merge marker

- **GIVEN** Git resolves an existing `MERGE_HEAD` path
- **WHEN** merge state is checked
- **THEN** the system SHALL report that a merge is active

#### Scenario: Cherry-pick marker

- **GIVEN** Git resolves an existing `CHERRY_PICK_HEAD` path
- **WHEN** cherry-pick state is checked
- **THEN** the system SHALL report that a cherry-pick is active

#### Scenario: Sequencer marker

- **GIVEN** Git resolves an existing `sequencer/todo` path
- **WHEN** cherry-pick or aggregate sequencer state is checked
- **THEN** the system SHALL report that an operation is active

#### Scenario: No operation markers

- **GIVEN** none of the recognized paths resolved by Git exists
- **WHEN** aggregate sequencer state is checked
- **THEN** the system SHALL report that no operation is active

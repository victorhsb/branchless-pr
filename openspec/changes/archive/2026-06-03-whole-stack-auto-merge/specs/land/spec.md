## MODIFIED Requirements

### Requirement: Whole-Stack Merge

When `whole-stack` style is active, the command SHALL queue the entire stack for GitHub-managed landing by retargeting the tip PR directly to the target branch and enabling GitHub rebase auto-merge for the tip PR. This mode SHALL require that the repository target branch uses GitHub merge queue.

#### Scenario: Repository must allow rebase merges

- **WHEN** `whole-stack` style is active
- **THEN** the command SHALL query the repository's merge settings via the GitHub API
- **AND** if `rebaseMergeAllowed` is false, the command SHALL print an error message explaining that rebase merges are disabled and exit without mutating state
- **AND** if the API call fails, the command SHALL propagate the error

#### Scenario: Target branch must use merge queue

- **WHEN** `whole-stack` style is active
- **THEN** the command SHALL verify that GitHub merge queue is enabled for the repository target branch before retargeting the tip PR
- **AND** if merge queue is not enabled, the command SHALL print `ERROR: --whole-stack only works for repositories with merge queue enabled`
- **AND** the command SHALL exit without editing PR bases, merging PRs, fetching, checking out branches, deleting local branches, rebasing local branches, or pushing branches

#### Scenario: Retarget tip PR to target

- **WHEN** the repository allows rebase merges
- **AND** the repository target branch has merge queue enabled
- **THEN** the command SHALL set the tip PR's base branch to the target branch with:
  - `gh pr edit <tip-pr> -B <target>`

#### Scenario: Queue rebase merge for the tip PR

- **WHEN** the tip PR is retargeted to the target branch
- **THEN** the command SHALL run:
  - `gh pr merge <tip-pr> --rebase --auto`
- **AND** GitHub SHALL own waiting for required checks, approvals, merge-queue grouping, and final merge
- **AND** the command SHALL NOT poll GitHub for CI or merge completion

#### Scenario: No per-entry rebase or push needed

- **WHEN** the tip PR has been queued for rebase auto-merge
- **THEN** the command SHALL NOT checkout, rebase, or force-push any remaining stack branches
- **AND** all commits from the stack SHALL be expected to appear linearly on the target branch when GitHub completes the queued merge

### Requirement: Cleanup and Restoration

After bottom-only landing, the command SHALL restore the local repository to a clean state. After whole-stack merge-queue scheduling, the command SHALL restore the original branch but SHALL NOT perform cleanup that assumes the stack has already landed.

#### Scenario: Original branch restored

- **WHEN** landing and rebasing complete
- **THEN** the command SHALL checkout the original recorded branch

#### Scenario: Original branch restored after whole-stack queue scheduling

- **WHEN** whole-stack merge-queue scheduling succeeds
- **THEN** the command SHALL checkout the original recorded branch
- **AND** the command SHALL print a message that whole-stack landing has been queued for the tip PR

#### Scenario: Local generated branches deleted after completed merge

- **WHEN** bottom-only landing and rebasing complete
- **THEN** all local stack generated branches SHALL be deleted

#### Scenario: Local generated branches retained after queued whole-stack merge

- **WHEN** whole-stack merge-queue scheduling succeeds
- **THEN** the command SHALL NOT delete local stack generated branches

#### Scenario: Local target branch rebased after completed merge

- **WHEN** bottom-only landing and rebasing complete
- **AND** a local branch exists whose name matches the remote target branch (e.g. `main`)
- **THEN** that branch SHALL be rebased onto `REMOTE/TARGET`

#### Scenario: Local target branch not rebased after queued whole-stack merge

- **WHEN** whole-stack merge-queue scheduling succeeds
- **THEN** the command SHALL NOT rebase the local target branch onto `REMOTE/TARGET`

#### Scenario: Original branch rebased after completed merge

- **WHEN** bottom-only cleanup completes
- **THEN** the original branch SHALL be rebased onto `REMOTE/TARGET`

#### Scenario: Original branch not rebased after queued whole-stack merge

- **WHEN** whole-stack merge-queue scheduling succeeds
- **THEN** the command SHALL NOT rebase the original branch onto `REMOTE/TARGET`

#### Scenario: No fetch after queued whole-stack merge

- **WHEN** whole-stack merge-queue scheduling succeeds
- **THEN** the command SHALL NOT fetch the remote after scheduling the queued merge

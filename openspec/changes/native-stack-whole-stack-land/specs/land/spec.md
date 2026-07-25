## MODIFIED Requirements

### Requirement: Whole-Stack Merge

When `whole-stack` style is active, the command SHALL queue the entire stack for GitHub-managed landing. For legacy (non-native) stacks, the command SHALL retarget the tip PR directly to the target branch and enable GitHub rebase auto-merge for the tip PR. For matching native Stacks, the command SHALL skip the base retarget and queue the tip PR directly, relying on GitHub's native Stack cascade. Both modes SHALL require that the repository target branch uses GitHub merge queue.

#### Scenario: Repository must allow rebase merges

- **WHEN** `whole-stack` style is active
- **THEN** the command SHALL query the repository's merge settings via the GitHub API
- **AND** if `rebaseMergeAllowed` is false, the command SHALL print an error message explaining that rebase merges are disabled and exit without mutating state
- **AND** if the API call fails, the command SHALL propagate the error

#### Scenario: Target branch must use merge queue

- **WHEN** `whole-stack` style is active
- **THEN** the command SHALL verify that GitHub merge queue is enabled for the repository target branch before retargeting or queuing the tip PR
- **AND** if merge queue is not enabled, the command SHALL print `ERROR: --whole-stack only works for repositories with merge queue enabled`
- **AND** the command SHALL exit without editing PR bases, merging PRs, fetching, checking out branches, deleting local branches, rebasing local branches, or pushing branches

#### Scenario: Retarget tip PR to target (legacy stacks)

- **GIVEN** the stack is not linked to a GitHub native Stack
- **WHEN** the repository allows rebase merges
- **AND** the repository target branch has merge queue enabled
- **THEN** the command SHALL set the tip PR's base branch to the target branch with:
  - `gh pr edit <tip-pr> -B <target>`

#### Scenario: Skip base retarget for native stacks

- **GIVEN** the stack is linked to a matching GitHub native Stack (action is `noop`)
- **WHEN** the repository allows rebase merges
- **AND** the repository target branch has merge queue enabled
- **THEN** the command SHALL NOT call `gh pr edit -B` on any PR
- **AND** the command SHALL NOT retarget the tip PR's base to the target branch
- **AND** the command SHALL rely on GitHub's native Stack mechanism to cascade merges from bottom to top

#### Scenario: Queue rebase merge for the tip PR

- **WHEN** the tip PR has been retargeted (legacy) or the stack is native (no retarget needed)
- **THEN** the command SHALL run:
  - `gh pr merge <tip-pr> --rebase --auto`
- **AND** GitHub SHALL own waiting for required checks, approvals, merge-queue grouping, and final merge
- **AND** the command SHALL NOT poll GitHub for CI or merge completion

#### Scenario: No per-entry rebase or push needed

- **WHEN** the tip PR has been queued for rebase auto-merge
- **THEN** the command SHALL NOT checkout, rebase, or force-push any remaining stack branches
- **AND** all commits from the stack SHALL be expected to appear linearly on the target branch when GitHub completes the queued merge

### Requirement: Native Landing Safety Gate

When native integration is enabled, land SHALL inspect GitHub membership before executing any existing bottom-only or whole-stack mutation. For a matching native Stack, whole-stack landing SHALL be allowed (the command queues the tip PR without base retargeting). Bottom-only landing for a matching native Stack SHALL be refused.

#### Scenario: Matching native bottom-only landing is refused

- **GIVEN** native integration is enabled
- **AND** every local PR belongs to one native Stack in the same complete bottom-to-top sequence
- **AND** the effective land style is `bottom-only`
- **WHEN** `stack-pr land` completes membership preflight
- **THEN** it SHALL return an actionable unsupported error before editing or merging a PR or mutating a branch
- **AND** the error SHALL identify the bottom PR URL for initiating the merge in GitHub's UI
- **AND** it SHALL explain that bottom-only landing requires client-side base edits that GitHub rejects for stacked PRs

#### Scenario: Matching native whole-stack landing is allowed

- **GIVEN** native integration is enabled
- **AND** every local PR belongs to one native Stack in the same complete bottom-to-top sequence
- **AND** the effective land style is `whole-stack`
- **WHEN** `stack-pr land` completes membership preflight
- **THEN** it SHALL proceed to the whole-stack landing flow
- **AND** it SHALL NOT return a refusal error
- **AND** it SHALL NOT call `gh pr edit -B` on any PR

#### Scenario: Native whole-stack with append is refused

- **GIVEN** native integration is enabled
- **AND** some local PRs belong to a native Stack and others are unstacked suffix (append)
- **AND** the effective land style is `whole-stack`
- **WHEN** `stack-pr land` completes membership preflight
- **THEN** it SHALL return an actionable unsupported error
- **AND** the error SHALL identify the top PR URL for initiating whole-stack landing in GitHub's UI

#### Scenario: Native refusal does not dissolve membership

- **GIVEN** local membership exactly matches a native Stack
- **WHEN** `stack-pr land` refuses native landing (bottom-only or append)
- **THEN** it SHALL NOT run `gh stack unstack`
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
- **AND** it SHALL NOT query native membership or invoke the gh-stack extension

### Requirement: Native Landing Is Deferred Pending Synchronization

The system SHALL treat supported native whole-stack landing as a queue-only operation: the command schedules the merge via `gh pr merge --rebase --auto` and returns immediately. The system SHALL NOT assume that `gh pr merge` provides a stacked-PR landing contract for bottom-only mode.

#### Scenario: Whole-stack native landing queues the tip PR

- **GIVEN** a matching native Stack and `whole-stack` style
- **WHEN** `stack-pr land` queues the tip PR for rebase auto-merge
- **THEN** GitHub's merge queue SHALL own the cascade from bottom to top
- **AND** the command SHALL NOT poll for merge completion
- **AND** the command SHALL print a message explaining the landing was queued

#### Scenario: Bottom-only native landing is not assumed stack-aware

- **GIVEN** the installed gh-stack extension exposes no supported merge command for bottom-only
- **WHEN** branchless-pr handles a matching native Stack with `bottom-only` style
- **THEN** it SHALL NOT assume `gh pr merge --squash` provides a stable stacked-PR landing contract
- **AND** it SHALL refuse with an actionable error

#### Scenario: GitHub-side rebase or merge can stale local state

- **GIVEN** a user rebases or merges a native Stack through GitHub's UI or via queued merge
- **WHEN** GitHub rewrites generated remote branches
- **THEN** branchless-pr SHALL NOT claim that its local commit stack has been synchronized
- **AND** documentation SHALL describe the limitation and manual recovery requirement

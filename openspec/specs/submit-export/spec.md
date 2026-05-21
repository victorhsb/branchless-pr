# Submit/Export Algorithm

## Purpose

Define the canonical behavior of `stack-pr submit` and its `stack-pr export` alias for creating or updating a stack of GitHub pull requests from an ordered set of local commits.

Submit/export mutates local Git state (branch creation, rebasing, commit amending, stashing), pushes generated branches to the remote, creates or updates GitHub PRs, adds `stack-info` metadata to commit messages, and manages cross-links between PRs. Dry-run mode previews these actions without mutation. Operation receipts provide opt-in machine-readable records of completed side effects.

## Requirements

### Requirement: Pre-flight Checks

Before any mutation, submit/export SHALL validate repository prerequisites.

#### Scenario: Rebase in progress blocks submit

- **WHEN** a rebase is detected as in-progress (`.git/rebase-merge` or `.git/rebase-apply` exists)
- **THEN** the command SHALL print an error and exit with status 1
- **AND** no mutation SHALL occur

#### Scenario: Current branch is recorded

- **WHEN** submit/export begins
- **THEN** the current branch name SHALL be recorded for later restoration

#### Scenario: Optional base fast-forward

- **WHEN** the local base is an ancestor of `REMOTE/TARGET`
- **AND** `REMOTE/TARGET` is an ancestor of `HEAD`
- **AND** the base hash differs from `REMOTE/TARGET`
- **THEN** the command SHALL run `git rebase REMOTE/TARGET base`
- **AND** the command SHALL checkout the original branch afterward

### Requirement: Stack Discovery and Validation

Submit/export SHALL discover the commit stack, validate it, and reject empty stacks.

#### Scenario: Stack loaded from base..head

- **WHEN** submit/export runs
- **THEN** the stack SHALL be loaded from commits in `base..head`
- **AND** stack entries SHALL be ordered oldest-to-newest internally

#### Scenario: Empty stack rejected

- **WHEN** the discovered stack contains no commits
- **THEN** the command SHALL print `Empty stack!` and return without further action

#### Scenario: Draft bitmask validation

- **WHEN** `--draft-bitmask` is provided
- **THEN** its length SHALL match the stack length
- **AND** each character SHALL be `0` or `1`
- **AND** on mismatch, the command SHALL print a validation message and return without submitting

- **WHEN** `--draft` is set together with a draft bitmask
- **THEN** `--draft` SHALL override the bitmask for all created PRs

### Requirement: Local Branch Initialization

Submit/export SHALL create or ensure local generated branches for each stack entry before remote interaction.

#### Scenario: Generated branch assignment

- **WHEN** local branches are initialized
- **THEN** the remote SHALL be fetched and pruned
- **AND** entries missing metadata heads SHALL receive generated head branches from the branch-name template
- **AND** for each entry, the command SHALL run `git checkout <commit-id> -B <entry.head>`

#### Scenario: Existing metadata head preserved

- **WHEN** a stack entry already has a head branch in its metadata
- **THEN** that head branch SHALL be reused

### Requirement: Base Branch Computation

Submit/export SHALL compute base branches for every stack entry so each PR targets the correct branch.

#### Scenario: Bottom entry targets remote target

- **WHEN** base branches are computed for a non-empty stack
- **THEN** the first (bottom) entry's base SHALL be the remote target branch (normally `main`)

#### Scenario: Higher entries target previous head

- **WHEN** base branches are computed
- **THEN** each subsequent entry's base SHALL be the previous entry's head branch

#### Scenario: Current branch rebase detection

- **WHEN** base branches are computed
- **THEN** the command SHALL determine whether the original current branch needs rebasing
- **AND** this SHALL be true if the top stack branch is an ancestor of the current branch

### Requirement: Existing PR Safeguard

Before creating new PRs, submit/export SHALL temporarily protect existing PRs from spurious merge notifications.

#### Scenario: Existing PRs marked temporary draft

- **WHEN** an entry has an existing PR
- **THEN** the command SHALL query the PR `isDraft` status via GitHub
- **AND** if the PR is not draft
- **THEN** the command SHALL mark it draft with `gh pr ready <pr> --undo`
- **AND** record `is_tmp_draft=True` for later restoration

#### Scenario: Existing PR base reset to target

- **WHEN** an entry has an existing PR
- **THEN** the command SHALL set its base branch to the target using `gh pr edit <pr> -B <target>`
- **AND** this prevents spurious merge notifications while the stack is repushed

### Requirement: Force-push Stack Branches

Submit/export SHALL push all generated head branches to the remote in a single batch.

#### Scenario: Single batch force-push

- **WHEN** local branches are initialized and existing PRs are safeguarded
- **THEN** the command SHALL force-push all stack head branches in one command:
  - `git push -f <remote> <head1>:<head1> <head2>:<head2> ...`

### Requirement: PR Creation for New Entries

Submit/export SHALL create a GitHub pull request for every stack entry that does not already have one.

#### Scenario: New PR creation

- **WHEN** a stack entry lacks PR metadata
- **THEN** the command SHALL create a PR with:
  - `gh pr create -B <base> -H <head> -t <commit-title> -F - [--reviewer <reviewer>] [--draft]`
- **AND** the body input SHALL be the full commit message
- **AND** the PR reference SHALL be parsed as the last whitespace-separated token of command output

#### Scenario: Draft from draft flag or bitmask

- **WHEN** a new PR is created and `--draft` is set
- **THEN** the PR SHALL be created as draft

- **WHEN** a new PR is created and a draft bitmask is provided
- **THEN** the PR SHALL be created as draft if the corresponding bitmask character is `1`

### Requirement: Stack Verification

After initial PR creation, submit/export SHALL verify that stack metadata and GitHub state are consistent.

#### Scenario: Verify after creation

- **WHEN** PR creation completes
- **THEN** the command SHALL run stack verification against GitHub
- **AND** each entry's PR, head, and base SHALL be present and match GitHub state

### Requirement: Metadata Addition

Submit/export SHALL amend commits to embed `stack-info` metadata so subsequent commands can reconstruct the stack.

#### Scenario: First commit amended without rebase

- **WHEN** metadata is added and no rebase is needed for the current branch
- **THEN** the first changed commit's head branch SHALL be checked out
- **AND** the `stack-info: PR: <pr-url>, branch: <head>` line SHALL be appended to its commit message
- **AND** the commit SHALL be amended with `git commit --amend -F -`

#### Scenario: Subsequent commits rebased and amended

- **WHEN** metadata is added for a later stack entry
- **THEN** if a prior commit was amended
- **AND** the entry's branch SHALL be rebased onto its base using `git rebase <base> <head> --committer-date-is-author-date`
- **AND** then the `stack-info` line SHALL be appended and amended

#### Scenario: Rebase cascades after first amendment

- **WHEN** one commit has been amended
- **THEN** all subsequent entries SHALL require rebasing before amendment

### Requirement: Final Push and Cross-linking

After metadata is embedded, submit/export SHALL force-push again and update all PR descriptions with cross-links.

#### Scenario: Second force-push after metadata

- **WHEN** metadata amendment completes
- **THEN** all stack head branches SHALL be force-pushed again to the remote

#### Scenario: Cross-links added for multi-PR stacks

- **WHEN** the stack has more than one PR
- **THEN** each PR body SHALL receive a stacked-PRs table of contents newest-to-oldest
- **AND** the current PR SHALL be marked with `__->__`
- **AND** the TOC SHALL be followed by the delimiter `--- --- ---`

#### Scenario: No cross-links for single-PR stack

- **WHEN** the stack contains exactly one PR
- **THEN** no table of contents SHALL be generated

#### Scenario: PR body construction

- **WHEN** a PR body is constructed
- **THEN** the PR title SHALL be the commit title
- **AND** the first line (title) SHALL be stripped from the commit message body
- **AND** the `stack-info` metadata line SHALL be stripped
- **AND** for multi-PR stacks, the body content SHALL start with `### <title>` followed by the stripped commit body

#### Scenario: Keep-body preserves existing content

- **WHEN** `--keep-body` is set
- **THEN** the existing PR body SHALL be fetched
- **AND** content after the delimiter `--- --- ---` SHALL be preserved instead of regenerating the body

#### Scenario: PR title, body, and base updated

- **WHEN** cross-links are added
- **THEN** each PR SHALL be updated with:
  - `gh pr edit <pr> -t <title> -F - -B <base>`

### Requirement: Cleanup and Restoration

Submit/export SHALL restore repository state after mutations.

#### Scenario: Temporary draft restored

- **WHEN** existing PRs were marked temporary draft during submission
- **THEN** after cross-linking completes
- **AND** those PRs SHALL be restored to ready state with `gh pr ready <pr>`

#### Scenario: Original branch restored

- **WHEN** cleanup begins
- **AND** if the current branch needs rebasing
- **THEN** it SHALL be rebased onto the top stack branch with `git rebase <top_branch> <current_branch> --committer-date-is-author-date`
- **AND** otherwise the original branch SHALL be checked out directly

#### Scenario: Local generated branches deleted

- **WHEN** cleanup completes
- **THEN** all local generated branches SHALL be deleted with `git branch -D ...`
- **AND** deletion errors SHALL be ignored (check=False)

#### Scenario: Post-export tips printed

- **WHEN** post-export tips are enabled
- **THEN** the command SHALL print guidance for the user after submission

### Requirement: Dry Run Behavior

Submit/export dry-run mode SHALL preview actions without any mutation.

#### Scenario: Dry-run flag accepted

- **WHEN** `stack-pr submit --dry-run` or `stack-pr export --dry-run` is invoked with otherwise valid options
- **THEN** the command SHALL execute dry-run behavior instead of real submit/export behavior

#### Scenario: Dry-run prints plan

- **WHEN** dry-run mode is invoked for a non-empty stack
- **THEN** output SHALL include each stack entry in stack order showing commit title, generated head branch, computed base branch, and whether the PR would be created or updated
- **AND** entries for new PRs SHALL show the draft state that would be used
- **AND** entries requiring metadata SHALL indicate metadata would be added

#### Scenario: Dry-run empty stack

- **WHEN** dry-run mode is invoked for an empty stack
- **THEN** output SHALL report that the stack is empty

#### Scenario: Dry-run no-changes note

- **WHEN** dry-run mode completes successfully
- **THEN** output SHALL clearly state that no local Git changes, remote pushes, or GitHub PR changes were made

#### Scenario: Dry-run mutation safety

- **WHEN** dry-run mode is invoked
- **THEN** the command SHALL NOT checkout generated branches, rebase, amend commits, create or delete local branches, save or pop a stash
- **AND** the command SHALL NOT push branches to the remote
- **AND** the command SHALL NOT create or edit PRs or change draft state

#### Scenario: Dry-run validation

- **WHEN** dry-run mode is invoked
- **THEN** it SHALL validate the draft bitmask and compute head/base branches using the same rules as real submit/export
- **AND** it SHALL fail the clean-repository check if tracked files have changes
- **AND** it SHALL NOT auto-stash changes

#### Scenario: Non-dry-run behavior preserved

- **WHEN** `stack-pr submit` or `stack-pr export` is invoked without `--dry-run`
- **THEN** the command SHALL perform full submit/export mutations as specified

### Requirement: Operation Receipts

Submit/export SHALL support opt-in machine-readable receipts for real executions.

#### Scenario: Receipt flag accepted on submit and export

- **WHEN** `stack-pr submit --receipt <destination>` or `stack-pr export --receipt <destination>` is invoked without `--dry-run`
- **THEN** the command SHALL attempt to emit a submit operation receipt

#### Scenario: Receipt disabled by default

- **WHEN** submit/export is invoked without a receipt flag and without receipt configuration
- **THEN** the command SHALL NOT emit a receipt
- **AND** existing human output SHALL remain unchanged

#### Scenario: Receipt destination values

- **WHEN** a receipt destination is provided
- **THEN** `off` SHALL disable receipt emission
- **AND** `-` SHALL emit one JSON document on standard output
- **AND** any other value SHALL be interpreted as a filesystem path

#### Scenario: Dry-run receipt rejected

- **WHEN** `--dry-run` and `--receipt <destination>` (other than `off`) are both provided
- **THEN** the command SHALL report an invocation error explaining receipts are only available for real executions
- **AND** the command SHALL NOT perform mutations

#### Scenario: Receipt configuration in .stack-pr.cfg

- **WHEN** `.stack-pr.cfg` contains `receipt.submit = <destination>`
- **THEN** submit/export SHALL use that destination unless `--receipt` overrides it
- **AND** the default when omitted SHALL be `off`

#### Scenario: Receipt JSON envelope

- **WHEN** a receipt is emitted
- **THEN** it SHALL be a single JSON object with fields:
  - `schema_version` (non-empty string)
  - `command` (`stack-pr submit` or `stack-pr export`)
  - `status` (`ok`, `failed`, or `partial_failure`)
  - `side_effects` (`true`)
  - `repo` (repository root, original branch, remote, target, base, head, template when available)
  - `stack` (size, per-entry commit SHA, title, head branch, base branch, PR URL when known)
  - `operations` (array of operation entries)

#### Scenario: Receipt operation entries

- **WHEN** a side-effecting operation completes successfully
- **THEN** the receipt SHALL append an entry with `type`, `status: ok`, and operation-specific details

#### Scenario: Receipt failure recording

- **WHEN** a side-effecting operation fails after receipt collection begins
- **THEN** the receipt SHALL append or update an entry with `status: failed` and an error message

#### Scenario: Receipt partial failure status

- **WHEN** at least one operation succeeds followed by a failed operation
- **THEN** the top-level `status` SHALL be `partial_failure`

#### Scenario: Receipt successful status

- **WHEN** submit/export completes successfully
- **THEN** the top-level `status` SHALL be `ok`

#### Scenario: Receipt operation coverage

- **WHEN** branch, push, PR, metadata, or cleanup operations occur
- **THEN** the receipt SHALL record entries identifying the affected branches, remotes, PRs, commits, or error messages

#### Scenario: Receipt recovery recording

- **WHEN** submit/export fails and recovery attempts original-branch checkout or stash pop
- **THEN** the receipt SHALL record recovery operation entries with success or failure status

#### Scenario: Receipt emission failure

- **WHEN** the effective receipt destination is a filesystem path and writing fails
- **THEN** the command SHALL return a non-zero error explaining receipt emission failed

#### Scenario: Disabled receipt suppresses errors

- **WHEN** the effective receipt destination is `off`
- **THEN** the command SHALL NOT attempt to write a receipt


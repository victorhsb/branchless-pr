## MODIFIED Requirements

### Requirement: Experimental Submit Engine Gate
Submit/export SHALL use the current submit/export algorithm by default and SHALL use the optimized submit/export engine only when an experimental feature gate opts in.

#### Scenario: Default submit/export path remains legacy

- **WHEN** `STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE` is not set to `1`
- **AND** `.stack-pr.cfg` does not set `submit.experimental_engine = true`
- **AND** a user runs `submit` or the `export` alias
- **THEN** the command SHALL use the current submit/export implementation path
- **AND** the optimized no-op skip behavior introduced by this change SHALL NOT be required on that invocation

#### Scenario: Experimental submit/export engine enabled by env flag

- **WHEN** `STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE=1`
- **AND** a user runs `submit` or the `export` alias
- **THEN** the command SHALL use the optimized submit/export engine
- **AND** the optimized engine SHALL preserve the same final local Git, remote branch, and GitHub PR state as the current submit/export algorithm

#### Scenario: Experimental submit/export engine enabled by config

- **WHEN** `.stack-pr.cfg` sets `submit.experimental_engine = true`
- **AND** a user runs `submit` or the `export` alias
- **THEN** the command SHALL use the optimized submit/export engine
- **AND** the optimized engine SHALL preserve the same final local Git, remote branch, and GitHub PR state as the current submit/export algorithm

#### Scenario: Dry-run uses the selected engine

- **WHEN** a user runs `submit --dry-run` or `export --dry-run`
- **THEN** dry-run planning SHALL use the same submit/export engine selection rule as the corresponding non-dry-run command
- **AND** dry-run SHALL remain free of local Git mutations, remote pushes, and GitHub PR writes

### Requirement: Existing PR Safeguard
When the experimental submit/export engine is enabled, submit/export SHALL temporarily protect existing PRs from spurious merge notifications while avoiding redundant GitHub mutations before creating new PRs.

#### Scenario: Existing PRs marked temporary draft only when needed

- **WHEN** an entry has an existing PR
- **THEN** the command SHALL determine the PR `isDraft` status via GitHub state available to submit/export
- **AND** if the PR is not draft
- **THEN** the command SHALL mark it draft with `gh pr ready <pr> --undo`
- **AND** record `is_tmp_draft=True` for later restoration

#### Scenario: Existing draft PRs do not need temporary draft mutation

- **WHEN** an entry has an existing PR
- **AND** the PR is already draft
- **THEN** the command SHALL NOT call `gh pr ready <pr> --undo` for that PR
- **AND** the PR SHALL NOT be recorded for ready-state restoration solely because it was already draft

#### Scenario: Existing PR base reset to target only when needed

- **WHEN** an entry has an existing PR
- **THEN** the command SHALL ensure its base branch is the target before stack branches are repushed
- **AND** if the PR base branch already equals the target
- **THEN** the command SHALL NOT call `gh pr edit <pr> -B <target>` for that temporary reset
- **AND** this prevents spurious merge notifications while avoiding no-op base edits

#### Scenario: Existing PR base changed to target when needed

- **WHEN** an entry has an existing PR
- **AND** the PR base branch differs from the target
- **THEN** the command SHALL set its base branch to the target using `gh pr edit <pr> -B <target>`

### Requirement: Final Push and Cross-linking
When the experimental submit/export engine is enabled, submit/export SHALL publish changed branch tips and update PR descriptions with cross-links while avoiding no-op pushes and PR edits after metadata is embedded.

#### Scenario: Second force-push after metadata changes

- **WHEN** metadata amendment or metadata-driven rebasing changes one or more stack head branch tips
- **THEN** all stack head branches SHALL be force-pushed again to the remote

#### Scenario: Second force-push skipped when metadata is unchanged

- **WHEN** no commit metadata was amended
- **AND** no metadata-driven rebasing changed stack head branch tips
- **THEN** submit/export SHALL NOT perform the second batch force-push

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
- **THEN** the existing PR body SHALL be fetched or reused from GitHub state available to submit/export
- **AND** content after the delimiter `--- --- ---` SHALL be preserved instead of regenerating the body

#### Scenario: PR title, body, and base updated when changed

- **WHEN** cross-links are added
- **AND** the desired PR title, body, or base branch differs from the current GitHub PR state
- **THEN** the PR SHALL be updated with:
  - `gh pr edit <pr> -t <title> -F - -B <base>`

#### Scenario: PR edit skipped when title, body, and base already match

- **WHEN** cross-links are added
- **AND** the desired PR title, body, and base branch already match the current GitHub PR state
- **THEN** submit/export SHALL NOT call `gh pr edit` for that PR

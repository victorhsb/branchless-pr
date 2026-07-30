---
title: Submit/Export
status: stable
---

# Submit/Export

## Overview

Define the canonical behavior of `stack-pr submit` and its `stack-pr export` alias for creating or updating a stack of GitHub pull requests from an ordered set of local commits.

Submit/export mutates local Git state (branch creation, rebasing, commit amending, stashing), pushes generated branches to the remote, creates or updates GitHub PRs, adds `stack-info` metadata to commit messages, and manages cross-links between PRs. Dry-run mode previews these actions without mutation. Operation receipts provide opt-in machine-readable records of completed side effects.

## Behavior

### Pre-flight checks

Before any mutation, submit/export validates repository prerequisites.

- A rebase is detected as in-progress (`.git/rebase-merge` or `.git/rebase-apply` exists) → print an error, exit with status 1, and perform no mutation.
- Submit/export begins → record the current branch name for later restoration.
- Local base is an ancestor of `REMOTE/TARGET` and `REMOTE/TARGET` is an ancestor of `HEAD` and the base hash differs from `REMOTE/TARGET` → run `git rebase REMOTE/TARGET base`, then check out the original branch afterward.

### Stack discovery and validation

- The stack is loaded from commits in `base..head`, ordered oldest-to-newest internally.
- Discovered stack contains no commits → print `Empty stack!` and return without further action.
- `--draft-bitmask` is provided → its length must match the stack length and each character must be `0` or `1`; on mismatch, print a validation message and return without submitting.
- `--draft` is set together with a draft bitmask → `--draft` overrides the bitmask for all created PRs.

### Local branch initialization

Submit/export creates or ensures local generated branches for each stack entry before remote interaction, without requiring a worktree checkout for every entry.

- Local branches are initialized → the remote is fetched and pruned.
- Entry is missing a metadata head → receive a generated head branch from the branch-name template.
- Each entry → ensure the local branch `<entry.head>` points at `<commit-id>`; initialization never requires checking out each stack entry.
- Entry already has a head branch in its metadata → reuse that head branch, and reset the corresponding local branch to the entry commit before the first batch force-push.
- Initialization preserves the current worktree branch unless a later submit/export step explicitly checks out or rebases a branch for metadata amendment, restoration, or cleanup.
- A generated head branch is the currently checked-out branch and already points at the corresponding stack commit → treat that branch as already initialized instead of force-updating it.
- A generated head branch is the currently checked-out branch and does not point at the corresponding stack commit → fail with an actionable error asking the user to switch to a non-generated branch before retrying.

### Base branch computation

- First (bottom) entry of a non-empty stack → base is the remote target branch (normally `main`).
- Each subsequent entry → base is the previous entry's head branch.
- Base branches are computed → determine whether the original current branch needs rebasing; true if the top stack branch is an ancestor of the current branch.

### Experimental submit engine gate

Submit/export uses the current submit/export algorithm by default; the optimized submit/export engine is used only when an experimental feature gate opts in.

- `STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE` is not set to `1` and `.stack-pr.cfg` does not set `submit.experimental_engine = true` → use the current submit/export implementation path; the optimized no-op skip behavior introduced by this change is not required on that invocation.
- `STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE=1` → use the optimized submit/export engine, which preserves the same final local Git, remote branch, and GitHub PR state as the current submit/export algorithm.
- `.stack-pr.cfg` sets `submit.experimental_engine = true` → use the optimized submit/export engine, which preserves the same final local Git, remote branch, and GitHub PR state as the current submit/export algorithm.
- `submit --dry-run` or `export --dry-run` → dry-run planning uses the same submit/export engine selection rule as the corresponding non-dry-run command, and dry-run remains free of local Git mutations, remote pushes, and GitHub PR writes.

### Existing PR safeguard

When the experimental submit/export engine is enabled, submit/export temporarily protects existing PRs from spurious merge notifications while avoiding redundant GitHub mutations before creating new PRs.

- Entry has an existing PR → determine the PR `isDraft` status via GitHub state available to submit/export; if the PR is not draft, mark it draft with `gh pr ready <pr> --undo` and record `is_tmp_draft=True` for later restoration.
- Existing PR is already draft → do not call `gh pr ready <pr> --undo` for that PR, and do not record the PR for ready-state restoration solely because it was already draft.
- Entry has an existing PR → ensure its base branch is the target before stack branches are repushed; if the PR base branch already equals the target, do not call `gh pr edit <pr> -B <target>` for that temporary reset (prevents spurious merge notifications while avoiding no-op base edits).
- Existing PR base branch differs from the target → set its base branch to the target using `gh pr edit <pr> -B <target>`.

### Force-push stack branches

- Local branches are initialized and existing PRs are safeguarded → force-push all stack head branches in one command: `git push -f <remote> <head1>:<head1> <head2>:<head2> ...`

### PR creation for new entries

- Stack entry lacks PR metadata → create a PR with `gh pr create -B <base> -H <head> -t <commit-title> -F - [--reviewer <reviewer>] [--draft]`; the body input is the full commit message; the PR reference is parsed as the last whitespace-separated token of command output.
- New PR is created and `--draft` is set → create the PR as draft.
- New PR is created and a draft bitmask is provided → create the PR as draft if the corresponding bitmask character is `1`.

### Stack verification

- PR creation completes → run stack verification against GitHub; each entry's PR, head, and base must be present and match GitHub state.

### Metadata addition

Submit/export amends commits to embed `stack-info` metadata so subsequent commands can reconstruct the stack.

- Metadata is added and no rebase is needed for the current branch → check out the first changed commit's head branch, append the `stack-info: PR: <pr-url>, branch: <head>` line to its commit message (separated from the commit title/body by at least one blank line), and amend with `git commit --amend -F -`.
- Metadata is added for a later stack entry and a prior commit was amended → rebase the entry's branch onto its base using `git rebase <base> <head> --committer-date-is-author-date`, then append the `stack-info` line and amend.
- One commit has been amended → all subsequent entries require rebasing before amendment.

### Final push and cross-linking

When the experimental submit/export engine is enabled, submit/export publishes changed branch tips and updates PR descriptions with cross-links while avoiding no-op pushes and PR edits after metadata is embedded.

- Metadata amendment or metadata-driven rebasing changes one or more stack head branch tips → force-push all stack head branches again to the remote.
- No commit metadata was amended and no metadata-driven rebasing changed stack head branch tips → do not perform the second batch force-push.
- Stack has more than one PR → each PR body receives a stacked-PRs table of contents newest-to-oldest, with the current PR marked with `__->__`, followed by the delimiter `--- --- ---`.
- Stack contains exactly one PR → no table of contents is generated.
- PR body is constructed → the PR title is the commit title; the first line (title) is stripped from the commit message body; the `stack-info` metadata line is stripped; for multi-PR stacks, the body content starts with `### <title>` followed by the stripped commit body.
- `--keep-body` is set → fetch or reuse the existing PR body from GitHub state available to submit/export, and preserve content after the delimiter `--- --- ---` instead of regenerating the body.
- Desired PR title, body, or base branch differs from the current GitHub PR state → update the PR with `gh pr edit <pr> -t <title> -F - -B <base>`.
- Desired PR title, body, and base branch already match the current GitHub PR state → do not call `gh pr edit` for that PR.

### Cleanup and restoration

- Existing PRs were marked temporary draft during submission → after cross-linking completes, restore those PRs to ready state with `gh pr ready <pr>`.
- Current branch needs rebasing → rebase it onto the top stack branch with `git rebase <top_branch> <current_branch> --committer-date-is-author-date`; otherwise check out the original branch directly.
- Cleanup completes → delete all local generated branches with `git branch -D ...`, ignoring deletion errors (check=False).
- Post-export tips are enabled → print guidance for the user after submission.

### Automatic stash lifecycle recovery

When non-dry-run submit/export creates an automatic stash, it attempts to restore that stash before returning from every subsequent success or failure path. This implements the Python-compatible `finally` semantics documented in `SPEC.md` section 8.

| Point of return | Behavior |
|-----------------|----------|
| Post-stash clean working-tree validation fails | attempt to restore the automatic stash before returning the validation error |
| Remote target validation fails before command dispatch | attempt to restore the automatic stash before returning the target error |
| Merge-base deduction fails before command dispatch | attempt to restore the automatic stash before returning the merge-base error |
| Command execution returns successfully | restore the automatic stash before the invocation returns |
| Command execution returns an error or panics | recovery attempts to restore the automatic stash before the invocation returns |
| Post-stash pre-run initialization fails and restoring the automatic stash also fails | the returned error preserves both the initialization failure and the restoration failure |
| Working tree was clean or dry-run prevented automatic stashing | do not attempt to pop a stash |

### Exact automatic stash identity

Non-dry-run submit/export determines automatic stash creation from Git reference state rather than human-facing command output, retains the exact created stash identity, and restores and removes only that stash. This is an explicit Go-port safety improvement over Python `stack-pr`'s boolean and top-pop behavior documented in `SPEC.md` section 8.

- No tracked working-tree changes exist → record no automatic stash regardless of Git's human-facing output; recovery never applies or drops any existing user stash.
- `git stash push` emits localized, empty, or unexpected human-facing output and Git changes `refs/stash` to a new stash commit → record the new commit as the automatic stash.
- A user stash exists before automatic stash creation → after the automatic stash is successfully restored, the pre-existing user stash remains unchanged.
- Another stash entry is added above the recorded automatic stash before recovery → apply and remove the recorded automatic stash; the newer user stash remains unchanged.
- A recorded automatic stash exists and applies cleanly → restore the exact automatic stash changes to the working tree, remove only its matching stash reflog entry, and clear the automatic stash from invocation state.
- The recorded automatic stash conflicts with current working-tree state → return an actionable error identifying the automatic stash, leave the automatic stash entry available for manual recovery, and retain the automatic stash identity in invocation state.
- Invocation state records an automatic stash whose reflog entry no longer exists → return an actionable error; never apply or remove a different stash entry.

### Native Stack reconciliation

When native integration is enabled, submit/export reconciles the final submitted PR chain with GitHub after ordinary PR and branch publication succeeds.

- Native integration is enabled for an eligible stack → after final branch pushes, commit metadata amendment, PR title/body/base updates, and temporary draft restoration, reload and validate candidate PR state and native membership, and apply only a `create`, `append`, or `noop` result.
- Either the current submit engine or the optimized submit engine reaches its final remote phase → both engines use the same native reconciliation rules and produce the same final GitHub Stack membership.

| Native membership state | Behavior |
|-------------------------|----------|
| Final eligible PR chain is entirely unstacked | POST every existing PR number bottom-to-top to `repos/{owner}/{repo}/stacks`; require a `201` Stack response with the exact resulting complete sequence |
| Remote native sequence is an exact prefix of the final local PR sequence and the local suffix PRs are unstacked | POST only the suffix PR numbers to `repos/{owner}/{repo}/stacks/{stack_number}/add`; require a `200` Stack response with the exact resulting complete sequence |
| Native membership exactly matches the final local PR sequence | no native Stack write occurs |
| Ordinary submit/export effects have completed and native membership is conflicting | return an error without changing native membership; the error states that earlier PR or branch updates may already have completed |
| A native create or append request fails with an uncertain server outcome | read current native membership and the complete affected Stack; accept an exact intended sequence as success; report unchanged, divergent, or unverified state without blindly repeating the write |

- A multi-PR stack is published as a GitHub native Stack → the existing stacked-PR table of contents and delimiter remain present in the finalized PR bodies.

### Native Stack push lease safety

When native mode is active, submit/export avoids overwriting server-side Stack rebases or concurrent remote updates with an unconditional force push.

- A generated remote branch existed when submit preflight read its head OID → the force-update push requires the remote branch still to equal the observed OID.
- GitHub or another actor updates a generated remote branch after preflight → the push with the observed lease fails without overwriting the newer remote head, and native reconciliation does not run.
- A generated remote branch did not exist at preflight → the first push of that branch requires that the branch is still absent.

### Native Stack availability preflight

Required native integration fails before submit-specific mutation when repository support is unavailable, and native writes use the documented REST endpoints through the base GitHub CLI.

- `github.native_stacks = required` and GitHub native Stacks is unavailable for the repository → fail before generated branch creation, commit amendment, remote push, PR mutation, or native Stack mutation.
- `github.native_stacks = auto` and GitHub native Stacks is unavailable for the repository → warn once and execute the legacy submit/export algorithm without native reconciliation.
- GitHub native Stacks is available and the final PR chain is eligible for create or append → proceed using `gh api`; never inspect, require, install, or upgrade the `github/gh-stack` extension.

### Dry-run behavior

`stack-pr submit --dry-run` and `stack-pr export --dry-run` preview submit/export actions without any local Git mutation, remote push, or GitHub write. The full dry-run contract — shared guarantees, plan output, validation, receipt unavailability, and the native stack dry-run plan — is specified in [Dry-run](dry-run.md).

### Operation receipts

Submit/export supports opt-in machine-readable receipts for real executions. Each receipt is a single JSON object with a stable, versioned schema so callers can inspect completed side effects, failures, and recovery attempts.

#### Receipt request and configuration

- `stack-pr submit --receipt <destination>` is invoked without `--dry-run` → attempt to emit a submit operation receipt to `<destination>`.
- `stack-pr export --receipt <destination>` is invoked without `--dry-run` → emit the same submit operation receipt as `stack-pr submit --receipt <destination>`.
- No receipt flag and no receipt configuration → emit no receipt; existing human output remains unchanged.

| Receipt destination | Behavior |
|---------------------|----------|
| `off` | disable receipt emission |
| `-` | emit one JSON document on standard output |
| any other value | interpret as a filesystem path |

- `--dry-run` and `--receipt <destination>` (other than `off`) are both provided → report an invocation error explaining receipts are only available for real executions, and perform no mutations.
- `.stack-pr.cfg` contains `receipt.submit = <destination>` → use that destination for both `submit` and `export` unless `--receipt` overrides it; the default when omitted is `off`.

#### Receipt JSON envelope

- A receipt is a single JSON object with fields:

| Field | Content |
|-------|---------|
| `schema_version` | non-empty string |
| `command` | `stack-pr submit` or `stack-pr export` |
| `status` | `ok`, `failed`, or `partial_failure` |
| `side_effects` | `true` |
| `repo` | repository root, original branch, remote, target, base, head, template when available |
| `stack` | size, per-entry commit SHA, title, head branch, base branch, PR URL when known |
| `operations` | array of operation entries |

- A receipt is emitted after stack discovery succeeds → `stack` includes the stack size and per-entry data as described above.
- The effective receipt destination is `-` → standard output contains exactly one valid JSON receipt document; human progress output is not interleaved into standard output.

#### Receipt operation entries

The receipt records high-value submit/export side effects in execution order.

- Side-effecting operation completes successfully → append an entry with `type`, `status: ok`, and operation-specific details.
- Side-effecting operation fails after receipt collection begins → append or update an entry with `status: failed` and an error message.
- At least one operation succeeds followed by a failed operation → top-level `status` is `partial_failure`.
- Submit/export fails before any side-effect operation succeeds and a receipt can be emitted → top-level `status` is `failed`.
- Submit/export completes successfully → top-level `status` is `ok`.
- Generated stack branches are created or checked out → record branch operation entries identifying the affected branch names and commits when available.
- Generated stack branches are force-pushed → record push operation entries identifying the remote and branch names.
- A pull request is created or updated → record pull request operation entries identifying the commit, head branch, base branch, title, and PR URL when available.
- Commits are amended to add `stack-info` metadata → record metadata operation entries identifying the affected head branch and commit when available.
- A best-effort cleanup operation fails without failing the command → record a warning operation entry identifying the cleanup operation and error message.

#### Recovery recording

- Submit/export fails and recovery attempts original-branch checkout → record a recovery operation entry with the target original branch and success or failure status.
- Submit/export fails after an auto-stash was created and recovery attempts to pop the stash → record a recovery operation entry with success or failure status.

#### Receipt emission failure

- Effective receipt destination is a filesystem path and writing fails → return a non-zero error explaining receipt emission failed.
- Effective receipt destination is `off` → never attempt to write a receipt.

#### Native Stack receipt operations

When native integration is enabled, receipts record the outcome of native Stack planning and reconciliation. All rows below assume receipt emission is enabled.

| Native reconciliation outcome | Receipt content |
|-------------------------------|-----------------|
| Submit/export creates a GitHub native Stack | `ok` operation containing the action `create`, Stack number, and ordered PR numbers |
| Submit/export appends PRs to a GitHub native Stack | `ok` operation containing the action `append`, Stack number, and appended PR numbers |
| Native reconciliation is a no-op | `ok` operation containing the action `noop` and Stack number |

- Auto mode falls back because native Stacks is unavailable or the stack is ineligible and submit/export completes through the legacy path → record the fallback reason.
- Native classification or mutation fails after earlier submit operations succeeded → include a failed native Stack operation, and the top-level receipt status is `partial_failure`.

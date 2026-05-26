## Context

Stack metadata is stored in commit messages as `stack-info: PR: <pr-url>, branch: <head>`. Submit/export can add this metadata while creating or updating a stack, but users can also have an existing PR whose local commit never received metadata. Running the whole submit flow is too broad for that repair because submit may create branches, push branches, create or edit PRs, update bases, and cross-link bodies.

The new command is a recovery tool: attach one explicitly selected existing PR to the current local `HEAD` commit and stop. Root `SPEC.md` is deprecated for this repo, so OpenSpec requirements are the behavioral source for this change.

## Goals / Non-Goals

**Goals:**

- Add `bpr fix --pr <number> [--replace] [--dry-run]`.
- Repair only `HEAD` by adding or replacing local `stack-info` metadata using PR data from `gh pr view`.
- Keep the command local-only: no branch creation, no branch reset, no push, no PR create/edit/retarget.
- Warn, but do not fail, when the selected PR head SHA differs from local `HEAD`.
- Require a clean working tree and block when Git has an in-progress sequencer operation.
- Provide dry-run output and advisory stack-readiness warnings.
- Add agent prompt guidance after the command contract is implemented.

**Non-Goals:**

- No automatic PR discovery in the first version.
- No support for repairing arbitrary non-HEAD commits.
- No auto-stash support.
- No remote push or GitHub PR mutation.
- No metadata format change to include base branch.
- No root `SPEC.md` update.

## Decisions

### Make `fix` a first-class command

`fix` is separate from `submit` because it has a narrower safety contract. Submit/export reconciles the full stack with GitHub; fix only amends local `HEAD` metadata from a manually named PR.

Alternative considered: `submit --fix --pr <number>`. This would mix recovery behavior into a command whose normal meaning includes remote pushes and PR writes, making the local-only boundary less obvious.

### Require explicit PR selection

The command requires `--pr <number>` and does not infer a PR from branch names, commit SHA, or open PR search. This avoids attaching incorrect metadata to a commit.

Alternative considered: automatic discovery. It can be added later when there is a clear, unambiguous matching rule.

### Amend `HEAD` directly and bypass stack discovery for the repair

The repair target is always `HEAD`. The command should read the current commit message, inspect or replace any existing metadata line, and call the existing amend path with the new message unless `--dry-run` is set.

Stack discovery is used only after the repair attempt for advisory readiness warnings. Those warnings must not determine whether the local repair succeeds.

Alternative considered: discover the full stack before fixing. That would make the recovery path depend on stack state that may be broken while the user is repairing metadata.

### Local-only side effects

`fix` may mutate local Git only by amending `HEAD`. It must not create/reset local branches, push, create PRs, edit PRs, or retarget PRs. After a successful repair, the output tells the user to run `bpr submit` to publish and reconcile the stack.

Alternative considered: force-updating the PR branch. The user explicitly wants pushing to remain the responsibility of submit.

### Existing metadata is protected by default

If `HEAD` already has metadata for the same PR and head branch, the command reports that the commit is already fixed and makes no change. If metadata exists but differs from the requested PR/head branch, the command refuses unless `--replace` is set.

Alternative considered: always overwrite. That risks corrupting an already valid stack relationship.

### PR head mismatch is a warning

The command should request `headRefOid` from `gh pr view` and compare it to local `HEAD`. A mismatch is printed as a warning but does not block the amend because the user chose a manual repair command.

Alternative considered: fail on mismatch. That is safer but too strict for recovery cases where the local commit is intentionally being prepared for a later submit.

### Preflight stays conservative

`fix` requires a clean working tree and blocks when any Git sequencer operation is in progress, including rebase, merge, or cherry-pick. It does not auto-stash.

The command still uses the shared repo and `gh` preflight. It should not need branch-name-template generation or any generated branch assignment.

### Agent guidance is part of the same change

Agent prompt updates belong in the same OpenSpec change because they describe how agents should use the new command. Tasks can sequence this after the command behavior is implemented and tested.

## Risks / Trade-offs

- Wrong manual PR number -> The command can attach wrong metadata. Mitigation: require explicit `--pr`, show PR URL/head/HEAD details, warn on SHA mismatch, and protect existing metadata by default.
- Advisory stack-readiness check fails for unrelated repo state -> Mitigation: print a warning and keep the repair result successful if the amend succeeded.
- `headRefOid` is not currently loaded by the PR wrapper -> Mitigation: extend the existing `gh pr view` wrapper or add a narrow fix-specific PR view helper; do not introduce a GitHub SDK.
- Sequencer detection may be incomplete today -> Mitigation: add explicit helpers for rebase, merge, and cherry-pick state and cover them with tests.
- Agent prompt JSON schema changes -> Mitigation: update supported topic requirements and bump affected prompt identifiers if the schema or semantics require it.

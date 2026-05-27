## 1. Command Plumbing and Preflight

- [x] 1.1 Register a new `fix` Cobra subcommand with `--pr`, `--replace`, and `--dry-run` flags.
- [x] 1.2 Add invocation policy support so `fix` uses normal repo and `gh` preflight, requires a clean tree, and does not use auto-stash.
- [x] 1.3 Add or extend Git state helpers to detect rebase, merge, and cherry-pick operations in progress.
- [x] 1.4 Block `fix` with an actionable error when any supported Git operation is in progress.

## 2. PR and Commit Metadata Repair

- [x] 2.1 Extend the PR inspection path to load `url`, `number`, `headRefName`, `baseRefName`, and `headRefOid` for the selected `--pr`.
- [x] 2.2 Add a `HEAD` commit-message read path that can detect, append, and replace the existing `stack-info` line.
- [x] 2.3 Implement missing-metadata repair by amending `HEAD` with `stack-info: PR: <pr-url>, branch: <head-branch>`.
- [x] 2.4 Implement already-fixed handling for matching metadata without amending the commit.
- [x] 2.5 Implement refusal for different existing metadata unless `--replace` is set.
- [x] 2.6 Implement `--replace` metadata replacement for different existing metadata.
- [x] 2.7 Print a warning, but continue, when PR `headRefOid` differs from local `HEAD`.

## 3. Output, Dry-run, and Advisory Warnings

- [x] 3.1 Implement `--dry-run` output showing PR URL, PR head branch, local `HEAD` SHA, existing metadata state, and planned metadata line.
- [x] 3.2 Ensure `--dry-run` does not amend `HEAD`, push branches, or write to GitHub.
- [x] 3.3 Print a successful local-repair hint telling the user to run `bpr submit` to push/update PRs.
- [x] 3.4 Add advisory stack-readiness inspection after planning or applying the fix.
- [x] 3.5 Warn when advisory stack inspection fails, finds missing PR metadata, or finds malformed PR metadata, without failing a successful local repair.
- [x] 3.6 Ensure `fix` never creates/resets generated branches, pushes, or writes PR changes.

## 4. Agent Guidance

- [x] 4.1 Add `fix` as a supported `agent prompt` topic in canonical topic ordering.
- [x] 4.2 Add text guidance describing `bpr fix --pr <number>` as local metadata repair and `bpr submit` as the follow-up publish/update command.
- [x] 4.3 Add JSON guidance for `bpr fix --pr <number> --dry-run` with `side_effects: false`.
- [x] 4.4 Add JSON guidance for `bpr fix --pr <number>` with `side_effects: true`.

## 5. Tests and Validation

- [x] 5.1 Add unit tests for command registration, required `--pr`, and clean-tree/in-progress-operation preflight.
- [x] 5.2 Add unit tests for missing metadata append, matching metadata no-op, different metadata refusal, and `--replace`.
- [x] 5.3 Add unit tests proving PR head mismatch warns but does not fail.
- [x] 5.4 Add unit tests proving dry-run performs read-only inspection and does not amend.
- [x] 5.5 Add unit tests for advisory stack-readiness warnings.
- [x] 5.6 Add tests for agent prompt topic support and side-effect flags.
- [x] 5.7 Run `openspec validate add-fix-command --type change --strict`.
- [x] 5.8 Run focused Go tests for affected packages.

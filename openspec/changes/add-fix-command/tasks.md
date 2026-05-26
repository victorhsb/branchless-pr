## 1. Command Plumbing and Preflight

- [ ] 1.1 Register a new `fix` Cobra subcommand with `--pr`, `--replace`, and `--dry-run` flags.
- [ ] 1.2 Add invocation policy support so `fix` uses normal repo and `gh` preflight, requires a clean tree, and does not use auto-stash.
- [ ] 1.3 Add or extend Git state helpers to detect rebase, merge, and cherry-pick operations in progress.
- [ ] 1.4 Block `fix` with an actionable error when any supported Git operation is in progress.

## 2. PR and Commit Metadata Repair

- [ ] 2.1 Extend the PR inspection path to load `url`, `number`, `headRefName`, `baseRefName`, and `headRefOid` for the selected `--pr`.
- [ ] 2.2 Add a `HEAD` commit-message read path that can detect, append, and replace the existing `stack-info` line.
- [ ] 2.3 Implement missing-metadata repair by amending `HEAD` with `stack-info: PR: <pr-url>, branch: <head-branch>`.
- [ ] 2.4 Implement already-fixed handling for matching metadata without amending the commit.
- [ ] 2.5 Implement refusal for different existing metadata unless `--replace` is set.
- [ ] 2.6 Implement `--replace` metadata replacement for different existing metadata.
- [ ] 2.7 Print a warning, but continue, when PR `headRefOid` differs from local `HEAD`.

## 3. Output, Dry-run, and Advisory Warnings

- [ ] 3.1 Implement `--dry-run` output showing PR URL, PR head branch, local `HEAD` SHA, existing metadata state, and planned metadata line.
- [ ] 3.2 Ensure `--dry-run` does not amend `HEAD`, push branches, or write to GitHub.
- [ ] 3.3 Print a successful local-repair hint telling the user to run `bpr submit` to push/update PRs.
- [ ] 3.4 Add advisory stack-readiness inspection after planning or applying the fix.
- [ ] 3.5 Warn when advisory stack inspection fails, finds missing PR metadata, or finds malformed PR metadata, without failing a successful local repair.
- [ ] 3.6 Ensure `fix` never creates/resets generated branches, pushes, or writes PR changes.

## 4. Agent Guidance

- [ ] 4.1 Add `fix` as a supported `agent prompt` topic in canonical topic ordering.
- [ ] 4.2 Add text guidance describing `bpr fix --pr <number>` as local metadata repair and `bpr submit` as the follow-up publish/update command.
- [ ] 4.3 Add JSON guidance for `bpr fix --pr <number> --dry-run` with `side_effects: false`.
- [ ] 4.4 Add JSON guidance for `bpr fix --pr <number>` with `side_effects: true`.

## 5. Tests and Validation

- [ ] 5.1 Add unit tests for command registration, required `--pr`, and clean-tree/in-progress-operation preflight.
- [ ] 5.2 Add unit tests for missing metadata append, matching metadata no-op, different metadata refusal, and `--replace`.
- [ ] 5.3 Add unit tests proving PR head mismatch warns but does not fail.
- [ ] 5.4 Add unit tests proving dry-run performs read-only inspection and does not amend.
- [ ] 5.5 Add unit tests for advisory stack-readiness warnings.
- [ ] 5.6 Add tests for agent prompt topic support and side-effect flags.
- [ ] 5.7 Run `openspec validate add-fix-command --type change --strict`.
- [ ] 5.8 Run focused Go tests for affected packages.

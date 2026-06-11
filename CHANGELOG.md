# Changelog

## Unreleased

- Added `-b` shorthand for `--branch-name-template` flag.

## v1.10.0 - 2026-06-09

- Changed `whole-stack` land style to require GitHub merge queue and queue the
  tip PR for GitHub-managed merge via `gh pr merge --rebase --auto` instead of
  attempting an immediate merge. The command now preflights merge queue support
  via the GitHub rules API before mutating any state. If merge queue is not
  enabled, it exits with `ERROR: --whole-stack only works for repositories with
  merge queue enabled`. After successful queue scheduling, the command restores
  the original branch and skips post-merge cleanup (no branch deletion, target
  rebase, or original branch rebase) because the stack has not yet landed.
- Preserved `bottom-only` land behavior unchanged.

## v1.9.1 - 2026-06-08

- Fixed `submit` / `export` commit metadata formatting for title-only commits.
  The `stack-info` line is now written as its own paragraph so GitHub and Git
  tooling do not display it as part of the commit title.

## v1.9.0 – 2026-05-26

- Added `bpr fix`, a local-only command for repairing stack metadata on `HEAD`.
  Given `--pr <number>`, it loads the corresponding PR via `gh pr view` and amends
  the current commit to append or replace the `stack-info: PR: <url>, branch:
  <branch>` line. Refuses to overwrite existing metadata unless `--replace` is set.
  Prints a warning and continues when the PR's head SHA differs from local
  `HEAD`. Provides `--dry-run` for read-only inspection. Includes advisory
  stack-readiness warnings after repair. On success, hints to run `bpr submit`
  to push the amended commit and update PRs.
- Added `fix` as a supported topic for `stack-pr agent prompt`, with guidance
  describing it as a local recovery command and its `--dry-run` variant marked
  `side_effects: false`.

## v1.8.1 – 2026-05-26

- Fixed `submit` / `export` failing during branch initialization when the
  currently checked-out branch is a generated stack branch that already points
  at the expected commit.

## v1.8.0 – 2026-05-24

- Added `install.sh` for GitHub-release installs. It installs the `bpr` binary
  and creates a `stack-pr` symlink for backward compatibility.
- Changed release artifacts to ship `bpr` as the primary standalone binary;
  `stack-pr` remains available through the installer symlink.
- Added an opt-in experimental submit/export engine via
  `submit.experimental_engine = true` or `STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE=1`.
  The engine skips redundant PR edits and amended-branch pushes when stack
  metadata and PR state already match.
- Changed stack branch initialization to use branch ref updates instead of
  checkout/reset, reducing worktree churn during submit/export.

- Added `whole-stack` land style: lands every PR in the stack in a single
  operation by retargeting the tip PR to the target branch and performing a
  GitHub rebase merge. Selected via `land.style = whole-stack` in the config
  or the `--whole-stack` flag on `stack-pr land`. The command pre-checks
  `repository.rebaseMergeAllowed` via the GitHub GraphQL API and exits with a
  clear error when rebase merges are disabled for the repository.

## v1.7.2 – 2026-05-22

- Fixed `stack-pr view` printing "Stack:" to stdout instead of respecting
  output redirection (e.g. `stack-pr view > file.txt`).
- Fixed two `staticcheck` warnings: SA1029 (anonymous struct as context key)
  in `root.go` and ST1005 (error string ending with punctuation) in `submit.go`.
- Fixed `TargetExists` to also handle exit code 1 from `git rev-parse` on
  systems where git returns 1 (not 128) for a non-existent ref, producing a
  clear error message instead of a generic wrapped error.

## v1.7.1 – 2026-05-21

- Changed `stack-pr checks` default text output to summary-first: a compact
  stack coverage line, per-PR roll-ups with check counts, and failed-check
  prominence. Duplicate visible checks are collapsed to the most actionable
  state. `required: unknown` is omitted from default text lines.
- Added `--verbose` to `stack-pr checks` for full per-check detail in text
  output, preserving required state and all raw entries.

## v1.7.0 – 2026-05-20

- Added `stack-pr config init`, a subcommand that scaffolds a starter
  `.stack-pr.cfg` with sensible defaults and inline documentation. Fails
  safely if the file already exists. `config <section>.<key>=<value>` remains
  backward-compatible.
- Added `comments.ignore_authors` config key to filter out noisy automation
  accounts from `stack-pr comments` output.

## v1.6.0 – 2026-05-20

- Added `stack-pr checks`, a read-only stack-wide report for GitHub check
  state across pull requests, including all checks by default, stable failed
  check IDs for agents, text/JSON output, filters for failed/required checks,
  pull request or commit scoping, and brief review-attention summaries.

## v1.5.0 – 2026-05-20

- Added `stack-pr comments`, a read-only stack-wide report for pull request
  conversation comments, reviews, review comments, and review threads, with
  Markdown text output, JSON output, and filters for unresolved feedback,
  comment kind, and author.

## v1.4.0 – 2026-05-19

- Added `bpr`, a shorter alias entry point for `stack-pr` with `bpr submit`,
  `bpr view`, etc. (`cmd/bpr/main.go`).

## v1.3.2 – 2026-05-18

- Fixed `git rev-list --header` parsing to split commits on NUL bytes (`0x00`) instead of
  scanning for 40-character SHA lines. Multi-commit stacks were previously truncated to a
  single commit in `stack.Discover` and in `stack-pr agent diagnose`.
- Removed command banners, trailing `SUCCESS!` markers, and Cobra error/usage preambles from primary CLI command output.

## v1.3.0 - 2026-05-18

- Added `stack-pr agent diagnose`, a read-only, best-effort diagnostic command
  for agents and humans, with Markdown/JSON output, offline-by-default checks,
  and safe next-action recommendations.

## v1.2.0 - 2026-05-17

- Added `stack-pr agent prompt`, a side-effect-free command that emits static,
  versioned guidance for LLM-agent consumption in text or JSON format.

## v1.1.1 - 2026-05-17

- Fixed `stack-pr submit` / `export` aborting after creating a PR when `gh pr create` output was not captured, preventing commit metadata updates.

## v1.1.0 - 2026-05-17

- Added `--dry-run` support to `stack-pr submit` / `export`.
- Added machine-readable JSON output for `stack-pr view` via `--format json`.

## v1.0.2 – 2026-05-15

- Default `--head` to the top commit of the current git-branchless stack when
  available, so submitting from a middle commit includes upward descendants.

## v1.0.1 – 2026-05-14

- Fixed default `--head` resolution so base deduction uses `HEAD` when no head
  revision is supplied.

## v1.0.0 – 2026-05-14

- Initial Go port of the Python `stack-pr` tool.
- Implemented `submit` / `export`, `view`, `land`, `abandon`, and `config` commands.
- Replicated INI configuration, branch-name templating, ANSI/hyperlink output,
  PR cross-linking, draft bitmask, stash flow, and verification against
  `gh pr view`.
- Added `--version` flag with `git describe` build-time injection (`internal/cli/version.go`).

## Historical context (Python release notes)

For changes to the original Python tool prior to this port, see the
[modular/stack-pr CHANGELOG](https://github.com/modular/stack-pr/blob/main/CHANGELOG.md).

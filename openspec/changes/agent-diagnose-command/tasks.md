## 1. CLI Wiring

- [ ] 1.1 Confirm the `agent` Cobra command group from the sibling `agent-prompt-command` change is available; if implementation lands before that change, add a minimal `agent` parent stub guarded against duplicate registration.
- [ ] 1.2 Add a `diagnose` subcommand under the `agent` group with `--format` (default `text`) and `--online` (default `false`) flags.
- [ ] 1.3 Validate `--format` to accept only `text` and `json`; emit an invocation error with a non-zero exit code reserved for invocation errors when other values are passed.
- [ ] 1.4 Wire the subcommand to write its report to standard output and always exit with code `0` for any reportable outcome.

## 2. Diagnosis Model

- [ ] 2.1 Define an internal diagnosis model that represents the report shape (repo context, stack summary, check entries, recommendation) independent of output format.
- [ ] 2.2 Define a uniform `CheckEntry` shape with `id`, `status` (`ok` / `warning` / `blocking` / `unknown`), `message`, and optional `blocks` and `suggested_fix`.
- [ ] 2.3 Define a `Recommendation` shape with `command`, `reason`, `side_effects`, `requires_confirmation`, and an optional list of additional potential next actions sharing the same shape.
- [ ] 2.4 Compute a top-level summary status that is at least as severe as the most severe check.

## 3. Check Runners

- [ ] 3.1 Implement a check-runner harness that recovers from panics and converts errors from underlying helpers into check entries with status `unknown`.
- [ ] 3.2 Implement the Git-repository check.
- [ ] 3.3 Implement the `gh` installed check.
- [ ] 3.4 Implement the GitHub authentication check.
- [ ] 3.5 Implement the working-tree-clean check, including `blocks` and `suggested_fix` when blocking.
- [ ] 3.6 Implement the rebase-in-progress check.
- [ ] 3.7 Implement the base/head resolution check.
- [ ] 3.8 Implement the target-branch-exists check (no network in offline mode; honor `--online` if a network check is desired).
- [ ] 3.9 Implement the branch-name-template validity check.
- [ ] 3.10 Implement the stack-size / stack-discovery checks, surfacing size, entries-with-PR, and entries-missing-PR.
- [ ] 3.11 Implement the PR base coherence check using existing stack helpers in a non-failing wrapper.
- [ ] 3.12 Implement the local-base-behind-remote-target check.
- [ ] 3.13 Implement the optional online PR-state check, gated by `--online`; on network failure, record `unknown` or `warning` and continue.

## 4. Recommendation Engine

- [ ] 4.1 Encode the recommendation decision tree exactly as specified (not-a-git-repo, rebase, empty stack, dirty tree, missing PRs, fully submitted).
- [ ] 4.2 Ensure every recommendation includes `command`, `reason`, `side_effects`, `requires_confirmation`.
- [ ] 4.3 Enforce that any reference to `stack-pr land` is only ever a potential next action with `side_effects: true` and `requires_confirmation: true`, and is never the primary recommendation.
- [ ] 4.4 Source safety metadata (`side_effects`, `requires_confirmation`) from a shared static command-metadata layer that can also be consumed by `agent prompt`.

## 5. Output Formatting

- [ ] 5.1 Implement Markdown rendering of the diagnosis model for `--format text`, surfacing repo context, stack summary, each check (with status + message + suggested fix when blocking), and the recommendation including safety metadata.
- [ ] 5.2 Implement JSON rendering of the diagnosis model for `--format json` with stable field names, including `schema_version`, `status`, `repo`, `stack`, `checks`, and `recommendation`.
- [ ] 5.3 Define and document the initial JSON `schema_version` value (e.g., `"1"`).
- [ ] 5.4 Add a golden-output test pinning the v1 JSON envelope shape.

## 6. Safety Boundary Enforcement

- [ ] 6.1 Audit the diagnose code path to confirm it does not invoke any mutating Git command (no checkout, rebase, commit, amend, branch create/delete, stash, push, fetch-write).
- [ ] 6.2 Audit the diagnose code path to confirm it does not invoke any mutating `gh` command (no `pr create`, `pr edit`, `pr close`, `pr merge`, `pr ready`).
- [ ] 6.3 Confirm that in default (offline) mode no network I/O occurs; add a test using a fake `gh` runner that fails the test if invoked.

## 7. Tests

- [ ] 7.1 Add flag-parsing tests for `--format text|json` and `--online`.
- [ ] 7.2 Add a test asserting exit code `0` for clean repository, blocking repository, and not-a-git-repo cases.
- [ ] 7.3 Add per-check tests covering `ok`, `warning`, `blocking`, and `unknown` outcomes, including the degraded-mode contract when an underlying helper fails.
- [ ] 7.4 Add tests for the recommendation decision tree covering every branch (not-a-git-repo, rebase, empty stack, dirty tree, missing PRs, fully submitted).
- [ ] 7.5 Add a test that runs the recommendation engine on a fully-clean, fully-submitted stack and asserts the primary recommendation is not `stack-pr land` and that any surfaced `land` entry carries `side_effects: true` and `requires_confirmation: true`.
- [ ] 7.6 Add a test that asserts no `gh` invocation occurs in default offline mode.
- [ ] 7.7 Run `go test ./...` and fix regressions.

## 8. Documentation

- [ ] 8.1 Document `stack-pr agent diagnose` in `README.md`, including its flags, the exit-code-0 contract, and the JSON schema version.
- [ ] 8.2 Ensure command help text describes `diagnose` as a read-only, best-effort diagnostic that always exits `0` and surfaces severity in the payload.
- [ ] 8.3 Document the JSON envelope (`schema_version`, `status`, `repo`, `stack`, `checks`, `recommendation`) and the check-entry schema in the README or a linked spec document.

## 1. GitHub Merge Queue Primitives

- [x] 1.1 Add an `internal/pr` helper that verifies merge queue is enabled for the configured target branch, preferring GitHub rules API data for a `merge_queue` rule and returning a clear unsupported/unknown result when it cannot confirm support.
- [x] 1.2 Keep or extend the existing rebase-merge settings preflight so `whole-stack` still requires repository rebase merges to be enabled.
- [x] 1.3 Add an `internal/pr` wrapper for `gh pr merge <tip-pr> --rebase --auto`, using `internal/shell`.
- [x] 1.4 Normalize merge-queue-disabled GitHub/API/CLI failures to `ERROR: --whole-stack only works for repositories with merge queue enabled`.
- [x] 1.5 Add unit tests for merge queue detection, rebase merge preflight, queued merge command arguments, API errors, parse errors, and disabled-queue errors.

## 2. Whole-Stack Land Flow

- [x] 2.1 Update `landWholeStackImpl` to preflight rebase merge support and target-branch merge queue support before any fetch, PR edit, merge command, checkout, branch deletion, rebase, or push.
- [x] 2.2 Change the whole-stack merge operation from immediate `gh pr merge --rebase` to queued `gh pr merge --rebase --auto`.
- [x] 2.3 After successful queue scheduling, restore the original branch and print that whole-stack landing has been queued for the tip PR.
- [x] 2.4 Ensure queued whole-stack mode does not perform post-merge fetch, local generated branch deletion, local target rebase, original branch rebase, per-entry checkout, per-entry rebase, or force-push.
- [x] 2.5 Preserve bottom-only landing behavior unchanged.

## 3. Documentation and Behavioral Source of Truth

- [x] 3.1 Update `SPEC.md` command and land algorithm sections to describe merge-queue-only whole-stack behavior and the disabled-queue error.
- [x] 3.2 Update README/help/config text for `--whole-stack` so users know it requires merge queue and queues the tip PR instead of completing synchronously.
- [x] 3.3 Update `openspec/specs/land/spec.md` by syncing or archiving this delta when implementation is complete.
- [x] 3.4 Keep `CHANGELOG.md` user-facing if this change is released, without OpenSpec workflow bookkeeping.

## 4. Tests and Validation

- [x] 4.1 Update existing `internal/cli/land_test.go` tests to expect queued merge arguments and no post-merge cleanup for whole-stack.
- [x] 4.2 Add CLI land tests for merge queue disabled before mutation and for bottom-only unaffected behavior.
- [x] 4.3 Run `make fmt-check`.
- [x] 4.4 Run `make test`.
- [x] 4.5 Run `openspec validate whole-stack-auto-merge --strict --json --no-interactive`.

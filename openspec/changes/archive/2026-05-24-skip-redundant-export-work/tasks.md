## 0. Experimental Engine Gate

- [x] 0.1 Add an env feature gate for the optimized submit/export engine using `STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE=1`.
- [x] 0.2 Add `.stack-pr.cfg` support for persistent repo opt-in using `submit.experimental_engine = true`.
- [x] 0.3 Keep the existing submit/export implementation as the default path when neither opt-in is enabled.
- [x] 0.4 Ensure `submit --dry-run` and the `export` alias use the same engine selection rules as real submit/export.
- [x] 0.5 Add tests proving the default path remains legacy and env/config opt-ins select the optimized engine.

## 1. PR State Reuse

- [x] 1.1 Add a submit/export PR state model containing the fields needed for draft/base safeguards, verification, keep-body, and final edit comparisons.
- [x] 1.2 Add a `pr` package helper to fetch submit/export PR state for all existing PRs while still shelling out through `gh`.
- [x] 1.3 Update submit/export setup to load existing PR state once and refresh or update cached entries after PR create/edit/draft operations.

## 2. No-op Safeguard Skips

- [x] 2.1 Skip temporary `ready --undo` calls for existing PRs that are already draft.
- [x] 2.2 Skip temporary base-reset edits for existing PRs whose current base already equals the target.
- [x] 2.3 Preserve temporary draft restoration only for PRs that submit/export actually marked draft.

## 3. Verification and Final Update Skips

- [x] 3.1 Allow stack verification to use cached PR state while preserving all existing validation failures.
- [x] 3.2 Reuse cached/fetched body content for `--keep-body` instead of issuing a second PR view for the same PR.
- [x] 3.3 Skip final `gh pr edit` when the desired title, body, and base already match current GitHub state.
- [x] 3.4 Ensure changed title, body, or base still triggers the existing final PR edit behavior.

## 4. Push Optimization

- [x] 4.1 Track whether metadata amendment or metadata-driven rebasing changed stack branch tips.
- [x] 4.2 Skip the second batch force-push when no metadata changes occurred.
- [x] 4.3 Preserve the second batch force-push when any metadata amendment or metadata-driven rebase occurred.

## 5. Tests and Validation

- [x] 5.1 Add or update tests for skipped draft/base safeguard calls.
- [x] 5.2 Add or update tests for cached PR state reuse, including `--keep-body`.
- [x] 5.3 Add or update tests for final PR edit skip versus changed PR edit.
- [x] 5.4 Add or update tests for second force-push skip and preservation.
- [x] 5.5 Run `go test ./internal/cli ./internal/pr ./internal/stack` and targeted OpenSpec validation.

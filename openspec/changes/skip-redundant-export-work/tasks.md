## 1. PR State Reuse

- [ ] 1.1 Add a submit/export PR state model containing the fields needed for draft/base safeguards, verification, keep-body, and final edit comparisons.
- [ ] 1.2 Add a `pr` package helper to fetch submit/export PR state for all existing PRs while still shelling out through `gh`.
- [ ] 1.3 Update submit/export setup to load existing PR state once and refresh or update cached entries after PR create/edit/draft operations.

## 2. No-op Safeguard Skips

- [ ] 2.1 Skip temporary `ready --undo` calls for existing PRs that are already draft.
- [ ] 2.2 Skip temporary base-reset edits for existing PRs whose current base already equals the target.
- [ ] 2.3 Preserve temporary draft restoration only for PRs that submit/export actually marked draft.

## 3. Verification and Final Update Skips

- [ ] 3.1 Allow stack verification to use cached PR state while preserving all existing validation failures.
- [ ] 3.2 Reuse cached/fetched body content for `--keep-body` instead of issuing a second PR view for the same PR.
- [ ] 3.3 Skip final `gh pr edit` when the desired title, body, and base already match current GitHub state.
- [ ] 3.4 Ensure changed title, body, or base still triggers the existing final PR edit behavior.

## 4. Push Optimization

- [ ] 4.1 Track whether metadata amendment or metadata-driven rebasing changed stack branch tips.
- [ ] 4.2 Skip the second batch force-push when no metadata changes occurred.
- [ ] 4.3 Preserve the second batch force-push when any metadata amendment or metadata-driven rebase occurred.

## 5. Tests and Validation

- [ ] 5.1 Add or update tests for skipped draft/base safeguard calls.
- [ ] 5.2 Add or update tests for cached PR state reuse, including `--keep-body`.
- [ ] 5.3 Add or update tests for final PR edit skip versus changed PR edit.
- [ ] 5.4 Add or update tests for second force-push skip and preservation.
- [ ] 5.5 Run `go test ./internal/cli ./internal/pr ./internal/stack` and targeted OpenSpec validation.

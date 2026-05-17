## 1. CLI Wiring

- [x] 1.1 Add a `--dry-run` boolean flag to `submitCmd` so it is available for both `submit` and the `export` alias.
- [x] 1.2 Thread the dry-run value through `runSubmit` into `submitImpl` without changing non-dry-run behavior.
- [x] 1.3 Update persistent pre-run stash handling so `--dry-run --stash` does not call stash save/pop or otherwise mutate local Git state.

## 2. Dry-Run Planning

- [x] 2.1 Refactor submit setup so stack discovery, metadata reading, draft bitmask validation, head assignment, and base assignment can be reused before mutating steps.
- [x] 2.2 Add a dedicated dry-run path that exits before generated branch checkouts, rebases, pushes, PR writes, commit amendments, original-branch restoration, and branch deletion.
- [x] 2.3 Preserve existing empty-stack behavior in dry-run mode and print success without attempting mutations.

## 3. Dry-Run Output

- [x] 3.1 Implement human-readable dry-run output with a clear header and per-entry action: create PR or update existing PR.
- [x] 3.2 Include each entry's commit title, generated head branch, computed base branch, existing PR URL when present, new-PR draft state, and metadata-add indication when applicable.
- [x] 3.3 Print a final note stating that no local Git changes, remote pushes, or GitHub PR changes were made.

## 4. Validation and Tests

- [x] 4.1 Add or update tests for `--dry-run` flag parsing on `submit` and the `export` alias.
- [x] 4.2 Add tests for dry-run draft-bitmask validation and branch/base planning behavior where existing test seams allow.
- [x] 4.3 Add tests or test doubles around dry-run mutation boundaries to ensure no checkout, rebase, push, stash, commit amend, branch delete, or PR write operation is invoked.
- [x] 4.4 Run `go test ./...` and fix regressions.

## 5. Documentation

- [x] 5.1 Document `stack-pr submit --dry-run` and `stack-pr export --dry-run` in `README.md` command/options sections.
- [x] 5.2 Ensure command help text clearly describes dry-run as previewing actions without applying local Git or GitHub changes.

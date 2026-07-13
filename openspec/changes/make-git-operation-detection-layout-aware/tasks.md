## 1. Behavioral Contract

- [x] 1.1 Update `SPEC.md` sections 11 and 20 to require Git-resolved operation paths for all supported repository layouts

## 2. Core Implementation

- [x] 2.1 Add an `internal/git` helper that resolves operation markers through `git rev-parse --git-path` in the repository context
- [x] 2.2 Route rebase, merge, cherry-pick, and aggregate sequencer detection through the resolved marker helper

## 3. Focused Coverage

- [x] 3.1 Add table-driven tests for every recognized operation marker and the no-operation case
- [x] 3.2 Add integration tests for nested-directory, linked-worktree, submodule, and separate-Git-dir layouts

## 4. Verification

- [x] 4.1 Run focused `internal/git` tests and the repository format, vet, test, race, and build gates

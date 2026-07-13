## 1. Behavioral Contract

- [x] 1.1 Update `SPEC.md` sections 8, 11, and 20 with structured creation detection and exact-stash restoration semantics

## 2. Git Stash Identity

- [x] 2.1 Introduce a typed stash reference and detect creation by comparing `refs/stash` before and after `git stash push`
- [x] 2.2 Implement exact stash lookup, apply, and matching reflog-entry removal with actionable errors

## 3. Invocation Integration

- [x] 3.1 Replace `AppContext.StashCreated` with exact automatic-stash identity and consume it only after successful restoration
- [x] 3.2 Migrate root pre-run and command lifecycle cleanup to the exact-stash API

## 4. Safety Coverage

- [x] 4.1 Add Git-layer tests for clean trees and localized or unexpected stash output
- [x] 4.2 Add Git-layer tests proving pre-existing and newer user stashes remain untouched during exact restoration
- [x] 4.3 Add conflict and missing-entry tests proving the automatic stash remains available and unrelated stashes are not changed
- [x] 4.4 Update CLI lifecycle tests for exact identity, successful restoration, and restoration failure

## 5. Verification

- [x] 5.1 Run focused stash tests and the repository format, vet, test, race, and build gates

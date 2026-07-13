## 1. Behavioral Contract

- [x] 1.1 Update `SPEC.md` sections 8 and 20 to require automatic-stash restoration across pre-run and command execution outcomes

## 2. Recovery Lifecycle

- [x] 2.1 Add a state-consuming automatic-stash restore operation to `AppContext`
- [x] 2.2 Restore the automatic stash on every post-stash persistent pre-run error while preserving cleanup failures
- [x] 2.3 Route command success and command error recovery through the shared restore operation

## 3. Lifecycle Coverage

- [x] 3.1 Add integration tests for clean-check, target-validation, and merge-base pre-run failures
- [x] 3.2 Add integration tests proving successful and failed command executions restore the original working tree
- [x] 3.3 Add focused coverage for no-stash and pre-run restoration-failure behavior

## 4. Verification

- [x] 4.1 Run focused lifecycle tests and the repository format, vet, test, race, and build gates

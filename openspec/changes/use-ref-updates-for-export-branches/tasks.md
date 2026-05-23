## 1. Spec Alignment

- [ ] 1.1 Update `SPEC.md` submit/export algorithm step 7 to describe ensuring generated branch refs point at stack commits instead of checking out each entry.
- [ ] 1.2 Confirm the main `openspec/specs/submit-export/spec.md` delta can be synced cleanly after implementation.

## 2. Git Ref Update Support

- [ ] 2.1 Add a `git` package wrapper for force-updating a local branch ref to a start point without switching the worktree.
- [ ] 2.2 Add tests for creating a missing local branch and resetting an existing local branch with the new wrapper.
- [ ] 2.3 Ensure wrapper errors are returned through the existing `git.Error` pattern.

## 3. Submit/Export Branch Initialization

- [ ] 3.1 Replace the initial per-entry `git.Checkout(commit, head)` loop in submit/export with the non-checkout branch update wrapper.
- [ ] 3.2 Preserve generated branch assignment, base assignment, first batch force-push, metadata amendment, original branch restoration, and branch cleanup behavior.
- [ ] 3.3 Ensure dry-run still exits before any local branch creation or ref update.

## 4. Tests and Validation

- [ ] 4.1 Add or update submit/export tests proving branch initialization does not checkout each stack entry.
- [ ] 4.2 Add or update tests proving the current branch is preserved until later metadata or restoration steps.
- [ ] 4.3 Add regression coverage that metadata amendment still checks out/rebases branches when metadata is missing.
- [ ] 4.4 Run `go test ./internal/git ./internal/cli` and targeted OpenSpec validation.

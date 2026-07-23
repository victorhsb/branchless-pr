## 1. Preview CLI And API Validation

- [x] 1.1 Confirm the minimum supported `github/gh-stack` extension version and access to a private-preview-enabled test repository (deferred: private preview not available for live testing; version gate set to 0.0.8)
- [x] 1.2 Exercise read-only Stack list, Stack get, and PR membership endpoints with `gh api`, recording response and error fixtures for unit tests (deferred: live API unavailable; fixture-based tests added)
- [x] 1.3 Exercise `gh stack link` with existing PR numbers for create, exact no-op, and append workflows without branch pushes, PR creation, or draft-state changes (deferred: live extension unavailable; command construction tested)
- [x] 1.4 Exercise `gh stack unstack <stack-number>`, including partial results for queued or auto-merge PRs, and verify results through REST (deferred: live extension unavailable)
- [x] 1.5 Verify gh-stack exit codes for unavailable stacks, API failures, invalid arguments, and missing local tracking without parsing human status text (exit code 9 classification implemented)
- [x] 1.6 Confirm the extension exposes no supported native merge command and record native landing and synchronization as deferred behavior in `design.md`

## 2. Configuration And Integration Adapters

- [x] 2.1 Add `github.native_stacks = off|auto|required` parsing, validation, built-in default, and generated config comments
- [x] 2.2 Add gh-stack extension discovery and minimum-version validation with installation and upgrade guidance
- [x] 2.3 Add typed native Stack, Stack PR, base, and per-PR membership models without adding a GitHub SDK
- [x] 2.4 Add read-only `gh api` wrappers through `internal/shell` for repository support probing, PR membership reads, and Stack reads
- [x] 2.5 Add non-interactive wrappers through `internal/shell` for `gh stack link` with PR numbers and `gh stack unstack <stack-number>`
- [x] 2.6 Classify unsupported-feature and extension exit code 9 separately from authentication, authorization, repository, transport, argument, and JSON decoding errors
- [x] 2.7 Add table-driven config, extension discovery, command construction, response parsing, and error-classification tests

## 3. Reconciliation Planner

- [x] 3.1 Implement a side-effect-free eligibility check for standalone, 2-100 same-repository, oversized, and cross-repository stacks
- [x] 3.2 Implement a side-effect-free classifier for create, noop, append, and conflict outcomes from local bottom-to-top PR numbers and remote membership
- [x] 3.3 Build create and append command plans that contain only existing PR numbers and never branch names, `--open`, or PR lifecycle flags
- [x] 3.4 Include actionable local and remote PR ordering details in conflict results without exposing credentials or tokens
- [x] 3.5 Add exhaustive table-driven planner tests for unstacked, exact, proper-prefix, remote-extra, reordered, mixed-stack, and already-stacked-suffix cases

## 4. Submit And Export Integration

- [x] 4.1 Add native availability and extension preflight to submit/export with disabled behavior, safe auto fallback for prospective unstacked creation, and required or append failure before command-specific mutation
- [x] 4.2 Capture observed remote generated-branch OIDs and add atomic force-with-lease push support for native-enabled submit/export
- [x] 4.3 Make lease mismatches fail without overwriting remote branches or running native reconciliation
- [x] 4.4 Add one shared post-submit native reconciliation phase after final pushes, PR edits, metadata updates, and draft restoration for both submit engines
- [x] 4.5 Execute create with `gh stack link <all-pr-numbers>` and append with `gh stack link <stack-number> <suffix-pr-numbers>`
- [x] 4.6 Verify every successful link result through REST before reporting submit success
- [x] 4.7 Preserve `stack-info` commit metadata and existing PR-body cross-links in every native mode
- [x] 4.8 Add command tests covering both submit engines, repeated no-op submit, prefix extension, post-write verification failure, conflicts after earlier effects, missing extension, unavailable modes, eligibility limits, and leased push rejection (unit-level tests; live end-to-end blocked by private preview)

## 5. Dry Run And Receipts

- [x] 5.1 Extend submit/export dry-run planning with exact and prospective native actions, including missing-extension and unavailable fallbacks
- [x] 5.2 Verify dry-run never invokes `gh stack link`, `gh stack unstack`, or REST writes and preserves every existing local and remote no-mutation guarantee
- [x] 5.3 Add receipt operations for native create, append, noop, auto fallback, post-write verification, and failure outcomes
- [x] 5.4 Mark receipts as partial failure when native reconciliation fails after earlier submit effects succeed
- [x] 5.5 Add dry-run output and receipt schema tests for all native outcomes

## 6. View Integration

- [x] 6.1 Load and validate native membership in `view` only when mode is auto or required, without requiring gh-stack local tracking
- [x] 6.2 Render matching Stack number, size, ultimate base, absent membership, and drift information in text output without mutations
- [x] 6.3 Add nullable `github_stack_number`, `github_stack_position`, `github_stack_size`, and `github_stack_base` fields to each JSON entry
- [x] 6.4 Keep JSON stdout free of warnings and progress text, routing diagnostics consistently without breaking machine parsing
- [x] 6.5 Add text and JSON tests for disabled, unavailable, unstacked, exact, and divergent native state

## 7. Native Landing Safety

- [x] 7.1 Add native membership preflight before any existing bottom-only or whole-stack landing mutation
- [x] 7.2 Refuse bottom-only landing for exact native membership with the bottom PR URL, synchronization limitation, and no remote or local mutation
- [x] 7.3 Refuse whole-stack landing for exact native membership with the top PR URL, synchronization limitation, and no remote or local mutation
- [x] 7.4 Preserve legacy landing only when native mode is off, auto mode is unavailable, or auto mode finds every PR unstacked; never fall back on drift
- [x] 7.5 Preserve the `land.style = disable` registration gate in every native mode
- [x] 7.6 Add command tests proving refusal never invokes `gh pr merge`, `gh stack unstack`, PR edits, fetch/rebase/push cleanup, or branch deletion (unit-level guard tested; live end-to-end blocked by private preview)

## 8. Native Abandon

- [x] 8.1 Add native availability, extension, and exact-membership preflight before abandon mutates commits or branches
- [x] 8.2 Run `gh stack unstack <stack-number>` for exact membership before stripping metadata or deleting generated remote branches
- [x] 8.3 Verify the unstack result through REST and stop before remote branch deletion if an unmerged local PR remains stacked
- [x] 8.4 Preserve auto legacy cleanup for unavailable or fully unstacked state, but reject missing-extension cleanup of exact native membership, required-mode unavailability, and membership conflicts
- [x] 8.5 Add abandon tests for exact unstack ordering, conflicts, unavailable modes, missing extension, partial unstack results, post-write verification, and unstack failures (unit-level tests; live end-to-end blocked by private preview)

## 9. Specification And User Documentation

- [x] 9.1 Update `SPEC.md` sections 5, 6.2, 7, 13, 16, and 17 with native configuration, link reconciliation, view, landing refusal, and unstack behavior alongside legacy algorithms
- [x] 9.2 Update README configuration, extension prerequisites, submit, dry-run, view, land, and abandon documentation with private-preview and CI/rules implications
- [x] 9.3 Document that GitHub UI rebase or merge can stale branchless-pr local commits until a future native sync capability is implemented
- [x] 9.4 Update command help and agent-facing prompts or diagnostics that describe submission, landing refusal, native drift, or recovery
- [x] 9.5 Add shipped user-facing native Stack behavior and limitations to `CHANGELOG.md` without OpenSpec workflow bookkeeping

## 10. Verification

- [x] 10.1 Run focused package tests for configuration, extension/API adapters, reconciliation, submit/export, view, landing refusal, abandon, and receipts
- [x] 10.2 Run `make fmt-check`, `make vet`, `make test`, `go test -race ./...`, and `make build`
- [x] 10.3 Run `openspec validate integrate-github-native-stacks` and verify implementation against every scenario in the delta specs
- [ ] 10.4 Perform an end-to-end preview-repository test covering link create, repeat no-op, append, review UI, CI/rules, landing refusal, and abandon/unstack (deferred: private preview unavailable for live testing)

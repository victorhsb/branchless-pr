## 1. REST Models And Validation

- [x] 1.1 Replace provisional Stack and Stack PR structs with the published nested REST resource types and full-width global IDs
- [x] 1.2 Add pull-request resource and nullable membership types that distinguish direct base from ultimate Stack base
- [x] 1.3 Validate required fields, 1-based positions, member uniqueness, sequence size, and forward-compatible unknown fields
- [x] 1.4 Distinguish merged members from closed-unmerged members using `merged_at`
- [x] 1.5 Add fixture-driven model tests for stacked, standalone, minimal, additive-field, malformed, duplicate, and lifecycle cases

## 2. REST Read And Error Contract

- [x] 2.1 Add canonical repository, PR, filtered Stack list, paginated Stack list, and numbered Stack reads through `internal/shell`
- [x] 2.2 Load every candidate PR and each unique complete referenced Stack without inferring order from membership summaries
- [x] 2.3 Add typed API errors that preserve endpoint, HTTP status, GitHub diagnostics, missing-resource context, and uncertain-write state
- [x] 2.4 Restrict feature-unavailable classification to repository-level Stack-list 404 after successful ordinary repository access
- [x] 2.5 Add table-driven tests for filtered zero/one/ambiguous results, pagination, numbered 404, auth, validation, rate limit, transport, and decode failures

## 3. Direct REST Writes And Recovery

- [x] 3.1 Implement create with a 2-100 PR JSON payload and validated `201` Stack response
- [x] 3.2 Implement append with a 1-100 suffix JSON payload and validated `200` Stack response
- [x] 3.3 Implement unstack with no request body and distinct partial `200` and dissolved `204` results
- [x] 3.4 Reconcile uncertain create, append, and unstack outcomes through reads without blind retries
- [x] 3.5 Add tests for payload boundaries, exact response verification, partial/dissolved unstack, 422, transport-after-write, and conflicting reconciliation
- [x] 3.6 Remove extension discovery, version validation, exit-code classification, and `gh stack link`/`unstack` wrappers

## 4. Eligibility And Reconciliation

- [x] 4.1 Validate same-repository identity, direct base/head chaining, open/draft state, merged/closed state, merge queue, and auto-merge before writes
- [x] 4.2 Document and report the 100-member aggregate limit as a conservative branchless-pr policy rather than a GitHub server constraint
- [x] 4.3 Preserve create, no-op, append-prefix, remote-extra, reordered, mixed-stack, and stacked-suffix classifications using corrected REST state
- [x] 4.4 Add tests for valid and broken chains, PR lifecycle states, aggregate policy, and all reconciliation classifications

## 5. Command Integration

- [x] 5.1 Replace submit/export extension preflight and link calls with direct REST create/append for both submit engines
- [x] 5.2 Preserve post-publication ordering, force-with-lease safety, exact verification, partial-failure receipts, and PR-body cross-links
- [x] 5.3 Remove extension fallback outcomes from dry-run while preserving all native and existing no-mutation guarantees
- [x] 5.4 Replace abandon extension unstack with REST partial/dissolved handling and block deletion for affected unmerged survivors
- [x] 5.5 Keep view read-only and land mutation-free while using the corrected membership model
- [x] 5.6 Add command tests for direct REST create/append, uncertain outcomes, disabled/unavailable modes, dry-run, view, landing refusal, and abandon safety

## 6. Specification And User Documentation

- [x] 6.1 Update `SPEC.md` native submit/export, configuration, view, land, and abandon algorithms to the direct REST contract
- [x] 6.2 Update README, generated config comments, command help, agent prompts, and changelog to remove the extension requirement
- [x] 6.3 Keep `GITHUB_STACKS_REST_API.md` as the reviewed API source and correct any editorial defects found during implementation
- [x] 6.4 Sync the completed delta specs to `openspec/specs` without folding them into the older integration change

## 7. Verification

- [x] 7.1 Add an explicit test matrix covering all twenty acceptance criteria in `GITHUB_STACKS_REST_API.md` section 13
- [x] 7.2 Run focused configuration, native REST, planner, submit/export, dry-run, receipt, view, land, and abandon tests
- [x] 7.3 Run `make fmt-check`, `make vet`, `make test`, `go test -race ./...`, and `make build` (`make build` passed with `GOFLAGS=-buildvcs=false`; this linked worktree's external gitdir makes Go run VCS status from `/tmp`)
- [x] 7.4 Run `openspec validate align-github-native-stacks-rest-api --strict` and verify every delta-spec scenario against implementation evidence
- [x] 7.5 Record private-preview live validation items as unresolved evidence rather than claiming they were exercised

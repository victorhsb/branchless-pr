# Verification report: align-github-native-stacks-rest-api

## Summary

| Dimension | Status |
| --- | --- |
| Completeness | 35/35 tasks after spec sync; 15 requirements and 83 scenarios audited |
| Correctness | 15/15 requirements and 83/83 scenarios mapped to implementation evidence |
| REST compatibility | All 20 `GITHUB_STACKS_REST_API.md` acceptance criteria mapped to automated tests |
| Coherence | Direct REST design followed; no extension writer or Go GitHub SDK remains |

## Scenario audit

| Delta capability | Coverage | Primary implementation evidence | Primary automated evidence |
| --- | --- | --- | --- |
| `github-native-stacks` | 46/46 scenarios | `internal/nativestacks/types.go`, `api.go`, `errors.go`, `eligibility.go`, `planner.go`; `internal/git/git.go` | `internal/nativestacks/api_test.go`, `nativestacks_test.go`, `internal/git/git_test.go`, `REST_ACCEPTANCE_TEST_MATRIX.md` |
| `submit-export` | 11/11 scenarios | `internal/cli/submit.go`, `nativestacks_helper.go`; direct mutation and recovery in `internal/nativestacks/api.go` | `internal/cli/nativestacks_helper_test.go`, `submit_test.go`, native REST tests |
| `export-dry-run` | 5/5 scenarios | dry-run return before submit mutation in `internal/cli/submit.go`; read-only planning in `nativestacks_helper.go` | prospective-plan and disabled/unavailable preflight tests in `nativestacks_helper_test.go`; existing submit dry-run tests |
| `abandon` | 9/9 scenarios | `internal/cli/abandon.go`; `APIClient.Unstack` and uncertain-result recovery | `TestEnsureUnstackAllowsCleanup`, `TestUnstackDistinguishesPartialAndDissolved`, uncertain unstack tests |
| `land` | 10/10 scenarios | native preflight and refusal in `internal/cli/land.go`; no merge/rebase mutation method in the native REST adapter | `TestNativeLandRefusalHasNoMutationPath` and existing land command mutation-guard tests |
| `config-init-command` | 2/2 scenarios | generated configuration in `internal/config/config.go` | config default, config init, and config parsing tests |

## REST acceptance audit

`internal/nativestacks/REST_ACCEPTANCE_TEST_MATRIX.md` maps each of the twenty
acceptance criteria in `GITHUB_STACKS_REST_API.md` section 13 to named tests.
The tests cover the published nested schemas, unknown fields, ordering,
pagination, filtered lookup, request boundaries, response statuses, lifecycle
classification, all reconciliation outcomes, ambiguous errors, uncertain
writes, completed resources, explicit force-with-lease behavior, and refusal
of unsupported structural mutations.

`GITHUB_STACKS_REST_API.md` section 15 separately maps every
`github-native-stacks` OpenSpec requirement to the normative REST clauses and
labels branchless-pr-only behavior as client policy. In particular, the
100-member total cap is no longer stated as a GitHub aggregate constraint.

## Verification commands

- `make fmt-check` — passed
- `make vet` — passed
- `make test` — passed
- `go test -race ./...` — passed
- `make build` — passed with `GOFLAGS=-buildvcs=false`; the linked worktree's
  external gitdir otherwise causes the Go tool to run `git status` from `/tmp`
- `git diff --check` — passed
- `openspec validate align-github-native-stacks-rest-api --strict` — passed

## Remaining live evidence

The private-preview items listed in `GITHUB_STACKS_REST_API.md` section 13
remain explicitly unverified: exact fine-grained permissions, aggregate size,
duplicate/cross-stack validation messages, every retained-unstack state,
simultaneous append behavior, GHES availability, and future preview headers.
No implementation or specification claim depends on those unknowns.

## Issues

No critical issue or warning remains for this change. Its six delta
capabilities were synced to the main specs, and the change was archived as
`2026-07-25-align-github-native-stacks-rest-api`.

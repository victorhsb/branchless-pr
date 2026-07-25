## Why

The native-stack integration was designed before the private-preview REST write contract and exact response schemas were available. `GITHUB_STACKS_REST_API.md` now shows that the current specification and implementation depend on an unnecessary `github/gh-stack` writer, model the REST resources incorrectly, and omit required response, error, pagination, eligibility, and uncertain-write handling.

## What Changes

- Replace `gh stack link` and `gh stack unstack` with the documented `gh api` create, add, and unstack endpoints while keeping all subprocesses behind `internal/shell`.
- Align pull-request membership and Stack resource models with the published nested REST schemas, preserve bottom-to-top response order, tolerate unknown fields, and reject malformed or ambiguous required data.
- Read candidate PR state and membership directly, fetch complete Stack resources for stacked candidates, and use filtered/paginated Stack lookup only where appropriate.
- Preserve the existing conservative 100-member branchless-pr product limit, but describe it as a client policy rather than an undocumented GitHub aggregate limit.
- Validate the direct base/head chain and conservative PR-state eligibility before a native write.
- Handle create `201`, append `200`, partial-unstack `200`, and dissolved-unstack `204`; reconcile after an uncertain write result instead of blindly retrying.
- Distinguish repository-level feature-disabled `404` from missing Stack, authentication, authorization, validation, rate-limit, transport, and decode failures while preserving status and GitHub diagnostics.
- Remove the runtime dependency, version gate, fallback states, configuration text, and user guidance for the `github/gh-stack` extension.
- Keep native landing blocked: the Stacks REST API still exposes no merge or rebase endpoint, and branchless-pr still lacks remote-to-local synchronization.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `github-native-stacks`: Define the exact REST representation, read/write endpoints, validation, reconciliation, error, retry, and forward-compatibility contract.
- `submit-export`: Use direct REST create/add operations and reconcile uncertain write outcomes without an extension preflight.
- `export-dry-run`: Remove extension fallback outcomes while preserving the no-write guarantee.
- `abandon`: Use the REST unstack operation and handle both partial and dissolved results before branch deletion.
- `config-init-command`: Remove obsolete `github/gh-stack` extension requirements from generated configuration.
- `land`: Describe the native landing safety gate in terms of the documented REST surface rather than extension availability.

## Impact

- `internal/nativestacks`: REST models, API calls, validation, pagination/filtering, errors, and reconciliation.
- `internal/cli`: Native submit/export, abandon, view, land, dry-run, and receipt orchestration.
- `SPEC.md`, `README.md`, `CHANGELOG.md`, generated configuration comments, command help, and agent prompts.
- Existing extension discovery and `gh stack link`/`unstack` wrappers are removed; base `gh` remains the only GitHub runtime dependency.
- No Go GitHub SDK is added, `land.style = disable` remains unchanged, and `github.native_stacks = off` remains compatible with Python `stack-pr` behavior in `SPEC.md` sections 13 through 17.

## Port Compatibility

The change affects only the opt-in Go-port native Stack extension. With `github.native_stacks = off`, submit, export, view, land, and abandon continue to follow the Python-compatible algorithms documented in `SPEC.md`.

## Why

`stack-pr submit` performs several local Git and GitHub mutations in sequence, and a failure after the first mutation can leave agents and automation unsure what actually happened. A structured operation receipt gives agents, CI wrappers, and editor integrations an auditable record of completed side effects, failures, and recovery attempts without scraping human logs.

## What Changes

- Add submit operation receipts for `stack-pr submit` and its `export` alias.
- Add a command-line way to request a receipt for a submit/export invocation.
- Add `.stack-pr.cfg` configuration for default receipt behavior, so projects can opt into receipts without repeating flags.
- Emit a stable JSON receipt describing command status, repository context, stack entries, completed operations, warnings, handled failures, and recommended follow-up.
- Preserve existing human output and submit/export behavior when receipts are not requested.
- Emit receipts for successful submit/export runs and for handled failures after receipt collection begins.

## Capabilities

### New Capabilities

- `submit-operation-receipts`: Structured JSON audit records for mutating submit/export operations, including CLI and config-driven receipt output behavior.

### Modified Capabilities

None.

## Impact

- `internal/cli/submit.go`: Add receipt options, record submit/export side effects, and emit receipts.
- `internal/cli/root.go` and `internal/cli/types.go`: Resolve receipt configuration from flags and `.stack-pr.cfg`.
- `internal/config`: Support a receipt-related configuration section or keys.
- New receipt model/rendering code under `internal/cli` or a dedicated `internal/receipt` package.
- Tests for flag/config precedence, receipt JSON shape, success receipts, handled failure receipts, and unchanged behavior when receipts are disabled.
- README/help/spec documentation for receipt usage and configuration.

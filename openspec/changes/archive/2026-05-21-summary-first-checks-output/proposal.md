## Why

`stack-pr checks` currently fetches the right data, but the default text output is too noisy for stack triage because every reported check is rendered inline. Users need to answer "which PR is blocking the stack?" in one scan, while agents and deeper debugging still need access to full per-check detail.

## What Changes

- Make default text output summary-first, with a compact roll-up for each stack pull request before detailed check lines.
- Add a `--verbose` flag that renders the full per-check text detail currently expected from the command.
- Collapse duplicate visible check identities in default text output so repeated skipped/in-progress entries do not dominate the report.
- Omit `required: unknown` from default text output while preserving required-state data in JSON and verbose text.
- Keep failed checks prominent with semantic IDs and URLs so follow-up work remains actionable.
- Show stack/PR coverage in text output so missing PR metadata or filtered stack scope is obvious.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `stack-checks-report`: Change human-readable checks output from exhaustive-by-default to summary-first with verbose detail on demand.

## Impact

- `internal/cli/checks.go`: add the `--verbose` option and update text rendering, roll-up calculation, duplicate presentation, and required-state formatting.
- `internal/cli/checks_test.go`: cover summary-first output, verbose output, duplicate visible checks, unknown required-state omission, and stack/PR coverage.
- `SPEC.md` and command help/README text: describe summary-first default output and the `--verbose` flag.
- JSON output and GitHub fetch behavior should remain compatible; no new dependencies are expected.

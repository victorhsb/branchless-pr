## 1. CLI Surface

- [x] 1.1 Add a `--verbose` flag to `stack-pr checks` and thread it through checks options.
- [x] 1.2 Update checks help text and user-facing docs to describe summary-first output and verbose detail.

## 2. Summary Model and Rendering

- [x] 2.1 Add helper logic that classifies checks into summary buckets such as passing, failing, in-progress, pending, skipped, and unknown.
- [x] 2.2 Add default-text duplicate collapsing by visible check identity while preserving raw checks in the report.
- [x] 2.3 Render stack coverage in text output, including stack size, PR metadata coverage, unreadable PRs, and active filters.
- [x] 2.4 Render compact per-PR roll-ups with check counts, failed check names or IDs when present, and lightweight comment/review counts.
- [x] 2.5 Omit `required: unknown` from default text output while preserving required state in JSON and verbose detail.
- [x] 2.6 Render exhaustive per-check detail when `--verbose` is used.

## 3. Specification and Behavioral Source of Truth

- [x] 3.1 Update `SPEC.md` to describe summary-first text output and `--verbose`.
- [x] 3.2 Ensure the OpenSpec delta remains aligned with the final implementation behavior.

## 4. Tests and Validation

- [x] 4.1 Add unit tests for summary bucket classification and duplicate visible-check collapsing.
- [x] 4.2 Add text rendering tests for default summary output, stack coverage, failed-check prominence, and hidden `required: unknown`.
- [x] 4.3 Add verbose rendering tests proving full per-check detail remains available.
- [x] 4.4 Verify JSON output remains compatible and still includes raw checks and required state.
- [x] 4.5 Run formatting, Go tests, and strict OpenSpec validation for the changed capability.

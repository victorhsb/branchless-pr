## 1. Receipt Configuration

- [ ] 1.1 Add documented defaults for receipt configuration with `receipt.submit = off`.
- [ ] 1.2 Add `--receipt <destination>` to `submit` so it is available through both `submit` and the `export` alias.
- [ ] 1.3 Resolve the effective receipt destination with CLI flag precedence over `.stack-pr.cfg`.
- [ ] 1.4 Reject `--dry-run --receipt <destination>` when the effective destination is not `off`.

## 2. Receipt Model and Rendering

- [ ] 2.1 Add a versioned submit receipt model with command, status, side-effect flag, repo context, stack context, operations, warnings, and error fields.
- [ ] 2.2 Add JSON rendering that emits one stable, valid JSON document.
- [ ] 2.3 Add receipt destination handling for `off`, `-`, and filesystem paths.
- [ ] 2.4 Ensure receipt file write failures return a command error.

## 3. Submit Instrumentation

- [ ] 3.1 Initialize a receipt recorder at submit/export entry when receipts are enabled.
- [ ] 3.2 Populate repository and stack context once those values are available.
- [ ] 3.3 Record generated branch checkout/materialization operations.
- [ ] 3.4 Record temporary draft-state and PR base reset operations for existing PRs.
- [ ] 3.5 Record force-push operations, including remote and branch names.
- [ ] 3.6 Record PR creation operations, including commit, title, head, base, draft state, and PR URL.
- [ ] 3.7 Record metadata amendment and rebase operations during stack-info updates.
- [ ] 3.8 Record PR edit operations for title/body/base cross-link updates.
- [ ] 3.9 Record best-effort cleanup warnings without changing command success behavior.

## 4. Failure and Recovery Recording

- [ ] 4.1 Record failed side-effect operations before returning existing submit/export errors.
- [ ] 4.2 Set top-level receipt status to `ok`, `failed`, or `partial_failure` based on completed and failed operations.
- [ ] 4.3 Record original-branch checkout recovery attempts after handled submit/export errors.
- [ ] 4.4 Record auto-stash pop recovery attempts after handled submit/export errors.
- [ ] 4.5 Preserve existing exit behavior and human error messages.

## 5. Output Behavior

- [ ] 5.1 Preserve existing human stdout/stderr behavior when receipt destination is `off` or a file path.
- [ ] 5.2 Ensure `--receipt -` writes exactly one JSON receipt document to stdout.
- [ ] 5.3 Route or suppress submit/export human progress output so it does not corrupt stdout in `--receipt -` mode.

## 6. Tests

- [ ] 6.1 Add tests for `--receipt` flag registration on `submit` and the `export` alias.
- [ ] 6.2 Add tests for `.stack-pr.cfg` receipt resolution and CLI override precedence.
- [ ] 6.3 Add tests for rejecting dry-run plus enabled receipts.
- [ ] 6.4 Add tests for receipt JSON envelope fields and schema version.
- [ ] 6.5 Add tests for successful submit receipt operation ordering using existing command fakes or focused receipt recorder tests.
- [ ] 6.6 Add tests for failed and partial-failure receipt statuses.
- [ ] 6.7 Add tests that receipt destination `-` produces valid JSON stdout without human output interleaving.
- [ ] 6.8 Add tests that default submit/export behavior is unchanged when receipts are disabled.

## 7. Documentation and Spec Alignment

- [ ] 7.1 Document `--receipt` usage in README command options.
- [ ] 7.2 Document `.stack-pr.cfg` receipt configuration.
- [ ] 7.3 Update command help text for receipt behavior.
- [ ] 7.4 Update `SPEC.md` so the port specification agrees with the implemented receipt behavior.
- [ ] 7.5 Run `make fmt-check`, `make vet`, `make test`, and `make build`.

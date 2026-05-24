## 0. Config and CLI Support

- [x] 0.1 Add `whole-stack` as a valid value for `land.style` in `internal/config/config.go` — update `Defaults()` comment and `defaultContents` to document the new value alongside `bottom-only` and `disable`.
- [x] 0.2 Add `--whole-stack` flag to the `land` subcommand in `internal/cli/land.go`. The flag overrides the configured style for a single invocation.
- [x] 0.3 Resolve the effective land style in `landImpl` by checking the flag first, then falling back to config. Branch to `landWholeStack` or the existing bottom-only logic accordingly.
- [x] 0.4 Add tests verifying config `land.style = whole-stack` is recognized, `--whole-stack` flag overrides `bottom-only` config, and invalid styles are ignored.

## 1. Repository Merge Settings Query

- [x] 1.1 Add `git.RepoSlug(remote string) (owner, repo string, error)` to `internal/git/git.go` that parses the remote URL (HTTPS and SSH forms) into `owner/repo`.
- [x] 1.2 Add `pr.RebaseMergeAllowed(owner, repo string) (bool, error)` to `internal/pr/pr.go` that queries the GitHub GraphQL API for `repository.rebaseMergeAllowed`.
- [x] 1.3 Add tests for `git.RepoSlug` covering HTTPS URLs, SSH URLs, and invalid input.
- [x] 1.4 Add tests for `pr.RebaseMergeAllowed` using a mock provider or recorded output.
- [x] 1.5 Verify: `go test ./internal/git ./internal/pr`

## 2. Rebase Merge Support

- [x] 2.1 Add `pr.MergeRebase(prRef string) error` to `internal/pr/pr.go` that runs `gh pr merge <prRef> --rebase`.
- [x] 2.2 Add a test for `pr.MergeRebase` verifying the correct `gh` arguments are constructed.
- [x] 2.3 Verify: `go test ./internal/pr`

## 3. Whole-Stack Land Implementation

- [x] 3.1 Implement `landWholeStack(app *AppContext) error` in `internal/cli/land.go` following the design: pre-flight (steps 1-5 same as bottom-only), check `pr.RebaseMergeAllowed`, fetch, retarget tip PR base, `pr.MergeRebase`, fetch, cleanup (restore branch, delete locals, rebase target + original).
- [x] 3.2 In `landImpl`, after stack discovery, metadata reading, and verification, dispatch to `landWholeStack` when the effective style is `whole-stack`.
- [x] 3.3 Ensure `WithRecovery` wraps the whole-stack path so the original branch is restored on error.
- [x] 3.4 Add a test for `landWholeStack` using mocked git/pr operations (or a table-driven integration test) covering: successful merge, rebase-not-allowed error, empty stack, single-PR stack, multi-PR stack.
- [x] 3.5 Verify: `go test ./internal/cli`

## 4. Spec and Documentation

- [x] 4.1 Sync the delta spec from `openspec/changes/land-whole-stack/specs/land/spec.md` to `openspec/specs/land/spec.md`.
- [x] 4.2 Update `SPEC.md` §6 and §15 to document the `whole-stack` land style and the `--whole-stack` flag.
- [x] 4.3 Update `CHANGELOG.md` with the new feature entry.

## 5. Final Validation

- [x] 5.1 Run `make vet && make fmt-check && make test`
- [x] 5.2 Run `go test -race ./...`
- [x] 5.3 Run `make build` and verify the `bpr land --whole-stack` flag appears in help output.

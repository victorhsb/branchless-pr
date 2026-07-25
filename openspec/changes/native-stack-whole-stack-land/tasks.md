## 1. Land Command — Native Whole-Stack Support

- [ ] 1.1 Add `landWholeStackNative(app *AppContext, st stack.Stack) error` function in `internal/cli/land.go`: check `RebaseMergeAllowed`, check `MergeQueueEnabled`, fetch, queue tip PR via `pr.MergeRebaseAuto`, checkout original branch, print queued message. No `pr.EditBase` call.
- [ ] 1.2 In `nativeLandPreflight`, change `ActionNoop` case to allow `whole-stack` (return `nil`) while still refusing `bottom-only` (call `nativeLandRefusal`).
- [ ] 1.3 In `landImpl`, dispatch to `landWholeStackNative` when the preflight detects a matching native Stack and style is `whole-stack`. Thread a `nativeWholeStack bool` flag from the preflight result through `landImpl`.
- [ ] 1.4 Update `nativeLandRefusal` to only be called for `bottom-only` style or `ActionAppend`; the error message for bottom-only should explain that bottom-only requires client-side base edits that GitHub rejects for stacked PRs.

## 2. SPEC.md Update

- [ ] 2.1 Update SPEC.md §6 (land) to document that `whole-stack` for matching native Stacks queues the tip PR without base retargeting, relying on GitHub's merge queue cascade.

## 3. Tests

- [ ] 3.1 Add test: `nativeLandPreflight` with `ActionNoop` and `whole-stack` returns nil (allowed).
- [ ] 3.2 Add test: `nativeLandPreflight` with `ActionNoop` and `bottom-only` returns refusal error.
- [ ] 3.3 Add test: `nativeLandPreflight` with `ActionAppend` and `whole-stack` returns refusal error.
- [ ] 3.4 Add test: `landWholeStackNative` calls `pr.MergeRebaseAuto` on the tip PR and does NOT call `pr.EditBase` (use fake gh logger).
- [ ] 3.5 Add test: `landWholeStackNative` checks `RebaseMergeAllowed` and `MergeQueueEnabled` before queuing.
- [ ] 3.6 Add test: `landWholeStackNative` does NOT delete local branches, rebase target, or rebase original branch after queuing.

## 4. Verification

- [ ] 4.1 Run `make test`, `make vet`, `make fmt-check`, `go test -race ./...`, `make build`.
- [ ] 4.2 Run `openspec validate native-stack-whole-stack-land`.
- [ ] 4.3 E2e test on connectlyai/connectly-backend: `bpr land --whole-stack` on a native-stacked 2-PR stack queues the tip PR for merge without error, no `gh pr edit -B` failures in stderr.

---
date: 2026-09-02
revised: 2026-09-03
status: active
type: refactor
title: Give the git integration a receiver, an injectable runner, and one home
---

# Git integration refactor

## Context

A review of `internal/git`, `internal/shell`, and their callers on 2026-09-02
found the wrappers correct and safe (argv validation, exact stash identity,
atomic force-with-lease) but structurally weak. Every finding below was
confirmed against the code at commit `f8a9a2a`. A Phase 0 boundary inventory on
2026-09-03 established the implementation scope below.

**Test seam is `$PATH`.** The functions in `internal/git/git.go` are
package-level and call `shell.Run` or `shell.Output` directly. Tests fake git
by writing a shell script into a temp dir and prepending it to `PATH`. This
pattern appears across nine test files:
`internal/git/git_test.go`, `internal/git/stash_test.go`,
`internal/cli/land_test.go`, `internal/cli/submit_test.go`,
`internal/cli/fix_test.go`, `internal/cli/root_stash_test.go`,
`internal/pr/pr_test.go`, `internal/pr/merge_rebase_test.go`, and
`internal/cli/agent_test.go`. The scripts are Unix-only, so these tests skip on
Windows (`runtime.GOOS == "windows"`). `internal/diagnose/runner.go` already
defines a `Runner` interface (`Output`, `Run`, `LookPath`) with a default
implementation and a `fakeRunner` in its tests; nothing outside `diagnose` uses
it.

- `agent_test.go` is a special case: it sets `PATH` to an empty temp dir to
  prove `agent` runs with no git/gh present. It is an assertion, not a fake.
  Post-refactor it becomes an assertion that the injected runner is never
  called (or it keeps the empty-PATH check unchanged).
- The stash tests in `git/stash_test.go` and
  `cli/root_stash_test.go` prepend a wrapper script that delegates to
  `$REAL_GIT` while injecting failures via `FORCE_DIRTY_STATUS` and
  `FAIL_STASH_RESTORE`. The reflog-behavior assertions want real git; the
  fault-injection ones only want a canned failure and become `shelltest.Fake`
  cases.

**`gh` lives in `internal/git`.** `CheckGHInstalled` and `GetGHUsername` call
the GitHub CLI. `internal/git/config.go` holds a package singleton whose only
job is letting tests override the username. It is used by three sites in
`cli/root_stash_test.go` and one in `git/git_test.go`.

**GitHub CLI spans three packages.** Besides
`internal/pr` (`pr.go` and `comments.go` both import `shell`),
`internal/nativestacks/api.go` runs `gh api` through its own `runFunc` seam,
and the two `gh` functions above live in `internal/git`. So the actual
production importers of `internal/shell` today are: `git`, `pr`, `diagnose`,
`nativestacks`, and `stack`.

**`internal/stack` shells out to git on its own.** `stack.Discover`
(`internal/stack/stack.go:16`) and `stack.NextID`
(`internal/stack/entry.go:310`) call `shell.Output` directly. `NextID` runs
`git ls-remote --heads <remote>` with no `--` and no `ValidateRemoteName`;
`git.ResolveRemoteRefs` runs the same command with both. Two `ls-remote`
parsers exist. `stack` already imports `git` (for `IsFullSHA` in `header.go`),
so passing a `*git.Repo` into `Discover` adds no new package edge.
`stack/verify.go` also calls `pr.View` directly as a fallback.

**`repoDir ...string` is a default argument in disguise.** Seven functions
take it and repeat the same four-line unpacking. One production caller passes
it: `git.BranchlessStackHead(repoRoot)` at `internal/cli/root.go:147`.

**Config-path resolution runs git before `PersistentPreRunE`.**
`config.FilePath` (`internal/config/config.go:31`) calls `git.RepoRoot()` from
inside `newRootCommand`, before the pre-run hook. Any injectable runner must
reach config-path resolution too, not only `PersistentPreRunE` — which is why
repo discovery and config load should move behind one `invocation.Bootstrap`
constructor.

**Argument order differs between siblings.** `Checkout(startPoint, branch)`,
`ForceUpdateBranch(branch, startPoint)`, `Rebase(onto, branch)`. All strings,
so a swap compiles. `Rebase` runs `git rebase <upstream> [<branch>]`, not
`--onto`, and its doc says extras go "between onto/upstream" while the code
appends them before `onto`.

**`shell.RunOpts.Check` is misdocumented.** The doc says Check is "the
default"; the zero value is false. Check only controls whether the error is
wrapped with the argv, never whether an error is returned. `shell.Output`
forces `Quiet` and `Check` to true, so the `Check: false` passed in
`TargetExists` is dead. `CurrentBranchName` and `RepoRoot` each have a
`NotARepo` branch identical to the fallback.

**`UncommittedChanges` returns `map[status]path`.** Two files with the same
status collapse to one entry. The only caller,
`invocation.RequireCleanRepo`, iterates keys and ignores values.

**`git.Error.Op` has one consumer, and it is a test.**
`cli/root_stash_test.go:90` asserts the substring `stash_apply`. No production
code does `errors.As` on `*git.Error` or reads `Op`. The values are
snake_case copies of the function name. Dropping `Op` touches that one
expectation.

**Validation is per wrapper, not per input.** Remote-facing wrappers
(`Fetch`, `ForcePush`, `ForcePushWithLease`, `DeleteRemoteBranches`,
`ResolveRemoteRefs`, `RepoSlug`, `TargetExists`) validate. Local wrappers
(`Checkout`, `CheckoutBranch`, `Rebase`, `RevParse`, `BranchExists`,
`ForceUpdateBranch`) do not and pass no `--`. Branch names reaching them come
from the template and from parsed `stack-info:` lines; the review found no
`ValidateRefName` call in `internal/stack` on the parsed path. Note: every
non-git caller of `RevParse`/`IsAncestor`/`MergeBase` passes `HEAD`,
`remote/target`, a full SHA, or a template branch, all of which the existing
`ValidateRefName` accepts. The naming split below makes the accepted argument
contract explicit.

**One 654-line file, no grouping.** `IsRebaseInProgress` is at line 148, its
siblings `IsMergeInProgress` and `IsCherryPickInProgress` at line 611.
`TargetExists` has no doc comment.

## Goals

1. One injectable subprocess seam, so no test needs to write a fake binary to
   `PATH`. The runner is injected at the composition root, not overridden
   per package.
2. `internal/git` and `internal/pr` are the only production importers of
   `internal/shell` besides `diagnose`. `nativestacks` transport folds into
   `pr`; `stack` stops shelling out.
3. Git wrappers hang off a `Repo` value that carries the working directory and
   the runner. No variadic `repoDir`.
4. All GitHub CLI (`gh`) usage lives in `internal/pr`, behind a `Client`.
   `internal/pr` is the single GitHub boundary.
5. Sibling wrappers share argument order and accurate docs.

## Non-goals

- Backwards compatibility. This is a solo project; intermediate commits do not
  need to preserve old APIs. Each old API is deleted in the same commit that
  introduces its replacement. No compatibility shims, no staged deprecation.
- Any user-visible behavior change from the structural commits. No spec in
  `docs/specs/` should need editing for commits 1–3 and 5. Commit 4 makes a
  deliberate, spec-backed safety change (see below), flagged as such.
- Replacing `git`/`gh` with an SDK (ADR-0001).
- Rewriting `diagnose`. It keeps the shared `shell.Runner` for now. Routing
  `diagnose` through `git`/`pr` so it stops building raw argv is real work and
  is recorded as **Future work** below, not done here.

## Constraints

- ADR-0002: `internal/shell` stays the only package that spawns subprocesses.
  A `Runner` interface in `shell` satisfies this; the fake lives in a test
  helper package and spawns nothing (it returns an in-memory error carrying
  the exit code, so no `sh -c "exit N"` and no reintroduced Unix dependency).
- ADR-0005: operation-state detection keeps going through
  `git rev-parse --git-path`. Moving `operationPathExists` onto `Repo` must
  not reintroduce literal `.git` paths.
- `docs/specs/git-operation-safety.md` section "Subprocess argument safety"
  describes the validation and `--` contract. Commit 4 adds validation the spec
  already requires; none is removed.
- Every commit must land on `main` already passing
  `make fmt-check && make vet && make test && go test -race ./... &&
  make build`. Commits go straight to main (no PR stack), but each one stays
  green so `git bisect` remains useful.

## Approach

Commits are aggregated by intent, not by mechanical step. Five commits, each
self-contained and green on `main`. Within a commit the old API and its
replacement change together — nothing survives as a shim.

1. **`shell`: injectable Runner + portable fake.**
2. **`git`: `Repo` receiver + composition-root injection** (`invocation.Bootstrap`).
3. **`pr`: single GitHub boundary** (absorbs `git`'s `gh` funcs and
   `nativestacks` transport; deletes package-level `shell.Run`/`Output`).
4. **`stack` + safety**: move `stack`'s two shell-outs behind `git.Repo`; add
   the `--`/validation the spec requires on the `NextID` and local-wrapper
   paths.
5. **`git`: cleanup** — signatures, docs, `HasTrackedChanges`, file split,
   optional `Error.Op` removal.

The subsections below give the detail for each commit.

### Commit 1 — Injectable runner in `internal/shell`

Add to `internal/shell`:

```go
type Runner interface {
    Run(args []string, opts RunOpts) ([]byte, []byte, error)
    Output(args []string, opts RunOpts) (string, error)
}

// Default runs commands through os/exec. It is the only Runner that spawns.
type Default struct{}
```

Delete the package-level `Run` and `Output` functions in this commit only if
no caller remains; they do not, so keep them here and delete them in commit 4
when the last callers convert. (They are not "compatibility" — they are the
same functions, kept alive only until their callers move.)

Add a portable exit-code accessor so tests never spawn a process to express an
exit status:

```go
// ExitCode reports the process exit code carried by err, if any.
// *exec.ExitError satisfies this, and so does the in-memory error the fake
// returns.
func ExitCode(err error) (int, bool)
```

Migrate call sites that only inspect the numeric status to `shell.ExitCode`.
Broaden `AsExitError` to the same exit-code interface for the two callers that
also preserve the underlying process error in their return value.

Add `internal/shell/shelltest` with a `Fake` that:

- implements `shell.Runner`;
- records every `args` slice and `RunOpts` it receives, in order;
- matches calls by a caller-supplied predicate or by argv prefix and returns
  canned stdout, stderr, and an in-memory error whose `ExitCode() int` returns
  the configured code — **no `sh -c "exit N"`, no real process, no Unix
  dependency**;
- fails the test on an unmatched call unless configured to log and return
  success, which is what the argv-logger fakes in `cli` do today;
- is safe for concurrent use (the `-race` suite runs commands that may call
  it from goroutines).

Fix the `RunOpts.Check` doc: state that the zero value is false, that Check
only adds the argv to the error text, and that `Output` overrides both `Quiet`
and `Check`.

Make `diagnose.DefaultRunner` delegate to `shell.Default{}`. Leave its
`LookPath` method and `fakeRunner` alone — `diagnose` stays on the shared
runner (see Future work).

### Commit 2 — `git.Repo` receiver + composition-root injection

Replace the package-level functions in `internal/git` with methods on:

```go
type Repo struct {
    Dir string       // "" means the process working directory
    run shell.Runner // nil means shell.Default{}
}

func New(dir string, run shell.Runner) *Repo
```

Rules for the conversion:

- Drop every `repoDir ...string` parameter. `Dir` replaces it.
- Fold the four-line unpacking into one private helper. Prefer
  `func (r *Repo) opts(o shell.RunOpts) shell.RunOpts { o.Dir = r.Dir; return o }`
  over an `opts(quiet bool)` helper, because call sites also vary `Check`,
  `Stdin`, and explicit output buffers, not just `Quiet`.
- Keep `IsFullSHA`, `ValidateRemoteName`, `ValidateRefName`, and `StashRef`
  as package-level: they touch no subprocess.

**Composition root.** Extract the bootstrap work now in
`PersistentPreRunE` (config load, repo discovery, gh-installed check, current
branch, branchless-head detection, username, stash) into
`invocation.Bootstrap(runner shell.Runner, args CommonArgs, ...) (*AppContext, error)`,
so `root.go` shrinks to Cobra wiring and the init sequence is testable without
a Cobra command.

- `newRootCommand` takes a `shell.Runner`; production `Execute` passes
  `shell.Default{}`, tests pass one `shelltest.Fake`.
- `config.FilePath` runs `git.RepoRoot()` *before* the pre-run hook, so
  config-path resolution must also use the injected runner. Either move config
  load into `Bootstrap` or give `config.FilePath` a `*git.Repo`. Prefer moving
  it into `Bootstrap` so there is one discovery path.
- `AppContext` gains `Git *git.Repo`. `RepoRoot`/`CurrentBranchName` run on a
  bootstrap `git.New("", runner)` before the root is known; every later call
  uses the `AppContext.Git` built once the root resolves.
- `invocation.RequireCleanRepo`, `WithRecovery`, and `RestoreStash` take the
  `*Repo` from `AppContext` instead of calling package functions.

**Tests.** Convert the git-side `PATH` fakes (`git/git_test.go`, and the
CLI-level `land`/`submit`/`fix` fakes for their git calls) to `shelltest.Fake`.
For the hybrid stash tests: keep the reflog-behavior assertions in
`git/stash_test.go` and `cli/root_stash_test.go` on `shell.Default{}` with
`Dir` set to a temp repo (real git), and convert the fault-injection cases
(`FORCE_DIRTY_STATUS`, `FAIL_STASH_RESTORE`) to `shelltest.Fake`. Drop the
`withWorkingDir` chdir helper and the wrapper scripts. Delete the
`runtime.GOOS == "windows"` skips that only existed for shell-script fakes.
This commit keeps argv byte-identical — shape only — so the argv-log tests pass
unchanged.

### Commit 3 — `pr` is the single GitHub boundary

Give `internal/pr` a `Client` carrying the runner and convert the
subprocess-backed functions (`pr.go` and `comments.go`) to methods:

```go
type Client struct { run shell.Runner }
func NewClient(run shell.Runner) *Client
```

Keep pure helpers (`ValidateRef`, `IsNativeStackBaseError`, parsers) package-
level. Fold the existing ad-hoc seams (`graphqlRunner`, `rulesRunner`, and
`comments.go`'s `runGHJSON`) into the `Client`'s runner.

- Move `CheckGHInstalled` and `GetGHUsername` from `internal/git` to the
  `Client`. Delete `internal/git/config.go`, `SetUsernameOverride`,
  `DefaultConfig`, and `TestUsernameOverride`. The three sites in
  `cli/root_stash_test.go` and the one in `git/git_test.go` instead have the
  fake return the login from the `gh api graphql` response.
- Absorb `nativestacks` transport into `pr`. `nativestacks.APIClient`'s
  `gh api` request/response plumbing, status parsing, and `IsAPIStatus` move
  onto the `pr.Client`. `internal/nativestacks` keeps the domain side
  (eligibility, `Classify`, `FeatureUnavailable`, PR/stack validation) and
  calls `pr` through a small interface. If little domain logic remains after
  the transport leaves, collapse `nativestacks` into `pr` — decide when the
  split is visible in the diff, not up front. The four
  `nativestacks.NewAPIClient(owner, repo)` construction sites
  (`abandon.go`, `land.go`, `nativestacks_helper.go`, `view.go`) switch to the
  injected client.
- `stack/verify.go`'s fallback call to `pr.View` becomes a call through the
  injected `Client`/provider.
- The `reports/checks` and `reports/comments` `Fetcher` seams stay as function
  types, now backed by `Client` methods; those packages need no shell
  knowledge.
After this commit, all `gh` execution is behind `pr.Client`. Convert the
remaining `PATH` fakes (`pr_test.go`, `merge_rebase_test.go`, and the gh side
of the CLI tests) to `shelltest.Fake`; `agent_test.go` keeps its empty-`PATH`
assertion or asserts the injected runner is never called.

This commit contains the only intentional behavior change in the whole plan.
The commit message must call it out.

- `stack.Discover(base, head)` becomes `stack.Discover(repo *git.Repo, base,
  head)` and calls a new `repo.RevListHeaders(base, head) (string, error)`
  that runs `git rev-list --header ^BASE HEAD`. Parsing stays in `stack`.
  `stack` already imports `git`, so no new package edge. Update all four
  `Discover` call sites (`abandon.go`, `fix.go`, `land.go`, `submit.go`,
  `view.go`) plus `internal/stackstate/stackstate.go`.
- `stack.NextID` stops running git. Change its signature to be pure over
  `remoteBranches []string` (return `int`, no error), and have
  `stack.AssignHeads` fetch them with a new
  `repo.RemoteBranches(remote) ([]string, error)` that shares the `ls-remote`
  parser with `ResolveRemoteRefs`.
- **Behavior change:** this routes the `NextID` path through the same `--` and
  `ValidateRemoteName` as `ResolveRemoteRefs`. `NextID` currently runs
  `git ls-remote --heads <remote>` with neither. The safety plan on 2026-08-07
  intended this and `docs/specs/git-operation-safety.md` §"Subprocess argument
  safety" already describes it, so the spec likely needs no edit — confirm.
- Also in this commit, tighten the local wrappers the spec covers:
  add `ValidateRefName` to `Checkout`, `CheckoutBranch`, `Rebase`,
  `BranchExists`, and `ForceUpdateBranch`. For revision-taking wrappers
  (`RevParse`, `IsAncestor`, `MergeBase`) the current `ValidateRefName` already
  accepts every value they receive (`HEAD`, `remote/target`, full SHAs,
  template branches), so guarding them is safe; the internal `^base` in
  `rev-list --header` is not a wrapper argument. Rename the validator used on
  revision positionals to make the contract legible — e.g. a
  `ValidateRevisionArg` alongside a branch-name `ValidateRefName` — rather than
  reusing one vaguely named check everywhere. Do **not** add `--` to
  `rev-parse` (the `TargetExists` comment explains why). Add a hostile-input
  table test mirroring `TestRemoteWrappersRejectHostileBranchName` for the
  local wrappers, asserting the runner received no command.
- Delete the package-level `shell.Run`/`shell.Output` after `stack` moves off
  them. `shell.Default{}` is then the only concrete runner.

After this commit,
`grep -rn --include='*.go' "internal/shell" internal | grep -v _test | grep -v "^internal/shell/"`
lists only `internal/git`, `internal/pr`, and `internal/diagnose`.

### Commit 5 — Cleanup: signatures, docs, clean-tree check, file split

Mechanical tail, no behavior change beyond `HasTrackedChanges` (which the only
caller already treats identically).

- `Checkout(startPoint, branch)` becomes `Checkout(branch, startPoint)` to
  match `ForceUpdateBranch`. Update the call sites (`abandon.go:149`,
  `land.go:193`, `land.go:229`) and the argv-log expectations in their tests.
- Rename `Rebase(onto, branch, extras...)` to `Rebase(upstream, branch,
  extras...)`. Fix the doc: extras precede `upstream`. Same for
  `RebaseWithAuthorDate`.
- Replace `UncommittedChanges() (map[string]string, error)` with
  `HasTrackedChanges() (bool, error)` that returns true for any porcelain line
  not starting with `??`. Update `invocation.RequireCleanRepo`.
- Delete the `NotARepo` branches in `CurrentBranchName` and `RepoRoot`; the
  fallback already returns the same error. Keep the `NotARepo` constant if
  anything else references it (`grep -rn NotARepo`), else delete it.
- Remove the dead `Check: false` in `TargetExists`.
- Give `TargetExists` a doc comment and the blank line before it.
- Split `git.go` by concern (bodies unchanged):
  - `worktree.go`: `RepoRoot`, `CurrentBranchName`, `HasTrackedChanges`,
    `Checkout`, `CheckoutBranch`, `ForceUpdateBranch`, `DeleteLocalBranches`,
    `BranchExists`, `CommitAmend`, `CommitMsg`
  - `refs.go`: `RevParse`, `MergeBase`, `IsAncestor`, `RevListHeaders`,
    `BranchlessStackHead`, `Rebase`, `RebaseWithAuthorDate`
  - `remote.go`: `Fetch`, `ForcePush`, `ForcePushWithLease`,
    `DeleteRemoteBranches`, `ResolveRemoteRefs`, `RemoteBranches`, `RepoSlug`,
    `parseRepoSlug`, `TargetExists`
  - `stash.go`: `StashRef`, `StashSave`, `StashRestore`, `stashHead`,
    `stashSelector`
  - `sequencer.go`: `IsRebaseInProgress`, `IsMergeInProgress`,
    `IsCherryPickInProgress`, `AnySequencerInProgress`, `operationPathExists`
  Keep `error.go` and `validate.go`. Move the test files to match.
- Optional, only if it stays small: drop `git.Error.Op`. Replace
  `&Error{Op: "x", Err: err}` with `fmt.Errorf("git x: %w", err)` and delete
  the `Error` type. The one consumer is `root_stash_test.go`'s `stash_apply`
  substring check, which becomes the new wording. Skip if the churn is not
  worth it; nothing depends on it.

## Future work (out of scope, record when closing the plan)

- **`diagnose` should stop building raw `git`/`gh` argv** and get its answers
  through `internal/git` and `internal/pr`, so the end state is `git` and `pr`
  as the *only* shell importers. `diagnose`'s checks map results to best-effort
  `ok|warning|blocking|unknown` statuses, so this is a real reshape, not a
  mechanical move. Track it as a follow-up.

## Verification

After every commit:

```
make fmt-check && make vet && make test && go test -race ./... && make build
```

After commit 2, run the end-to-end check that exercised the most wrappers and
confirm the argv logs are byte-identical (commits 1–3 change shape only):

```
go test ./internal/cli -run 'TestLand|TestSubmit|TestFix' -v
```

After commit 4, confirm the import boundary:

```
grep -rln --include='*.go' '"github.com/victorhsb/branchless-pr/internal/shell"' internal | grep -v _test
```

Expected output is files under `internal/git`, `internal/pr`,
`internal/diagnose`, and `internal/shell` only.

## Tasks

- [x] Phase 0: boundary inventory (folded into Context above)
- [x] Commit 1 — `shell`: `Runner`, `Default`, portable `ExitCode`,
      `shelltest.Fake` (process-free); fix `Check` doc; `diagnose.DefaultRunner`
      delegates to `shell.Default`
- [ ] Commit 2 — `git.Repo` receiver; drop `repoDir` variadics;
      `invocation.Bootstrap(runner, ...)`; runner injected at `newRootCommand`;
      config-path resolution uses the injected runner; `AppContext.Git`;
      convert git-side `PATH` fakes + hybrid stash fault-injection cases to
      `shelltest.Fake`; remove Windows skips; argv byte-identical
- [ ] Commit 3 — `pr.Client` single GitHub boundary: absorb `CheckGHInstalled`/
      `GetGHUsername` (delete `git/config.go` + username override) and
      `nativestacks` transport; migrate `stack/verify.go` and the four
      `NewAPIClient` sites; convert remaining `PATH` fakes
- [ ] Commit 4 — `Repo.RevListHeaders`, `Repo.RemoteBranches`; pure
      `stack.NextID(remoteBranches)`; `stack.Discover(repo, ...)` + all callers
      incl. `stackstate`; **safety change** flagged in commit message
      (`--`/validation on `NextID` and local wrappers, revision-arg validator
      rename, hostile-input tests); delete package-level
      `shell.Run`/`shell.Output`
- [ ] Commit 5 — `Checkout(branch, startPoint)`; `Rebase(upstream, ...)` rename
      and doc; `HasTrackedChanges`; delete dead `NotARepo` branches and
      `Check: false`; split `git.go`; optional `Error.Op` removal
- [ ] Confirm `docs/specs/` needed no edits; add a CHANGELOG entry only if a
      user-visible message changed
- [ ] Record the "`pr` is the single GitHub boundary" decision as an ADR and
      the `diagnose` reshape as future work when closing the plan
- [ ] Run the `document` skill to close the plan

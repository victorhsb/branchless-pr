# stack-pr (Go Port) – Task Tracker

## Legend

- `[ ]` Not started
- `[~]` In progress
- `[x]` Done

---

## Phase 1: Foundation

- [x] Initialize Go module (`go.mod`) for `github.com/victorhsb/branchless-pr`
- [x] Scaffold directory layout (`cmd/`, `internal/`)
- [x] Add `.gitignore` for Go artefacts (`*.exe`, `bin/`, `vendor/`)

## Phase 2: Core Packages (bottom-up)

- [x] `internal/shell/shell.go` — subprocess wrapper
- [x] `internal/git/error.go` — `GitError`, `GIT_NOT_A_REPO_ERROR = 128`, `GIT_SHA_LENGTH = 40`
- [x] `internal/git/git.go` — Git helper functions
- [x] `internal/git/config.go` — `GitConfig` with `username_override`
- [x] `internal/config/config.go` — INI config parsing, defaults, write path
- [x] `internal/stack/header.go` — `CommitHeader` struct and parser
- [x] `internal/stack/entry.go` — `StackEntry` struct + metadata read/write
- [x] `internal/stack/stack.go` — stack discovery, range resolution, base/head assignment
- [x] `internal/stack/print.go` — ANSI colours, terminal hyperlinks, stack line formatting
- [x] `internal/stack/crosslink.go` — PR body and cross-link generation
- [x] `internal/stack/verify.go` — stack verification against `gh pr view`
- [x] `internal/pr/pr.go` — thin wrappers around `gh pr *`

## Phase 3: CLI Commands

- [x] `internal/cli/types.go` — `CommonArgs`, config resolution, `RequireCleanRepo`
- [x] `internal/cli/root.go` — arg parsing via `spf13/cobra`, persistent flags (SPEC §6.1)
- [x] `internal/cli/config.go` — `config` command (SPEC §6.2 / §7)
- [x] `internal/cli/submit.go` — `submit` / `export` algorithm (SPEC §13)
- [x] `internal/cli/view.go` — `view` algorithm (SPEC §17)
- [x] `internal/cli/land.go` — `land` algorithm (SPEC §15)
- [x] `internal/cli/abandon.go` — `abandon` algorithm (SPEC §16)

## Phase 4: Testing

- [x] `internal/shell/shell_test.go` — quiet vs non-quiet behaviour, output strip
- [x] `internal/git/git_test.go` — rebase-in-progress, username override, SHA validation
- [x] `internal/stack/entry_test.go` — metadata parsing, branch template generation/match/extract
- [x] `internal/config/config_test.go` — read/write roundtrip, parse arg, defaults, getbool

## Phase 5: Polish

- [x] Stash pop on success (SPEC §8 step 21)
- [x] `--stash` scoped to submit/export only (SPEC §6.2)
- [x] `IsRebaseInProgress` reads `.git/rebase-*` directly per SPEC §11
- [x] ANSI / hyperlink output matches SPEC §19
- [x] Error messages with red `ERROR:` prefix per SPEC §20

## Phase 6: Remaining Commands

- [x] `internal/cli/land.go` — implemented SPEC §15
- [x] `internal/cli/abandon.go` — implemented SPEC §16

## Phase 7: Documentation & CI

- [x] `Makefile` with build/test/vet/fmt targets
- [x] `README.md` for the Go port
- [x] `CHANGELOG.md` (Go port history + pointer to Python history)
- [x] `LICENSE` (Apache-2.0 with LLVM Exceptions)
- [x] `CONTRIBUTING.md`
- [x] `.github/workflows/ci.yml` — `go test`, `go vet`, `gofmt -l`, build

---

## Currently Working On

Nothing — port is feature-complete per SPEC.md. End-to-end smoke test against
a real Git+gh environment remains the responsibility of release engineering.

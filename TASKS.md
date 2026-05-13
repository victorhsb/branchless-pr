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
  - Replicate `run_shell_command` semantics (quiet mode, forbid `shell`, debug logging)
  - Replicate `get_command_output` (capture output, strip trailing whitespace)
- [x] `internal/git/error.go` — `GitError` type, error constants (`GIT_NOT_A_REPO_ERROR = 128`, `GIT_SHA_LENGTH = 40`)
- [x] `internal/git/git.go` — Git helper functions
  - `is_full_git_sha`
  - `branch_exists`
  - `get_current_branch_name`
  - `get_repo_root`
  - `get_uncommitted_changes`
  - `check_gh_installed`
  - `get_gh_username`
  - `get_changed_files`
  - `get_changed_dirs`
  - `is_rebase_in_progress`
- [x] `internal/git/config.go` — `GitConfig` singleton with `username_override`
- [x] `internal/config/config.go` — INI config parsing
  - `STACKPR_CONFIG` env override
  - Defaults: `common.*`, `repo.*`, `land.*`
  - `getboolean` equivalent
  - `config` command write path
- [x] `internal/stack/header.go` — `CommitHeader` struct and parser
- [x] `internal/stack/entry.go` — `StackEntry` struct + metadata read/write
- [x] `internal/stack/stack.go` — stack discovery, range resolution, base/head assignment
- [x] `internal/stack/print.go` — ANSI colours, terminal hyperlinks, stack line formatting
- [x] `internal/pr/pr.go` — thin wrappers around `gh pr *` commands

## Phase 3: CLI Commands

- [x] `internal/cli/types.go` — `CommonArgs`, config resolution, `RequireCleanRepo`
- [x] `internal/cli/root.go` — arg parsing via `spf13/cobra`, persistent flags matching SPEC §6.1
- [x] `internal/cli/config.go` — `config` command (§6.2 / §7 of SPEC)
- [~] `internal/cli/submit.go` — `submit` / `export` algorithm (§13 of SPEC) — stub only
- [~] `internal/cli/view.go` — `view` algorithm (§17 of SPEC) — stub only
- [~] `internal/cli/land.go` — `land` algorithm (§15 of SPEC) — stub only
- [~] `internal/cli/abandon.go` — `abandon` algorithm (§16 of SPEC) — stub only

## Phase 4: Entrypoint & Build

- [x] `cmd/stack-pr/main.go` — calls `cli.Execute()`
- [x] `go build ./cmd/stack-pr` passes
- [x] All commands + global flags render help correctly per SPEC
- [ ] Add `Makefile` or simple build script (optional)

## Phase 5: Documentation & CI

- [ ] Port `README.md` from Python version to Go install instructions (`go install`)
- [ ] Port `CHANGELOG.md` (reset for Go port, keep historical context)
- [ ] Port `LICENSE` / `CONTRIBUTING.md`
- [ ] `.github/workflows/ci.yml`
  - `go test ./...`
  - `go vet ./...`
  - `gofmt -l` check
  - Build `cmd/stack-pr`

## Phase 6: Testing

- [ ] `internal/shell/shell_test.go` — quiet vs non-quiet behaviour
- [ ] `internal/git/git_test.go` — rebase-in-progress detection, username override, branch ID extraction
- [ ] `internal/stack/entry_test.go` — metadata parsing, branch name generation
- [ ] `internal/config/config_test.go` — read/write roundtrip
- [ ] Integration-style tests where feasible (mock `gh` / `git` in temp dirs)

## Phase 7: Polish

- [ ] Review SPEC §18–20 (cleanliness, safety, error messages, output formatting)
- [ ] Ensure all error messages match SPEC descriptions
- [ ] Ensure ANSI / hyperlink output matches SPEC
- [ ] End-to-end smoke test in a real Git repo

---

## Currently Working On

Phase 3: Implementing command RunE functions (submit, view, land, abandon)

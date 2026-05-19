# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go port of the Python `stack-pr` CLI (https://github.com/modular/stack-pr). Creates, updates, views, abandons, and lands stacked GitHub pull requests. A "stack" is the ordered list of commits in `BASE..HEAD`; each commit maps to exactly one PR, with the bottom PR targeting `main` and each higher PR targeting the generated branch of the commit below it.

`SPEC.md` is the behavioral source of truth (mirrors the Python tool's algorithms). If a port decision contradicts SPEC.md, the spec wins — update both in the same PR if behavior must change.

## Build, test, lint

```bash
make build         # go build -o stack-pr ./cmd/stack-pr (injects version via -ldflags)
make test          # go test ./...
make vet           # go vet ./...
make fmt-check     # fails if gofmt -l reports anything
make fmt           # gofmt -w .
make tidy          # go mod tidy
```

Single test: `go test ./internal/cli -run TestSubmitDryRun`. The package layout makes `./internal/<pkg>` the right granularity for `-run` filters.

CI (`.github/workflows/ci.yml`) runs gofmt check → go vet → go test → go build on Go 1.23. Match this locally before pushing.

## Architecture

Entry: `cmd/stack-pr/main.go` → `internal/cli.Execute()` → Cobra root command in `internal/cli/root.go`.

### Package map (internal/)

- `cli/` — Cobra subcommands (`submit`/`export`, `view`, `land`, `abandon`, `config`, `agent`). `root.go` wires shared flags, loads config, resolves `CommonArgs`, sets up `AppContext`, and gates the `land` subcommand on `land.style != disable`.
- `stack/` — Core model: `Entry`, `CommitHeader`, stack discovery via `git rev-list --header ^BASE HEAD` (NUL-delimited), header parsing, branch name templating (`$USERNAME/stack` etc.), TOC/crosslink rendering.
- `git/` — Typed wrappers around `git` (merge-base, current branch, stash, push, branchless stack head detection, `gh` install check, GH username).
- `pr/` — `gh` CLI wrappers for PR create/edit/view.
- `shell/` — The single subprocess wrapper. **Do not call `os/exec` directly outside this package** (per CONTRIBUTING.md).
- `config/` — INI parsing for `<repo-root>/.stack-pr.cfg` (override path with `STACKPR_CONFIG`). Sections: `[common]`, `[repo]`, `[land]`. Defaults merged in `cli/root.go`.
- `agent/` — Static, deterministic LLM-facing prompts for `stack-pr agent prompt [topic]`. Side-effect-free; runs outside a repo.
- `diagnose/` — Read-only diagnostic engine for `stack-pr agent diagnose`. Best-effort: reportable failure modes (dirty tree, missing PR metadata, rebase in progress, not in a repo) appear in the JSON envelope with `status` of `ok|warning|blocking|unknown` rather than causing the command to exit non-zero. `--online` opt-in enables `gh` checks; default is fully offline.

### Cross-cutting flow

`PersistentPreRunE` in `root.go` does heavy lifting for non-agent commands: merges config + flags into `CommonArgs`, validates the branch name template (`$ID` is required), checks `gh` is installed, finds repo root, resolves the current branch, auto-detects the git-branchless stack top when `--head` is not explicit, fetches the GH username, optionally stashes (submit/export only, skipped under `--dry-run`), enforces a clean tree except for `view`/`config`, checks `REMOTE/TARGET` exists (hint about `master` if `main` missing), and deduces `BASE` via `git merge-base` if not supplied. The `agent` subtree is short-circuited: it skips repo discovery, gh checks, and config-path resolution so it works outside a git repo.

`AppContext` (cli/types.go) is the resolved per-invocation state, threaded through `context.Context` via `FromContext`. `WithRecovery` wraps mutating commands to restore the original branch and pop the auto-stash on error/panic.

### Port invariants

- **Shell out to `git` and `gh`.** No Go GitHub SDK.
- **Each commit ↔ one PR.** Stack metadata is encoded in the commit message; `abandon` strips it; `land` squash-merges the bottom and rebases the rest.
- **`--dry-run` (submit/export) performs no local Git mutation, no remote push, no PR write.** Stash is skipped under dry-run for the same reason.
- **`land` is removable.** If `land.style = disable` in config, the subcommand is not registered at all.
- **Branch template must contain `$ID`** (or implicitly via `/$ID`).

## Spec-driven workflow

This repo uses OpenSpec (`openspec/`). New behavioral changes go through a change proposal in `openspec/changes/` and archived specs live in `openspec/specs/`. Use the `openspec-*` / `opsx:*` skills for the workflow (propose → continue → apply → verify → archive). When porting behavior, the corresponding `SPEC.md` section must agree.

## Conventions

- Errors propagate via explicit returns; no panics for control flow.
- Table-driven tests are the norm; see `internal/cli/*_test.go` and `internal/stack/entry_test.go`.
- `CHANGELOG.md` documents user-facing shipped behavior only — keep OpenSpec workflow bookkeeping out of it.
- Don't add a Go GitHub SDK dependency; don't bypass `internal/shell`.

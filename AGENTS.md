# AGENTS.md

## Project

Go port of the Python `stack-pr` CLI (https://github.com/modular/stack-pr). Creates, updates, views, abandons, comments, checks, and lands stacked GitHub pull requests. A "stack" is the ordered list of commits in `BASE..HEAD`; each commit maps to exactly one PR, with the bottom PR targeting `main` and each higher PR targeting the generated branch of the commit below it.

Behavioral specs live in `docs/specs/` (flattened, one file per area). If a port decision contradicts a spec, the spec wins — update both in the same PR if behavior must change. The upstream Python tool (https://github.com/modular/stack-pr) is the original reference.

## Build, test, lint

Requires **Go 1.23+**.

```bash
make build         # go build -o bpr ./cmd/bpr (injects version via -ldflags)
make test          # go test ./...
make vet           # go vet ./...
make fmt-check     # fails if gofmt -l reports anything
make fmt           # gofmt -w .
make tidy          # go mod tidy
```

**`make build` produces the `bpr` binary.** The `stack-pr` standalone binary is deprecated; `install.sh` creates a `stack-pr` symlink to `bpr` for backward compatibility.

Single test: `go test ./internal/cli -run TestSubmitDryRun`. The package layout makes `./internal/<pkg>` the right granularity for `-run` filters.

CI (`.github/workflows/ci.yml`) runs: **gofmt check → go vet → go test → go test -race ./... → go build**. Match this locally before pushing.

## Architecture

Entry: `cmd/bpr/main.go` → `internal/cli.Execute()` → Cobra root command in `internal/cli/root.go`.

### Package map (internal/)

- `cli/` — Cobra subcommands (`submit`/`export`, `view`, `land`, `abandon`, `config`, `agent`, `comments`, `checks`). `root.go` wires shared flags, loads config, resolves `CommonArgs`, sets up `AppContext`, and gates the `land` subcommand on `land.style != disable`.
- `stack/` — Core model: `Entry`, `CommitHeader`, stack discovery via `git rev-list --header ^BASE HEAD` (NUL-delimited), header parsing, branch name templating (`$USERNAME/stack` etc.), TOC/crosslink rendering.
- `git/` — Typed wrappers around `git` (merge-base, current branch, stash, push, branchless stack head detection, `gh` install check, GH username).
- `pr/` — `gh` CLI wrappers for PR create/edit/view/comments/checks.
- `shell/` — The **only** subprocess wrapper. **Do not call `os/exec` directly outside this package** (per `CONTRIBUTING.md`).
- `config/` — INI parsing for `<repo-root>/.stack-pr.cfg` (override path with `STACKPR_CONFIG`). Sections: `[common]`, `[repo]`, `[comments]`, `[land]`. Defaults merged in `cli/root.go`.
- `agent/` — Static, deterministic LLM-facing prompts for `stack-pr agent prompt [topic]`. Side-effect-free; runs outside a repo.
- `diagnose/` — Read-only diagnostic engine for `stack-pr agent diagnose`. Best-effort: reportable failure modes (dirty tree, missing PR metadata, rebase in progress, not in a repo) appear in the JSON envelope with `status` of `ok|warning|blocking|unknown` rather than causing the command to exit non-zero. `--online` opt-in enables `gh` checks; default is fully offline.

### Cross-cutting flow

`PersistentPreRunE` in `root.go` does heavy lifting for non-agent commands: merges config + flags into `CommonArgs`, validates the remote name and target branch so config values cannot be read by `git` as options or transport URLs, checks `gh` is installed, finds repo root, resolves the current branch, auto-detects the git-branchless stack top when `--head` is not explicit, fetches the GH username, optionally stashes (submit/export only, skipped under `--dry-run`), enforces a clean tree except for `view`/`config`, checks `REMOTE/TARGET` exists (hint about `master` if `main` missing), and deduces `BASE` via `git merge-base` if not supplied. The `agent` subtree is short-circuited: it skips repo discovery, gh checks, and config-path resolution so it works outside a git repo.

`AppContext` (`cli/types.go`) is the resolved per-invocation state, threaded through `context.Context` via `FromContext`. `WithRecovery` wraps mutating commands to restore the original branch and pop the auto-stash on error/panic.

### Port invariants

- **Shell out to `git` and `gh`.** No Go GitHub SDK.
- **Each commit ↔ one PR.** Stack metadata is encoded in the commit message; `abandon` strips it; `land` squash-merges the bottom and rebases the rest.
- **`--dry-run` (submit/export) performs no local Git mutation, no remote push, no PR write.** Stash is skipped under dry-run for the same reason.
- **`land` is removable.** If `land.style = disable` in config, the subcommand is not registered at all.
- **Branch template always yields an `$ID`.** `stack.ParseTemplate` appends `/$ID` when the template omits it, so any non-empty template is usable; the default `$USERNAME/stack` relies on this. There is no rejection path.
- **Config- and commit-authored values are validated before they reach `git`/`gh` argv** (`git.ValidateRemoteName`, `git.ValidateRefName`, `pr.ValidateRef`); remote-authored text is stripped of control characters before terminal output (`internal/textsafe`).

## Spec-driven workflow

Behavioral specs live in `docs/specs/`. Behavioral changes go through the LocalSpec workflow: the `plan` skill writes a single-file plan to `docs/plans/<name>.md`, implementation updates code and specs together, and the `document` skill finalizes (verifies specs, prompts for ADRs in `docs/adr/`, deletes the plan). When changing behavior, the corresponding `docs/specs/` section must agree.

- **Specs** (`docs/specs/`): Flattened current behavioral contracts, one file per area. Read by agents before touching code.
- **Plans** (`docs/plans/`): Active implementation work; single file per plan with frontmatter and task checklist. Ephemeral — deleted after completion.
- **ADRs** (`docs/adr/`): Architecture Decision Records for constraints and "why we can't do X"; managed via `adrm`, sequential from 0001.

## Conventions

- Errors propagate via explicit returns; no panics for control flow.
- Table-driven tests are the norm; see `internal/cli/*_test.go` and `internal/stack/entry_test.go`.
- `CHANGELOG.md` documents user-facing shipped behavior only — keep workflow bookkeeping out of it.
- Don't add a Go GitHub SDK dependency; don't bypass `internal/shell`.

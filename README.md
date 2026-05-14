# stack-pr (Go port)

`stack-pr` is a command-line tool for creating, updating, viewing, abandoning, and landing stacked GitHub pull requests. This is the Go port of the [Modular `stack-pr`](https://github.com/modular/stack-pr) Python tool. It preserves the original tool's algorithms and CLI surface; see `SPEC.md` for the full specification.

A stack is the ordered list of local commits in a Git revision range (`BASE..HEAD`). Each commit corresponds to exactly one GitHub PR. The bottom PR targets the repository target branch (normally `main`); every higher PR targets the generated branch for the previous commit. This way each PR review shows only one logical commit while still preserving dependency order.

## Install

```bash
go install github.com/victorhsb/branchless-pr/cmd/stack-pr@latest
```

Or build from source:

```bash
git clone https://github.com/victorhsb/branchless-pr
cd branchless-pr
go build -o stack-pr ./cmd/stack-pr
```

## Requirements

- Go 1.23+
- `git`
- `gh` (GitHub CLI) authenticated via `gh auth login`. SSH auth is recommended.

The tool shells out to `git` and `gh`; no Go GitHub SDK is used.

## Quick start

```bash
# create some commits on a feature branch
git checkout -b my-feature main
# ... commit a few times ...

# inspect the stack
stack-pr view

# submit / update the stack of PRs
stack-pr submit

# land the bottom-most PR
stack-pr land

# remove all stack metadata and clean up generated branches
stack-pr abandon
```

## Commands

- `stack-pr submit` (alias: `export`) — create or update PRs for each commit.
- `stack-pr view` — inspect the stack without modifying anything.
- `stack-pr land` — squash-merge the bottom PR and rebase the rest.
- `stack-pr abandon` — strip stack metadata and delete generated branches.
- `stack-pr config <section>.<key>=<value>` — write a setting to `.stack-pr.cfg`.

## Shared options

| Flag | Description |
|---|---|
| `-R, --remote` | Remote name (default `origin`). |
| `-B, --base` | Local base revision (default merge-base). |
| `-H, --head` | Local head revision (default `HEAD`). |
| `-T, --target` | Remote target branch (default `main`). |
| `--hyperlinks` / `--no-hyperlinks` | Enable terminal hyperlinks. |
| `-V, --verbose` | Verbose subprocess output. |
| `--branch-name-template` | Generated branch template (default `$USERNAME/stack`). |
| `--show-tips` / `--no-show-tips` | Post-command guidance. |

## Submit-only options

| Flag | Description |
|---|---|
| `--keep-body` | Preserve current PR body after the stack TOC. |
| `-d, --draft` | Create new PRs as draft. |
| `--draft-bitmask` | Per-PR draft bitmask (e.g. `010`). |
| `--reviewer` | Reviewer list. |
| `-s, --stash` | Stash uncommitted changes during submit. |

## Configuration

Config lives at `<repo-root>/.stack-pr.cfg` (override with `STACKPR_CONFIG`). Example:

```ini
[common]
verbose = false
hyperlinks = true
show_tips = true

[repo]
remote = origin
target = main
reviewer = someuser
branch_name_template = $USERNAME/stack

[land]
style = bottom-only
```

Setting `land.style = disable` removes the `land` subcommand entirely.

## License

Apache-2.0 with LLVM Exceptions. See [`LICENSE`](LICENSE).

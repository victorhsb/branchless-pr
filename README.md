# stack-pr (Go port)

`stack-pr` is a command-line tool for creating, updating, viewing, abandoning, and landing stacked GitHub pull requests. This is the Go port of the [Modular `stack-pr`](https://github.com/modular/stack-pr) Python tool. It preserves the original tool's algorithms and CLI surface; see `SPEC.md` for the full specification.

A stack is the ordered list of local commits in a Git revision range (`BASE..HEAD`). Each commit corresponds to exactly one GitHub PR. The bottom PR targets the repository target branch (normally `main`); every higher PR targets the generated branch for the previous commit. This way each PR review shows only one logical commit while still preserving dependency order.

## Install

### Homebrew (macOS/Linux)

```bash
brew tap victorhsb/tap
brew install stack-pr
```

### Docker

```bash
docker pull ghcr.io/victorhsb/branchless-pr:latest
docker run --rm ghcr.io/victorhsb/branchless-pr:latest version
```

### Pre-built binaries

Download from [Releases](https://github.com/victorhsb/branchless-pr/releases). Extract and place `stack-pr` in your `$PATH`.

### Go install

```bash
go install github.com/victorhsb/branchless-pr/cmd/stack-pr@latest
```

### Build from source

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
- `stack-pr agent prompt [topic]` — emit static, versioned guidance for LLM agents.
- `stack-pr agent diagnose [--format text|json] [--online]` — emit a read-only, best-effort diagnostic report for agents.

## Shared options

| Flag                               | Description                                                                                          |
| ---------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `-R, --remote`                     | Remote name (default `origin`).                                                                      |
| `-B, --base`                       | Local base revision (default merge-base).                                                            |
| `-H, --head`                       | Local head revision (default: top of current git-branchless stack when available, otherwise `HEAD`). |
| `-T, --target`                     | Remote target branch (default `main`).                                                               |
| `--hyperlinks` / `--no-hyperlinks` | Enable terminal hyperlinks.                                                                          |
| `-V, --verbose`                    | Verbose subprocess output.                                                                           |
| `--branch-name-template`           | Generated branch template (default `$USERNAME/stack`).                                               |
| `--show-tips` / `--no-show-tips`   | Post-command guidance.                                                                               |

## Submit-only options

| Flag              | Description                                                                 |
| ----------------- | --------------------------------------------------------------------------- |
| `--keep-body`     | Preserve current PR body after the stack TOC.                               |
| `-d, --draft`     | Create new PRs as draft.                                                    |
| `--draft-bitmask` | Per-PR draft bitmask (e.g. `010`).                                          |
| `--reviewer`      | Reviewer list.                                                              |
| `-s, --stash`     | Stash uncommitted changes during submit. Ignored under `--dry-run`.         |
| `--dry-run`       | Preview submit/export actions without applying local Git or GitHub changes. |

### Previewing with `--dry-run`

`stack-pr submit --dry-run` (and its alias `stack-pr export --dry-run`) prints
the plan that a real submit would execute — per stack entry: the action
(create or update PR), commit title, generated head branch, computed base
branch, existing PR URL when present, draft state for new PRs, and whether
stack metadata would be added to the commit. No local Git mutations, remote
pushes, or GitHub PR writes are performed.

## Agent prompt

`stack-pr agent prompt [topic]` prints deterministic guidance for LLM agents.
It is side-effect-free and runs without a git repository or authenticated `gh`.
Supported topics are `overview`, `view`, `submit`, `land`, `abandon`,
`recovery`, and `all` (the default).

```bash
stack-pr agent prompt
stack-pr agent prompt submit
stack-pr agent prompt submit --format json
```

Use `--format text` for markdown (default) or `--format json` for a structured
agent-consumable envelope with versioned `id` values and command side-effect
metadata.

## Agent diagnose

`stack-pr agent diagnose` inspects repository, stack, and PR metadata state and
prints a read-only diagnostic report. It is best-effort: reportable conditions
such as a dirty working tree, missing PR metadata, a rebase in progress, or even
being outside a Git repository are represented in the payload instead of causing
the command to fail. The command exits `0` for those reportable outcomes; check
the top-level `status` and individual check entries for severity.

```bash
stack-pr agent diagnose
stack-pr agent diagnose --format json
stack-pr agent diagnose --online
```

Flags:

- `--format text|json`: output Markdown text (default) or a single JSON document.
- `--online`: allow optional GitHub checks via `gh`, such as live PR state.
  Without this flag, diagnose performs no `gh` command invocations and does not
  contact GitHub.

The initial JSON schema version is `"1"`. The JSON envelope contains:

- `schema_version`: currently `"1"`.
- `status`: one of `ok`, `warning`, `blocking`, or `unknown`.
- `repo`: repository context (`root`, `current_branch`, `remote`, `target`,
  `base`, `head`, `branch_name_template`, `online`).
- `stack`: stack summary (`size`, `entries_with_pr`, `entries_missing_pr`).
- `checks`: check entries with `id`, `status`, and `message`; blocking entries
  also include `blocks` and `suggested_fix`.
- `recommendation`: a safe next action with `command`, `reason`,
  `side_effects`, `requires_confirmation`, and optional
  `potential_next_actions`. `stack-pr land` is never the primary
  recommendation; if surfaced, it requires explicit confirmation.

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

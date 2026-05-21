# stack-pr (Go port)

`stack-pr` is a command-line tool for creating, updating, viewing, abandoning, and landing stacked GitHub pull requests. This is the Go port of the [Modular `stack-pr`](https://github.com/modular/stack-pr) Python tool. It preserves the original tool's algorithms and CLI surface; see `SPEC.md` for the full specification.

A stack is the ordered list of local commits in a Git revision range (`BASE..HEAD`). Each commit corresponds to exactly one GitHub PR. The bottom PR targets the repository target branch (normally `main`); every higher PR targets the generated branch for the previous commit. This way each PR review shows only one logical commit while still preserving dependency order.

> **Alias:** `bpr` ("branchless PR") is a shorter alias for `stack-pr`. It
> ships as an identical binary with the same commands, flags, and config file.
> See the [Install](#install) section below for setup options.

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
# shorter alias binary
alias bpr='stack-pr'
```

> Installing the binary via `go install` only provides `stack-pr`. For a
> standalone `bpr` binary, use `make install-bpr` (see [Build from
> source](#build-from-source)) or add a shell alias.

### Build from source

```bash
git clone https://github.com/victorhsb/branchless-pr
cd branchless-pr
make build        # produces both stack-pr and bpr binaries
```

The `make build` target now produces two binaries:

- `stack-pr` — the main CLI entrypoint
- `bpr` — the shorter alias binary (`bpr --help` works identically)

> **Homebrew & Docker:** when installing via Homebrew or Docker, `bpr` is
> already included. No extra setup is needed.

#### Shell aliases (optional)

If you only installed the `stack-pr` binary, you can also get `bpr` via a
shell alias:

```bash
# Bash / Zsh — add to ~/.bashrc or ~/.zshrc
alias bpr='stack-pr'

# Fish
alias bpr 'stack-pr'
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

# collect review comments across the stack
stack-pr comments

# inspect CI checks and brief review-attention state across the stack
stack-pr checks

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
- `stack-pr comments` — collect PR comments, reviews, and review threads across the stack.
- `stack-pr checks` — report all CI checks and brief review-attention state across the stack.
- `stack-pr land` — squash-merge the bottom PR and rebase the rest.
- `stack-pr abandon` — strip stack metadata and delete generated branches.
- `stack-pr config init` — scaffold a starter `.stack-pr.cfg` with sensible defaults.
- `stack-pr config set <section>.<key>=<value>` (or legacy `config <section>.<key>=<value>`) — write a setting to `.stack-pr.cfg`.

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

## Stack comments

`stack-pr comments` prints a read-only report of pull request feedback across
the current stack. It groups conversation comments, submitted reviews, review
comments, and review threads by stack entry and PR. The command does not
checkout branches, amend commits, push, or write to GitHub.

Set `comments.ignore_authors` in `.stack-pr.cfg` to hide noisy automation
accounts from comments output by default.

```bash
stack-pr comments
stack-pr comments --unresolved-only
stack-pr comments --kind review_thread --format json
stack-pr comments --author octocat
```

Flags:

- `--format text|json`: output Markdown-compatible text (default) or a single
  JSON object for agents.
- `--unresolved-only`: include only unresolved or attention-required feedback.
- `--kind`: comma-separated kinds: `conversation`, `review`,
  `review_comment`, `review_thread`.
- `--author`: include feedback authored by the given GitHub login.

## Stack checks

`stack-pr checks` prints a read-only report of GitHub check state across the
current stack. It reports all checks by default, not only required checks, and
includes stable failed-check IDs so humans and agents can identify what to fix.
It also includes brief comment/review counts and bounded snippets; use
`stack-pr comments` for full comment inspection.

```bash
stack-pr checks
stack-pr checks --failed-only
stack-pr checks --required-only
stack-pr checks --pr 123 --format json
stack-pr checks --commit abc123
```

Flags:

- `--format text|json`: output Markdown-compatible text (default) or a single
  JSON object for agents.
- `--failed-only`: include only failed checks and the stack context needed to
  understand them.
- `--required-only`: include only checks known to be required. Checks whose
  required state is unknown are excluded by this filter.
- `--pr`: include only the stack entry associated with the given pull request
  number.
- `--commit`: include only the stack entry matching a full or unambiguous
  abbreviated commit SHA.

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
- `--online`: allow optional GitHub checks via `gh`, including GitHub
  availability and live PR state. Without this flag, diagnose performs no `gh`
  command invocations and does not contact GitHub. If GitHub appears
  unavailable, diagnose marks that as blocking for mutating stack operations
  such as `submit`, `land`, and `abandon`.

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

## Config init

`stack-pr config init` scaffolds a starter `.stack-pr.cfg` at the repository root with sensible defaults and inline documentation. It fails safely if the file already exists.

```bash
stack-pr config init
```

After the file is created you can edit it by hand or set individual values inline with `stack-pr config set`.

## Configuration

Config lives at `<repo-root>/.stack-pr.cfg` (override with `STACKPR_CONFIG`). The file uses INI syntax: `[section]` headers followed by `key = value` lines.

### All settings

#### `[common]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `verbose` | bool | `false` | Show verbose subprocess output (`git` / `gh`) for every command. |
| `hyperlinks` | bool | `true` | Enable terminal hyperlinks (e.g. clickable PR URLs). Use `--no-hyperlinks` to disable on a single run. |
| `draft` | bool | `false` | Create **new** PRs as drafts by default. Only affects PRs created with `stack-pr submit`. |
| `keep_body` | bool | `false` | Preserve the existing PR body after the generated stack TOC on update. Without this, the body is replaced. |
| `stash` | bool | `false` | Automatically stash uncommitted changes before `submit` / `export`. Skipped under `--dry-run`. |
| `show_tips` | bool | `true` | Show contextual tips/hints after commands (e.g. next recommended action). |

#### `[repo]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `remote` | string | `origin` | Git remote name used for pushes and merge-base calculation. |
| `target` | string | `main` | Remote branch that the bottom PR targets (e.g. `main`, `master`). |
| `reviewer` | string | *(empty)* | Comma-separated GitHub usernames to add as reviewers on new PRs. |
| `branch_name_template` | string | `$USERNAME/stack` | Template for generated branch names. **Must contain `$ID`**. Supported substitutions: `$USERNAME`, `$ID`. |

#### `[comments]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `ignore_authors` | string | *(empty)* | Comma-separated GitHub usernames whose review comments are hidden from `stack-pr comments` output by default. |

#### `[land]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `style` | string | `bottom-only` | `bottom-only` merges the bottom PR and rebases the rest. `all` merges the whole stack. `disable` removes the `land` subcommand entirely. |

### Example file

```ini
[common]
verbose = false
hyperlinks = true
show_tips = true
stash = false

[repo]
remote = origin
target = main
reviewer = someuser
branch_name_template = $USERNAME/stack

[comments]
ignore_authors = ci-bot,release-bot

[land]
style = bottom-only
```

Setting `land.style = disable` removes the `land` subcommand entirely.


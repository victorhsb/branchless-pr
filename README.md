# branchless-pr

`branchless-pr` is a command-line tool for creating, updating, viewing, abandoning, and landing stacked GitHub pull requests. It is the Go successor to the [Modular `stack-pr`](https://github.com/modular/stack-pr) Python tool; it remains compatible with that tool's commit-message stack metadata while evolving its own behavior and CLI.

A stack is the ordered list of local commits in a Git revision range (`BASE..HEAD`). Each commit corresponds to exactly one GitHub PR. The bottom PR targets the repository target branch (normally `main`); every higher PR targets the generated branch for the previous commit. This way each PR review shows only one logical commit while still preserving dependency order.

> **Alias:** `bpr` ("branchless PR") is the primary executable name. A `stack-pr`
> symlink is provided for backward compatibility with the original Python tool.

## Install

### One-line installer (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/victorhsb/branchless-pr/main/install.sh | sh
```

Installs `bpr` to `~/.local/bin` and creates a `stack-pr` symlink for backward compatibility. Pass `-y` to skip the confirmation prompt. See `install.sh --help` for options.

### Pre-built binaries

Download from [Releases](https://github.com/victorhsb/branchless-pr/releases). Extract and place `bpr` in your `$PATH`. Optionally create a symlink: `ln -s bpr stack-pr`.

### Go install

```bash
go install github.com/victorhsb/branchless-pr/cmd/bpr@latest
```

### Build from source

```bash
git clone https://github.com/victorhsb/branchless-pr
cd branchless-pr
make build        # produces the bpr binary
```

> **Note:** `stack-pr` is no longer built as a separate binary. The install script and release archives create a `stack-pr` symlink to `bpr` for backward compatibility.

## Requirements

| Requirement | Notes |
| ----------- | ----- |
| Go 1.23+ | Only needed when building from source or using `go install`. |
| `git` | Required for all stack operations. |
| `gh` | GitHub CLI, authenticated via `gh auth login`. SSH auth is recommended. |

The tool shells out to `git` and `gh`; no Go GitHub SDK is used.

## For LLM agents

If you are an LLM agent setting this tool up for a user:

1. Install: `curl -fsSL https://raw.githubusercontent.com/victorhsb/branchless-pr/main/install.sh | sh -s -- -y`
2. Learn how to use it: `bpr agent prompt`
3. Before mutating the stack, inspect state: `bpr view` and `bpr agent diagnose`

The `agent prompt` subcommand emits versioned, deterministic guidance covering each operation, safety model, and recovery procedures. The `agent diagnose` subcommand emits a read-only diagnostic report with a recommended next action.

<details>
<summary><strong>Agent setup and usage guide</strong> (click to expand)</summary>

### Installation for agents

Agents should install `branchless-pr` non-interactively:

```bash
# One-line installer (skips confirmation prompt)
curl -fsSL https://raw.githubusercontent.com/victorhsb/branchless-pr/main/install.sh | sh -s -- -y

# Or via Go
go install github.com/victorhsb/branchless-pr/cmd/bpr@latest
```

Verify installation:

```bash
bpr --version
bpr agent prompt overview
```

### Quick orientation

Before performing any stack operations, gather context:

```bash
# View the current stack (read-only)
bpr view

# Get a diagnostic report with recommended next action
bpr agent diagnose
bpr agent diagnose --format json  # machine-readable output
```

### Common agent workflows

**Inspecting a stack:**
```bash
bpr view                          # human-readable
bpr view --format json            # machine-readable
```

**Previewing changes (no side effects):**
```bash
bpr submit --dry-run              # shows what would happen without mutating anything
```

**Submitting a stack:**
```bash
bpr submit                        # creates/updates PRs
bpr submit --reviewer alice,bob   # add reviewers
```

**Checking CI status:**
```bash
bpr checks                        # human-readable
bpr checks --format json          # machine-readable
bpr checks --failed-only          # filter to failures only
```

**Collecting review feedback:**
```bash
bpr comments                      # human-readable
bpr comments --format json        # machine-readable
bpr comments --unresolved-only    # filter to unresolved items
```

**Landing the bottom PR:**
```bash
bpr land                          # squash-merge bottom PR, rebase the rest
```

**Cleaning up:**
```bash
bpr abandon                       # strip stack metadata, delete generated branches
```

### Safety model

- `--dry-run` performs **no local Git mutations**, **no remote pushes**, and **no GitHub PR writes**. Use it to preview changes safely.
- `bpr agent diagnose` is read-only and safe to run at any time.
- `bpr view` is read-only and safe to run at any time.
- Mutating commands (`submit`, `land`, `abandon`) should be preceded by `bpr view` and `bpr agent diagnose`.

### Configuration for agents

Agents can scaffold a config file:

```bash
bpr config init
```

Or set individual values:

```bash
bpr config set common.draft=true
bpr config set repo.reviewer=octocat
```

See the full [Configuration](#configuration) section for all options.

### Machine-readable output

All major commands support `--format json` for agent consumption:

| Command | JSON flag |
| ------- | --------- |
| `bpr view` | `--format json` |
| `bpr comments` | `--format json` |
| `bpr checks` | `--format json` |
| `bpr agent diagnose` | `--format json` |

### Getting help

```bash
bpr agent prompt                  # full agent guidance (all topics)
bpr agent prompt submit           # topic-specific guidance
bpr agent prompt recovery         # recovery procedures
bpr --help                        # general help
bpr <command> --help              # command-specific help
```

</details>

## Quick start

> All examples use `bpr` (the primary executable name). `bpr` and the historical `stack-pr` symlink are interchangeable.

```bash
# create some commits on a feature branch
git checkout -b my-feature main
# ... commit a few times ...

# inspect the stack
bpr view

# collect review comments across the stack
bpr comments

# inspect CI checks and brief review-attention state across the stack
bpr checks

# submit / update the stack of PRs
bpr submit

# land the bottom-most PR
bpr land

# remove all stack metadata and clean up generated branches
bpr abandon
```

## Commands

| Command | Description |
| ------- | ----------- |
| `bpr submit` (alias: `export`) | Create or update PRs for each commit. Optionally reconcile with a GitHub native Stack. |
| `bpr view` | Inspect the stack without modifying anything. Includes native Stack metadata in JSON when enabled. |
| `bpr comments` | Collect PR comments, reviews, and review threads across the stack. |
| `bpr checks` | Report all CI checks and brief review-attention state across the stack. |
| `bpr land` | Squash-merge the bottom PR and rebase the rest. `--whole-stack` queues the tip PR for merge queue landing. Refuses to land stacks linked to a GitHub native Stack. |
| `bpr abandon` | Strip stack metadata and delete generated branches. Unstacks matching GitHub native Stacks before deleting remote branches. |
| `bpr fix --pr <number>` | Repair the local `HEAD` commit's stack metadata from an existing PR. Local-only: amends the commit message, never touches remotes or GitHub. Supports `--dry-run` and `--replace`. |
| `bpr config init` | Scaffold a starter `.stack-pr.cfg` with sensible defaults. |
| `bpr config set <section>.<key>=<value>` | Write a setting to `.stack-pr.cfg` (legacy: `config <section>.<key>=<value>`). |
| `bpr agent prompt [topic]` | Emit static, versioned guidance for LLM agents. |
| `bpr agent diagnose [--format text\|json] [--online]` | Emit a read-only, best-effort diagnostic report for agents. |

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
| `--receipt`       | Emit a JSON operation receipt to a file, `-`, or `off`.                      |

### Operation receipts

`bpr submit --receipt <path>` records a machine-readable JSON receipt of the
execution: repo and stack context plus every operation (PR creates/updates,
pushes, native Stack calls) with an `ok` or `failed` status and, on failure,
the error. The top-level `status` is `ok`, `failed`, or `partial_failure`. Use
`--receipt -` to write the receipt to stdout or `--receipt off` to suppress
it. Receipts are only available for real executions, not `--dry-run`.

## Previewing with `--dry-run`

`bpr submit --dry-run` (and its alias `bpr export --dry-run`) prints
the plan that a real submit would execute — per stack entry: the action
(create or update PR), commit title, generated head branch, computed base
branch, existing PR URL when present, draft state for new PRs, and whether
stack metadata would be added to the commit. No local Git mutations, remote
pushes, or GitHub PR writes are performed. `bpr fix --dry-run` reports the
planned metadata repair the same way. `land` and `abandon` do not support
`--dry-run`.

## GitHub native Stacked PRs (private preview)

The `github.native_stacks` setting opts into publishing branchless-pr stacks as GitHub native Stacks:

```ini
[github]
native_stacks = off      # default: legacy behavior only
# native_stacks = auto   # use native stacks when available and eligible
# native_stacks = required # fail if native stacks cannot be used
```

Requirements for native mode:

- The repository must have the GitHub native Stacks private preview enabled.
- Native Stack reads and writes use the documented REST API through the base `gh` CLI; no extension is required.
- Only multi-PR, same-repository PR chains are eligible. branchless-pr conservatively limits native publication to 100 total PRs; GitHub currently documents 100 per create or append request, not an aggregate limit.

Behavior by command:

- `submit`/`export`: after ordinary PR/branch publication, creates via `POST /stacks`, appends via `POST /stacks/{number}/add`, or confirms a native Stack. Returned membership must exactly match the intended bottom-to-top sequence. Uses force-with-lease pushes to avoid overwriting server-side rebases. `--dry-run` reports the planned native action without making GitHub writes.
- `view`: shows native Stack number, position, size, and base in text and JSON output.
- `land`: refuses to land a stack that exactly matches a native Stack; merge must be initiated in the GitHub UI until a supported CLI landing contract exists.
- `abandon`: calls `POST /stacks/{number}/unstack` before deleting generated remote branches, handling both partial `200` and dissolved `204` results.

Native writes are not blindly retried. If a transport or server failure leaves the outcome uncertain, branchless-pr reads the current Stack and reconciles the observed sequence first.

> **Synchronization limitation:** Using GitHub's Rebase Stack or merge controls can rewrite generated remote branches and leave the branchless-pr local commit stack stale. Recovery requires manual synchronization; native `land` is blocked until GitHub documents a supported non-interactive merge interface.

## Stack comments

`bpr comments` prints a read-only report of pull request feedback across
the current stack. It groups conversation comments, submitted reviews, review
comments, and review threads by stack entry and PR. The command does not
checkout branches, amend commits, push, or write to GitHub.

Set `comments.ignore_authors` in `.stack-pr.cfg` to hide noisy automation
accounts from comments output by default.

```bash
bpr comments
bpr comments --unresolved-only
bpr comments --kind review_thread --format json
bpr comments --author octocat
```

| Flag | Description |
| ---- | ----------- |
| `--format text\|json` | Output Markdown-compatible text (default) or a single JSON object for agents. |
| `--unresolved-only` | Include only unresolved or attention-required feedback. |
| `--kind` | Comma-separated kinds: `conversation`, `review`, `review_comment`, `review_thread`. |
| `--author` | Include feedback authored by the given GitHub login. |

## Stack checks

`bpr checks` prints a read-only report of GitHub check state across the
current stack. It reports all checks by default, not only required checks, and
includes stable failed-check IDs so humans and agents can identify what to fix.
It also includes brief comment/review counts and bounded snippets; use
`bpr comments` for full comment inspection.

```bash
bpr checks
bpr checks --failed-only
bpr checks --required-only
bpr checks --pr 123 --format json
bpr checks --commit abc123
```

| Flag | Description |
| ---- | ----------- |
| `--format text\|json` | Output Markdown-compatible text (default) or a single JSON object for agents. |
| `--failed-only` | Include only failed checks and the stack context needed to understand them. |
| `--required-only` | Include only checks known to be required. Checks whose required state is unknown are excluded. |
| `--pr` | Include only the stack entry associated with the given pull request number. |
| `--commit` | Include only the stack entry matching a full or unambiguous abbreviated commit SHA. |

## Agent prompt

`bpr agent prompt [topic]` prints deterministic guidance for LLM agents.
It is side-effect-free and runs without a git repository or authenticated `gh`.
Supported topics are `overview`, `view`, `submit`, `land`, `abandon`, `fix`,
`recovery`, and `all` (the default).

```bash
bpr agent prompt
bpr agent prompt submit
bpr agent prompt submit --format json
```

Use `--format text` for markdown (default) or `--format json` for a structured
agent-consumable envelope with versioned `id` values and command side-effect
metadata.

## Agent diagnose

`bpr agent diagnose` inspects repository, stack, and PR metadata state and
prints a read-only diagnostic report. It is best-effort: reportable conditions
such as a dirty working tree, missing PR metadata, a rebase in progress, or even
being outside a Git repository are represented in the payload instead of causing
the command to fail. The command exits `0` for those reportable outcomes; check
the top-level `status` and individual check entries for severity.

```bash
bpr agent diagnose
bpr agent diagnose --format json
bpr agent diagnose --online
```

| Flag | Description |
| ---- | ----------- |
| `--format text\|json` | Output Markdown text (default) or a single JSON document. |
| `--online` | Allow optional GitHub checks via `gh`, including GitHub availability and live PR state. Without this flag, diagnose performs no `gh` invocations and does not contact GitHub. If GitHub appears unavailable, diagnose marks that as blocking for mutating stack operations such as `submit`, `land`, and `abandon`. |

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
  `potential_next_actions`. `bpr land` is never the primary
  recommendation; if surfaced, it requires explicit confirmation.

## Config init

`bpr config init` scaffolds a starter `.stack-pr.cfg` at the repository root with sensible defaults and inline documentation. It fails safely if the file already exists.

```bash
bpr config init
```

After the file is created you can edit it by hand or set individual values inline with `bpr config set`.

## Configuration

Config lives at `<repo-root>/.stack-pr.cfg` (override with `STACKPR_CONFIG`). The file uses INI syntax: `[section]` headers followed by `key = value` lines.

### All settings

#### `[common]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `verbose` | bool | `false` | Show verbose subprocess output (`git` / `gh`) for every command. |
| `hyperlinks` | bool | `true` | Enable terminal hyperlinks (e.g. clickable PR URLs). Use `--no-hyperlinks` to disable on a single run. |
| `draft` | bool | `false` | Create **new** PRs as drafts by default. Only affects PRs created with `bpr submit`. |
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
| `ignore_authors` | string | *(empty)* | Comma-separated GitHub usernames whose review comments are hidden from `bpr comments` output by default. |

#### `[land]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `style` | string | `bottom-only` | `bottom-only` merges the bottom PR and rebases the rest. `whole-stack` queues the tip PR for merge-queue landing. `disable` removes the `land` subcommand entirely. |

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

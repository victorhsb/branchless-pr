# stack-pr (Go Port) – Agent Instructions

## Project Goal

Port the Python CLI tool `stack-pr` to Go, replicating its behavior for creating,
updating, viewing, abandoning, and landing stacked GitHub pull requests.

The tool models a stack as an ordered list of local commits in a Git revision
range (`BASE..HEAD`). Each commit maps to exactly one GitHub PR. The bottom PR
targets the repository's target branch (normally `main`), and every higher PR
targets the generated branch for the previous commit.

## Source of Truth

- `SPEC.md` — complete specification of the original Python tool's behavior,
  data model, algorithms, and packaging.
- `TASKS.md` — persistent task state; updated as work progresses.

## Repository Layout (proposed)

```
.
├── cmd/stack-pr/       # entrypoint
├── internal/
│   ├── cli/            # command implementations (submit, view, land, abandon, config)
│   ├── git/            # git shell helpers and GitError
│   ├── shell/          # subprocess wrapper
│   ├── stack/          # CommitHeader, StackEntry, stack discovery, formatting
│   ├── config/         # INI config parsing and defaults
│   └── pr/             # gh CLI wrappers
├── go.mod / go.sum
├── AGENTS.md           # this file
├── TASKS.md            # task tracker
├── SPEC.md             # baseline specification
├── README.md           # user-facing docs
├── CHANGELOG.md
├── LICENSE
├── CONTRIBUTING.md
└── .github/workflows/
```

## Key Port Decisions

- **Shelling out is intentional** — we call `git` and `gh` via subprocess, just
  like the Python version.
- **Go idioms first** — explicit error returns, small packages, table-driven
  tests.
- **No external GitHub SDK** — `gh` CLI handles auth and API.

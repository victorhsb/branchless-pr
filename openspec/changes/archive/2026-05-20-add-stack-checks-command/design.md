## Context

`stack-pr` can already discover a local commit stack and map commits to pull requests through stack metadata. Users still need a concise way to answer whether every PR in the stack is review-ready, which CI checks are failing, and which failure a human or agent should inspect first. GitHub exposes this state through `gh`, but the raw shape mixes check runs, status contexts, required-check metadata, review/comment signals, and per-PR failures.

The command should remain read-only and should not become a full comment browser. It can summarize comment pressure so users see that a PR needs review attention, while detailed thread/comment inspection remains the responsibility of `stack-pr comments`.

## Goals / Non-Goals

**Goals:**

- Add `stack-pr checks` as a read-only stack inspection command.
- Report all checks for every stack PR, not only required checks.
- Preserve required-check information when GitHub exposes it.
- Emit stable check IDs that are useful to agents and exact provider IDs when available.
- Summarize failed checks at top level in JSON and visibly in text output.
- Include brief comment/review signals without duplicating the full `comments` command.
- Keep output deterministic and suitable for automation.

**Non-Goals:**

- Do not rerun checks, approve PRs, resolve comments, edit PRs, push branches, or mutate local Git.
- Do not replace `stack-pr comments`; `checks` only shows brief comment/review summaries.
- Do not introduce a Go GitHub SDK dependency.
- Do not guarantee that the semantic check ID survives arbitrary workflow/job renames; include provider IDs for exact lookup when available.
- Do not require branch protection to be configured. All-check reporting works even when required-check status is unknown.

## Decisions

### Use a dedicated top-level command

`stack-pr checks` should be a top-level command rather than an `agent diagnose` mode. Diagnose answers "what should I do next?" across many local and remote blockers. Checks answers a narrower operational question: "what is the check/review state of each PR in this stack?"

Alternative considered: extend `view` with check fields. That would make `view` heavier and blur its current stack metadata focus.

### Fetch check state through `gh`

The implementation should use `gh pr view` and/or `gh api graphql` through `internal/shell`, with normalization in `internal/pr` or a sibling internal package. This preserves the repo invariant of shelling out to `git` and `gh` and avoids a GitHub SDK dependency.

The likely data sources are:

- PR metadata and `statusCheckRollup`/check summaries via `gh pr view --json` when sufficient.
- GraphQL queries for fields that need check run IDs, workflow names, required status, or richer commit/check-suite data.
- Optional lightweight PR comments/reviews fields, limited to counts and short recent snippets.

### Emit both semantic and provider identifiers

Each check entry should include a stable semantic ID plus provider identifiers when available:

- `id`: normalized agent-facing identifier such as `github-actions:ci.yml:test` or `checks:codecov/project`.
- `provider`: source category such as `github_actions`, `github_status`, or `github_check`.
- `provider_id`, `run_id`, `check_run_id`, `workflow`, `name`, and `url` when GitHub exposes them.

The semantic ID is for routing and deduplication. Provider IDs are for exact lookup, log links, and future follow-up commands.

### Report all checks by default

The default output should include every check GitHub reports for each PR head commit. Required-only views are a filter, not the default. Each check carries `required: true|false|unknown` so callers can distinguish hard merge blockers from optional checks without losing information.

### Keep comments brief

`checks` should include comment/review pressure as a compact PR-level summary:

- counts by category when available, such as conversation comments, reviews, review comments, unresolved review threads, and requested changes,
- short snippets for the most recent or highest-priority items when available,
- a pointer command such as `stack-pr comments --pr <number>` or equivalent supported filter once available.

Full bodies, full thread trees, and detailed resolution context belong in `stack-pr comments`.

### Failure handling is report-oriented

Missing PR metadata, GitHub authentication failures, unavailable check data, and per-PR read errors should be represented clearly. Invocation errors such as invalid flags should still exit non-zero. Reportable partial failures should preserve any available check data for other PRs.

## Risks / Trade-offs

- GitHub check APIs have multiple shapes -> Normalize into one internal model and test with fixtures for check runs, status contexts, skipped checks, pending checks, and missing fields.
- Semantic IDs can change when workflows or job names are renamed -> Include exact provider IDs and URLs when available, and document the ID stability boundary.
- Required-check detection may be unavailable without branch protection data -> Use `required: unknown` rather than guessing.
- Comment summaries could become expensive -> Keep them lightweight, bounded, and optional to omit when unavailable.
- Text output can become noisy on large stacks -> Default to grouped summaries and make JSON the complete machine-readable contract.

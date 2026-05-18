## Context

`stack-pr` is a Cobra-based CLI that drives a branchless PR workflow on top of Git and GitHub (`gh`). Existing commands (`submit`, `export`, `view`, `land`, `abandon`) are *executive* — they assume a well-formed environment and abort early when preconditions fail. They are not well suited for agent orientation: an LLM-driven caller that does not yet know whether the repository is in a usable state, whether the stack has been created, or whether PRs already exist needs a single read-only entry point that always returns a parseable description.

A sibling change is introducing the `agent` Cobra command group along with an `agent prompt` subcommand that emits agent-facing usage guidance. This change introduces a second subcommand, `agent diagnose`, under the same parent. The two subcommands share a need to expose consistent, machine-readable safety metadata about destructive operations (e.g., `side_effects`, `requires_confirmation`), but their specific surfaces are independent.

The diagnosis surface must remain useful even when the repository is broken. That means rejecting the "fail fast" idiom used elsewhere in `stack-pr` in favor of a check-driven model where each individual check reports a `status` and the command itself almost never aborts.

## Goals / Non-Goals

**Goals:**

- Provide a read-only `stack-pr agent diagnose` subcommand that always returns parseable output describing the repository, stack, and check state.
- Support a Markdown text format (human-default) and a stable JSON format.
- Gate any GitHub network access behind `--online`; default to a local-only mode that still produces a useful diagnosis.
- Adopt a degraded-mode contract: individual checks report `ok` / `warning` / `blocking` / `unknown` rather than aborting the command.
- Always exit with code `0` for any reportable outcome so agents can rely on parsing stdout.
- Define a stable, versioned JSON output schema with explicit `schema_version` (or equivalent) so consumers can detect drift.
- Define a recommendation contract: every recommendation includes `command`, `reason`, `side_effects`, and `requires_confirmation`. `land` is always conservative (potential next action, requires confirmation), never an outright recommendation.
- Reuse existing stack discovery and metadata helpers from `internal/stack` for stack inspection, but in a non-failing wrapper that surfaces problems as checks rather than errors.

**Non-Goals:**

- Mutating the repository, the working tree, the index, or any remote.
- Implementing `agent prompt` or owning the `agent` command group itself (sibling change).
- Replacing or changing the behavior of `submit`, `export`, `view`, `land`, or `abandon`.
- Guaranteeing that a follow-up command recommended by `diagnose` will succeed; the repository can change between calls.
- Producing a stable Markdown contract — only the information surfaced is constrained; exact headings and wording are an implementation choice.
- Defining the full shape of the `AgentCommandSpec` Go struct as a spec-level requirement; the spec constrains the externally observable JSON only.

## Decisions

1. **`diagnose` is read-only by construction.**
   - The implementation must not invoke any Git plumbing or `gh` subcommand that mutates state. No `git add`, `git commit`, `git checkout`, `git rebase`, `git stash`, `git push`, `git reset`, no `gh pr create/edit/merge/close`, no `gh auth login`.
   - Rationale: the value of `diagnose` to an agent depends on it being side-effect-free; a hidden mutation would undermine recommendations that defer destructive actions to other commands.
   - Alternative considered: allow `--fix` to auto-remediate. Rejected because remediation is the job of other commands (`submit`, `abandon`); mixing diagnosis with remediation breaks the safety contract.

2. **Always exit with code `0`; severity lives in the payload.**
   - The command exits non-zero only for catastrophic unexpected failures (e.g., panic, OOM, the JSON encoder fails). Every reportable repository condition — including "not a git repo", "rebase in progress", or "all checks blocking" — yields exit `0`.
   - Rationale: agents parse output to make decisions. A non-zero exit can cause callers to short-circuit before reading the payload, defeating the purpose.
   - Alternative considered: exit `1` when any blocking check is present. Rejected because the payload already carries severity (`status` on each check and at the top level), and agents have repeatedly proven unreliable at recovering from non-zero exits.

3. **Check-driven model with a uniform check record.**
   - Each check is an entry with at minimum `id` (stable string), `status` (`ok` / `warning` / `blocking` / `unknown`), and `message`. Blocking entries additionally carry `blocks` (list of commands they block) and `suggested_fix` (human-readable remediation hint).
   - Rationale: a uniform shape lets agents iterate over checks generically and decide whether each gate is open.
   - Alternative considered: ad-hoc fields per check. Rejected because it would force consumers to special-case each check type.

4. **Network access is opt-in via `--online`.**
   - By default, `diagnose` performs zero network I/O. With `--online`, it may consult GitHub (via `gh`) to fetch live PR state for entries with metadata.
   - Rationale: the default should be cheap, deterministic, and usable in environments without GitHub auth. Network calls can fail in many ways and would otherwise need their own status handling on every invocation.
   - Alternative considered: always go online. Rejected because it breaks offline use and air-gapped CI.

5. **JSON schema is versioned.**
   - The JSON envelope carries a `schema_version` field with a stable string value (initially `"1"` or `"1.0"`; exact value chosen at implementation time but documented in command help and `README.md`).
   - Rationale: agents need a way to detect drift and either adapt or refuse.
   - Alternative considered: ship without a version and rely on backwards compatibility. Rejected because the agent-facing JSON is the contract; explicit versioning is cheap insurance.

6. **Recommendations always carry safety metadata, and `land` is special.**
   - Every recommendation object includes `command`, `reason`, `side_effects` (boolean), and `requires_confirmation` (boolean). For agents, these fields are the gate on whether to execute autonomously.
   - When the stack looks fully ready (all PRs exist, working tree clean, no rebase in progress), `diagnose` SHALL NOT recommend `stack-pr land` outright. It MAY surface `land` as a separate "potential next action" entry, but that entry MUST carry `requires_confirmation: true` and SHOULD be presented as conservative guidance rather than the primary recommendation.
   - Rationale: `land` mutates remote PRs and merges to the target branch; an agent acting on a blanket recommendation could merge prematurely.
   - Alternative considered: recommend `land` when checks are green. Rejected; the human is the merge gate.

7. **Reuse `internal/stack` helpers behind a non-failing wrapper.**
   - Stack discovery, metadata reading, and head/base assignment helpers are reused, but invoked through wrappers that translate errors into `unknown` or `blocking` check entries.
   - Rationale: keeps stack semantics consistent with `submit`/`view` while preserving the degraded-mode contract of `diagnose`.
   - Alternative considered: reimplement stack inspection. Rejected as duplication.

8. **Share a static command-metadata layer with `agent prompt` (design hint only).**
   - The implementation is expected to define a shared `AgentCommandSpec` (Go) describing safety metadata for each stack-pr command, consumed by both `agent diagnose` (to populate recommendation `side_effects`/`requires_confirmation`) and `agent prompt` (to render guidance).
   - This is a design hint, not a spec requirement. The spec only constrains what `diagnose` emits externally.

9. **Markdown text format prioritizes human readability; only the information set is contractual.**
   - The text format MUST surface the same information set as the JSON (repo metadata, stack summary, each check with at minimum id + status + message, and the recommendation). Exact section headings, ordering, and prose are an implementation choice and not constrained by the spec.
   - Rationale: keeps text rendering flexible; JSON remains the stable contract.

## Risks / Trade-offs

- **Risk: A check accidentally aborts the command.** → Mitigate by routing every check through a wrapper that recovers from panics and converts errors to `unknown` status entries; add tests that simulate each underlying-helper failure path and assert exit `0`.
- **Risk: Network access leaks in even without `--online`.** → Mitigate by isolating all `gh` calls behind an "online" gate at the boundary and adding a test that asserts no `gh` (or other network) invocation occurs in default mode.
- **Risk: JSON schema drift over time.** → Mitigate via explicit `schema_version` and a golden-output test pinning the v1 envelope shape.
- **Risk: Recommendation logic recommends `land` because the stack looks ready.** → Mitigate by encoding the "never recommend land outright" rule as both a spec scenario and a unit test that runs the recommendation engine on a fully-clean stack.
- **Risk: Reused stack helpers introduce coupling and may be changed by other features.** → Mitigate by keeping the non-failing wrapper thin and by treating helper signature changes as a test-detected break.
- **Risk: Markdown output diverges from JSON information set.** → Mitigate with a shared in-memory diagnosis model that both formatters consume, so a missing field is impossible to render in only one format.
- **Trade-off: Always exiting `0` makes failures invisible to shell-level success checks.** → Accepted; this is the explicit cost of the agent-friendly contract. Users who want a non-zero exit can grep/jq the payload.

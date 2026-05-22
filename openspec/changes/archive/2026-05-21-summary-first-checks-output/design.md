## Context

`stack-pr checks` already builds a complete stack-wide report with normalized checks, stable semantic IDs, failed-check summaries, and lightweight comment summaries. The current text renderer presents every check line by default, which makes common stack triage noisy when GitHub reports multiple entries for the same workflow/job or when most checks are skipped, pending, or in progress.

The data model should remain complete because JSON output is useful for agents and follow-up automation. The main change is to separate the human scanning view from the exhaustive debugging view.

## Goals / Non-Goals

**Goals:**

- Make default text output answer which PRs are blocking, waiting, clean, missing metadata, or unreadable.
- Preserve complete per-check text detail behind `--verbose`.
- Keep failed checks visible with semantic IDs and URLs in both default and verbose text output.
- Avoid printing low-value unknown metadata in default text output.
- Preserve existing JSON shape and check-fetch behavior unless a small additive summary field is needed for implementation clarity.

**Non-Goals:**

- Do not infer GitHub branch-protection requirements from check names.
- Do not remove skipped, pending, or unknown-required checks from JSON output.
- Do not replace `stack-pr comments`; checks output remains a lightweight review-attention summary.
- Do not add new GitHub API dependencies or move away from `gh`.

## Decisions

1. Default text output will be summary-first.

   The renderer should produce a compact per-PR roll-up before any detailed check listing. Each roll-up should include PR identity, overall state, check counts by useful bucket, failed check names when present, and lightweight review/comment counts. This makes the common scan path fast without changing the fetched report.

   Alternative considered: keep the current output and add a separate `--summary` flag. Rejected because the feedback is that the current default is too noisy for the primary workflow.

2. `--verbose` will render full per-check detail.

   `--verbose` should preserve the debugging value of the current text output by listing all checks in deterministic order, including skipped, pending, in-progress, optional, and unknown-required checks. It should still include the summary so users do not lose the high-level scan when asking for details.

   Alternative considered: name the flag `--all`. Rejected because the command already includes all checks in the report model; the difference is text verbosity, not collection scope.

3. Default text will collapse duplicate visible check identities.

   When multiple checks share the same semantic check ID or otherwise same visible identity, the summary should count and display the most actionable state rather than printing each duplicate line. A practical priority order is failed, in progress, queued/pending, action-required/neutral/cancelled, success, skipped, unknown. Verbose output should still show every raw check entry.

   Alternative considered: dedupe in the underlying report. Rejected because agents may need exact raw entries and provider IDs.

4. Unknown required state remains structured data, not default text noise.

   JSON and verbose detail should preserve `required: unknown`. Default text should omit required-state labels unless a check is known required or the output is specifically describing required-check counts.

   Alternative considered: drop unknown required state from all outputs. Rejected because the existing spec requires preserving that GitHub did not expose the value.

5. Stack coverage should be explicit.

   Text output should report the number of stack entries, entries with PR metadata, missing entries, unreadable PRs, and any active `--pr` or `--commit` filter. This addresses the case where a user expected a stack-wide report but only one PR appears because only one stack entry had usable PR metadata or because a filter was applied.

## Risks / Trade-offs

- Duplicate collapsing may hide a raw skipped entry that explains why another in-progress entry exists -> Mitigation: keep verbose output exhaustive and mention duplicate counts where useful.
- Summary status categories can oversimplify GitHub's status/conclusion combinations -> Mitigation: derive categories from existing normalized fields and keep failed/pending/in-progress behavior conservative.
- Text snapshots in tests may become brittle -> Mitigation: test key substrings and helper-level summary calculations rather than large golden outputs only.

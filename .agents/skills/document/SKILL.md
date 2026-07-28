---
name: document
description: Finalize a completed implementation plan — verify specs in docs/specs/ reflect the shipped behavior, prompt for an ADR if an architectural decision emerged, update AGENTS.md if needed, and delete the plan file. Use when the user says work on a plan is done, asks to "document", "finalize", or "close out" a plan. Replaces the old openspec verify/archive workflow.
---

# Document

Close out a completed plan in `docs/plans/` and leave the repository documentation consistent with the code.

**Input**: A plan name (kebab-case, matching `docs/plans/<name>.md`). If the user doesn't name one, list the files in `docs/plans/` and ask which to finalize.

## Steps

1. **Confirm the work is actually done.**
   - Read the plan file. Every item in `Tasks` should be checked or accounted for; if some aren't, ask the user whether they're done, dropped, or still pending. Do not finalize a plan with outstanding work unless the user confirms.
   - Verify the code state: run `make test` and `make vet` (see AGENTS.md for the full local CI sequence). If they fail, stop and report — do not document over a broken tree.

2. **Verify specs match shipped behavior.**
   - For each spec in the plan's `related-specs` frontmatter (and any other spec the change plausibly touched), read `docs/specs/<name>.md` and judge whether its rules describe the behavior as it now exists in code.
   - If a rule is missing, stale, or contradicted, update the spec directly. Follow the tiered format conventions in `docs/plans/localspec.md` (rules → tables → rare scenario blocks; one rule per bullet).
   - This is agent judgment, not a scripted diff. If you're unsure whether behavior changed in a user-visible way, check the code and tests, and ask the user if still ambiguous.

3. **Prompt for an ADR if an architectural decision emerged.**
   - Ask: "Did this work make a lasting architectural decision (a constraint on future changes, a rejected alternative, a new invariant)?"
   - If yes, draft it via adrm: `adrm new --kind adr --title "Short title" --dry-run` to preview, then without `--dry-run` to create, then fill in Context/Decision/Consequences and `adrm accept <id>`.
   - ADRs are for constraints and "why we can't do X", not implementation details. When in doubt, don't create one.

4. **Update AGENTS.md if needed.** If the change altered anything AGENTS.md describes (architecture, commands, conventions, workflows), update it in the same pass.

5. **Delete the plan file.**
   - Check `git log --oneline -- docs/plans/<name>.md`: if the plan was never committed, warn the user that deleting it loses the record and ask whether to commit it first or delete anyway.
   - Otherwise `git rm docs/plans/<name>.md` (or delete the file and let the user stage it, matching how they manage commits).

6. **Report**: specs updated (or confirmed already accurate), ADR created (number/title) or explicitly not needed, AGENTS.md touched or not, plan file deleted.

## Guardrails

- Never delete a plan whose tasks are incomplete without explicit user confirmation.
- Never delete more than the one plan file being finalized.
- Do not create CHANGELOG entries here — CHANGELOG.md documents shipped user-facing behavior and is maintained separately (see AGENTS.md conventions).
- If the plan is being abandoned rather than completed, set `status: abandoned` in its frontmatter and ask the user whether to keep or delete it — don't run the spec-verification steps.

---
name: plan
description: Create a structured implementation plan for a behavioral change to this repository. Use when the user describes a feature, fix, or behavior change they want to build, asks to "plan this", or is about to start non-trivial work that will touch behavior documented in docs/specs/.
---

# Plan

Create a single-file implementation plan in `docs/plans/<name>.md`.

**Input**: A description of the change (feature, fix, or behavior modification). If the user's request is vague, ask what they want to build before proceeding — do not plan without a clear goal.

## Steps

1. **Ground yourself in constraints and current behavior.**
   - Skim `docs/adr/` and read any ADR whose decision constrains this change (e.g. ADR-0001 forbids a Go GitHub SDK, ADR-0002 forbids `os/exec` outside `internal/shell`). These are hard constraints — the plan must not violate them.
   - Read the relevant spec(s) in `docs/specs/` for the behavior being changed. The plan's "Spec Changes" section must be accurate against these.
   - Look at the relevant code only as needed to make the Approach section realistic.

2. **Derive a kebab-case name** (e.g. "retry failed pushes" → `retry-failed-pushes`). Check `docs/plans/` for an existing plan with the same or a conflicting name; if one exists, ask the user whether to continue it or supersede it.

3. **Write `docs/plans/<name>.md`** using exactly this format:

   ```markdown
   ---
   date: YYYY-MM-DD
   status: active
   title: Short descriptive title
   related-specs: [spec-name-1, spec-name-2]
   related-adrs: [0001, 0003]
   ---

   # Plan Title

   ## Context
   Why this change is needed.

   ## Goals
   - Specific, measurable objectives

   ## Approach
   Technical strategy.

   ## Tasks
   - [ ] Implementation step 1
   - [ ] Implementation step 2
   - [ ] Update specs

   ## Spec Changes
   Which specs will be modified and how.
   ```

   - `related-specs` / `related-adrs`: omit from frontmatter if none apply.
   - `Tasks` is a checklist; keep items specific and actionable. Always include a spec-update task when behavior changes.
   - `Spec Changes` is a preview, not a delta spec — name the spec files and describe the rule-level changes in prose or bullets. If no spec changes, write "None".

4. **Summarize** for the user: plan path, goals, task count, and which specs will change. Tell them they can ask you to start implementing.

## Guardrails

- One file only. Do not create proposal/design/tasks splits or delta spec directories.
- Do not implement anything; this skill ends at a written plan.
- Keep the plan proportional: a small fix gets a short plan. Do not pad sections.
- Plans are ephemeral scratchpads. They get deleted by the `document` skill when the work is done, so don't put anything in a plan that belongs permanently in a spec or ADR.

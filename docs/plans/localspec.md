---
date: 2026-07-28
status: active
type: meta
title: Migrate from OpenSpec to LocalSpec workflow
---

# LocalSpec Migration Plan

## Context

The project currently uses OpenSpec for behavioral documentation and change management. While the structured planning and delta-application steps provide value, the full workflow (proposal → design → tasks → spec deltas → verification → archival) has become too heavyweight for this codebase. The user wants to preserve the "structured plan" and "delta confirmation" aspects while eliminating the ceremony.

## Current State (As-Is)

### Directory Structure
```
openspec/
├── specs/                    # 15 behavioral specifications
│   ├── submit-export/
│   │   └── spec.md          # Detailed behavior for submit/export commands
│   ├── land/
│   │   └── spec.md
│   ├── view/
│   │   └── spec.md
│   └── ... (12 more)
└── changes/                  # Active and completed changes
    ├── make-git-operation-detection-layout-aware/
    │   ├── proposal.md      # Problem statement and goals
    │   ├── design.md        # Technical approach
    │   ├── tasks.md         # Implementation checklist
    │   └── specs/           # Delta specs (showing changes)
    │       └── git-operation-safety/
    │           └── spec.md
    ├── restore-stash-on-prerun-failure/
    └── ... (several more)
```

### Workflow
1. **Propose**: Create `proposal.md` with problem/goals
2. **Design**: Create `design.md` with technical approach
3. **Tasks**: Create `tasks.md` with implementation steps
4. **Delta Specs**: Create modified spec files showing changes
5. **Implement**: Write code
6. **Verify**: Check implementation matches deltas
7. **Archive**: Merge deltas into main specs, move change to archive

### Pain Points
- **146 files** to maintain for a CLI tool
- **8 skills** required for the workflow (openspec-propose, openspec-continue, openspec-apply, etc.)
- **Directory sprawl**: Deep nesting for simple changes
- **Synchronization overhead**: Keeping specs and code in sync requires formal delta application
- **Context switching**: Multiple files to create before implementation can begin

## Target State (To-Be)

### Three Pillars

1. **Specs** (`docs/specs/`)
   - Current behavioral contracts
   - Flattened structure (no nesting)
   - Read by agents before touching code
   - Updated directly during implementation or via `document` skill

2. **ADRs** (`docs/adr/`)
   - Architecture Decision Records
   - Constraints and "why we can't do X"
   - Grounding for new plans
   - Managed via `adrm` CLI tool (starting at 0001)
   - Immutable history of significant decisions

3. **Plans** (`docs/plans/`)
   - Active implementation work
   - Single file per plan: `<name>.md` with frontmatter
   - Transient scratchpads for implementation
   - Archived or deleted after completion

### New Directory Structure
```
docs/
├── specs/                  # Flattened from openspec/specs/
│   ├── submit-export.md   # Was: submit-export/spec.md
│   ├── land.md            # Was: land/spec.md
│   ├── view.md
│   └── ...
├── adr/                   # New: Architecture Decision Records
│   ├── 0001-no-go-github-sdk.md
│   ├── 0002-shell-out-to-git.md
│   └── ...
└── plans/                 # Active plans only
    └── localspec.md       # This document
```

### Workflow

**Planning Phase** (replaces: propose → design → tasks)
- Single file: `docs/plans/<name>.md`
- Frontmatter: `date`, `status: active`, `title`, `related-specs`, `related-adrs`
- Body: Context, Goals, Approach, Tasks (checklist)
- Optional: Spec delta preview (what will change)

**Implementation Phase**
- Code changes
- Update specs directly in `docs/specs/` (or use `document` skill at end)

**Documentation Phase** (replaces: verify → archive)
- Run `document` skill
- Deletes the plan file (git history preserves the record)
- Ensures specs are updated
- Prompts for ADR creation if architectural decisions emerged

### Skills (2 total, replacing 8)

1. **`plan` skill**
   - Input: Feature description
   - Reads: Relevant ADRs (constraints) + Specs (current behavior)
   - Creates: `docs/plans/<name>.md`
   - Output: Structured plan with tasks and spec impact

2. **`document` skill**
   - Input: Completed plan name
   - Actions:
     - Validates specs are updated
     - Deletes the plan file (git history preserves it)
     - Updates AGENTS.md if needed
     - Prompts for ADR if architectural decision made
   - Output: Clean state with updated documentation

## Migration Path

### Phase 1: Documentation Migration
- [x] Create `docs/specs/` directory
- [x] Flatten `openspec/specs/*/spec.md` → `docs/specs/*.md`
- [x] Update AGENTS.md references from `openspec/specs/` to `docs/specs/`
- [x] Create `docs/adr/` directory
- [x] Extract constraints from AGENTS.md into initial ADRs:
  - No Go GitHub SDK (shell out to gh) — ADR-0001
  - Shell package exclusivity (no os/exec outside shell/) — ADR-0002
  - Branch template requires $ID — ADR-0003
  - Destructive commands must be optional (land removable via config) — ADR-0004

### Phase 1b: Spec Format Simplification
- [x] Define tiered spec format (rules / tables / rare scenario blocks) — see Format Specifications
- [x] Rewrite `docs/specs/view.md` as reference exemplar
- [x] Rewrite remaining 14 specs, preserving all behavioral content
- [x] Verify no behavioral facts lost vs. git history

### Phase 2: Skill Creation
- [x] Create `.agents/skills/plan/SKILL.md`
- [x] Create `.agents/skills/document/SKILL.md`
- [x] Remove or deprecate openspec-* skills (deleted, along with the `.opencode/commands/opsx-*` slash commands that invoked them)

### Phase 3: Cleanup
- [x] Delete `openspec/` directory entirely (including changes/)
- [x] Update AGENTS.md architecture section
- [x] Remove leftover `openspec-*` skill copies and `/opsx-*` workflow files (`.pi/skills/`, `.codex/skills/`, `.agent/workflows/`)

## Format Specifications

### Plan Format (`docs/plans/<name>.md`)
```markdown
---
date: YYYY-MM-DD
status: active|completed|abandoned
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
Which specs will be modified and how (optional preview).
```

### ADR Format (`docs/adr/NNNN-title.md`)
Managed via `adrm` CLI. Format determined by the tool, typically:
- Sequential numbering starting at 0001
- Standard sections: Status, Context, Decision, Consequences
- Stored in `docs/adr/`

### Spec Format (`docs/specs/<name>.md`)

Tiered format — use the cheapest tier that expresses the behavior precisely. One rule per bullet keeps test traceability (one rule ≈ one test case), replacing the old OpenSpec Requirement/Scenario scaffolding.

```markdown
---
title: Command/Feature Name
status: stable|beta|deprecated
---

# <Title>

## Overview
What this does and why it exists. One short paragraph.

## Behavior
All rules are normative; "must" is implied. (State this convention here in
the format guide, not inside each spec.)

### <Rule group>            ← Tier 1 (default): condition → outcome
- <condition> → <outcome>
- <condition> → <outcome>

### <Enumeration group>     ← Tier 2: table for families differing only
| Condition | Behavior |       in an enumerated value (statuses,
|--------------------------|  styles, formats, fields)

### <Sequence group>        ← Tier 3 (rare): only for multi-step,
#### Scenario: <name>         ordering-sensitive flows (stash/mutate/
- **WHEN** ...                restore, error recovery)
- **THEN** ...

## Examples (optional)
Common usage patterns.
```

Tier selection rules:
1. **Rules** (`- condition → outcome`) are the default. Multi-clause conditions joined with "and"; negations kept as "does not" / "never".
2. **Tables** when 3+ scenarios differ only in an enumerated value.
3. **Scenario blocks** only when step ordering or teardown semantics matter. If a scenario has no meaningful sequence, demote it to a rule.

## Benefits

1. **Reduced overhead**: 2 skills instead of 8
2. **Fewer files**: ~20 files instead of 146
3. **Faster planning**: One file vs three
4. **Clearer separation**: Constraints (ADRs) vs Contracts (Specs) vs Work (Plans)
5. **Easier onboarding**: New agents read specs + ADRs, not complex workflow docs

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Losing change history | Git history preserves everything; plans are ephemeral |
| Specs becoming stale | `document` skill enforces updates; CI check possible |
| ADR bloat | Strict criteria: only architectural constraints, not implementation details |
| ADR numbering conflicts | `adrm` tool manages sequential numbering |

## Success Criteria

- [x] AGENTS.md points to `docs/specs/` not `openspec/`
- [x] `plan` skill creates valid plan files
- [x] `document` skill deletes plans and updates specs
- [x] No references to `openspec` in codebase (except git history)
- [x] New workflow documented in AGENTS.md

## Decisions Log

- **2026-07-28**: Plans are ephemeral; delete after completion (git history preserves them)
- **2026-07-28**: ADRs managed via `adrm` CLI, starting at 0001
- **2026-07-28**: Specs use simplified, flexible template (not rigid structure)
- **2026-07-28**: No `docs/plans/completed/` archive directory; plans are transient
- **2026-07-28**: Specs use a tiered format (rules → tables → rare scenario blocks) instead of uniform GIVEN/WHEN/THEN scenarios; ~40–50% of the old corpus was scenario boilerplate. `docs/specs/view.md` is the reference exemplar.
- **2026-07-28**: New skills live in `.agents/skills/` (tool-agnostic, consistent with `gh-stack`), not `.opencode/skills/`.
- **2026-07-28**: openspec-* skills deleted outright (not deprecated); the `/opsx-*` slash commands that invoked them were deleted too. Git history preserves both.
- **2026-07-28**: Plan files are named `docs/plans/<name>.md` (no date prefix; the date lives in frontmatter).
- **2026-07-28**: `document` skill validates specs via agent judgment, not scripted diffing.

## Next Steps

Phase 3 (Cleanup): delete `openspec/` entirely (including `changes/`), then finish the AGENTS.md workflow section so it describes only the LocalSpec workflow.

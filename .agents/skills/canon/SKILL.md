---
name: canon
description: Manage Architecture Decision Records (ADRs), SPECs, and domain entries with the canon CLI. Use whenever creating, recording, or revisiting an architectural decision; defining or updating a canonical domain concept; transitioning an ADR, SPEC, or domain entry through its lifecycle (accept, reject, supersede, deprecate, append); querying decision history or the domain model; or initializing document storage - even if the user does not mention canon by name.
---
<!-- canon-skill-version: 10 -->
<!-- canon-skill-hash: sha256:42873dc2d50d4a1d4de86a2d7339393a1f3e3b45f990c40cdc09869c64b189d0 -->

# CANON Agent Skill

Use canon to manage Architecture Decision Records without guessing repository state.

## Operating rules

1. Start with `canon commands` to inspect command metadata, side effects, selectors, examples, and dry-run availability.
2. Run `canon doctor` before mutating documents. If it reports a missing ADR, SPEC, or domain directory, preview initialization with `canon adr init --dry-run`, `canon spec init --dry-run`, or `canon domain init --dry-run` before applying. Doctor also flags domain-model integrity problems: duplicate accepted titles and references to superseded or deprecated entries. For deep integrity checks (malformed files, duplicate ids, broken references, reciprocity, metadata validity), run `canon validate` — doctor answers "can I work here?", validate answers "is my corpus healthy?".
3. Use JSON output unless a human explicitly asks for text or a bounded prompt projection. For prompt injection, `canon --format context adr list --status accepted` emits concise Markdown; use context format only with list commands. Every JSON response has `schema_version`, `status`, `data`, and optional `error` / `next_actions`.
4. For every mutating command, run the same command with `--dry-run` first and verify the returned plan. The plan response is how you confirm selectors and side effects before anything touches disk; a correct dry-run carries `No changes were made.` in warnings.
5. Use `canon list` (all kinds), `canon adr list` / `canon spec list` / `canon domain list`, `canon search --query ...`, and `canon show --id ...` to gather context before changing a document.
6. Prefer selectors from CLI output. Document ids are stable strings like `ADR-0001`, `SPEC-0001`, and `DM-0001` and can be passed to `--id` or `--by`.

## Common commands

Preview each of these with `--dry-run` first (rule 4), then apply:

```sh
canon adr new --title "Use SQLite for local query index" --status proposed --context "Agents need fast local lookup." --decision "Use SQLite-backed indexes."
canon spec new --title "Local query index" --requirements "Return ADRs by tag." --acceptance "canon list --tag storage returns ADR-0001."
canon domain new --title "ADR" --definition "A dated, narrowly-scoped architecture commitment." --avoid "design doc: too broad; ticket: tracks work, not decisions"
canon accept --id ADR-0001 --reason "Approved by the team."
canon reject --id ADR-0001 --reason "Chose a different approach."
canon supersede --id ADR-0001 --by ADR-0002 --reason "ADR-0002 captures the current storage approach."
canon deprecate --id ADR-0003 --reason "The system no longer uses this component."
canon append --id ADR-0002 --title "Implementation note" --body "The initial rollout used the default local index."
canon skill install
canon skill update
```

## When to create or change an ADR

`canon` stores *Architecture Decision Records*, not plans, tickets, or changelogs. Use an ADR when these four tests are all true:

1. **It is a commitment, not an intention.** Past-tense: "We decided X". Not "We will add X".
2. **It is architectural.** It shapes the system's structure, contract, data model, or cross-cutting policy, and reversal would ripple.
3. **It is non-obvious.** Reasonable people might choose differently, so the reasoning is worth preserving.
4. **It is narrow.** One ADR per decision. Bundles hide the real tradeoff.

### Technical vs product decisions

A pure product decision (market, prioritization, pricing) is **not** architecture and does not belong in an ADR. It belongs in an ADR only when it **forces an architectural commitment**. In that case, the product driver goes in **Context** as a force, and the **Decision** is the architectural commitment it produced.

### `canon` trigger list

Decisions that affect the CLI contract, ADR file format, query behavior, lifecycle semantics, output schema, or storage layout are architectural when they establish structural or cross-cutting commitments. Project processes, agent workflows, and skill behavior belong in `AGENTS.md` or the relevant `SKILL.md`, not in ADRs.

### Anti-patterns

Do not create an ADR for:

- a roadmap ("we will add A, B, C")
- a ticket ("add command X")
- a changelog entry ("implemented back-references")
- a bundle of unrelated decisions
- a product strategy with no architectural consequence
- a project process, agent workflow, or skill instruction
- vague commitments ("be flexible")
- obvious decisions with no real alternatives

## When to create or change a domain entry

Domain entries are the project's single source of truth for what things mean: one canonical concept per entry, with a definition, avoided terms (each with a reason), and relationships to other entries as relative markdown links.

1. **Search before defining.** Run `canon domain search --query ...` first; if an entry exists, sharpen it instead of creating a parallel one.
2. **One concept per entry.** The title is the canonical term. Do not bundle several terms into one entry.
3. **Definitions carry no implementation details.** An entry says what a concept is, not how it is built.
4. **Record rejected wording.** Every avoided term gets a reason, so future readers know why "design doc" is not an ADR.
5. **Supersede is for redefinitions.** Renaming a concept retitles the same entry (title is content, not lifecycle metadata) plus a `canon append` history note; a changed meaning gets a new entry superseding the old.

## Recovery

If a command fails, read `error.code`, `error.category`, and `error.suggested_fix`. Prefer the suggested next diagnostic command over guessing. For missing or unreadable ADR state, run `canon doctor`.

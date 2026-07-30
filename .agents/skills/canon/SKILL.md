---
name: canon
description: Manage Architecture Decision Records (ADRs) and SPECs with the canon CLI. Use whenever creating, recording, or revisiting an architectural decision; transitioning an ADR or SPEC through its lifecycle (accept, reject, supersede, deprecate, append); querying decision history; or initializing ADR storage - even if the user does not mention canon by name.
---
<!-- canon-skill-version: 5 -->
<!-- canon-skill-hash: sha256:62e4cc90292f13c4650291e67ca9299eb5c9180cf35bfdc2ec5d1db3286cc26b -->

# CANON Agent Skill

Use canon to manage Architecture Decision Records without guessing repository state.

## Operating rules

1. Start with `canon commands` to inspect command metadata, side effects, selectors, examples, and dry-run availability.
2. Run `canon doctor` before mutating ADRs. If it reports a missing ADR or SPEC directory, preview initialization with `canon adr init --dry-run` or `canon spec init --dry-run` before applying.
3. Use JSON output unless a human explicitly asks for text. Every JSON response has `schema_version`, `status`, `data`, and optional `error` / `next_actions`.
4. For every mutating command, run the same command with `--dry-run` first and verify the returned plan. The plan response is how you confirm selectors and side effects before anything touches disk; a correct dry-run carries `No changes were made.` in warnings.
5. Use `canon list` (both kinds), `canon adr list` / `canon spec list`, `canon search --query ...`, and `canon show --id ...` to gather context before changing an ADR.
6. Prefer selectors from CLI output. ADR ids are stable strings like `ADR-0001` and can be passed to `--id` or `--by`.

## Common commands

Preview each of these with `--dry-run` first (rule 4), then apply:

```sh
canon adr new --title "Use SQLite for local query index" --status proposed --context "Agents need fast local lookup." --decision "Use SQLite-backed indexes."
canon spec new --title "Local query index" --requirements "Return ADRs by tag." --acceptance "canon list --tag storage returns ADR-0001."
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

Decisions that affect the CLI contract, ADR file format, query behavior, lifecycle semantics, output schema, storage layout, or agent operating model are almost always architectural. Not every change to those surfaces is a commitment, but a change that fixes a contract downstream consumers will depend on is.

### Anti-patterns

Do not create an ADR for:

- a roadmap ("we will add A, B, C")
- a ticket ("add command X")
- a changelog entry ("implemented back-references")
- a bundle of unrelated decisions
- a product strategy with no architectural consequence
- vague commitments ("be flexible")
- obvious decisions with no real alternatives

## Recovery

If a command fails, read `error.code`, `error.category`, and `error.suggested_fix`. Prefer the suggested next diagnostic command over guessing. For missing or unreadable ADR state, run `canon doctor`.

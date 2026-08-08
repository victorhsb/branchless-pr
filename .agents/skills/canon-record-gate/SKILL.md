---
name: canon-record-gate
description: Classify candidate knowledge as an ADR, SPEC, Domain Entry, or None, and validate whether a proposed record uses the right kind. Use whenever deciding whether something should be recorded as an architecture decision, behavioral specification, or canonical domain concept; reviewing a proposed record's kind; or separating mixed decisions, requirements, and terminology. This skill judges kind fit and record readiness only, not document format, lifecycle, CLI usage, duplication, or retention.
---
<!-- canon-skill-version: 2 -->
<!-- canon-skill-hash: sha256:af53a2b2b7a204fcd1d9d4df26caa38968749070784431f5365639213e5740ca -->

# Canon Record Gate

Judge what kind of record, if any, a candidate deserves. Apply every gate for a
kind rather than choosing by keywords or by the candidate's requested label.

## Establish The Evidence

Support two modes:

- **Classification:** determine the candidate's kind.
- **Selected-kind validation:** test a caller's proposed kind and identify a
  better-fitting kind when the proposed one fails.

Use the candidate, supplied context, and available project evidence. Inspect
relevant documents or code when doing so is practical. Do not invent agreement,
architectural impact, observable behavior, or established terminology. Report
insufficient evidence when a required fact cannot be established.

Treat bare normative wording inside proposed record text as a claim to test,
not proof that the claim is agreed or established. Words such as "must" or
"means" do not by themselves establish commitment or settled usage. Explicit
context that a team approved or decided something, demonstrated corpus usage,
or equivalent evidence can establish it.

Separate distinct claims before applying the gates. A proposal that combines an
architecture choice, required behavior, and canonical terminology is not one
record; judge each concern independently.

## Apply The Kind Gates

### ADR

Record an ADR only when all four tests pass:

1. **Commitment:** it records a settled decision, not an intention, option, or
   plan.
2. **Architectural:** it shapes durable system contracts, interfaces,
   boundaries, data, lifecycle semantics, cross-cutting policy, or
   deployment/runtime structure. Reversal would have effects beyond one local
   implementation detail.
3. **Non-obvious:** reasonable people could choose differently, so preserving
   the trade-off and reasoning has future value.
4. **Narrow:** it records one decision.

Product or operational drivers may explain an architecture decision, but they
do not become architecture decisions without a system-shaping commitment.

### SPEC

Record a SPEC only when all three tests pass:

1. **Commitment:** it records agreed requirements, not a request, desire, or
   exploratory story.
2. **Behavioral:** it defines observable behavior for one capability and has
   testable acceptance criteria.
3. **Narrow:** it specifies one capability.

A SPEC says what must be observable. Reasoning behind a system-shaping choice
belongs to an ADR; implementation work that does not change agreed behavior is
not a new SPEC.

### Domain Entry

Record a Domain Entry only when all five tests pass:

1. **Relevant:** the concept appears in project knowledge and affects decisions
   or interpretation, or another canonical definition relies on it.
2. **Load-bearing:** a wrong interpretation would steer a human or agent in the
   wrong direction.
3. **Non-obvious:** common meaning is insufficient, or the project gives the
   concept a consequential specialization.
4. **Narrow:** it defines one concept.
5. **Self-supporting:** it answers a distinct reader question on its own; it is
   neither broad enough to split nor shallow enough to absorb elsewhere.

The meaning must also be settled enough to prescribe canonical usage. An
ephemeral term can fit the concern but is not ready to record.

## Decide

Use these outcomes:

- **Ready:** exactly one kind passes all its gates with enough evidence.
- **Not ready:** the likely kind is clear, but its decision, requirements, or
  meaning is not settled enough to record. Use this only when the other
  substantive gates establish the likely kind.
- **Insufficient evidence:** kind or readiness depends on facts that are not
  available. Name the unresolved gates instead of guessing.
- **None:** no kind passes. State the failed gates, but do not route the content
  to another artifact or workflow.
- **Split:** independent concerns fit different kinds or violate narrowness.
  Give each proposed record its own gate result; do not bless every piece merely
  because the original proposal mentioned it.

Use `Multiple` and `Split` only when at least two concerns independently fit or
likely fit record kinds. An unresolved local implementation detail is not a
second record merely because it appears beside a valid SPEC or Domain Entry.
Exclude that detail from the recordable concern instead of letting it make the
valid concern not ready; explain the tighter scope in the rationale.
When substantive gates such as architectural impact, relevance, or
load-bearing meaning fail, use `None` rather than preserving the caller's label
as a likely fit.

Do not add document-format checks, lifecycle advice, creation commands,
duplication analysis, or judgments about whether an existing corpus entry
should be retained. Those are separate concerns.

## Verdict Format

Keep the result compact:

```text
MODE: <classification | selected-kind validation>
REQUESTED KIND: <ADR | SPEC | Domain Entry | not supplied>
KIND: <ADR | SPEC | Domain Entry | None | Multiple | Undetermined>
READINESS: <ready | not ready | insufficient evidence>
GATES: <one concise pass/fail/unknown line per applicable gate>
RATIONALE: <why this kind and readiness follow from the evidence>
SPLIT: <separate concerns and their individual results; omit unless needed>
```

For selected-kind validation, `KIND` reports the actual fit, not the requested
label. Keep the likely kind with `not ready` when the concern fits but lacks a
settled commitment or meaning. Use `None` when no kind fits, and `Undetermined`
only when missing evidence prevents classification.

Read [boundary examples](references/boundary-examples.md) when the distinction
between kinds, readiness, or mixed concerns is unclear.

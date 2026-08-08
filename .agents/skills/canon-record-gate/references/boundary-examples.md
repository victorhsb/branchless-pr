<!-- canon-skill-version: 2 -->
<!-- canon-skill-hash: sha256:03000a23e13ce852b888e6908a0bbb222d7a774f0de2dcf99f2b6c1ded81fbfa -->

# Boundary Examples

These examples illustrate the gates; their subjects are not automatic trigger
lists. Real verdicts still depend on project evidence.

## ADR

**Candidate:** The team decided to isolate tenants with separate databases
after comparing shared-schema cost against failure containment. The choice
changes provisioning, backup, and connection boundaries.

**Result:** ADR, ready. It is a settled, system-shaping, non-obvious, narrow
decision. Required tenant behavior, if any, should be judged separately as a
SPEC.

## SPEC

**Candidate:** The approved password-reset capability must expire links after
30 minutes and show an expired-link response. Acceptance checks exercise both
conditions through the user-visible interface.

**Result:** SPEC, ready. It records agreed, observable, testable behavior for
one capability without explaining an architecture choice.

## Domain Entry

**Candidate:** Project documents use "settlement date" to mean the business day
on which ownership transfers, not the payment processor's capture timestamp.
Scheduling decisions depend on the distinction, and the usage is established.

**Result:** Domain Entry, ready. The concept is relevant, load-bearing,
project-specific, narrow, settled, and substantial enough to answer a distinct
reader question.

## None

**Candidate:** Add retry logging before the next sprint ends.

**Result:** None, not ready. This is an intention rather than a settled
architecture decision or agreed behavioral requirement, and it defines no
canonical concept.

## Split

**Candidate:** We decided to use an event store; the create-order operation must
return an order identifier; and "accepted order" means an order that passed
policy checks but has not yet been allocated.

**Result:** Split. Judge the event-store decision as a possible ADR, the
observable operation result as a possible SPEC, and the specialized term as a
possible Domain Entry. Each piece must independently pass its full gate; the
combined text fails every kind's narrowness test.

Normative phrasing inside a candidate does not prove readiness. If the only
evidence is that the text says "must" or "means," mark commitment or settled
usage unknown until context establishes it.

## Tighten Without Splitting

**Candidate:** An approved, acceptance-tested requirement says failed invoice
emails receive three attempts. The same proposal says to use exponential
backoff, but no retry mechanism has been selected.

**Result:** SPEC, ready, scoped to three attempts. The unselected backoff detail
is not a second record kind and does not make the approved behavioral concern
unready. Exclude it and explain the tighter scope.

## Right Kind, Not Ready

**Candidate:** The team is comparing a queue and a database outbox for durable
delivery across service boundaries. No option has been selected.

**Result:** ADR, not ready. The concern is architectural and non-obvious, but
the commitment gate fails until a decision is settled.

## Insufficient Evidence

**Candidate:** Document our account-closure policy.

**Result:** Undetermined with insufficient evidence. The text does not reveal
whether this is an architecture decision, agreed observable behavior, a
specialized canonical concept, or merely a request.

## Failed Substantive Gates

**Candidate:** Create a Domain Entry for "active customer." The phrase occurs
once in a brainstorming note, has several current meanings, and affects no
decisions.

**Result:** None, not ready. Ambiguity alone does not make a canonical concept.
Relevance and load-bearing meaning fail, and usage is not settled; this is not a
Domain Entry fit waiting only for maturity.

## Misplaced Selected Kind

**Candidate:** Validate as an ADR: "A workspace is the billing boundary that
owns projects and members."

**Result:** The requested ADR kind fails because no decision or architectural
trade-off is recorded. Domain Entry is the likely fit, but readiness depends on
evidence that the meaning is relevant, load-bearing, non-obvious, narrow,
self-supporting, and settled.

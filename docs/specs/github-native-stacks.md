---
title: GitHub native stacks
status: stable
---

# GitHub Native Stacks

## Overview

Define the opt-in integration between branchless-pr's commit-oriented stacks and GitHub's native Stack REST resources while preserving safe reconciliation, local metadata authority, and legacy behavior when the integration is disabled or unavailable.

## Behavior

### Integration mode

`github.native_stacks` takes values `off`, `auto`, or `required` and defaults to `off`.

| Mode | Repository feature state | Behavior |
|------|--------------------------|----------|
| unset or `off` | any | use legacy branchless-pr behavior; never call a GitHub native Stacks endpoint |
| `auto` | native Stacks supported | use the native behavior defined for that command |
| `auto` | ordinary repository access succeeds but the repository-level GitHub Stacks endpoint reports the feature unavailable | warn that native stacks are unavailable; use legacy behavior |
| `required` | repository-level GitHub Stacks endpoint reports the feature unavailable | fail before command-specific mutation |

- `github.native_stacks` has a value other than `off`, `auto`, or `required` → configuration error when configuration is resolved.
- Native integration is `auto` or `required` and a native read or write is required → invoke the REST API through the base `gh api` command; never require the `github/gh-stack` extension.

### Eligibility

Only branchless-pr stacks satisfying the documented native Stack request, repository, ref-chain, and PR-state constraints are published, preserving the one-commit-per-PR invariant (one commit maps to exactly one PR).

- Stack contains between 2 and 100 PRs, every PR head belongs to the repository in the endpoint path, the bottom PR targets the ultimate Stack base, every later PR directly targets the previous PR's head, and every PR to be added is open or draft, unmerged, not merge-queued, and not auto-merge-enabled → eligible for native reconciliation.
- Stack contains exactly one PR → the PR remains a standalone PR in any mode; no GitHub Stack is created.

| Condition | `auto` | `required` |
|-----------|--------|------------|
| Stack contains more than 100 PRs | warn that branchless-pr conservatively limits native Stacks to 100 total PRs; continue with legacy behavior | fail before a native Stack write; the error identifies 100 as a branchless-pr client policy rather than a documented GitHub aggregate limit |
| Stack PR heads do not all belong to the repository in the endpoint path | do not attempt to create or extend a native Stack; warn and use legacy behavior | do not attempt to create or extend a native Stack; fail |

- Bottom PR does not target the ultimate base, or a later PR does not target the preceding PR's head → fail before a native Stack write and report the mismatched PR and refs.
- A PR to be created or appended is merged, closed-unmerged, merge-queued, or auto-merge-enabled → fail before a native Stack write; preserve the server response as authoritative if GitHub applies stricter validation.

### Reconciliation classification

Local and GitHub membership are classified before writing a native Stack; local PR order is authoritative.

| Local vs remote membership | Action | Details |
|---------------------------|--------|---------|
| Every local PR is unstacked on GitHub | `create` | create payload lists PR numbers bottom-to-top |
| Every local PR belongs to one GitHub Stack whose complete PR sequence exactly equals the local bottom-to-top sequence | `noop` | no Stack write occurs |
| One GitHub Stack's complete PR sequence is an exact proper prefix of the local sequence and every local PR in the remaining suffix is unstacked | `append` | only the suffix PR numbers are sent to the append endpoint, bottom-to-top |
| The local sequence is a proper prefix of the GitHub Stack sequence | `conflict` | no Stack write occurs |
| Local PRs are reordered remotely, belong to multiple native Stacks, or contain a suffix PR already stacked elsewhere | `conflict` | error describes the local and remote membership sufficiently for manual resolution |

### Non-destructive reconciliation

Native Stacks are created or appended only for safe classifications, writes use the documented REST interface, and a conflicting Stack is never automatically replaced.

- Classification is `create` → POST all local PR numbers bottom-to-top to `repos/{owner}/{repo}/stacks`; pass PR numbers rather than branch names; do not modify PR lifecycle state as part of the Stack write.
- Classification is `append` → POST only the unstacked local suffix to `repos/{owner}/{repo}/stacks/{stack_number}/add`, preserving suffix bottom-to-top order; never send the complete existing sequence.
- A create or append REST request returns successfully → the complete returned or re-read remote PR sequence must exactly equal the planned bottom-to-top sequence before the operation is considered successful.
- branchless-pr invokes a native Stack REST write → branchless-pr remains responsible for branch pushes, PR creation, PR title and body, reviewers, direct bases, and draft state.
- Classification is `conflict` in `auto` or `required` mode → fail; never unstack, replace, reorder, or append the conflicting GitHub Stack.

### API error classification

REST failure context is preserved, distinguishing feature unavailability from authentication, authorization, missing resources, validation, rate limiting, transport uncertainty, and malformed responses.

| GitHub response | Behavior |
|-----------------|----------|
| Repository-level Stack list endpoint returns `404` after ordinary repository access succeeds | classify native Stacks as unavailable |
| Numbered Stack get, add, or unstack endpoint returns `404` | preserve the missing-resource context; never classify that response as repository-level feature unavailability without a separate availability probe |
| `401` or `403` | return the underlying status and GitHub diagnostic; never silently classify the feature as unavailable |
| `422` for a create, append, or unstack request | preserve the status and GitHub message; re-read relevant PR and Stack state before any new plan; never repeat the same write blindly |
| Rate-limit response | preserve the status and available retry diagnostics; never relabel the response as feature unavailable |
| Transport failure or `5xx` after a native write | mark the write outcome as uncertain; the next operation is read and reconciliation rather than an unconditional retry |
| Response cannot be decoded or violates required fields or ordering invariants | return an error; perform no native Stack write based on that response |

### Local commit metadata authority

Native integration does not replace branchless-pr's commit-oriented local identity model.

- Native reconciliation creates, appends, or confirms a GitHub Stack and submit/export completes → every submitted commit retains its `stack-info` PR and generated-branch metadata, and subsequent local discovery continues to use `BASE..HEAD`.
- A GitHub native Stack was not constructed from the current branchless-pr commit stack → never infer or rewrite local commits solely from that remote Stack.

### Explicit Go port behavior

Native Stack integration is an explicit Go-port extension to the base behavior documented in `docs/specs/`.

- `github.native_stacks = off` and submit, export, view, land, or abandon runs → behavior remains compatible with the non-native algorithms in `docs/specs/` (`submit-export`, `land`, `abandon`, and `view`).

### REST representation

The published pull-request membership and Stack resource schemas are decoded and validated (implemented in `internal/nativestacks`; see `REST_ACCEPTANCE_TEST_MATRIX.md` for acceptance evidence).

- Pull-request resource contains `stack.id`, `stack.number`, `stack.size`, `stack.position`, `stack.base.ref`, and `stack.base.sha` → preserve the global ID and repository-scoped Stack number as distinct values, interpret position as 1-based bottom-to-top order, and distinguish the PR's direct `base.ref` from the Stack's ultimate `stack.base.ref`.
- Pull-request resource contains `stack: null` → classify the PR as unstacked.
- Stack resource contains `id`, `number`, `node_id`, `url`, `base.ref`, `open`, `created_at`, and `pull_requests` → preserve the `pull_requests` array in returned bottom-to-top order; each member decodes `number`, `state`, `draft`, nullable `merged_at`, `head.ref`, and `head.sha`; do not require `base.sha` on the Stack resource.
- Membership or Stack response contains additional preview fields → ignore the unknown fields and continue to validate every documented required field.
- Response omits a required field, contains an impossible membership position, contains duplicate PR numbers, or cannot be decoded → return a schema error; perform no native Stack write based on that response.
- Stack member has `state: "closed"` → `merged_at != null` classifies the member as merged; `merged_at == null` classifies the member as closed but unmerged.

### REST read contract

Reconciliation uses the canonical repository-scoped REST paths and complete remote resources.

- Repository support is probed → first confirm ordinary repository access, then query the repository-level Stack list endpoint; only a Stack-list `404` after successful repository access is classified as feature unavailable.
- Local entries already have PR numbers → load each candidate PR from the repository and validate repository identity, state, head ref, direct base ref, and nullable Stack membership.
- One or more candidate PRs contain Stack membership → load every unique referenced Stack by its repository-scoped `stack.number`; never infer the complete ordered member list from membership `size` and `position`.

| `pull_request={number}` list-endpoint result | Meaning |
|----------------------------------------------|---------|
| empty array | the PR is unstacked |
| one result | identifies the Stack |
| more than one result | ambiguous membership |

- An operation enumerates repository Stacks rather than using a filtered lookup and the Stack list spans multiple pages → follow GitHub pagination until no next page remains; never treat one page as the complete repository set.

### REST mutation contract

Native Stacks are created, appended, and unstacked through the documented REST endpoints using JSON request bodies and repository-scoped identifiers.

- Classification is `create` → POST `{"pull_requests":[...]}` to `repos/{owner}/{repo}/stacks`; the array contains 2 through 100 PR numbers in bottom-to-top order; a successful response is `201` with the created Stack resource.
- Classification is `append` → POST `{"pull_requests":[...]}` to `repos/{owner}/{repo}/stacks/{stack_number}/add`; the array contains only 1 through 100 new suffix PR numbers from the current top upward; a successful response is `200` with the updated Stack resource.
- Unstack is requested → POST with no body to `repos/{owner}/{repo}/stacks/{stack_number}/unstack`.

| Unstack outcome | Response | Behavior |
|-----------------|----------|----------|
| One or more server-locked members remain | `200` | decode and report the returned surviving Stack; never assume the Stack was dissolved |
| Every member removed | `204` with an empty body | classify the Stack as dissolved |

- The desired change requires insertion below the top, arbitrary removal, movement, reorder, replacement, rebase, or merge → never emulate that operation with REST appends; report the operation as unsupported or conflicting.

### Write concurrency and recovery

Native Stack mutations are non-idempotent read-modify-write operations; uncertain outcomes are reconciled before any retry.

- Create or append returns a successful Stack resource → its complete bottom-to-top PR sequence must exactly equal the intended sequence; fail as conflicting if it does not.
- A create request fails after its server outcome becomes uncertain → query membership for a candidate PR and load any resulting complete Stack; accept an exact intended sequence as completed; never blindly repeat the create request.
- An append request fails after its server outcome becomes uncertain → reload the numbered Stack and recompute the ordered relationship; accept an exact intended sequence as completed; never blindly repeat the append request.
- An unstack request fails after its server outcome becomes uncertain → reload the numbered Stack and distinguish a dissolved Stack, a partial Stack, and an unverified result; never blindly repeat the unstack request.
- GitHub rebases or retargets remaining Stack branches and branchless-pr later updates a generated remote branch → use the previously observed remote OID as a force-with-lease expectation; fail rather than overwrite an unexpected server-side rewrite.

## Context

The first native-stack integration was intentionally built from provisional fixtures because no preview repository or published write contract was available. It uses `gh api` for reads but delegates writes to `github/gh-stack`. The published contract captured in `GITHUB_STACKS_REST_API.md` now documents direct create, add, and unstack endpoints and shows that the implemented response structs do not match the wire format.

The correction crosses `internal/nativestacks`, submit/export, dry-run, view, land, abandon, configuration, receipts, and user documentation. It must keep `github.native_stacks = off` unchanged, use only `internal/shell` for subprocesses, avoid a Go GitHub SDK, preserve `--dry-run` no-mutation guarantees, and update `SPEC.md` with the behavior.

## Goals / Non-Goals

**Goals:**

- Make every native Stack read and write conform to the REST contract in `GITHUB_STACKS_REST_API.md`.
- Decode the published PR membership and Stack resource shapes exactly while tolerating additive preview fields.
- Preserve bottom-to-top ordering and reject incomplete, duplicate, impossible, or ambiguous membership before mutation.
- Use direct REST create, append, and unstack operations through `gh api`.
- Reconcile safely after an uncertain write outcome and never blindly retry.
- Remove the `github/gh-stack` runtime dependency and its fallback states.
- Cover all twenty integration acceptance criteria in `GITHUB_STACKS_REST_API.md` with automated tests.

**Non-Goals:**

- Native stack landing, rebasing, merge-queue control, arbitrary insertion, reordering, replacement, or single-member removal.
- Importing arbitrary remote Stacks into the local commit model.
- Changing the default native mode from `off`.
- Removing the conservative branchless-pr limit of 100 total PRs until GitHub documents or live testing confirms the aggregate limit.
- Claiming GitHub Enterprise Server support or preview-specific permissions that GitHub has not documented.

## Decisions

1. **Use the documented REST writes directly through `gh api`.**

   Create uses `POST repos/{owner}/{repo}/stacks`, append uses `POST repos/{owner}/{repo}/stacks/{number}/add`, and abandon uses `POST repos/{owner}/{repo}/stacks/{number}/unstack`. JSON is supplied on stdin. The returned Stack is the mutation contract; create therefore obtains the new repository-scoped Stack number without parsing CLI prose.

   The former extension adapter is removed. It adds a second versioned runtime dependency, obscures HTTP status and response bodies, and cannot provide the direct response contract required for safe reconciliation.

2. **Represent wire resources separately from local reconciliation state.**

   REST types mirror the published nested objects:

   - PR membership: `id`, `number`, `size`, `position`, and `base.{ref,sha}`;
   - Stack: `id`, `number`, `node_id`, `url`, `base.ref`, `open`, `created_at`, and ordered `pull_requests`;
   - Stack member: `number`, `state`, `draft`, nullable `merged_at`, and `head.{ref,sha}`.

   Global IDs use `int64`; repository-scoped numbers remain `int`. Local `Membership` remains a planner-friendly derived type. Standard JSON decoding ignores unknown fields, followed by explicit validation of required values, unique PR numbers, membership bounds, and array order invariants.

3. **Load candidate PRs first, then complete Stacks.**

   Each existing local PR is read from `GET repos/{owner}/{repo}/pulls/{number}` to obtain repository identity, direct base, head, state, draft status, and nullable Stack membership. Every unique referenced Stack is then loaded by repository-scoped Stack number. Planning never infers the complete member list from membership summaries.

   Repository availability uses an ordinary repository read followed by a repository-level Stack list probe. A filtered list query is used to rediscover a Stack after a create result is uncertain. Full enumeration, when needed, uses `gh api --paginate`; normal membership loading does not enumerate all repository Stacks.

4. **Validate at both wire and product layers.**

   The REST adapter enforces documented request boundaries: create accepts 2–100 PR numbers and append accepts 1–100 suffix numbers. The branchless-pr planner retains its conservative 100-member total limit as an explicit client policy, not as a claimed GitHub aggregate limit.

   Before create or append, candidate PRs must be in the same repository, open or draft, not merged, not closed-unmerged, not merge-queued, and not auto-merge-enabled. Their direct bases and heads must form the intended chain. The server remains authoritative and validation failures are surfaced.

5. **Use typed API failures with operation context.**

   API errors preserve the HTTP status when available, GitHub's message/body, endpoint, and whether a failed write may have reached the server. A repository-level Stack-list `404` becomes `FeatureUnavailable` only after ordinary repository access succeeds. A numbered Stack `404` is `StackNotFound`; `401`, `403`, `422`, `429`, transport errors, and decode errors remain distinct failures.

   Unknown response fields are ignored. Missing required fields, duplicate members, impossible positions, and malformed JSON fail closed.

6. **Reconcile uncertain write outcomes by reading.**

   Successful create and append responses are validated immediately and must contain the exact intended sequence. If a write fails in a way that may have reached GitHub:

   - create looks up membership by a candidate PR, then loads and compares the complete Stack;
   - append reloads the numbered Stack;
   - unstack reloads the numbered Stack, accepting not-found as dissolved only in the context of that attempted unstack.

   An exact intended result is accepted as success. Unchanged or conflicting state returns a diagnostic error. No write is retried automatically.

7. **Model unstack as a two-result operation.**

   A `200` response returns and validates the surviving partial Stack. A `204` response has no body and means the Stack was dissolved. Abandon may delete generated branches only when the result or reconciliation proves that every affected unmerged local PR is no longer stacked. `state: closed` is not treated as merged; `merged_at != null` is authoritative.

8. **Keep native landing blocked for the same synchronization reason.**

   Direct REST writes do not add merge or rebase endpoints. The existing land gate remains and is reworded around the documented REST limitation. The `land.style = disable` registration behavior is unaffected.

## Risks / Trade-offs

- **Private-preview schemas change** → Keep native mode default-off, ignore additive fields, validate required fields, and return decode/schema errors without mutation.
- **`gh api` diagnostics vary by version** → Capture stdout and stderr, preserve raw diagnostics, parse status conservatively, and reconcile any ambiguous write failure.
- **Per-PR reads add requests** → Prefer correctness and required eligibility data; fetch each candidate once and deduplicate complete Stack reads.
- **The aggregate Stack limit may exceed 100** → Keep the current conservative product cap explicitly documented until live preview evidence supports safe multi-request creation.
- **Concurrent append or unstack races** → Re-read immediately before mutation, verify afterward, and never blindly retry a write.
- **A partial unstack retains members** → Return the surviving Stack and block branch deletion for affected unmerged members.

## Migration Plan

1. Add corrected wire models, validation, typed errors, read endpoints, and fixtures.
2. Add direct create, append, and unstack methods plus uncertain-outcome reconciliation.
3. Replace extension calls in submit/export and abandon; remove extension discovery and command wrappers.
4. Update dry-run, receipts, view/land diagnostics, config generation, `SPEC.md`, README, agent prompts, and changelog.
5. Run focused tests, all twenty REST acceptance tests, the full CI-equivalent suite, and OpenSpec validation.

Rollback is configuration-only for users: `github.native_stacks = off` preserves legacy behavior. Code rollback can restore the prior release, but any native Stack already created remains a GitHub resource until explicitly unstacked.

## Open Questions

- Is 100 a total Stack limit or only a per-request limit? Keep the conservative product cap pending preview validation.
- Which fine-grained token permissions and headers will GitHub require at general availability? Surface current errors and avoid inventing preview requirements.
- Does GitHub Enterprise Server support these endpoints? Treat support as unknown and rely on the availability probe.

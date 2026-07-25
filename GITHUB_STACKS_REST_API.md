# GitHub Native Stacked Pull Requests REST API

Status: research specification
Source snapshot: 2026-07-25
Product status: private preview

## 1. Purpose

This document specifies the REST representation and management contract for
GitHub native stacks of pull requests. It covers:

- how a pull request refers to its stack;
- how a stack is represented;
- how to list, read, create, extend, and dissolve stacks;
- the ordering and branch-chain requirements behind those operations;
- the lifecycle and error cases a stack-management client must handle; and
- operations that the Stacks REST API does not offer.

The primary source is GitHub's
[Stacked PRs REST API reference](https://github.github.com/gh-stack/reference/rest-api/).
Product-level behavior comes from the official
[overview](https://github.github.com/gh-stack/introduction/overview/),
[working guide](https://github.github.com/gh-stack/guides/stacked-prs/), and
[FAQ](https://github.github.com/gh-stack/faq/). Generic HTTP requirements come
from GitHub's
[REST API getting-started guide](https://docs.github.com/en/rest/using-the-rest-api/getting-started-with-the-rest-api)
and
[pagination guide](https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api).

The words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are
normative requirements for an API client implementing this specification.
"GitHub requires" identifies a server-side rule documented by GitHub.

## 2. Capability summary

| Capability | REST surface | Result |
| --- | --- | --- |
| Read a PR's stack membership | A `stack` field on each pull-request resource returned by REST | Membership summary or `null` |
| Find a stack containing a PR | `GET /repos/{owner}/{repo}/stacks?pull_request={number}` | Zero- or one-element stack array |
| List repository stacks | `GET /repos/{owner}/{repo}/stacks` | Paginated stack array, newest stack number first |
| Read one stack | `GET /repos/{owner}/{repo}/stacks/{stack_number}` | One stack resource |
| Create a stack | `POST /repos/{owner}/{repo}/stacks` | New stack, `201 Created` |
| Append PRs to the top | `POST /repos/{owner}/{repo}/stacks/{stack_number}/add` | Updated stack, `200 OK` |
| Remove eligible membership | `POST /repos/{owner}/{repo}/stacks/{stack_number}/unstack` | Updated partial stack (`200`) or dissolved stack (`204`) |
| Insert, reorder, replace, or remove one arbitrary member | Not offered | Unstack and reconstruct where possible |
| Rebase a stack | Not offered by the Stacks REST API | Use GitHub UI or a local stacking tool |
| Merge all or part of a stack | Not offered by the documented Stacks REST API | Use GitHub's stacked-PR merge UI/other documented product surface |

The dedicated Stacks API is deliberately small and asymmetric: creation and
growth are additive, while the only removal operation is a bulk best-effort
unstack.

## 3. Availability and protocol

### 3.1 Private-preview availability

`AVAIL-001` — A client MUST treat native Stacked PRs as a private-preview
feature whose schema and behavior may change.

`AVAIL-002` — The dedicated `/stacks` endpoints are available only for
repositories where GitHub enabled the feature.

`AVAIL-003` — GitHub documents `404 Not Found` when Stacked PRs is not enabled.
A `404` is not, by itself, proof of feature unavailability: GitHub may also
return `404` for a missing repository, a missing stack, or an inaccessible
private resource.

`AVAIL-004` — Before classifying a repository-level `/stacks` `404` as
"feature unavailable," a client SHOULD first prove that the authenticated
principal can read the repository through an ordinary repository endpoint.

`AVAIL-005` — The published Stacks reference does not document GitHub
Enterprise Server support. A client MUST NOT assume that the endpoints exist on
GHES solely because the host implements the ordinary pull-request API.

### 3.2 HTTP conventions

`HTTP-001` — Direct HTTP clients MUST use the repository API host and the
canonical paths in this document. For github.com, the base URL is
`https://api.github.com`.

`HTTP-002` — Requests SHOULD send:

```http
Accept: application/vnd.github+json
Authorization: Bearer <token>
X-GitHub-Api-Version: <supported-version>
User-Agent: <application-name>
```

GitHub requires a valid `User-Agent`. `gh api` supplies authentication and
standard GitHub headers for an authenticated GitHub CLI session.

`HTTP-003` — `POST` request bodies MUST be JSON objects and SHOULD use
`Content-Type: application/json`.

`HTTP-004` — The private-preview reference does not publish a special preview
media type, custom request header, or endpoint-specific token permission.
Clients MUST NOT invent one. They SHOULD surface authentication and
authorization errors without relabeling them as feature unavailability.

`HTTP-005` — Clients MUST tolerate additional response fields. They MUST NOT
require undocumented fields observed in a live response.

## 4. Domain model and invariants

### 4.1 Stack topology

A native stack is an ordered, linear chain of ordinary pull requests in one
repository:

```text
stack.base.ref
  <- PR at position 1 (bottom)
  <- PR at position 2
  <- ...
  <- PR at position size (top)
```

`MODEL-001` — Stack order is always **bottom to top**. Position `1` is the PR
closest to the ultimate base; the highest position is the top.

`MODEL-002` — Every PR in a stack remains an ordinary pull request with its own
reviews, draft state, checks, head branch, and direct base branch.

`MODEL-003` — All PR head branches in one stack must belong to the same
repository. GitHub does not support cross-fork stacks.

`MODEL-004` — The bottom PR's `base.ref` must equal the stack's ultimate
`base.ref`.

`MODEL-005` — For every PR above the bottom, that PR's direct `base.ref` must
equal the immediately preceding PR's `head.ref`.

`MODEL-006` — The REST create and append operations validate the ref chain.
Linear commit ancestry is additionally required for merging, but the REST
reference does not say that commit-graph linearity is a create/append
precondition. Clients SHOULD keep the branch history linear so the resulting
stack is mergeable.

`MODEL-007` — A stack has at least two PRs when created. A single PR remains a
standalone PR.

`MODEL-008` — The documented create payload contains 2–100 PR numbers. The
documented append payload contains 1–100 new PR numbers. The REST reference
does **not** explicitly say that 100 is the maximum total size after one or more
append operations. Clients that impose a 100-member total cap must describe it
as a conservative product/client constraint until GitHub documents or verifies
the aggregate limit.

`MODEL-009` — The API model exposes singular stack membership on each PR. A
client SHOULD enforce that a PR belongs to at most one stack and treat mixed
membership as a conflict. The REST prose does not separately document the
validation response for attempting to add a PR that belongs to another stack.

### 4.2 Identifier types

| Identifier | Scope | Use |
| --- | --- | --- |
| `stack.id` | Global integer | Stable resource identity |
| `stack.node_id` | Global string | GraphQL/global node identity |
| `stack.number` | Repository-scoped integer | Human-facing number and REST path identifier |
| PR `number` | Repository-scoped integer | Member identifiers in request bodies |

`ID-001` — `{stack_number}` path parameters MUST use `stack.number`, not
`stack.id`.

`ID-002` — A stack number is meaningful only with its `{owner}/{repo}`.

`ID-003` — Create and append payloads MUST contain PR numbers from the
repository named in the path.

## 5. Pull-request stack membership

GitHub adds a nullable `stack` property to pull-request resources returned by
the REST API. GitHub says this applies to every endpoint that returns a pull
request, including:

```http
GET /repos/{owner}/{repo}/pulls
GET /repos/{owner}/{repo}/pulls/{pull_number}
```

Example:

```json
{
  "id": 123456,
  "number": 50,
  "size": 5,
  "position": 2,
  "base": {
    "ref": "main",
    "sha": "def456..."
  }
}
```

### 5.1 Membership schema

| Field | Type | Required meaning |
| --- | --- | --- |
| `id` | integer | Global stack identifier |
| `number` | integer | Repository-scoped stack number |
| `size` | integer | Total number of PRs in the stack |
| `position` | integer | This PR's 1-based bottom-to-top position |
| `base.ref` | string | Ultimate target branch of the whole stack |
| `base.sha` | string | Current HEAD SHA of the ultimate base branch |

`MEM-001` — For a stacked PR, the client MUST interpret the PR resource's
`stack` field as the membership summary above.

`MEM-002` — For a standalone PR, the client MUST accept
`stack: null`.

`MEM-003` — A client MUST distinguish:

- the PR resource's `base.ref`: the PR's direct parent branch; and
- the PR resource's `stack.base.ref`: the whole stack's ultimate target.

They are equal for the bottom PR and normally differ for every PR above it.

`MEM-004` — A client MUST NOT infer the ordered list of PR numbers from
`size` and `position`. It must read the stack resource to obtain all members.

`MEM-005` — When reading several local PRs, a client SHOULD verify that their
`stack.number`, `stack.size`, `stack.position`, and `stack.base.ref` values
describe one coherent, contiguous sequence before acting on the membership.

## 6. Stack resource

The list, get, create, add, and partial-unstack responses use a stack resource.

```json
{
  "id": 9876543,
  "number": 42,
  "node_id": "S_kwDOABCDEF4AAAAA",
  "url": "https://api.github.com/repos/octocat/hello-world/stacks/42",
  "base": {
    "ref": "main"
  },
  "open": true,
  "created_at": "2026-04-15T10:00:00Z",
  "pull_requests": [
    {
      "number": 101,
      "state": "open",
      "draft": false,
      "merged_at": null,
      "head": {
        "ref": "user-model",
        "sha": "aaa1111..."
      }
    }
  ]
}
```

### 6.1 Stack fields

| Field | Type | Required meaning |
| --- | --- | --- |
| `id` | integer | Global stack identifier |
| `number` | integer | Repository-scoped stack number |
| `node_id` | string | Global node ID |
| `url` | string | API URL for this stack |
| `base.ref` | string | Ultimate target branch |
| `open` | boolean | `true` if any member PR is open |
| `created_at` | string | ISO 8601 creation timestamp |
| `pull_requests` | array | Members ordered bottom to top |

`RESOURCE-001` — Clients MUST preserve the array order returned in
`pull_requests`; sorting by PR number, date, or branch name is incorrect.

`RESOURCE-002` — `open: false` means no member PR is open. It does not mean the
stack resource has been dissolved.

`RESOURCE-003` — The stack resource documents `base.ref`, but not
`base.sha`. Clients MUST NOT require `base.sha` on this resource even though
the PR membership object includes it.

### 6.2 Minimal member fields

| Field | Type | Required meaning |
| --- | --- | --- |
| `number` | integer | Pull-request number |
| `state` | string | `open` or `closed` |
| `draft` | boolean | Whether the PR is a draft |
| `merged_at` | string or `null` | Merge time, or `null` when unmerged |
| `head.ref` | string | Head branch |
| `head.sha` | string | Current head SHA |

`RESOURCE-004` — A client MUST use `merged_at != null` to distinguish merged
PRs from closed-but-unmerged PRs because both use `state: "closed"`.

`RESOURCE-005` — A client MAY consume extra nested PR fields when present, but
MUST operate correctly with only the documented minimal fields.

As observed on 2026-07-25, the live list endpoint returned the minimal member
shape, while the detail endpoint returned extra PR fields such as `id`, `url`,
`base`, `title`, and `user`. Those extra detail fields are not part of the
published minimum contract.

## 7. Endpoint contract

### 7.1 List stacks

```http
GET /repos/{owner}/{repo}/stacks
```

The response is an array of stack resources ordered by stack number descending
(newest first).

Query parameters:

| Name | Type | Meaning |
| --- | --- | --- |
| `pull_request` | integer | Return the stack containing this PR number |
| `per_page` | integer | Results per page, maximum 100 |
| `page` | integer | Page number |

Success:

```http
200 OK
```

`LIST-001` — A client enumerating all stacks MUST follow the standard GitHub
`Link` response header until no `rel="next"` link remains. It MUST NOT assume
one page is complete.

`LIST-002` — A client looking up membership for one known PR SHOULD use
`pull_request={pr_number}` instead of listing every stack.

`LIST-003` — A filtered lookup MUST accept an empty array as "the PR is not in a
stack."

`LIST-004` — A filtered lookup SHOULD expect at most one result. More than one
result violates the singular-membership model and MUST be treated as ambiguous
rather than silently selecting one.

Examples:

```bash
gh api --paginate repos/OWNER/REPO/stacks
gh api "repos/OWNER/REPO/stacks?pull_request=102"
```

### 7.2 Get a stack

```http
GET /repos/{owner}/{repo}/stacks/{stack_number}
```

Success:

```http
200 OK
```

`GET-001` — The client MUST address the stack by its repository-scoped stack
number.

`GET-002` — The client SHOULD use this endpoint before append, unstack, or any
operation that depends on the complete current member order.

`GET-003` — A missing stack is expected to produce `404 Not Found`; clients
MUST distinguish that resource-level case from a repository-level feature
probe where possible.

Example:

```bash
gh api repos/OWNER/REPO/stacks/42
```

### 7.3 Create a stack

```http
POST /repos/{owner}/{repo}/stacks
Content-Type: application/json

{
  "pull_requests": [101, 102, 103]
}
```

Success:

```http
201 Created
```

with the created stack resource.

`CREATE-001` — `pull_requests` is required.

`CREATE-002` — The array MUST contain 2–100 repository-scoped PR numbers.

`CREATE-003` — The array MUST be ordered bottom to top.

`CREATE-004` — The bottom PR MUST target the intended ultimate stack base.

`CREATE-005` — Every later PR's `base.ref` MUST equal the previous PR's
`head.ref`.

`CREATE-006` — All PRs MUST belong to the repository in the endpoint path.

`CREATE-007` — Before creating, a client SHOULD verify that none of the PRs
belongs to another stack. Reissuing create without this preflight is not an
idempotent no-op contract.

`CREATE-008` — A client SHOULD verify the returned stack number and exact
bottom-to-top member sequence before reporting success.

`CREATE-009` — The REST reference does not publish a complete PR-state
eligibility table. As a conservative compatibility policy, the official
`github/gh-stack` client allows newly added open PRs (including drafts) and
rejects merged, closed, merge-queued, and auto-merge-enabled PRs. Clients SHOULD
preflight those states but MUST still treat the server response as authoritative.

Example:

```bash
gh api \
  --method POST \
  repos/OWNER/REPO/stacks \
  --input - <<'JSON'
{"pull_requests":[101,102,103]}
JSON
```

### 7.4 Add pull requests to a stack

```http
POST /repos/{owner}/{repo}/stacks/{stack_number}/add
Content-Type: application/json

{
  "pull_requests": [104, 105]
}
```

Success:

```http
200 OK
```

with the updated stack resource.

`ADD-001` — `pull_requests` is required.

`ADD-002` — The array MUST contain 1–100 PR numbers.

`ADD-003` — The request MUST contain only the new suffix, not the complete
existing stack.

`ADD-004` — New PRs MUST be ordered from the current top upward.

`ADD-005` — The first new PR's `base.ref` MUST equal the current top PR's
`head.ref`.

`ADD-006` — Each later new PR's `base.ref` MUST equal the preceding new PR's
`head.ref`.

`ADD-007` — A client MUST NOT use this endpoint to insert a PR below the top,
reorder existing members, replace a member, or remove a member.

`ADD-008` — A client SHOULD read the current stack immediately before append
and classify the desired sequence:

- exact current sequence: no-op;
- current sequence is an exact prefix and the remaining PRs are unstacked:
  append only the suffix;
- any other relationship: conflict; do not append.

`ADD-009` — A client SHOULD verify that the response contains the original
sequence followed by exactly the requested suffix.

`ADD-010` — The endpoint does not publish an idempotency key or conditional
mutation parameter. A retry after an uncertain network result MUST first re-read
the stack and recompute the delta.

`ADD-011` — New suffix members SHOULD pass the same conservative PR-state
preflight described by `CREATE-009`.

### 7.5 Unstack

```http
POST /repos/{owner}/{repo}/stacks/{stack_number}/unstack
```

The request takes no body.

Possible successes:

```http
200 OK
```

with an updated stack resource when one or more PRs remain, or:

```http
204 No Content
```

when no PR remains and the stack is dissolved.

`UNSTACK-001` — A client MUST send no request body.

`UNSTACK-002` — Unstack is a bulk best-effort operation over eligible,
unmerged members. It is not an arbitrary member-deletion endpoint.

`UNSTACK-003` — GitHub documents that merged, currently merging, and
merge-queued PRs cannot be unstacked and remain in the stack.

`UNSTACK-004` — On `200`, the client MUST parse the returned stack and treat it
as a partial unstack. It MUST NOT assume that the stack was deleted.

`UNSTACK-005` — On `204`, the client MUST accept the empty body and treat the
stack resource as dissolved.

`UNSTACK-006` — A client SHOULD generically preserve and report any members
returned after unstack, regardless of why the server retained them. The
official `gh-stack` client also anticipates auto-merge-enabled PRs being
non-removable; the published REST prose currently names merged, merging, and
queued PRs.

`UNSTACK-007` — A retry after `204` may receive `404`; callers that already
observed a successful dissolution MAY treat that as idempotent completion.
A caller targeting an unknown stack without prior success SHOULD report not
found.

`UNSTACK-008` — If downstream cleanup would delete PR head branches or local
tracking, it MUST occur only after the response proves that every affected
unmerged PR was removed from native membership.

Example:

```bash
gh api --method POST repos/OWNER/REPO/stacks/42/unstack
```

## 8. Lifecycle semantics

### 8.1 Standalone to active

Two or more ordinary PRs with correct base/head chaining become a native stack
through create. Existing PR state remains attached to each PR.

### 8.2 Active growth

An active stack grows only at the top through append. Changing PR base refs
alone creates a branch chain, but native Stack membership must still be created
or updated through the Stacks product surface.

### 8.3 Partial merge

GitHub's product behavior allows merging a contiguous prefix from the lowest
unmerged PR through a selected higher PR. Remaining PRs stay open, and GitHub
automatically rebases and retargets the remaining chain so the new lowest
unmerged PR targets the stack base.

`LIFE-001` — A client MUST expect head SHAs and direct base refs to change after
a GitHub-side partial merge or rebase.

`LIFE-002` — A client that also manages local branches SHOULD use observed
remote SHAs and safe force-push leases. It MUST NOT overwrite an unexpected
server-side rewrite.

### 8.4 Completed stack

When every PR is merged or closed, `open` is `false`. GitHub's product guide
says a fully merged stack is complete and cannot be extended; new work must
start a new stack rooted on the trunk.

`LIFE-003` — A client MUST NOT treat `open: false` as permission to append.

`LIFE-004` — A client MAY retain completed stacks in history and discovery
results. Completion is not dissolution.

### 8.5 Closed middle PR

Closing an unmerged PR in the middle preserves stack membership and blocks PRs
above it from being mergeable.

`LIFE-005` — A client MUST NOT infer that `state: "closed"` removes a PR from
the stack.

### 8.6 Partial or full unstack

An unstack may:

- remove all removable members and dissolve the resource (`204`); or
- retain locked members and return the surviving resource (`200`).

The operation does not mean "close PRs" or "delete branches"; it removes native
stack association where allowed.

## 9. Required management algorithm

This section defines a safe client workflow over the REST primitives.

### 9.1 Read and classify

`CLIENT-001` — Resolve repository access and native-stack availability before
planning a write.

`CLIENT-002` — Load every candidate PR and validate repository identity,
state, head ref, direct base ref, and nullable stack membership.

`CLIENT-003` — When any candidate PR is stacked, load its complete stack
resource. Do not plan from membership summaries alone.

`CLIENT-004` — Compare ordered PR-number sequences, never unordered sets.

`CLIENT-005` — The safe reconciliation states are:

| Observed remote membership | Desired local sequence | Safe action |
| --- | --- | --- |
| Every desired PR unstacked | Valid chain of at least two PRs | Create |
| One stack exactly equals desired sequence | Same sequence | No-op |
| One stack is an exact prefix; desired suffix is unstacked | Existing prefix + valid suffix | Append suffix |
| Desired sequence is a prefix of a longer remote stack | Remote has extra members | Conflict |
| Order differs | Reordered membership | Conflict |
| Desired PRs span multiple stacks | Mixed ownership | Conflict |
| A would-be suffix PR belongs to another stack | Conflicting ownership | Conflict |

`CLIENT-006` — A conflict MUST NOT trigger automatic unstack-and-recreate
unless the user explicitly requested destructive restructuring and the client
has accounted for members that cannot be unstacked.

### 9.2 Write ordering

`CLIENT-007` — Before create or append, ensure all head branches exist remotely
and all PR direct base refs form the intended chain.

`CLIENT-008` — PR creation, title/body editing, reviewer assignment, draft
state, and direct base updates are ordinary PR operations. The Stacks API does
not own them.

`CLIENT-009` — Create or append only after ordinary PR mutations succeed.

`CLIENT-010` — After a successful response, compare the complete returned or
re-read member sequence to the intended sequence.

`CLIENT-011` — If verification fails, report remote state as uncertain or
conflicting. Do not issue a second blind mutation.

### 9.3 Concurrency and retries

`CLIENT-012` — The Stacks reference exposes no ETag precondition,
`If-Match` requirement, resource version, idempotency key, or replace
operation for mutations.

`CLIENT-013` — Treat create, append, and unstack as read-modify-write
operations subject to races.

`CLIENT-014` — Retry reads for transient failures according to GitHub rate-limit
and retry headers. Do not blindly retry writes whose server outcome is unknown.

`CLIENT-015` — On append `404`, re-read availability and stack membership. The
stack may have been dissolved between preflight and write.

`CLIENT-016` — On validation failure, re-read PR bases, heads, states, and stack
membership before deciding whether a new plan is safe.

### 9.4 Forward-compatible parsing

`CLIENT-017` — Ignore unknown JSON fields.

`CLIENT-018` — Reject malformed required fields, impossible positions, or
duplicate PR numbers rather than mutating from ambiguous state.

`CLIENT-019` — Preserve full-width JSON integers supported by the client
language; do not assume stack global IDs fit in 32 bits.

## 10. Error classification

The Stacks reference explicitly documents feature-disabled `404` and the
success statuses. It does not provide a complete endpoint-by-endpoint error
table. The official `github/gh-stack` client handles `404` and `422` for stack
writes, but that implementation behavior is not a substitute for a published
REST error schema.

| Status/category | Meaning to consider | Required client behavior |
| --- | --- | --- |
| `400` | Malformed JSON or unsupported API version | Fix request; do not retry unchanged |
| `401` | Missing/invalid authentication | Surface authentication failure |
| `403` | Permission, rate limit, or policy failure | Inspect body and rate-limit headers |
| `404` on repository list path | Feature disabled, repository missing, or hidden by auth | Disambiguate with ordinary repository access |
| `404` on numbered path | Stack missing, feature disabled, repository missing, or hidden by auth | Use operation context and repository probe |
| `422` | Validation/ref-chain/state conflict | Surface server message; re-read before replanning |
| `429` | Rate limit | Respect `Retry-After`/rate-limit headers |
| `5xx`/transport failure | Transient or outcome unknown | Safe reads may retry; writes require reconciliation first |
| Decode/schema failure | Preview drift or malformed response | Fail closed; do not write from partial state |

`ERROR-001` — Clients MUST preserve the response status and GitHub error
message for diagnosis.

`ERROR-002` — Clients MUST NOT parse human-oriented CLI success text as the
REST correctness contract.

`ERROR-003` — Clients MUST distinguish "request failed" from "write outcome
unknown." After an unknown write outcome, the next operation is a read and
reconciliation, not an unconditional repeat.

`ERROR-004` — The Stacks reference does not publish endpoint-specific
fine-grained token permissions. Clients SHOULD inspect
`X-Accepted-GitHub-Permissions` when GitHub returns an integration-permission
error and SHOULD document the permissions verified in a preview-enabled test
repository.

## 11. Operations not offered by this REST API

The dedicated Stacks REST API has no documented endpoint for:

- changing the stack's ultimate base;
- replacing the complete member list;
- inserting below the current top;
- moving a member;
- reordering members;
- removing one selected member;
- moving a PR between stacks;
- merging a whole or partial stack;
- rebasing stack branches;
- enqueueing/dequeueing a stack in a merge queue;
- creating PRs, editing PR metadata, or pushing branches; or
- importing/exporting local stack-tracking state.

`LIMIT-001` — A client MUST NOT emulate unsupported structural edits by a
sequence of appends.

`LIMIT-002` — Restructuring generally requires: unstack, verify the result,
repair branches and PR base refs, then create a new stack. This is not always
possible while merged, merging, queued, or otherwise server-locked members
remain.

`LIMIT-003` — The Stacks API is a remote membership API, not a local Git branch
manager.

`LIMIT-004` — Native merge behavior exists in the GitHub product, but the
documented Stacks REST surface in scope here exposes no stack merge endpoint.

## 12. Reference commands

Read membership from one PR:

```bash
gh api repos/OWNER/REPO/pulls/42 --jq '.stack'
```

Find the stack containing a PR:

```bash
gh api "repos/OWNER/REPO/stacks?pull_request=42"
```

List all stacks with pagination:

```bash
gh api --paginate repos/OWNER/REPO/stacks
```

Get a complete stack:

```bash
gh api repos/OWNER/REPO/stacks/7
```

Create from an existing bottom-to-top PR chain:

```bash
printf '%s\n' '{"pull_requests":[101,102,103]}' |
  gh api --method POST repos/OWNER/REPO/stacks --input -
```

Append only the new top suffix:

```bash
printf '%s\n' '{"pull_requests":[104,105]}' |
  gh api --method POST repos/OWNER/REPO/stacks/7/add --input -
```

Unstack and inspect whether the result was partial (`200`) or dissolved
(`204`):

```bash
gh api --include --method POST repos/OWNER/REPO/stacks/7/unstack
```

## 13. Acceptance criteria for an integration

An implementation conforms to this specification when automated tests cover:

1. stacked and standalone PR membership decoding;
2. direct PR base versus ultimate stack base;
3. stack resource decoding with unknown fields;
4. bottom-to-top order preservation;
5. merged versus closed-unmerged detection;
6. multi-page stack listing and filtered lookup;
7. create boundaries of 2 and 100 request members;
8. append boundaries of 1 and 100 delta members;
9. valid and broken base/head chains;
10. create, exact no-op, and append-only-prefix reconciliation;
11. remote-extra, reordered, mixed-stack, and already-stacked-suffix conflicts;
12. create `201` and append `200` verification;
13. unstack partial `200` with retained members;
14. unstack dissolved `204` with an empty body;
15. ambiguous `404` classification;
16. validation `422` without blind retry;
17. transport failure after a write followed by read/reconciliation;
18. completed `open: false` stacks remaining discoverable;
19. safe handling of server-side SHA/base changes; and
20. refusal to perform unsupported insert, remove-one, reorder, replace,
    rebase, or merge operations through the Stacks REST API.

Live private-preview validation is still required for:

- exact authentication and fine-grained permission requirements;
- whether 100 is an aggregate stack limit or only a per-request limit;
- exact validation messages for duplicate membership and cross-stack adds;
- all states that cause unstack to retain a member;
- concurrency behavior when two clients append simultaneously;
- GitHub Enterprise Server availability; and
- whether any preview-specific headers become required later.

## 14. Source and implementation notes

The normative API shape comes from the
[REST reference](https://github.github.com/gh-stack/reference/rest-api/).
The lifecycle constraints come from the official
[working guide](https://github.github.com/gh-stack/guides/stacked-prs/) and
[FAQ](https://github.github.com/gh-stack/faq/).

For corroboration, the official `github/gh-stack` implementation at commit
[`6dcf9f0`](https://github.com/github/gh-stack/tree/6dcf9f050ae922aa0beea2027e5d456118d972b3)
contains typed wrappers for list, filtered lookup, get, create, append, and
unstack in
[`internal/github/github.go`](https://github.com/github/gh-stack/blob/6dcf9f050ae922aa0beea2027e5d456118d972b3/internal/github/github.go).
Its
[`link` command](https://github.com/github/gh-stack/blob/6dcf9f050ae922aa0beea2027e5d456118d972b3/cmd/link.go)
uses additive create/no-op/append reconciliation and rejects PRs it considers
ineligible. Its
[`unstack` command](https://github.com/github/gh-stack/blob/6dcf9f050ae922aa0beea2027e5d456118d972b3/cmd/unstack.go)
handles `200`, `204`, `404`, and `422`.

Those client choices are useful evidence but are not promoted to server
guarantees unless the published REST reference or product documentation also
states them.

## 15. Compatibility with the branchless-pr OpenSpec

The `openspec/specs/github-native-stacks` capability combines this REST
contract with branchless-pr product policy. The following traceability table is
normative for deciding whether that capability is compatible with this
document:

| OpenSpec requirement | REST coverage | Compatibility rule |
| --- | --- | --- |
| Native REST Representation | `MEM-001`–`MEM-005`, `RESOURCE-001`–`RESOURCE-005`, `CLIENT-017`–`CLIENT-019` | Wire types and validation must follow the published nested schemas, preserve order, tolerate additive fields, and fail closed on ambiguous required data. |
| Native REST Read Contract | `AVAIL-002`–`AVAIL-005`, `LIST-001`–`LIST-004`, `GET-001`–`GET-003`, `CLIENT-001`–`CLIENT-004` | Repository availability, PR membership, filtered lookup, pagination, and complete Stack reads must retain their distinct semantics. |
| Native REST Mutation Contract | `CREATE-001`–`CREATE-009`, `ADD-001`–`ADD-011`, `UNSTACK-001`–`UNSTACK-008`, `LIMIT-001`–`LIMIT-004` | Create, append, and unstack must use only the documented endpoints, request shapes, order, and result statuses. Unsupported structural operations remain unsupported. |
| Native Write Concurrency and Recovery | `LIFE-001`–`LIFE-002`, `ADD-008`–`ADD-010`, `CLIENT-007`–`CLIENT-016`, `ERROR-001`–`ERROR-003` | A client must verify successful results, protect server-side rewrites, reconcile uncertain writes through reads, and never retry a write blindly. |
| Native Stack Integration Mode | `AVAIL-001`–`AVAIL-005` | `off`, `auto`, and `required` are branchless-pr policy. They are compatible only when `404` feature detection is disambiguated through ordinary repository access and no extension-only contract is presented as a REST requirement. |
| Native Stack Eligibility | `MODEL-001`–`MODEL-009`, `CREATE-002`–`CREATE-009`, `ADD-002`–`ADD-011` | Same-repository, ref-chain, ordering, and conservative PR-state checks follow this document. A 100-member total cap is allowed only when labeled a branchless-pr client policy because GitHub documents per-request, not aggregate, maxima. |
| Native Stack Reconciliation Classification | `MEM-004`–`MEM-005`, `ADD-003`–`ADD-010`, `CLIENT-003`–`CLIENT-006` | Create, exact no-op, append-only prefix, and conflict states must compare complete ordered PR-number sequences. |
| Non-Destructive Native Reconciliation | `CREATE-003`–`CREATE-008`, `ADD-003`–`ADD-010`, `CLIENT-006`–`CLIENT-011`, `LIMIT-001`–`LIMIT-003` | Safe create and append are additive. A conflict must not trigger automatic unstack, replace, reorder, or append. |
| Native API Error Classification | `AVAIL-003`–`AVAIL-004`, section 10, `CLIENT-012`–`CLIENT-016` | Authentication, authorization, missing resource, validation, rate limit, transport, uncertain write, and schema failures must not be collapsed into feature unavailability. |
| Local Commit Metadata Remains Authoritative | `CLIENT-008`, `LIMIT-003` | Local Git discovery and commit metadata are outside the Stack resource. Keeping them authoritative does not contradict the REST contract. |
| Explicit Go Port Behavior | Sections 2 and 11 | Legacy-mode behavior is outside the REST surface. The opt-in Go-port extension is compatible when its native operations remain within the capabilities and limits documented here. |

Branchless-pr command requirements for configuration, dry-run output, receipts,
view rendering, landing refusal, and local cleanup are client concerns outside
the server schema. They remain compatible only when they do not weaken the
REST invariants above or claim an undocumented GitHub capability.

## Context

branchless-pr models a stack as the commits in `BASE..HEAD`, ordered bottom-to-top. Each commit receives a generated branch and one PR; `stack-info` in the commit message persists the PR URL and branch identity. PR bases already form the same chain required by GitHub native Stacked PRs, but GitHub currently sees them only as related ordinary PRs and branchless-pr maintains a duplicate table of contents in every PR body.

GitHub's private-preview Stacks REST API adds a repository-scoped Stack resource, while the versioned `github/gh-stack` extension provides supported non-interactive `link` and `unstack` workflows for external stack managers. Native membership can be created or extended additively, but neither surface exposes arbitrary replacement, removal, or reordering. Native membership changes rules, Actions triggering, review context, server-side cascading rebases, and landing behavior, so this cannot be enabled as an invisible no-op.

The integration crosses submit/export, dry-run, view, land, abandon, configuration, receipts, and documentation. `SPEC.md` sections 5, 6.2, 7, 13, 16, and 17 must be updated with the implementation because the Python source behavior predates native stacks.

## Goals / Non-Goals

**Goals:**

- Publish eligible branchless-pr PR chains as native GitHub stacks without changing the local commit-oriented workflow.
- Preserve offline reconstruction from commit metadata and all existing behavior when native integration is disabled.
- Make reconciliation deterministic and idempotent for unstacked, exact, and append-only states.
- Refuse ambiguous or destructive remote reconciliation.
- Preserve dry-run's no-write guarantee and provide useful plans and operation receipts.
- Prevent legacy landing and force-push behavior from corrupting matching native stacks.
- Keep every subprocess invocation behind `internal/shell` and continue using `gh` rather than a Go GitHub SDK.

**Non-Goals:**

- Import arbitrary GitHub stacks into the branchless-pr commit model.
- Support native stacks across forks, with more than 100 PRs, or with multiple unique commits per PR.
- Adopt `gh stack` local tracking or delegate branch management to it.
- Automatically dissolve and recreate a divergent native stack.
- Remove `stack-info` metadata or generated PR-body cross-links in the first release.
- Make native integration the default while the GitHub feature remains in private preview.
- Add a general-purpose stack restructuring or remote-to-local sync command.
- Merge native stacks from branchless-pr before GitHub exposes and documents a supported non-interactive landing contract.

## Decisions

1. **Keep the local commit stack authoritative.**

   `BASE..HEAD`, generated heads, and `stack-info` remain the source used to discover and operate on branchless-pr stacks. The GitHub Stack object is an opt-in remote projection that must agree with local PR order. This preserves offline operation and the one-commit-per-PR invariant.

   Alternative considered: replace commit metadata with GitHub membership. Rejected because commands would become online-only, rebases would lose stable commit-to-PR identity, and generic native stacks permit multiple commits per PR.

2. **Gate integration with `github.native_stacks = off|auto|required`.**

   `off` is the default and performs no native-stack API or extension calls. `auto` probes repository support and uses legacy behavior when the feature is unavailable. If the extension is missing, `auto` may fall back only when every PR is unstacked and skipping a prospective create leaves no existing native Stack inconsistent; exact no-op reads require no extension. Append and unstack operations against an existing native Stack fail rather than falling back without the required writer. `required` fails before command-specific mutation when support or a required extension write is unavailable or eligible PRs are not in the required state. Unknown values are configuration errors.

   Single-PR stacks are ordinary PRs in every mode because GitHub requires at least two PRs. Stacks over 100 entries and cross-repository/fork PR chains cannot be published; `auto` warns and remains legacy, while `required` fails before native reconciliation.

   Alternative considered: enable automatically whenever the API responds. Rejected because native membership changes CI, protection, review, and merge semantics.

3. **Use a hybrid REST-read and gh-stack-write adapter.**

   Add typed native Stack and membership structures and wrappers near the existing PR integration, using `git.RepoSlug` for repository identity and `internal/shell` for execution. Use `gh api` only for availability probing, PR membership reads, Stack reads, and post-write verification. Apply safe write plans through the extension's external-tool interface: `gh stack link <pr...>` for create, `gh stack link <stack-number> <suffix-pr...>` for append, and `gh stack unstack <stack-number>` for dissolution. Always pass PR numbers rather than branch names so the extension does not push branches or create PRs, and never pass `--open` so it does not alter draft state.

   Do not parse success text from the extension. Classify its documented exit codes, including exit code 9 for unavailable native stacks, then verify final membership through REST. A documented Stacks endpoint `404` is classified as unavailable only after ordinary repository access succeeds; authentication and repository lookup failures remain errors.

   Alternative considered: write directly through the private-preview REST create, append, and unstack endpoints. Rejected for the first integration because `gh stack link` is explicitly intended for external stack managers and insulates branchless-pr from preview write-protocol and header changes. Direct REST remains a future fallback if GitHub stabilizes those endpoints before the extension interface.

4. **Use a pure reconciliation classifier before writes.**

   The classifier consumes local PR numbers bottom-to-top plus fetched native membership and returns one of:

   - `ineligible`: fewer than two PRs, more than 100 PRs, or unsupported repository topology.
   - `create`: every local PR is unstacked.
   - `noop`: one native Stack contains exactly the local PR sequence.
   - `append`: the native sequence is an exact proper prefix of the local sequence and every suffix PR is unstacked.
   - `conflict`: any other state, including reordered membership, a remote suffix absent locally, mixed stacks, or a suffix PR already in another stack.

   Create and append writes occur only after final branch pushes, metadata amendment, PR title/body/base edits, and draft restoration. Conflict never triggers automatic unstacking. After every write, REST membership must equal the planned sequence before submit succeeds. This makes repeated submit idempotent and prevents unexpected loss of GitHub-side state.

5. **Preserve both submit engines and protect native branch updates with leases.**

   Native reconciliation is a shared final phase used by both current and optimized submit engines. When native mode is active and an observed remote head already exists, pushes use atomic force-with-lease expectations derived from the preflight state instead of an unconditional force push. A lease failure stops submit rather than overwriting a GitHub server-side cascade or another actor's update.

   Alternative considered: leave `git push -f` unchanged. Rejected because GitHub can now force-push stack branches during server-side rebases.

6. **Make dry-run plan native intent without performing writes.**

   Dry-run may perform the same read-only support and membership queries needed to classify existing PRs. It reports `disabled`, `ineligible`, `missing extension fallback`, `unavailable fallback`, `create`, `append`, `noop`, or `conflict`. New PR numbers do not exist during dry-run, so a stack containing new PRs is reported as a prospective create or append based on the known prefix. No `gh stack link`, `gh stack unstack`, or REST Stack write is invoked.

7. **Expose native state only when configured, with stable JSON fields.**

   In `off` mode, view behavior and network access remain unchanged. In `auto` or `required`, view reads membership for entries with PRs, validates order, and prints a concise native Stack summary or drift warning. JSON entries add nullable `github_stack_number`, `github_stack_position`, `github_stack_size`, and `github_stack_base` fields; they are `null` when an entry is not in a native stack or membership is unavailable.

   Alternative considered: replace the flat view schema with a nested Stack object. Rejected to keep the existing per-entry schema and filtering consumers simple.

8. **Block native landing until GitHub exposes a supported contract.**

   Before any landing mutation, native-enabled land loads membership. `auto` falls back to legacy landing only when native support is unavailable or every PR is unstacked; it does not fall back on drift. `required` rejects an eligible unstacked stack.

   When membership exactly matches a native Stack, `land` returns an actionable unsupported error before editing or merging anything. For `bottom-only`, it identifies the bottom PR URL; for `whole-stack`, it identifies the top PR URL. It explains that GitHub's current CLI extension does not support merging stacked PRs and that merge must be initiated in the GitHub UI.

   It does not automatically call `gh stack unstack` and then use legacy landing because dissolving shared remote membership is a destructive policy decision. Native landing and post-merge local synchronization should be proposed together after GitHub documents a supported non-interactive merge interface.

   Alternative considered: invoke `gh pr merge` directly. Rejected because gh-stack v0.0.8 documents stacked PR merging as unsupported from the CLI and exposes no merge command. An implementation spike may observe behavior, but undocumented behavior is not a stable product contract.

9. **Unstack before abandon removes remote branches.**

   Native-enabled abandon verifies exact membership before local commit amendments. It runs `gh stack unstack <stack-number>` before deleting generated remote branches, then verifies the result through REST, preventing a native Stack from retaining broken open members. A conflict stops abandon before mutation. In `auto`, unavailable or fully unstacked state uses legacy cleanup, but exact native membership with a missing extension fails before cleanup. `required` treats unavailable support or a missing extension as an error.

   Alternative considered: delete branches first and rely on GitHub cleanup. Rejected because GitHub preserves Stack identity independently of branch deletion and middle-stack closure can block PRs above it.

10. **Retain PR-body cross-links during the compatibility period.**

    Native Stack UI becomes the richer reviewer representation, but existing tables remain useful in notifications, APIs, repositories without the preview, and after unstacking. Removing or making them conditional can be a separate behavioral change after native support stabilizes.

11. **Treat GitHub-side rebases and merges as an explicit synchronization limitation.**

    The extension can synchronize server-updated branches only for stacks it tracks locally, and this integration deliberately does not adopt that local tracking model. Force-with-lease protects against remote changes after submit preflight, but cannot prove ownership of a rewrite that already existed when preflight began. Documentation must therefore warn that using GitHub's Rebase Stack or merge controls can leave the branchless-pr local commit stack stale. Native `land` remains blocked, and a future native sync design must solve remote-to-local reconciliation before branchless-pr automates those operations.

    Alternative considered: temporarily adopt generated branches into gh-stack local tracking and run `gh stack sync`. Rejected because it introduces a second local source of truth and conflicts with branchless-pr deleting generated local branches after submit.

## Risks / Trade-offs

- **Private-preview API changes or disappears** -> Keep the feature off by default, isolate typed API wrappers, distinguish `auto` fallback from `required` failure, and test response parsing with fixtures.
- **Native membership multiplies CI usage** -> Require explicit configuration and document that Actions and protection are evaluated against the ultimate Stack base.
- **Dual local and remote representations drift** -> Validate every PR sequence before writes and never auto-recreate divergent stacks.
- **Server-side rebases race submit pushes** -> Use observed force-with-lease expectations whenever native mode is active.
- **The optional extension is missing or changes behavior** -> Detect a supported extension version before native writes, use documented exit codes rather than parsing status text, verify writes through REST, and preserve auto fallback.
- **Users rebase or merge through GitHub and local commits become stale** -> Document the limitation, preserve force-with-lease protection, block native `land`, and design synchronization before automating native merge flows.
- **`auto` masks an authentication error as unsupported** -> Classify only the documented Stacks-endpoint `404` as unavailable after ordinary repository access is confirmed.
- **Unstack cannot remove merged, merging, or queued PRs** -> Validate the returned Stack resource and stop before branch deletion if unmerged local members remain unexpectedly stacked.

## Migration Plan

1. Add config parsing, API models/wrappers, availability probing, and the pure reconciliation classifier behind the default-off gate.
2. Add submit/export reconciliation, leased native pushes, dry-run planning, receipts, and view metadata.
3. Validate link create/append/no-op, Actions, branch protection, exit code 9, and unstack behavior in a preview-enabled test repository.
4. Add native landing refusal and native abandon only after membership verification is reliable.
5. Update `SPEC.md`, README/help, generated config, agent guidance, and `CHANGELOG.md` with the shipped opt-in behavior.
6. Rollback is configuration-only: set `github.native_stacks = off`. Existing native Stack objects remain on GitHub until explicitly unstacked; branchless-pr continues to operate from commit metadata using legacy behavior.

## Open Questions

- What minimum gh-stack extension version should native writes require after testing against the first supported release? Keep version checking isolated so the minimum can advance independently.
- Should native synchronization and landing become one follow-up change, or should synchronization ship first so GitHub UI rebases and merges can be recovered safely before CLI landing is added?

## Port Compatibility

The Python `stack-pr` algorithms know only ordinary PRs and generated branch chains. This design preserves those algorithms under the default `off` mode and introduces explicit Go-port behavior only when native mode is selected. `SPEC.md` will document both paths so the Go implementation does not silently diverge from its behavioral source of truth.

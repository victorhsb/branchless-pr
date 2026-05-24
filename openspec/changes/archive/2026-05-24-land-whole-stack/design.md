## Context

The current `land` command (SPEC §15) implements a `bottom-only` strategy: squash-merge the bottom PR, then rebase and push every remaining branch onto the new target. For a stack of N PRs this requires N invocations, each performing a squash-merge, a fetch per remaining entry, a checkout, rebase, and force-push per remaining entry. The total network calls scale as O(N) per invocation, O(N²) across a full stack landing.

The `whole-stack` strategy eliminates this by performing a single GitHub rebase merge of the tip PR directly into the target branch. All commits land linearly in one operation, removing the need for per-entry fetch/checkout/rebase/push cycles.

## Goals / Non-Goals

**Goals:**

- Add a `whole-stack` land style that atomically lands all PRs in a stack by rebase-merging the tip PR into the target branch.
- Check that the repository allows rebase merges before attempting the operation; error out with a clear message if not.
- Retarget the tip PR base to the target branch before merging.
- Clean up local state after the merge: restore original branch, delete local stack branches, rebase onto new target.
- Support both CLI flag (`--whole-stack`) and config (`land.style = whole-stack`) for selecting the strategy.
- Preserve the existing `bottom-only` behavior unchanged.

**Non-Goals:**

- Do not add a Go GitHub SDK dependency.
- Do not bypass `internal/shell` for subprocess calls.
- Do not handle intermediate PR closure in this change; GitHub's auto-detection or manual closure is sufficient for now.
- Do not add commit message annotation (e.g. appending `(#PR_N)`) for rebase-merged commits; stack metadata already includes PR references in commit messages.
- Do not validate that all intermediate PRs are approved or have passing CI before merging; that responsibility remains with the user.
- Do not parallelize any operations within `whole-stack` landing.

## Decisions

0. **`whole-stack` is a new `land.style` value alongside `bottom-only` and `disable`.**
   - Config: `land.style = whole-stack` makes `bpr land` default to the whole-stack strategy.
   - CLI: `bpr land --whole-stack` overrides the configured style for a single invocation.
   - The `--whole-stack` flag is only available when `land.style` is not `disable` (i.e. the `land` subcommand is registered).
   - When both config and flag specify a style, the flag takes precedence. If config says `bottom-only` and the user passes `--whole-stack`, the whole-stack path runs.
   - Alternative considered: a separate `land-all` subcommand. Rejected because it fragments the CLI surface and the behavior is a style variant, not a fundamentally different command.

1. **Dispatch happens inside `landImpl`, not via separate subcommands.**
   - `landImpl` reads the effective style (config + flag override) and branches to either the existing bottom-only logic or the new `landWholeStack` function.
   - This keeps `land.go` as the single entry point and avoids duplicating `WithRecovery` wrapping, pre-flight checks, etc.
   - Alternative considered: separate `landWholeStackCmd()`. Rejected because both styles share the same pre-flight (discover stack, read metadata, assign bases, verify) — only the merge and post-merge steps differ.

2. **Query repository merge settings via `gh api graphql`.**
   - Add `pr.RebaseMergeAllowed(owner, repo string) (bool, error)` to `internal/pr` that runs a single GraphQL query for `repository.rebaseMergeAllowed`.
   - The owner and repo are derived from the `gh` context (the authenticated user's default repo or from `git remote get-url`). A helper `git.RepoSlug(remote string) (owner, repo string, error)` extracts the `owner/repo` pair from the remote URL.
   - If `rebaseMergeAllowed` is false, the command prints a clear error: `"ERROR: Repository does not allow rebase merges. Enable rebase merges in repository settings or use land.style = bottom-only."` and exits without mutating state.
   - Alternative considered: try the merge and handle the rejection. Rejected because a pre-flight check is faster (no mutation attempt) and gives a clearer error message.

3. **The whole-stack flow:**
   ```
   1. maybeRebaseBase            (same as bottom-only)
   2. stack.Discover             (same)
   3. ReadMetadata × N           (same)
   4. AssignBases + PrintStack   (same)
   5. Verify(st, checkBase=true) (same)
   6. pr.RebaseMergeAllowed()    ← NEW: 1 GraphQL call
   7. git fetch --prune          (same, needed before merge)
   8. pr.EditBase(tip.PR, target) ← retarget tip to target
   9. pr.MergeRebase(tip.PR)    ← NEW: gh pr merge <tip> --rebase
   10. git fetch --prune         (same, needed after merge)
   11. Checkout original branch
   12. Delete local stack branches
   13. Rebase local target + original branch onto REMOTE/TARGET
   ```
   - Steps 1-5 are identical to bottom-only because the pre-flight verification is the same: all PRs must be OPEN with correct metadata.
   - Step 8 retargets the tip PR base to the target branch so GitHub shows the correct merge target.
   - Step 9 performs the rebase merge. This is the only network-heavy operation (waits for GitHub merge machinery).
   - No per-entry checkout/rebase/push cycle is needed because all commits are already on the tip branch and GitHub's rebase merge applies them linearly to main.
   - Steps 10-13 mirror the cleanup in bottom-only but without the intermediate rebase+push loop.

4. **Add `pr.MergeRebase` to `internal/pr`.**
   - Signature: `MergeRebase(prRef string) error`
   - Implementation: `gh pr merge <prRef> --rebase`
   - This mirrors the existing `MergeSquash` pattern.

5. **Add `pr.RebaseMergeAllowed` to `internal/pr`.**
   - Signature: `RebaseMergeAllowed(owner, repo string) (bool, error)`
   - Implementation: single `gh api graphql` call querying `repository(owner, name) { rebaseMergeAllowed }`
   - Returns `(false, nil)` when the setting is explicitly false. Returns `(false, err)` on API failure (network error, auth failure, etc.) — the caller should propagate the error rather than silently falling back.

6. **Add `git.RepoSlug` to `internal/git`.**
   - Signature: `RepoSlug(remote string) (owner, repo string, error)`
   - Implementation: `git remote get-url <remote>` parsed to extract `owner/repo` from the URL.
   - Supports both HTTPS (`https://github.com/owner/repo.git`) and SSH (`git@github.com:owner/repo.git`) forms.

7. **Land command remains registered for both `bottom-only` and `whole-stack` styles.**
   - The current `root.go` check `landStyle != "disable"` already covers `whole-stack` since `whole-stack != disable`.
   - No change needed to the registration gate.

## Risks / Trade-offs

- **Rebase merge not allowed** → Pre-flight GraphQL check prevents mutation. Clear error message guides the user.
- **GitHub auto-close of intermediate PRs is unreliable** → Accepted for now. Intermediate PRs may remain open after the tip merges. Users can manually close them or rely on GitHub's eventual detection. A follow-up change could add explicit `gh pr close` calls.
- **Commit messages lack `(#PR_N)` annotation** → Stack metadata in commit messages already contains PR URL references. The squash-merge annotation pattern from `bottom-only` does not apply to rebase merges. Accepted as-is.
- **Branch protection may require checks on each PR individually** → Accepted. The whole-stack strategy intentionally merges through a single PR. Users who need per-PR CI gates should use `bottom-only`.
- **GraphQL query for merge settings requires `owner/repo`** → Need a reliable way to extract this from the git remote URL. The `git.RepoSlug` helper handles both HTTPS and SSH URL forms.

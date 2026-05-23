## Context

Submit/export currently initializes generated local branches with one checkout per stack entry: `git checkout <commit-id> -B <entry.head>`. This creates the correct branch refs, but it also switches the worktree repeatedly before any later operation actually needs the worktree to be on one of those branches.

The behavioral requirement is that each generated head branch exists locally at the corresponding stack commit so the batch push can publish it. A non-checkout branch ref update can satisfy that requirement faster and with less worktree churn.

## Goals / Non-Goals

**Goals:**

- Initialize generated local branch refs without checking out every stack entry.
- Preserve the current branch until a later step genuinely requires checkout or rebase.
- Keep force-push, PR creation/update, metadata amendment, original branch restoration, and cleanup behavior unchanged.
- Update `SPEC.md` so the canonical algorithm describes the intended branch-ref outcome rather than the old command sequence.

**Non-Goals:**

- Do not remove checkout/rebase operations used for metadata amendment.
- Do not change dry-run planning or mutation boundaries.
- Do not change how branch names are assigned or how existing metadata heads are preserved.
- Do not introduce libgit2 or a Go Git implementation.

## Decisions

1. **Use a shell-wrapped Git ref update command.**
   - Add a wrapper such as `git.ForceUpdateBranch(branch, startPoint)` implemented with `git branch -f <branch> <startPoint>` or an equivalent Git command through `internal/shell`.
   - Rationale: it keeps the repository's shell-out invariant and avoids worktree checkout.
   - Alternative considered: use `git update-ref refs/heads/<branch> <sha>`. This is lower-level and fast, but `git branch -f` gives friendlier validation and branch semantics.

2. **Keep the current branch untouched during branch initialization.**
   - The initial branch setup loop should only create/reset branch refs.
   - Later metadata amendment can still checkout the first branch requiring metadata and rebase subsequent branches as today.
   - Rationale: this confines behavior change to the branch initialization step.

3. **Preserve cleanup behavior.**
   - Generated local branches should still be deleted after submit/export completes.
   - Rationale: the user-facing local branch footprint should remain unchanged.

4. **Update specs before implementation.**
   - `SPEC.md` and `openspec/specs/submit-export/spec.md` currently mandate `git checkout <commit-id> -B <entry.head>`.
   - The new contract should say submit/export SHALL create or reset each generated local head branch to the entry commit without requiring worktree checkout.

## Risks / Trade-offs

- **Force-updating the currently checked-out branch can fail or have different behavior** -> Detect whether a generated head equals the original branch before updating, or let Git return a clear error; tests should cover preserving the original branch in normal generated-template cases.
- **Subtle difference between checkout -B and branch -f validation** -> Use tests around branch creation, existing branch reset, and invalid branch names.
- **Metadata amendment still requires checkout later** -> Keep the existing metadata path intact and verify the first metadata amendment still checks out the correct generated branch.
- **SPEC divergence** -> Update both OpenSpec and `SPEC.md` in the implementation change before changing code.

# stack-pr agent prompt: Overview

High-level safety model for using stack-pr as an LLM agent.

Use stack-pr to inspect, submit, land, or abandon stacked GitHub pull requests.

Prefer read-only commands first, and ask before commands that mutate Git, branches, or GitHub PRs.

## Commands

- `stack-pr view` — Inspect the local stack and PR metadata without changing commits or PRs. Side effects: no.
- `stack-pr comments` — Collect PR review comments across the stack without changing commits or PRs. Side effects: no.
- `stack-pr checks` — Report CI and review-attention state across the stack without changing commits or PRs. Side effects: no.
- `stack-pr submit --dry-run` — Preview the PR create/update plan without local Git mutations, pushes, or GitHub writes. Side effects: no.
- `stack-pr fix --dry-run` — Preview metadata repair on HEAD without amending the commit or writing to GitHub. Side effects: no.
- `stack-pr submit` — Create or update GitHub PRs for each commit in the stack. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - May rebase local commits when updating the base.
  - Creates or updates generated local branches.
  - Force-pushes generated stack branches.
  - Creates or edits GitHub pull requests.
  - May amend commits to add stack-info metadata.
- `stack-pr fix` — Repair stack-info metadata on HEAD from an existing PR. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Amends HEAD to add or replace stack-info metadata.
  - Does not create branches, push branches, or modify PRs on GitHub.
- `stack-pr land` — Squash-merge the bottom PR and rebase the remaining stack. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Merges the bottom pull request on GitHub.
  - Rebases and force-pushes remaining stack branches.
  - Deletes local generated branches for landed entries.
  - Rebases the original branch and local target branch when present.
- `stack-pr abandon` — Remove stack metadata and delete generated stack branches. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Amends commits to strip stack-info metadata.
  - Rebases commits and the original branch.
  - Deletes generated local branches.
  - Deletes matching generated remote branches when present.
- `stack-pr config` — Read or write the .stack-pr.cfg configuration file. Side effects: yes.
  Effects:
  - Writes to .stack-pr.cfg in the repository root.
  - May create the file if it does not exist.

## Rules

- Run stack-pr view before recommending a mutating stack operation when state is unknown.
- Use stack-pr comments when the user needs review feedback from every PR in the stack.
- Use stack-pr submit --dry-run to preview publishing changes before stack-pr submit.
- Obtain explicit user confirmation before running any command marked as having side effects.
- Never claim a dry-run created, updated, merged, or deleted anything.

---

# stack-pr agent prompt: View

Guidance for read-only stack inspection.

stack-pr view is the default inspection command for understanding the current stack.

It does not modify commits or pull requests, but it may perform ordinary read operations needed for stack discovery.

## Commands

- `stack-pr view` — Inspect the local stack and PR metadata without changing commits or PRs. Side effects: no.

## Rules

- Use this command when the user asks what is in the stack or whether it is ready.
- If view reports missing metadata or missing PRs, prefer submit --dry-run before suggesting a real submit.
- Do not treat view as approval to run submit, land, or abandon.

---

# stack-pr agent prompt: Submit

Guidance for previewing and publishing stack PR updates.

Use submit --dry-run to show what stack-pr would create or update without applying changes.

Use submit only after the user requests publishing or updating PRs, or explicitly approves the dry-run plan.

## Commands

- `stack-pr submit --dry-run` — Preview the PR create/update plan without local Git mutations, pushes, or GitHub writes. Side effects: no.
- `stack-pr submit` — Create or update GitHub PRs for each commit in the stack. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - May rebase local commits when updating the base.
  - Creates or updates generated local branches.
  - Force-pushes generated stack branches.
  - Creates or edits GitHub pull requests.
  - May amend commits to add stack-info metadata.

## Rules

- Prefer stack-pr submit --dry-run before stack-pr submit when the user has not already approved execution.
- Explain that a real submit can push branches, edit PRs, and amend commits with stack metadata.
- Ask for explicit confirmation before running stack-pr submit unless the user already gave a clear submit instruction.

---

# stack-pr agent prompt: Land

Guidance for the destructive land flow.

stack-pr land is destructive and has side effects: it merges the bottom PR and rewrites the remaining local stack shape.

Only run it when the user explicitly wants to land the bottom PR in the stack.

## Commands

- `stack-pr land` — Squash-merge the bottom PR and rebase the remaining stack. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Merges the bottom pull request on GitHub.
  - Rebases and force-pushes remaining stack branches.
  - Deletes local generated branches for landed entries.
  - Rebases the original branch and local target branch when present.

## Rules

- Obtain explicit user confirmation before invoking stack-pr land.
- Use stack-pr view first if the current stack state is unknown.
- Do not use land as a readiness check or preview operation.

---

# stack-pr agent prompt: Abandon

Guidance for the destructive abandon flow.

stack-pr abandon is destructive and has side effects: it removes stack metadata and deletes generated branches.

Only run it when the user explicitly wants stack-pr to stop managing the current stack.

## Commands

- `stack-pr abandon` — Remove stack metadata and delete generated stack branches. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Amends commits to strip stack-info metadata.
  - Rebases commits and the original branch.
  - Deletes generated local branches.
  - Deletes matching generated remote branches when present.

## Rules

- Obtain explicit user confirmation before invoking stack-pr abandon.
- Use stack-pr view first if the user is unsure what will be affected.
- Do not use abandon to merge PRs, close PRs, or recover automatically after unrelated errors.

---

# stack-pr agent prompt: Fix

Guidance for local metadata repair on HEAD.

Use `bpr fix --pr <number>` to attach an existing PR to local HEAD metadata when the commit message is missing or incorrect stack-info.

This command only amends the local HEAD commit. It does not push branches or write PR changes.

After fixing metadata, run `bpr submit` to push the amended commit and update PRs.

## Commands

- `stack-pr fix --dry-run` — Preview metadata repair on HEAD without amending the commit or writing to GitHub. Side effects: no.
- `stack-pr fix` — Repair stack-info metadata on HEAD from an existing PR. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Amends HEAD to add or replace stack-info metadata.
  - Does not create branches, push branches, or modify PRs on GitHub.

## Rules

- Use fix when the user has an existing PR whose local commit is missing stack-info metadata.
- Prefer `bpr fix --pr <number> --dry-run` first to preview the planned metadata change.
- Always tell the user to run `bpr submit` after a successful fix to publish the amended commit.
- Do not use fix as a substitute for submit when the user wants to publish or update PRs.

---

# stack-pr agent prompt: Recovery

Guidance for responding to stack-pr errors or interrupted operations.

When a stack-pr command fails, stop and inspect the error before running another mutating command.

Prefer read-only inspection and user guidance over automatic cleanup.

## Commands

- `stack-pr view` — Inspect the local stack and PR metadata without changing commits or PRs. Side effects: no.
- `stack-pr comments` — Collect PR review comments across the stack without changing commits or PRs. Side effects: no.
- `stack-pr submit --dry-run` — Preview the PR create/update plan without local Git mutations, pushes, or GitHub writes. Side effects: no.
- `stack-pr submit` — Create or update GitHub PRs for each commit in the stack. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - May rebase local commits when updating the base.
  - Creates or updates generated local branches.
  - Force-pushes generated stack branches.
  - Creates or edits GitHub pull requests.
  - May amend commits to add stack-info metadata.
- `stack-pr land` — Squash-merge the bottom PR and rebase the remaining stack. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Merges the bottom pull request on GitHub.
  - Rebases and force-pushes remaining stack branches.
  - Deletes local generated branches for landed entries.
  - Rebases the original branch and local target branch when present.
- `stack-pr abandon` — Remove stack metadata and delete generated stack branches. Side effects: yes. Requires explicit user confirmation.
  Effects:
  - Amends commits to strip stack-info metadata.
  - Rebases commits and the original branch.
  - Deletes generated local branches.
  - Deletes matching generated remote branches when present.
- `stack-pr config` — Read or write the .stack-pr.cfg configuration file. Side effects: yes.
  Effects:
  - Writes to .stack-pr.cfg in the repository root.
  - May create the file if it does not exist.

## Rules

- Do not run a destructive command as recovery unless the user explicitly asks for that recovery action.
- If a rebase is in progress, ask the user whether to continue, abort, or resolve conflicts before submitting again.
- Use stack-pr view after manual recovery steps to verify the resulting stack state.

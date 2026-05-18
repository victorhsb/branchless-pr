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

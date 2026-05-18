# stack-pr agent prompt: Recovery

Guidance for responding to stack-pr errors or interrupted operations.

When a stack-pr command fails, stop and inspect the error before running another mutating command.

Prefer read-only inspection and user guidance over automatic cleanup.

## Commands

- `stack-pr view` — Inspect the local stack and PR metadata without changing commits or PRs. Side effects: no.
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

## Rules

- Do not run a destructive command as recovery unless the user explicitly asks for that recovery action.
- If a rebase is in progress, ask the user whether to continue, abort, or resolve conflicts before submitting again.
- Use stack-pr view after manual recovery steps to verify the resulting stack state.

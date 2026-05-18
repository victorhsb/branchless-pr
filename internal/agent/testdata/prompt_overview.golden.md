# stack-pr agent prompt: Overview

High-level safety model for using stack-pr as an LLM agent.

Use stack-pr to inspect, submit, land, or abandon stacked GitHub pull requests.

Prefer read-only commands first, and ask before commands that mutate Git, branches, or GitHub PRs.

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

- Run stack-pr view before recommending a mutating stack operation when state is unknown.
- Use stack-pr submit --dry-run to preview publishing changes before stack-pr submit.
- Obtain explicit user confirmation before running any command marked as having side effects.
- Never claim a dry-run created, updated, merged, or deleted anything.

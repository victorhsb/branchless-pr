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

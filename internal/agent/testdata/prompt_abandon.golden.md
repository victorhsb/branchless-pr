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

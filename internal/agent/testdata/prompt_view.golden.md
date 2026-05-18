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

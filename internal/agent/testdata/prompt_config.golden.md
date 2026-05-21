# stack-pr agent prompt: Config

Guidance for managing stack-pr configuration.

stack-pr config is used to read or write the local .stack-pr.cfg file.

Configuration changes are local-only and do not affect remote repositories or pull requests.

## Commands

- `stack-pr config` — Read or write the .stack-pr.cfg configuration file. Side effects: yes.
  Effects:
  - Writes to .stack-pr.cfg in the repository root.
  - May create the file if it does not exist.

## Rules

- Only modify configuration when the user explicitly requests a change.
- Ensure the working directory is at the repository root before writing configuration.
- Do not modify configuration as part of normal stack operations.

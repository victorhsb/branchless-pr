---
title: view --json
status: stable
---

# View JSON output

## Overview

`stack-pr view --format json` produces machine-readable JSON for inspecting a stack, instead of the default ANSI-colored text.

## Behavior

### Format selection

- `stack-pr view` without `--format` → output uses the existing ANSI-colored, hyperlink-enabled text format.
- `stack-pr view --format json` → output is a JSON array ordered newest-to-oldest; each array element is a flat object with fields: `commit`, `short_sha`, `title`, `author`, `author_name`, `author_email`, `pr_url`, `pr_number`, `head_branch`, `base_branch`, `github_stack_number`, `github_stack_position`, `github_stack_size`, `github_stack_base`; output contains no ANSI escape sequences or terminal hyperlinks.
- Stack entry has no associated PR → `pr_url` is `""` and `pr_number` is `0`.
- `--format` with a value other than `text` or `json` → exit with an error returning a clear message.

### Native stack fields

- Native integration enabled and entry belongs to a GitHub Stack → `github_stack_number` is the repository-scoped Stack number, `github_stack_position` is the 1-based bottom-to-top position, `github_stack_size` is the Stack size, and `github_stack_base` is the ultimate Stack base ref.
- Native integration enabled and entry's PR is unstacked or membership is unavailable in auto mode → every `github_stack_*` field is `null`.
- `github.native_stacks = off` → every `github_stack_*` field is `null`; no native membership query occurs.

---
title: config init
status: stable
---

# Config init

## Overview

`stack-pr config init` scaffolds a new `.stack-pr.cfg` file with documented defaults, without overwriting an existing file.

## Behavior

### File generation

- Run inside a repository that has no `.stack-pr.cfg` → create `<repo-root>/.stack-pr.cfg` containing all default sections and keys, each with a descriptive comment.
- Run inside a repository that already has `.stack-pr.cfg` → exit non-zero and print an error indicating the file already exists.

### Defaults parity

The generated configuration contains, at minimum, the same keys and values as the built-in `config.Defaults()` map, organised into sections `[common]`, `[repo]`, `[github]`, `[land]`, and `[comments]`.

- Successful generation → parsing the generated file with `config.Load` and merging with `config.Defaults()` produces no new keys in either direction.
- Successful generation → the `[github]` section contains `native_stacks = off`; inline comments document the `off`, `auto`, and `required` values, note that enabling native stacks changes GitHub CI, rules, review, and landing behavior, and state that native Stack operations use the REST API through the base `gh` CLI; the comments do not require the `github/gh-stack` extension.

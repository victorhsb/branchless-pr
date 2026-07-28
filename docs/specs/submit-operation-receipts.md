---
title: Submit operation receipts
status: stable
---

# Submit operation receipts

## Overview

Provide an opt-in machine-readable receipt for real submit/export executions so callers can inspect completed side effects, failures, and recovery attempts. Each receipt is a single JSON object with a stable, versioned schema.

## Behavior

### Receipt request

The `submit` command and its `export` alias support an opt-in receipt destination for real submit/export executions.

- `stack-pr submit --receipt <destination>` is invoked without `--dry-run` → attempt to emit a submit operation receipt to `<destination>`.
- `stack-pr export --receipt <destination>` is invoked without `--dry-run` → emit the same submit operation receipt as `stack-pr submit --receipt <destination>`.
- `stack-pr submit` is invoked without a receipt flag and without receipt configuration → emit no receipt; existing human output behavior remains unchanged.

| Receipt destination | Behavior |
|---------------------|----------|
| `off` | disable receipt emission |
| `-` | emit one JSON receipt document on standard output |
| any other value | interpret as a filesystem path where the receipt JSON document is written |

- `stack-pr submit --dry-run --receipt <destination>` is invoked with a destination other than `off` → report a clear invocation error explaining that operation receipts are only available for real submit/export executions, and perform no submit/export mutations.

### Receipt configuration

`.stack-pr.cfg` configures default submit/export receipt behavior.

- `.stack-pr.cfg` contains `receipt.submit = <destination>` and `stack-pr submit` is invoked without `--receipt` → use `<destination>` as the effective receipt destination.
- `.stack-pr.cfg` contains `receipt.submit = <destination>` and `stack-pr export` is invoked without `--receipt` → use `<destination>` as the effective receipt destination.
- `.stack-pr.cfg` contains `receipt.submit = <configured-destination>` and `stack-pr submit --receipt <flag-destination>` is invoked → use `<flag-destination>` as the effective receipt destination.
- `.stack-pr.cfg` omits `receipt.submit` → the effective receipt destination is `off`.

### Receipt JSON envelope

- A submit operation receipt is emitted → the JSON object includes `schema_version`, `command`, `status`, `side_effects`, `repo`, `stack`, and `operations`.
- `schema_version` is a non-empty string; `command` identifies the invoked operation as `stack-pr submit` or `stack-pr export`; `side_effects` is `true`.
- `status` is one of `ok`, `failed`, or `partial_failure`.
- A submit operation receipt is emitted → `repo` includes the resolved repository root, original branch, remote, target, base, head, and branch-name template when those values are available.
- A submit operation receipt is emitted after stack discovery succeeds → `stack` includes the stack size and per-entry commit SHA, title, head branch, base branch, and PR URL when known.
- The effective receipt destination is `-` → standard output contains exactly one valid JSON receipt document; human progress output is not interleaved into standard output.

### Receipt operation entries

The receipt records high-value submit/export side effects in execution order.

- Submit/export successfully completes a side-effecting operation → append an operation entry with `type`, `status`, and operation-specific details; `status` is `ok`.
- Submit/export fails during a side-effecting operation after receipt collection begins → append or update an operation entry for the failed operation with `status` set to `failed` and an error message.
- A receipt contains at least one successful side-effect operation followed by a failed operation → the top-level receipt `status` is `partial_failure`.
- Submit/export fails before any side-effect operation succeeds and a receipt can be emitted → the top-level receipt `status` is `failed`.
- Submit/export completes successfully → the top-level receipt `status` is `ok`.

### Submit operation coverage

Receipts record the main categories of submit/export side effects.

- Submit/export creates or checks out generated stack branches → record branch operation entries identifying the affected branch names and commits when available.
- Submit/export force-pushes generated stack branches → record push operation entries identifying the remote and branch names.
- Submit/export creates or updates a pull request → record pull request operation entries identifying the commit, head branch, base branch, title, and PR URL when available.
- Submit/export amends commits to add `stack-info` metadata → record metadata operation entries identifying the affected head branch and commit when available.
- Submit/export performs a best-effort cleanup operation that fails without failing the command → record a warning operation entry identifying the cleanup operation and error message.

### Recovery recording

Receipts record best-effort recovery attempts made after handled errors.

- Submit/export fails and recovery attempts to checkout the original branch → record a recovery operation entry with the target original branch and success or failure status.
- Submit/export fails after an auto-stash was created and recovery attempts to pop the stash → record a recovery operation entry with success or failure status.

### Receipt emission failure

Receipt emission failures are visible to callers.

- The effective receipt destination is a filesystem path and the receipt JSON document cannot be written to that path → return a non-zero error explaining that receipt emission failed.
- The effective receipt destination is `off` → do not attempt to write a receipt.

### Native Stack receipt operations

When native integration is enabled, receipts record the outcome of native Stack planning and reconciliation. All rows below assume receipt emission is enabled.

| Native reconciliation outcome | Receipt content |
|-------------------------------|-----------------|
| Submit/export creates a GitHub native Stack | `ok` operation containing the action `create`, Stack number, and ordered PR numbers |
| Submit/export appends PRs to a GitHub native Stack | `ok` operation containing the action `append`, Stack number, and appended PR numbers |
| Native reconciliation is a no-op | `ok` operation containing the action `noop` and Stack number |

- Auto mode falls back because native Stacks is unavailable or the stack is ineligible and submit/export completes through the legacy path → record the fallback reason.
- Native classification or mutation fails after earlier submit operations succeeded → include a failed native Stack operation, and the top-level receipt status is `partial_failure`.

---
kind: adr
id: ADR-0008
title: Constrain shell execution to Git and GitHub boundaries
status: accepted
date: 2026-09-04
tags: architecture, git, github, subprocess
supersedes:
superseded_by:
deprecated_by:
---
# ADR-0008: Constrain shell execution to Git and GitHub boundaries

## Status

accepted

## Context

ADR-0002 requires `internal/shell` to own process spawning, but its generic runner does not define which packages may construct commands or invoke it. Git and GitHub CLI calls had spread across `internal/git`, `internal/pr`, `internal/nativestacks`, and `internal/stack`. That spread made side effects harder to audit and led to package-specific test seams that replaced binaries through `PATH`.

## Decision

Constrain normal production shell execution to two typed boundaries. `internal/git.Repo` constructs and executes `git` commands. `internal/pr.Client` constructs and executes `gh` commands. No other normal production package may invoke `shell.Runner` or construct subprocess argv.

Composition-root packages may import `internal/shell` to create and inject a runner, but they must not invoke it. Domain packages consume `git.Repo`, `pr.Client`, or narrow interfaces backed by those types.

`internal/diagnose` remains a temporary exception. Its best-effort checks map raw command failures into diagnostic statuses, so moving them behind the typed boundaries requires a separate design change. New subprocess calls must not extend this exception.

## Consequences

The composition root creates one `git.Repo` and one `pr.Client` with the invocation's `shell.Runner`. Tests provide `shelltest.Fake` through the same constructors instead of replacing binaries through `PATH` or adding package-specific callbacks.

`internal/nativestacks` owns Stack domain behavior and depends on a narrow transport interface implemented by `pr.Client`. GitHub API response and transport error types live in `internal/pr`, which keeps command execution and status parsing at the GitHub boundary.

Adding a Git or GitHub operation now requires a `git.Repo` or `pr.Client` method, or an interface backed by one. Production shell execution of commands other than `git` and `gh` requires revisiting this decision.

The remaining direct `git` and `gh` calls in `internal/diagnose` must move through these boundaries when its diagnostic result mapping is redesigned.

## History: Accepted

Date: 2026-09-04

The completed refactor establishes the production Git and GitHub subprocess boundaries.

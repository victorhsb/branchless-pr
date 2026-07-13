# Engineering TODO

This file tracks identified engineering improvements. It is a non-binding
backlog: behavioral changes still require an OpenSpec proposal and corresponding
`SPEC.md` updates where applicable.

## P0 - Correctness and Recovery

- [x] Make Git operation detection repository-layout aware.
  - Resolve rebase, merge, cherry-pick, and sequencer paths through Git instead
    of assuming a `.git` directory exists in the current working directory.
  - Support invocation from repository subdirectories, linked worktrees,
    submodules, and repositories using a separate Git directory.
  - Add focused tests for each supported repository layout and operation state.

- [x] Restore an automatic stash when pre-run initialization fails.
  - Cover failures after stashing but before command execution, including clean
    checks, target validation, and merge-base resolution.
  - Manage stash cleanup across the complete post-stash invocation lifecycle,
    not only inside the command mutation recovery wrapper.
  - Add tests proving both successful and failed invocations restore the
    original working tree.

- [x] Track and restore the exact automatic stash created by the command.
  - Stop detecting stash creation by matching localized Git output.
  - Preserve a stable stash object or reference and restore only that stash,
    without popping a pre-existing user stash.
  - Add tests for clean trees, existing user stashes, localized or unexpected
    command output, successful restoration, and restoration conflicts.

## P1 - Safety and Diagnostics

- [ ] Preserve Git diagnostics and honor verbose subprocess output.
  - Include command, exit status, and captured stderr in errors returned from
    the shell and Git layers.
  - Explicitly connect subprocess stdout and stderr to the parent streams in
    non-quiet mode so `--verbose` behaves as documented.
  - Clarify or simplify `RunOpts.Check` semantics and align its documentation
    and tests with the implementation.
  - Add tests for quiet and non-quiet output, stderr propagation, and wrapped
    exit errors.

- [ ] Harden remote branch force-push and deletion operations.
  - Use fully qualified `refs/heads/<branch>` source and destination refspecs to
    avoid ambiguity with tags or other refs.
  - Reject empty branch lists so a helper cannot accidentally invoke a default
    push operation.
  - Evaluate fetched-state `--force-with-lease` protection in place of
    unconditional `-f`, including stale-remote behavior and recovery guidance.
  - Create an OpenSpec change and update `SPEC.md` before changing force-push
    semantics, because the current specification requires `-f`.

## P2 - Contract and Maintainability

- [ ] Reconcile changed-path helpers with their documented contract.
  - Decide whether changed files use `git diff <base> HEAD` or the three-dot
    merge-base range, then align implementation, `SPEC.md`, and tests.
  - Make changed-directory results match the documented top-level-directory
    behavior, including files at the repository root.
  - Add divergent-history, nested-path, root-file, rename, and unusual-filename
    test cases before these currently unused helpers gain callers.

- [ ] Replace ad hoc Git remote URL parsing with structured parsing.
  - Parse standard HTTPS, SSH URL, and SCP-style remote forms without silently
    ignoring extra path components.
  - Define and validate supported GitHub.com and GitHub Enterprise hosts,
    including ports, trailing slashes, and `.git` suffixes.
  - Return actionable errors for unsupported hosts and malformed owner/repository
    paths, with table-driven tests for accepted and rejected forms.

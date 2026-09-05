# Working on bpr

`bpr` is an agentic-first Go CLI for stacked GitHub pull requests: each commit
maps to one PR. It is a successor to Python `stack-pr`, expected to diverge.
Python is a legacy reference for intent, not a constraint on behavior or design.

## Design philosophy

- **Simplicity is cheap future change.** Judge a design by how much a future
  change requires a reader to know and how many places it requires them to edit,
  not by line count. Prefer clear module boundaries and isolate unavoidable
  complexity behind small interfaces. *A Philosophy of Software Design* is the
  reference frame.
- **Leave it simpler than found.** New behavior should justify its complexity.
  When an integration must add complexity that cannot reasonably be resolved in
  the change, leave a concrete TODO naming what needs to be isolated or
  refactored and why.
- **Idiomatic Go, always.** Prefer straightforward Go over mirroring Python or
  importing ceremony-heavy patterns from other languages. Constructors and
  interfaces should serve a concrete need, rather than a factory framework or
  architectural pattern chosen in advance. Return errors with useful operation
  and domain context; preserve the underlying cause when wrapping.
- **Let abstractions earn their place.** Two or three similar implementations
  are acceptable until the common concept is understood. Share code when it
  hides complexity or enforces a genuinely shared rule, not merely because the
  implementations look alike. Established, well-supported community dependencies
  are welcome when they reduce custom machinery within the architectural
  boundaries.
- **Simplify over backward compatibility.** Prefer a simpler design over
  preserving historical flags, output, or implementation shape. Preserve
  commit-message stack metadata used by existing PRs: ask before changing
  persistent metadata in a way that could break those PRs. Safety takes
  precedence over compatibility.
- **Agent-first throughout.** Consider agent consumers when designing every
  command, not only the `agent` subtree or explicit JSON modes.

## Authority and decisions

Use current Go behavior, tests, and maintained documentation to understand the
project. If they disagree, investigate the discrepancy rather than treating
either an old test or prose as proof of intended behavior. Ask when the intended
outcome remains unclear. Specs and the former plan/document workflow are retired;
they are not implementation gates.

**Architectural boundaries:** before changing a boundary, inspect the relevant
accepted decisions in `docs/adr/`. Follow them; discuss a conflicting approach
before implementing it. Keep individual architectural rules in their ADRs rather
than duplicating them here.

**ADR candidates:** before creating an ADR or changing an architectural decision,
apply the `canon-record-gate` skill and establish its verdict. Use the `canon`
skill to manage records that pass the gate. A coding preference, local refactor,
or completed task does not by itself warrant an ADR.

## Autonomy and scope

- Choose and explain the best option when it is reasonable, serves the requested
  goal, and fits these principles and the ADRs. Ask when the choice falls outside
  those boundaries or the product tradeoff remains unresolved.
- If the requested solution reinforces the wrong abstraction, propose an
  alternative and discuss it before implementing.
- Fix the rule, not only the reported example. Propose broader work when a fix
  exposes a shared problem; if a broader redesign is deferred, cover every path
  governed by the same rule in the focused fix.
- Fix adjacent problems when cheap; otherwise leave a concrete note. Keep broader
  cleanup separate when it would obscure the requested change.

## Completion

Work is complete when the requested outcome works. Each bug fix includes a new
test covering the bug; demonstrate that it catches the regression when practical.
Run `make test` before declaring completion and report any failures or checks
that could not run.

Update documentation when explicitly requested or when it meaningfully helps
someone use or understand the change; there is no mandatory spec, plan, or
documentation-finalization step. `CHANGELOG.md` records user-facing behavior
shipped in previous versions or intended for the next release, not workflow
bookkeeping.

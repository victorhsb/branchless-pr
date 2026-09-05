# Contributing

Thanks for your interest in `bpr`!

## Filing issues

- Search existing issues before opening a new one.
- Include the command you ran, the output, and your Git / `gh` / `go` versions.

## Pull requests

- Fork the repo and open your PR against `main`.
- Run `make test`, `make vet`, and `make fmt-check` before pushing.
- Keep changes focused — one feature or fix per PR.
- Add a regression test for each bug fix.
- Current Go behavior, tests, and maintained documentation are the authority.
  There is no spec or plan phase; consult `docs/adr/` before changing an
  architectural boundary.

## Style

- Standard `gofmt`.
- Errors propagate via explicit returns; do not use panics for control flow.
- Shell wrappers live in `internal/shell`; never call `exec.Command` outside it.

Maintainers control merges. Reviews aim for a one-business-day turnaround.

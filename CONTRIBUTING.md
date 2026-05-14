# Contributing

Thanks for your interest in `stack-pr`!

## Filing issues

- Search existing issues before opening a new one.
- Include the command you ran, the output, and your Git / `gh` / `go` versions.

## Pull requests

- Fork the repo and open your PR against `main`.
- Run `go vet ./...`, `gofmt -l .`, and `go test ./...` before pushing.
- Keep changes focused — one feature or fix per PR.
- Follow the algorithms in `SPEC.md`; if behavior should change, update the
  spec in the same PR.

## Style

- Standard `gofmt`.
- Errors propagate via explicit returns; do not use panics for control flow.
- Shell wrappers live in `internal/shell`; never call `exec.Command` outside it
  unless there is a clear reason.

Maintainers control merges. Reviews aim for a one-business-day turnaround.

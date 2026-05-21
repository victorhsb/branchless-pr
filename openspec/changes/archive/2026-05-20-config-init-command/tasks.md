## 1. Add `config init` subcommand wiring

- [x] 1.1 Modify `internal/cli/config.go` to introduce a `config init` subcommand under the existing `config` Cobra command, changing `configCmd()` from a leaf command to a parent command with a default `init` subcommand and the existing inline-set functionality as a `set` subcommand (or alias).
- [x] 1.2 Ensure the `init` subcommand bypasses git checks (`PersistentPreRunE: nil` or similar) so it behaves like the existing `config` command, but still resolves `config.FilePath()` to find the repo root.

## 2. Implement INI generation

- [x] 2.1 Add `internal/config/init.go` (or extend `config.go`) with a `WriteDefaults(path string) error` function that writes a `.stack-pr.cfg` file containing all default sections and keys with inline comments. Use a hard-coded template string to preserve comments and section ordering.
- [x] 2.2 Include overwrite guard logic: if the file already exists, return an error without writing.
- [x] 2.3 Wire the CLI `init` subcommand to call `WriteDefaults`, printing the created path on success or the guard error on failure.

## 3. Validate generated output

- [x] 3.1 Add unit tests in `internal/config/` that load the generated file with `config.Load` and assert it matches `config.Defaults()` key-for-key.
- [x] 3.2 Add CLI-level test in `internal/cli/` (or existing CLI test file) covering the overwrite-guard case: invoking `config init` twice returns an error on the second call.
- [x] 3.3 Run `make test`, `make vet`, and `make fmt-check` locally.

## 4. Documentation

- [x] 4.1 Update `SPEC.md` (search for the config section) to document `config init` behaviour: file generation, content, and overwrite guard.
- [x] 4.2 Add user-facing entry to `CHANGELOG.md` describing the new `config init` subcommand.

---
date: 2026-08-07
status: active
type: behavioral
title: Harden subprocess argument handling and remote-authored output
---

# Security hardening: argument injection and terminal escapes

## Context

A codebase review found that config- and commit-message-controlled values reach
`git` and `gh` argument vectors as bare positionals, and that GitHub-authored
text is printed to the terminal verbatim.

The `remote` value from `.stack-pr.cfg` (`[repo] remote`) is the sole positional
to `git fetch --prune <remote>`, `git ls-remote --heads <remote>`, and
`git push <remote>`, with no `--` terminator. A checked-in config containing
`remote = --upload-pack=<cmd>` is parsed by `git` as an option, and with a
local/file-transport origin this is local code execution.

This was reproduced in an isolated sandbox against git 2.43.0:

```
git fetch --prune '--upload-pack=touch PWNED'   # -> PWNED created
git fetch --prune -- '--upload-pack=touch PWNED' # -> fatal: strange pathname ... blocked
```

The `ext::sh -c '<cmd>'` transport-URL vector (which `--` does *not* stop, since
git accepts a URL wherever it accepts a remote name) was also tested and is
blocked by default: `fatal: transport 'ext' not allowed`. It is not relied upon
as a defense here — name validation closes it regardless of `protocol.ext.allow`.

The path is reachable from the read-only `bpr view` command via
`stack.AssignHeads` → `NextID` → `ResolveRemoteRefs`, so no mutating command is
required to trigger it.

The same root cause applies in bounded form to the `stack-info:` PR reference,
whose regex captures `(.+)` unvalidated before the raw value reaches
`gh pr view/edit/merge <ref>`.

Separately, GitHub-authored comment and check-run bodies are printed verbatim
under the default `--format=text`, permitting ANSI/OSC escape-sequence abuse of
the user's terminal. The `json` format is unaffected.

## Goals

1. Config- and remote-authored values can never be interpreted by `git`/`gh` as
   options, transports, or paths.
2. Remote-authored text cannot emit terminal control sequences.

## Non-goals

- Sandboxing `git`/`gh` themselves.
- Validating the *content* of legitimately-named remotes or branches beyond what
  is needed to keep them positional.

## Approach

### 1. Remote / target / PR-ref validation (closes HIGH + LOW)

Two layers, both cheap:

- **Syntactic validation.** A remote name may not be empty, begin with `-`,
  begin with `/` or `.`, contain `:`, or contain whitespace or control
  characters. This rejects `--upload-pack=`, `ext::`, `file://`, scp-style
  `host:path`, and absolute/relative paths, while accepting every real remote
  name (`origin`, `upstream`, a fork name).

  Name-only is already the de facto contract: `RepoSlug` runs
  `git remote get-url <remote>` and `TargetExists` builds `<remote>/<target>`
  and rev-parses it. Neither works for a URL. So this is behavior-preserving.

- **`--` terminators.** Insert `--` before every remote/refspec positional in
  the `internal/git` wrappers. This additionally protects the refspec
  positionals in `ForcePush`/`ForcePushWithLease`/`DeleteRemoteBranches`, which
  are built from the config-controlled branch template. Each placement was
  verified against git 2.43.0 to still parse correctly.

Validation is wired into `PersistentPreRunE` in `cli/root.go` (which already
validates the branch template, and which `view` passes through), with
defense-in-depth checks inside the exported `internal/git` wrappers.

For the PR ref, reject leading `-`, whitespace, and control characters. The ref
is a PR URL or a bare number per `docs/specs/submit-export.md:85`, so this
rejects nothing legitimate.

### 2. Control-character stripping (closes MEDIUM)

Strip C0 controls except `\n`, `\r`, `\t`, plus DEL and C1 (U+0080–U+009F), from
remote-authored text before printing in text renderers. Audit beyond
`reports/*` for other places remote-authored text reaches stdout (PR titles in
`view`, TOC rendering) rather than assuming two files is the whole surface.

## Out of scope for this plan (behavior-preserving, no spec impact)

- Deleting the vestigial `stack.BranchTemplate.HasID` field. `ParseTemplate`
  auto-appends `/$ID` when absent and sets `HasID: true` on both paths, which is
  exactly the contract AGENTS.md documents ("or implicitly via `/$ID`"). The
  `$ID` invariant is enforced by construction, not by rejection, so removing the
  unreachable rejection branches drops no invariant and needs no spec change.
- Dead exported code removal.
- Unifying the duplicated submit engines.

## Spec impact

- `docs/specs/git-operation-safety.md` — add an argument-safety section covering
  remote/ref validation and positional termination.
- `docs/specs/stack-reports.md` — note control-character sanitization in text
  output.
- `CHANGELOG.md` — user-facing: hostile config values are now rejected.

## Tasks

- [ ] Add remote/target/PR-ref validation helpers with table tests
- [ ] Wire validation into `cli/root.go` and the `internal/git` wrappers
- [ ] Add `--` terminators to all remote-positional `git` invocations
- [ ] Strip control characters from remote-authored text renderers, with an
      ANSI-payload table test
- [ ] Update `docs/specs/` + `CHANGELOG.md`
- [ ] `make fmt-check && make vet && make test && go test -race ./... && make build`

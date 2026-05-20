## Context

`stack-pr view --format json` already gives agents a structured view of the local stack and PR metadata, but it does not include review feedback. GitHub stores PR feedback across several surfaces: issue-style conversation comments, review summaries, review comments, and review threads that may be resolved or unresolved. Users currently have to inspect each PR manually, which is inefficient for a stack where each commit maps to one PR.

The command should be read-only, but unlike `agent prompt` it must run inside a repository and contact GitHub. It should therefore be a normal top-level command, likely `stack-pr comments`, sharing the existing root preflight and stack discovery behavior rather than living under the static `agent` subtree.

## Goals / Non-Goals

**Goals:**

- Gather comments for every PR in the current stack and preserve the stack entry that each comment came from.
- Produce Markdown by default for humans and a stable JSON schema for agents.
- Include enough provenance for action: PR number/URL, stack index, commit SHA/title, author, timestamps, state/resolution when available, source URL, body, path, line/range, and reply nesting when available.
- Support filters that reduce noise for review work, especially unresolved-only output and selecting comment kinds.
- Avoid local Git mutations and GitHub write operations.

**Non-Goals:**

- Posting, editing, resolving, or deleting comments.
- Summarizing comments with an LLM or deciding which comments are important.
- Replacing `stack-pr view`; stack comments are a review-feedback report, not the primary stack status report.
- Supporting non-GitHub hosting providers.

## Decisions

1. Add a top-level read-only `comments` command.
   - The command needs repo state, stack metadata, `gh`, and GitHub network access, so it should use the normal root command setup.
   - It should be exempt from the clean-worktree requirement like `view`, because reading review comments is useful while code is being edited.
   - Alternative considered: `stack-pr agent comments`. Rejected because the current `agent` namespace is primarily side-effect-free prompt/diagnose tooling, and `comments` necessarily depends on a repository and GitHub reads.

2. Fetch comments through `gh` and `internal/shell`, with no GitHub SDK.
   - `gh pr view --json comments,reviews` can provide conversation comments and review summaries.
   - Review threads and resolved state likely require `gh api graphql` because the normal `gh pr view` JSON fields do not expose all thread-level resolution and reply structure.
   - The implementation should live in `internal/pr` or a small adjacent package that wraps read-only GitHub comment queries through `internal/shell`.
   - Alternative considered: parse `gh pr view` text output. Rejected because it is presentation-oriented and less stable for agents.

3. Normalize GitHub comment surfaces into one internal report model.
   - A stack-level report should contain metadata for the command invocation, a per-PR entry list in stack order, warnings/errors for individual PRs, and normalized comment/thread items.
   - Each item should carry a `kind` value such as `conversation`, `review`, `review_thread`, or `review_comment`.
   - Thread items should preserve `resolved` when GitHub provides it; if resolution cannot be determined for a comment kind, the value should be omitted or marked unknown instead of guessed.
   - Alternative considered: emit raw GitHub JSON. Rejected because callers would have to merge multiple GitHub schemas and rebuild stack context themselves.

4. Make Markdown the default and JSON explicit.
   - `--format text` should produce Markdown-friendly output grouped by stack entry and PR, with unresolved items easy to spot.
   - `--format json` should emit a single JSON object with no ANSI escapes, progress logs, or terminal hyperlinks.
   - Alternative considered: add comments to `view --format json`. Rejected because comment retrieval requires online GitHub reads, can be large, and has different filtering/error semantics from local stack inspection.

5. Degrade per PR rather than failing the whole report when possible.
   - Missing PR metadata should be reported on the relevant stack entry.
   - Inaccessible PRs, deleted PRs, authentication failures, rate limits, and GraphQL failures should produce structured errors or warnings.
   - Global invocation errors, such as unsupported `--format`, should still return non-zero.
   - Alternative considered: fail on the first bad PR. Rejected because stacked review context is still useful when one PR in the stack cannot be read.

## Risks / Trade-offs

- **Risk: Comment retrieval is slower on large stacks.** -> Fetch per PR in stack order first; parallel fetches can be added later if latency becomes a problem and output ordering remains deterministic.
- **Risk: GitHub API fields differ between issue comments, reviews, and review threads.** -> Normalize only stable common fields and keep optional fields nullable/omitted when unavailable.
- **Risk: `--unresolved-only` cannot apply cleanly to every comment kind.** -> Apply it to review threads with known resolution and include only unresolved-capable kinds unless the user explicitly includes all kinds.
- **Risk: JSON output becomes noisy or invalid if progress logs share stdout.** -> Send diagnostics to stderr or suppress them in JSON mode so stdout remains a single parseable object.
- **Risk: GraphQL query shape may be complex.** -> Keep GraphQL use isolated behind a read-only helper with fixture-based tests for parsing and degradation.

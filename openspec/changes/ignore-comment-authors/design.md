## Context

`stack-pr comments` currently fetches normalized PR feedback and filters it by kind, unresolved state, and an optional positive `--author` login. The command already receives the loaded `.stack-pr.cfg` through `AppContext`, but comments-specific options are command-local and do not resolve any config defaults.

CI systems and automation accounts often leave repeated comments or review-thread replies. Those entries are valid GitHub feedback, but they are usually lower signal when a user is trying to scan human review context across a stack.

## Goals / Non-Goals

**Goals:**

- Let repositories configure a default ignored-author list for `stack-pr comments` and the `bpr comments` alias.
- Apply ignored-author filtering consistently to conversation comments, reviews, standalone review comments, review threads, and nested thread replies.
- Preserve existing defaults when `.stack-pr.cfg` does not configure ignored authors.
- Keep the report read-only and avoid adding GitHub calls or dependencies.

**Non-Goals:**

- Add global user-level config outside `.stack-pr.cfg`.
- Add comment body, bot-type, or organization-based filtering.
- Mutate or hide comments in GitHub.
- Change `stack-pr checks` comment summaries.

## Decisions

1. Use `comments.ignore_authors` in `.stack-pr.cfg`.

   A dedicated `[comments]` section keeps comments-report configuration separate from repository settings such as `repo.reviewer`. The value is a comma-separated list of GitHub logins, matching existing comma-separated config and flag patterns. Missing or empty values mean no ignored authors.

   Alternative considered: `repo.ignore_comment_authors`. That would avoid a new section, but it mixes command-output preferences into repository identity settings and scales poorly if `comments` gains more config later.

2. Resolve ignored authors in the comments command, not in shared args.

   The setting is command-specific, so it should stay in `commentsOptions` or a small comments-specific resolver rather than expanding `CommonArgs`. This keeps unrelated commands from carrying unused config state.

   Alternative considered: add a field to `CommonArgs`. That would make the value easy to access everywhere, but it broadens shared invocation state for a single command.

3. Apply ignored-author filtering before positive `--author` filtering.

   The configured list is a default exclusion policy for noisy accounts. If `comments.ignore_authors = ci-bot` and the user runs `stack-pr comments --author ci-bot`, the effective result is empty because the author was excluded by configuration first. This should be documented so users can temporarily edit config when they intentionally want to inspect ignored authors.

   Alternative considered: make `--author` override ignored authors. That is convenient for one-off inspection, but it makes command output depend on an implicit precedence exception and weakens the meaning of an ignored-author policy.

4. Filter nested review-thread replies without dropping useful thread context unnecessarily.

   For simple comment/review items, an ignored author removes the item. For review threads, ignored replies are removed from `Replies`; the thread item remains only if its own effective author is not ignored or at least one non-ignored reply remains. If all replies are ignored, the thread is removed from output.

   Alternative considered: remove an entire thread when any reply author is ignored. That is simpler but would hide human replies in mixed bot/human threads.

## Risks / Trade-offs

- Configured ignore list can hide relevant automation failures or bot-authored review comments -> Document the setting as an output filter and keep the default empty.
- Users cannot one-off include ignored authors with a flag -> Keep the first implementation small and add an override flag later only if the workflow proves necessary.
- GitHub login casing may differ across APIs -> Compare ignored authors case-insensitively while preserving original author casing in output.
- Nested reply filtering may leave a review-thread item with trimmed replies -> Ensure JSON/text output reflects the filtered result exactly and tests cover mixed-author threads.

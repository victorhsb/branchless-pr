## 1. Configuration Resolution

- [x] 1.1 Add a documented default for `comments.ignore_authors` with an empty value.
- [x] 1.2 Parse `comments.ignore_authors` as a comma-separated GitHub login list, trimming whitespace and dropping empty entries.
- [x] 1.3 Resolve ignored authors inside the comments command path without adding command-specific fields to shared `CommonArgs`.

## 2. Comment Filtering

- [x] 2.1 Extend comments filtering so ignored authors are excluded case-insensitively before `--author` filtering.
- [x] 2.2 Apply ignored-author filtering to conversation comments, reviews, standalone review comments, review-thread items, and review-thread replies.
- [x] 2.3 Preserve mixed-author review threads when at least one non-ignored reply remains, and remove threads with no reportable feedback.
- [x] 2.4 Ensure text and JSON output both render the filtered result consistently.

## 3. Tests

- [x] 3.1 Add config parsing/resolution tests for missing, empty, single-author, multi-author, and whitespace-padded ignored-author values.
- [x] 3.2 Add comments filtering tests for ignored conversation comments, reviews, review comments, and case-insensitive login matching.
- [x] 3.3 Add review-thread tests covering mixed ignored/non-ignored replies and all-ignored replies.
- [x] 3.4 Add a test showing `--author` does not re-include a configured ignored author.

## 4. Documentation

- [x] 4.1 Update `SPEC.md` comments command and configuration sections with `comments.ignore_authors`.
- [x] 4.2 Update README or command documentation examples if they describe `.stack-pr.cfg` comments behavior.
- [x] 4.3 Run `make test`, and run `make fmt-check` if any Go files changed.

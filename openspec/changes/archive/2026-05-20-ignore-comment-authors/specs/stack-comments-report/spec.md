## ADDED Requirements

### Requirement: Ignored Comment Authors Configuration
The `comments` command SHALL support `.stack-pr.cfg` configuration that excludes feedback authored by configured GitHub logins from comments report output.

#### Scenario: Ignored authors are omitted by default
- **WHEN** `.stack-pr.cfg` contains `comments.ignore_authors = ci-bot,release-bot`
- **AND** `stack-pr comments` is invoked without an author override
- **THEN** the report SHALL exclude comments, reviews, review comments, review-thread items, and review-thread replies authored by `ci-bot` or `release-bot`

#### Scenario: Missing ignore configuration preserves existing behavior
- **WHEN** `.stack-pr.cfg` omits `comments.ignore_authors`
- **THEN** `stack-pr comments` SHALL NOT exclude any feedback because of ignored-author configuration

#### Scenario: Empty ignore configuration preserves existing behavior
- **WHEN** `.stack-pr.cfg` contains `comments.ignore_authors =`
- **THEN** `stack-pr comments` SHALL NOT exclude any feedback because of ignored-author configuration

#### Scenario: Ignored author matching is case-insensitive
- **WHEN** `.stack-pr.cfg` contains `comments.ignore_authors = CI-Bot`
- **AND** GitHub returns feedback authored by `ci-bot`
- **THEN** that feedback SHALL be treated as authored by an ignored login

#### Scenario: Ignored authors combine with author filtering
- **WHEN** `.stack-pr.cfg` contains `comments.ignore_authors = ci-bot`
- **AND** `stack-pr comments --author ci-bot` is invoked
- **THEN** the report SHALL contain no `ci-bot` feedback because ignored-author filtering is applied before positive author filtering

#### Scenario: Mixed-author review threads retain non-ignored replies
- **WHEN** a review thread contains replies from both an ignored author and a non-ignored author
- **THEN** the report SHALL exclude the ignored-author replies
- **AND** it SHALL retain the thread with the remaining non-ignored replies when the thread still has reportable feedback

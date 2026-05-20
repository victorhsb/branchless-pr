# stack-pr Project Specification

This document specifies the current `stack-pr` repository in enough detail to re-create its behavior, source layout, packaging, tests, and release automation.

## 1. Purpose

`stack-pr` is a command-line tool for creating, updating, viewing, abandoning, and landing stacked GitHub pull requests.

A stack is modeled as the ordered list of local commits in a Git revision range (`BASE..HEAD`). Each commit corresponds to exactly one GitHub PR. The bottom PR targets the repository target branch (normally `main`), and every higher PR targets the generated branch for the previous commit. This makes each PR review show only one logical commit while still preserving dependency order.

The installed console command is:

```bash
stack-pr
```

A shorter alias, `bpr` ("branchless PR"), is available as an identical binary
or via a shell alias. It accepts the exact same commands, flags, and config
file as `stack-pr`.

In the Python port, module execution is also supported:

```bash
python -m stack_pr
```

## 2. Repository layout

```text
.
├── .gitattributes
├── .github/
│   └── workflows/
│       ├── check_tests.yml
│       ├── lint.yml
│       └── release.yml
├── .gitignore
├── CHANGELOG.md
├── CONTRIBUTING.md
├── LICENSE
├── pdm.lock
├── pyproject.toml
├── README.md
├── SPEC.md
├── src/
│   └── stack_pr/
│       ├── __init__.py
│       ├── __main__.py
│       ├── cli.py
│       ├── git.py
│       ├── py.typed
│       └── shell_commands.py
└── tests/
    ├── __init__.py
    ├── test_misc.py
    └── test_shell_commands.py
```

`src/stack_pr/__init__.py` and `src/stack_pr/py.typed` are intentionally empty. `py.typed` marks the package as typed for type checkers.

## 3. Packaging and metadata

The project is defined by `pyproject.toml`.

- Build backend: `pdm.backend` from `pdm-backend`.
- Distribution name: `stack-pr`.
- Import package: `stack_pr`.
- Console script: `stack-pr = stack_pr.cli:main`.
- Python requirement: `>=3.9`.
- Runtime dependencies: `typing_extensions` only when `python_version < "3.13"`.
- Optional development dependencies: `pytest`, `pytest-mock`, `mypy`, `ruff`.
- Versioning: dynamic SCM version via PDM, fallback version `0.1.0`.
- License file: `LICENSE`.
- URLs: homepage, repository, and bug tracker point to `https://github.com/modular/stack-pr`.
- Keywords: stacked PRs, GitHub, pull requests, git, version control.
- Classifiers describe a production/stable console utility for Python 3.9 through 3.13.

Pixi metadata also exists:

- Channels: `conda-forge`.
- Platforms: `osx-arm64`, `osx-64`, `linux-64`, `linux-aarch64`.
- Python pin: `3.9.*`.
- Editable local PyPI dependency: `stack-pr = { path = ".", editable = true }`.
- PDM version constraint: `>=2.17.1,<2.18`.

`pdm.lock` is generated and contains no normal runtime package entries beyond metadata.

## 4. External runtime requirements

The CLI expects to run inside a Git repository that has at least one commit and has a configured GitHub remote.

External commands used:

- `git`
- `gh` (GitHub CLI)

`gh` must be installed and authenticated. The README recommends `gh auth login` with SSH.

The tool shells out to Git and GitHub CLI rather than using Python GitHub libraries.

## 5. Core concepts

### 5.1 Stack range

For most commands, the stack is discovered from commits in `BASE..HEAD`.

- `HEAD` defaults to the Git revision `HEAD`.
- `TARGET` defaults to remote target branch `main`.
- `REMOTE` defaults to `origin`.
- If `BASE` is not explicitly supplied, it is deduced as:

```bash
git merge-base HEAD REMOTE/TARGET
```

The tool requires `BASE` to be an ancestor of `HEAD`; otherwise it exits after printing an error.

### 5.2 Stack order

Internally, Git returns commits with:

```bash
git rev-list --header ^BASE HEAD
```

That result is reversed so stack entries are ordered oldest-to-newest. Display output prints entries newest-to-oldest.

### 5.3 Commit metadata

Submitted commits are amended with a metadata line in their commit message:

```text
stack-info: PR: <pr-url-or-ref>, branch: <generated-head-branch>
```

The parser recognizes this exact line pattern with a leading newline:

```regex
\n^stack-info: PR: (.+), branch: (.+)\n?
```

The PR URL/ref and head branch are read from commit metadata. Base branches are recomputed from stack order.

### 5.4 Generated branches

Generated branches are stack implementation details. Users are expected not to manually push, delete, or merge them.

Default branch template:

```text
$USERNAME/stack
```

If a template does not contain `$ID`, `/$ID` is appended, so the default becomes logically:

```text
$USERNAME/stack/$ID
```

Supported template variables:

- `$USERNAME`: current GitHub login from `gh api graphql`.
- `$BRANCH`: current local branch name.
- `$ID`: numeric branch identifier.

The tool fetches/prunes remotes, scans matching remote refs, selects the maximum existing numeric ID, and uses the next integer. If none exist, ID `1` is used.

Example generated branches:

```text
alice/stack/1
alice/stack/2
alice/my-feature/3
feature-123-desc
```

### 5.5 Base branch assignment

For a stack ordered bottom-to-top (`oldest -> newest`):

- First entry base = target branch, normally `main`.
- Each later entry base = previous entry head branch.

Example with three commits:

```text
commit A: head alice/stack/1, base main
commit B: head alice/stack/2, base alice/stack/1
commit C: head alice/stack/3, base alice/stack/2
```

## 6. Command-line interface

### 6.1 Common options

These options are shared by `submit`, `export`, `view`, `comments`, `land`, and `abandon`:

```text
-R, --remote                Remote name; default from config repo.remote or origin
-B, --base                  Local base revision; default deduced merge-base
-H, --head                  Local head revision; default HEAD
-T, --target                Remote target branch; default from config repo.target or main
--hyperlinks / --no-hyperlinks
                            Enable terminal hyperlinks; default true
-V, --verbose               Show verbose Git/GH subprocess output; default false
--branch-name-template      Generated branch template; default $USERNAME/stack
--show-tips / --no-show-tips
                            Show post-command guidance; default true
```

### 6.2 Commands

#### `stack-pr submit`

Creates or updates the stack of PRs. Alias: `stack-pr export`.

Options:

```text
--keep-body         Preserve current PR body after stack cross-link section
-d, --draft         Create all new PRs as draft
--draft-bitmask     Per-PR draft bitmask; chars must be 0 or 1
--reviewer          Reviewer list; default from STACK_PR_DEFAULT_REVIEWER or config repo.reviewer
-s, --stash         Stash uncommitted changes before submitting and pop afterward
```

Draft bitmask semantics:

- Parsed to a `list[bool]`.
- Length must match stack length.
- `1` means draft for the corresponding stack entry.
- If `--draft` is set, it overrides the bitmask and makes all created PRs draft.

#### `stack-pr view`

Safely inspects the current stack. It does not modify commits or push branches, but it may fetch/prune the remote while assigning hypothetical head branches for display.

#### `stack-pr comments`

Collects pull request conversation comments, reviews, review comments, and review threads across the current stack. It is read-only: it does not modify commits, branches, remotes, pull requests, or comments.

Options:

```text
--format text|json       Output Markdown-compatible text or machine-readable JSON; default text
--unresolved-only        Show only unresolved or attention-required feedback
--kind                   Comma-separated kinds: conversation, review, review_comment, review_thread
--author                 Show only feedback authored by the given GitHub login
```

#### `stack-pr land`

Lands the bottom-most PR in the stack using GitHub squash merge, then rebases remaining stack branches onto the latest remote target. This command is only registered when config `land.style` is `bottom-only` (the default). If `land.style=disable`, the command is unavailable.

#### `stack-pr abandon`

Removes stack metadata from commits, deletes local generated branches, and deletes matching remote generated branches. The current implementation does not call `gh pr close`; although README text describes closing PRs, the code only strips metadata and deletes branches.

#### `stack-pr config <section>.<key>=<value>`

Creates or updates a setting in the config file. It does not require command-specific options. It expects exactly the form:

```text
<section>.<key>=<value>
```

Invalid formats print a usage error and exit with status 1.

## 7. Configuration

Config file path:

1. Environment variable `STACKPR_CONFIG`, if set.
2. Otherwise `<repo-root>/.stack-pr.cfg`.

The file uses INI format through Python `configparser`.

Recognized defaults:

```ini
[common]
verbose=True|False
hyperlinks=True|False
draft=True|False
keep_body=True|False
stash=True|False
show_tips=True|False

[repo]
remote=origin
target=main
reviewer=user1,user2
branch_name_template=$USERNAME/stack

[land]
style=bottom-only|disable
```

The config command writes values as strings. Boolean values are later read with `ConfigParser.getboolean`.

Reviewer default precedence for submit:

1. `--reviewer` CLI argument.
2. `STACK_PR_DEFAULT_REVIEWER` environment variable.
3. `repo.reviewer` config key.
4. Empty string.

## 8. Main execution flow

`stack_pr.cli.main()` performs these steps:

1. Compute repo config file as `get_repo_root() / ".stack-pr.cfg"`.
2. Override with `STACKPR_CONFIG` if present.
3. Load config.
4. Build argparse parser.
5. Parse arguments.
6. Set global verbose flag if the parsed command has `verbose`.
7. If no command: print invalid usage plus help and return.
8. If command is `config`: update config file and return.
9. Ensure branch name template contains `$ID`.
10. Create `CommonArgs` from parsed args.
11. Enable logger debug level when verbose.
12. Check `gh` installation by invoking `gh`.
13. Record current branch.
14. Resolve the branch name base, which also validates current GitHub username lookup.
15. For `submit/export --stash`: run `git stash save` and remember whether anything was stashed.
16. For all commands except `view`: require the repo to be clean, ignoring untracked files.
17. Check that `REMOTE/TARGET` exists. If target is `main` and `REMOTE/master` exists, print a targeted hint for master-based repos.
18. Deduce base if missing.
19. Dispatch to the command implementation.
20. On exceptions: checkout the original branch, print subprocess failure details when applicable, and re-raise.
21. Finally, for `submit/export --stash`, pop the stash if one was actually created.

## 9. Data model

### 9.1 `CommitHeader`

Represents parsed output from `git rev-list --header`.

Fields:

- `raw_header: str`

Methods extract:

- `tree()` from `tree <hash>`.
- `title()` from the first indented commit message line.
- `commit_id()` from a raw commit SHA line.
- `parents()` from all `parent <sha>` lines.
- `author()` from `author Name <email>`.
- `author_name()`.
- `author_email()`.
- `commit_msg()` as all indented message lines joined by newlines.

If a required field is missing, a `ValueError` is raised.

### 9.2 `StackEntry`

Represents one stack commit and associated PR state.

Fields:

- `commit: CommitHeader`
- `_pr: str | None`
- `_base: str | None`
- `_head: str | None`
- `is_tmp_draft: bool`

Properties `pr` and `head` raise `ValueError` when unset. `base` can be `None`.

Important methods:

- `has_pr()`, `has_head()`, `has_base()`.
- `has_missing_info()` returns true when any of PR, head, or base is missing.
- `read_metadata()` parses `stack-info` from the commit message and sets PR/head.
- `pprint(links: bool)` formats one stack line with ANSI colors and optional terminal hyperlink.

### 9.3 `CommonArgs`

Typed container for shared command arguments:

- `base`
- `head`
- `remote`
- `target`
- `hyperlinks`
- `verbose`
- `branch_name_template`
- `show_tips`
- `land_disabled`

## 10. Shell command behavior

`stack_pr.shell_commands.run_shell_command(cmd, *, quiet, check=True, **kwargs)` wraps `subprocess.run`.

Required behavior:

- `cmd` is an iterable of strings or `pathlib.Path` objects.
- `shell` keyword is forbidden and raises `ValueError("shell support has been removed")`.
- All command elements are converted to strings.
- `check=True` by default.
- When `quiet=True`, stdout and stderr default to `subprocess.PIPE` unless the caller explicitly provides them.
- When `quiet=False`, stdout/stderr inherit the console unless explicitly provided.
- The command is debug-logged.

`get_command_output(cmd, **kwargs)`:

- Rejects a `capture_output` keyword with `ValueError`.
- Calls `run_shell_command(..., capture_output=True, quiet=False, **kwargs)`.
- Returns decoded UTF-8 stdout with trailing whitespace stripped via `rstrip()`.

## 11. Git helper behavior

`stack_pr.git` defines:

- `GitError`, raised for selected Git/GH helper failures.
- `GIT_NOT_A_REPO_ERROR = 128`.
- `GIT_SHA_LENGTH = 40`.
- `GitConfig(username_override: str | None = None)` singleton `git_config` for tests.

Functions:

- `is_full_git_sha(s)`: true iff `s` is exactly 40 lowercase hexadecimal characters. Implementation accepts characters from `string.hexdigits.lower()`.
- `branch_exists(branch, repo_dir=None)`: runs `git show-ref -q refs/heads/<branch>`; return code 0 -> true, 1 -> false, anything else -> `GitError`.
- `get_current_branch_name(repo_dir=None)`: `git rev-parse --abbrev-ref HEAD`; converts git return code 128 to `GitError`.
- `get_repo_root(repo_dir=None)`: `git rev-parse --show-toplevel`; converts git return code 128 to `GitError`.
- `get_uncommitted_changes(repo_dir=None)`: parses `git status --porcelain` into a dict keyed by the first two status chars, with values from `line[3:]`.
- `check_gh_installed()`: runs `gh`; any `CalledProcessError` becomes `GitError` instructing installation from `https://cli.github.com/`.
- `get_gh_username()`: returns `git_config.username_override` when set; otherwise runs a GraphQL query through `gh api graphql` and extracts `"login":"..."` with regex.
- `get_changed_files(base=None, repo_dir=None)`: `git diff --name-only <base or main> HEAD`, returns `Path` objects split on newline.
- `get_changed_dirs(base=None, repo_dir=None)`: top-level directory set from changed files.
- `is_rebase_in_progress(repo_dir=None)`: checks whether `.git/rebase-merge` or `.git/rebase-apply` exists. With a repo_dir, it uses `repo_dir / ".git"`; without one, it uses `Path(".git")`.

## 12. Stack verification

`verify(st, check_base=False)` validates every `StackEntry`.

For each entry:

1. PR, head, and base must all be present.
2. Last path component of PR must be numeric.
3. Query GitHub:

```bash
gh pr view <pr> --json baseRefName,headRefName,number,state,body,title,url,mergeStateStatus
```

4. Response must include `state`, `number`, `baseRefName`, `headRefName`.
5. `state` must equal `OPEN`.
6. PR number from metadata must match GitHub number.
7. Head branch must match GitHub `headRefName`.
8. If `check_base=True`, base branch must match GitHub `baseRefName`.
9. If `check_base=True` and this is the first entry, `mergeStateStatus` must be one of `CLEAN`, `UNKNOWN`, or `UNSTABLE`.

Failures print a specific ANSI-red error message and raise `RuntimeError`.

## 13. Submit/export algorithm

`command_submit(args, draft, reviewer, keep_body, draft_bitmask=None)` implements submission.

Detailed behavior:

1. If a rebase is in progress, print an error and exit status 1.
2. Record current branch.
3. If local base can be fast-forwarded/rebased to `REMOTE/TARGET` because:
   - `base` is ancestor of `REMOTE/TARGET`,
   - `REMOTE/TARGET` is ancestor of `head`, and
   - hashes differ,
     then run `git rebase REMOTE/TARGET base` and checkout the original branch.
4. Load stack from `base..head`.
5. If empty: print `Empty stack!`.
6. Validate draft bitmask length if provided; on mismatch, print a message and return without submitting.
7. Initialize local branches:
   - Fetch/prune remote.
   - Assign generated head branches to entries missing metadata heads.
   - For each entry, run `git checkout <commit-id> -B <entry.head>`.
8. Compute base branches.
9. Determine whether the original current branch needs rebasing: true if the top stack branch is an ancestor of the current branch.
10. Reset remote base branches for existing PRs:

- For every entry with an existing PR, query `isDraft`.
- If not draft, mark draft using `gh pr ready <pr> --undo` and set `is_tmp_draft=True`.
- Set PR base to the target branch using `gh pr edit <pr> -B <target>`.

11. Force-push all stack head branches in one command:

```bash
git push -f <remote> <head1>:<head1> <head2>:<head2> ...
```

12. For each stack entry without a PR, create one:

```bash
gh pr create -B <base> -H <head> -t <commit-title> -F - [--reviewer <reviewer>] [--draft]
```

The PR body input is the full commit message. The PR reference is parsed as the last whitespace-separated token of command output.

13. Verify stack metadata and GitHub state.
14. Print the stack.
15. Add metadata to commit messages:

- For the first changed commit, checkout its head branch if no rebase is needed.
- For later changed commits, rebase branch onto its base using `--committer-date-is-author-date`.
- If metadata is absent, append the `stack-info` line and amend using `git commit --amend -F -`.
- Once one commit is amended, later entries need rebasing.

16. Force-push all branches again.
17. Add cross-links and update PR titles/bodies/base branches.
18. Restore PRs that were made temporary draft using `gh pr ready <pr>`.
19. Rebase or checkout the original branch:

- If needed, `git rebase <top_branch> <current_branch> --committer-date-is-author-date`.
- Otherwise checkout current branch.

20. Delete local generated branches with `git branch -D ...` using `check=False`.
21. Print post-export tips if enabled.

## 14. PR cross-linking

For multi-PR stacks, each PR body receives a table-of-contents header:

```text
Stacked PRs:
 * #<top-pr>
 * __->__#<current-pr>
 * #<bottom-pr>

--- --- ---
```

The stack is listed newest-to-oldest. The current PR is marked by prefixing its entry with `__->__`.

For a single-PR stack, no table of contents is generated.

When constructing a PR body:

- PR title is the commit title.
- The first line/title is stripped from the commit message body.
- The `stack-info` metadata line is stripped.
- For multi-PR stacks, body content starts with `### <title>` followed by the stripped commit body.
- If `--keep-body` is set, the existing PR body is fetched and content after the delimiter `--- --- ---` is preserved instead.

Each PR is updated with:

```bash
gh pr edit <pr> -t <title> -F - -B <base>
```

## 15. Land algorithm

`command_land(args)` implements bottom-only landing.

Detailed behavior:

1. Record current branch.
2. Optionally update local base the same way `submit` does.
3. Load stack.
4. If empty: print `Empty stack!`.
5. Set base branches and print stack.
6. Verify with `check_base=True`.
7. Land the bottom-most PR:
   - Fetch/prune remote.
   - Checkout remote head branch locally with `git checkout REMOTE/<head> -B <head>`.
   - Set PR base to target with `gh pr edit <pr> -B <target>`.
   - Build squash merge title as `<original first commit-message line> (#<pr-number>)`.
   - Build squash body from the remaining commit message after stripping stack metadata; if empty, use one space.
   - Run `gh pr merge <pr> --squash -t <title> -F -`.
8. If more PRs remain:
   - Print `Rebasing the rest of the stack` and those entries.
   - For each remaining entry:
     - Fetch/prune remote.
     - Checkout `REMOTE/<head>` to local branch `<head>`.
     - Rebase branch onto `REMOTE/TARGET` with `--committer-date-is-author-date`.
     - Force-push `<head>:<head>`.
   - Set the new bottom PR base to the target branch.
9. Checkout original branch.
10. Delete local stack branches.
11. If a local branch named target exists, rebase it onto `REMOTE/TARGET`.
12. Rebase the original branch onto `REMOTE/TARGET`.

The land command does not delete remote branches directly; GitHub may delete merged PR branches depending on repository settings.

## 16. Abandon algorithm

`command_abandon(args)` implements abandonment.

Detailed behavior:

1. Load stack.
2. If empty: print `Empty stack!`.
3. Record current branch.
4. Initialize local branches for every stack commit, preserving existing metadata heads or assigning new ones if absent.
5. Set base branches.
6. Print stack.
7. For each entry, strip metadata:
   - Remove `stack-info` from commit message.
   - First entry checks out its head branch.
   - Later entries rebase their head branch onto their base with `--committer-date-is-author-date`.
   - Amend commit message with `git commit --amend -F -`.
   - Record new hash from `git rev-parse <head>`.
8. Rebase current branch onto the final stripped top commit hash.
9. Delete local generated branches.
10. Delete remote branches that:
    - match the configured branch name base, and
    - are heads for stack entries.

Remote deletion command form:

```bash
git push -f <remote> :<branch1> :<branch2> ...
```

## 17. View algorithm

`command_view(args)` implements inspection.

Detailed behavior:

1. If local base appears behind remote target in the auto-updatable way, print a warning and suggested commands instead of modifying anything.
2. Load stack.
3. If empty: print `Empty stack!`.
4. Assign head branches to entries missing metadata heads by scanning the remote, but do not create branches or push.
5. Set base branches.
6. Print stack newest-to-oldest.
7. Print tips:
   - If every entry has PR/head/base metadata, say stack is ready to land and show update/land commands.
   - Otherwise say stack cannot be landed yet and show export command.

## 18. Comments algorithm

`command_comments(args)` implements stack-wide review feedback inspection.

Detailed behavior:

1. Load the stack using the same base/head range as `view`.
2. If empty, print an empty comments report and do not query GitHub for comments.
3. Read `stack-info` metadata for each entry.
4. Assign head branches to entries missing metadata heads by scanning remote refs, but do not create branches or push.
5. Set base branches.
6. For every entry with PR metadata, fetch read-only comment data through `gh`:
   - `gh pr view <pr> --json number,url,comments,reviews` for conversation comments and reviews.
   - `gh api graphql` for review threads and thread resolution state.
7. For every entry without PR metadata, include a report entry with status `missing`.
8. If an individual PR cannot be read, include a report entry with status `failed` and continue with other PRs.
9. If GitHub authentication or authorization fails globally, exit non-zero with a clear error.
10. Apply filters in this order: comment kind, unresolved-only, author.
11. Render Markdown-compatible text by default, grouped by stack entry in stack order.
12. With `--format json`, render one JSON object containing `schema_version`, `command`, `repository`, `range`, `stack`, and `pull_requests`, with no ANSI escape sequences, terminal hyperlinks, progress logs, or extra stdout text.

Normalized comment kinds:

- `conversation`: issue-style pull request conversation comments.
- `review`: submitted pull request reviews.
- `review_comment`: line-level review comments when exposed separately or as thread replies.
- `review_thread`: review thread containers with resolution state when GitHub provides it.

## 19. Cleanliness and safety rules

- All commands except `view` and `comments` require no tracked/staged/unstaged changes.
- Untracked files (`??`) are ignored for cleanliness checks.
- `submit/export --stash` can stash changes before the clean check and pop them afterward.
- Submit refuses to run while `.git/rebase-merge` or `.git/rebase-apply` exists.
- On exceptions during main command dispatch, the original branch is checked out before re-raising.
- Subprocess stdout/stderr are captured in quiet mode, which allows failures to print exit code/stdout/stderr.
- Shell invocation is disallowed in the subprocess wrapper.

## 20. Output formatting

ANSI color helpers:

- Header: `\033[95m`.
- Blue: `\033[94m`.
- Green: `\033[92m`.
- Red/fail: `\033[91m`.
- Bold: `\033[1m`.
- Reset: `\033[0m`.

Terminal hyperlinks use OSC 8:

```text
\033]8;;<location>\033\\<text>\033]8;;\033\\
```

Stack lines include:

```text
* <short-sha> (#<pr-number or no PR>, '<head>' -> '<base>'): <commit title>
```

Commands do not print command banners such as `SUBMIT`, `VIEW`, `LAND`, or `ABANDON`, and do not print generic success/failure markers such as `SUCCESS!`; output is limited to command results, warnings, tips, and errors. Runtime command errors are printed without Cobra's extra `Error:` or usage preambles.

Comments text output is Markdown-compatible. It starts with `# stack-pr comments`, prints the inspected range, then groups results by stack entry and PR. Empty stacks, entries without PR metadata, empty filtered results, and per-PR read failures are rendered explicitly. JSON comments output is a single parseable object and does not include ANSI styling or terminal hyperlinks.

Verbosity:

- `log(..., level=1)` always prints.
- `log(..., level>=2)` prints only when global verbose is enabled.

## 21. Error behavior

The CLI contains explicit multi-line error messages for these scenarios:

- Cannot update stack metadata.
- Cannot create a PR.
- Cannot rebase or checkout remote branch while landing.
- Stack metadata missing.
- Malformed/bad PR link in stack metadata.
- Malformed GitHub response.
- Associated PR not open.
- PR number/head/base mismatch.
- Bottom PR not mergeable.
- Dirty repository.
- Rebase in progress.
- Invalid config setting format.
- Target branch missing.
- Target `main` missing while `master` exists.
- Unsupported comments output format or comment kind.
- GitHub authentication/authorization failure while reading stack comments.

Errors are printed with a red `ERROR:` prefix. Many validation failures raise `RuntimeError`; target branch/config/rebase-progress failures use `sys.exit(1)` or return paths as implemented.

## 22. Tests

The repository currently has 14 tests, all passing.

Command used during this specification pass:

```bash
pytest -q
```

Result:

```text
14 passed
```

### 22.1 `tests/test_misc.py`

Covers:

- GitHub username override via `git_config.set_username_override("TestBot")`.
- Branch ID extraction for templates including `$USERNAME` and `$ID`.
- Non-match cases for branch ID extraction.
- Branch name generation.
- Extracting taken branch IDs from remote refs.
- Choosing next available branch name.
- Rebase-in-progress detection for `.git/rebase-merge` and `.git/rebase-apply`.

### 22.2 `tests/test_shell_commands.py`

Covers `run_shell_command` behavior:

- `quiet=False` prints stdout/stderr and does not capture them on success.
- `quiet=True` captures stdout/stderr and does not print them on success.
- `quiet=True` captures stdout/stderr on `CalledProcessError`.
- `quiet=False` prints stdout/stderr on failure.

## 23. Linting, formatting, and type checking configuration

### 23.1 Ruff

Ruff configuration:

- Line length: 88.
- Target Python: 3.9.
- Enabled rule groups include pycodestyle, Pyflakes, isort, bugbear, comprehensions, pyupgrade, pep8-naming, simplify, ruff-specific, warnings, flake8-2020, annotations, bandit, blind-except, boolean-trap, builtins, datetimez, debugger, implicit string concat, logging-format, no-pep420, pie, print, pytest-style, quotes, raise, return, self, tidy-imports, unused-arguments, pathlib, eradicate, pandas-vet, pygrep-hooks, pylint, tryceratops, and selected future/type-stub rules.
- Globally ignored rules:
  - `S603`
  - `TRY003`
  - `PLR0913`
  - `T201`
  - `E501`
- Import sorting treats `stack_pr` as first-party.
- McCabe max complexity: 16.
- `__init__.py` ignores `F401`.
- Pydocstyle convention: Google.
- Quote style: double quotes for docstrings, inline strings, and multiline strings.
- Tests additionally ignore `S101`, `ARG`, `FBT`, `ANN401`, `T201`, `PLR2004`.

### 23.2 Pytest

Pytest config:

- `testpaths = ["tests"]`.
- Test files: `test_*.py`.
- Addopts: `-v`.
- Asyncio mode: `auto`.
- Asyncio default fixture loop scope: `function`.

### 23.3 Mypy

Mypy strictness:

- Python version: 3.9.
- Warn return Any and unused configs.
- Disallow untyped and incomplete defs.
- Check untyped defs.
- Disallow untyped decorators.
- No implicit optional.
- Warn redundant casts, unused ignores, no return, unreachable.
- Strict optional enabled.

## 24. GitHub Actions

### 24.1 Test workflow

File: `.github/workflows/check_tests.yml`

- Name: `Check Tests`.
- Trigger: pull requests.
- Runner: `ubuntu-latest`.
- Steps:
  1. Checkout via pinned `actions/checkout` v4 SHA.
  2. Setup Python via pinned `actions/setup-python` v4 SHA with `python-version: '3.x'` and pip cache.
  3. Upgrade pip and install pytest.
  4. Run `pytest tests/`.

### 24.2 Lint workflow

File: `.github/workflows/lint.yml`

- Name: `Lint and check`.
- Trigger: pull requests.
- Runner: `ubuntu-latest`.
- Steps:
  1. Checkout via pinned `actions/checkout` v4 SHA.
  2. Run pinned `chartboost/ruff-action` v1.
  3. Run pinned `chartboost/ruff-action` v1 with `args: 'format --check'`.

### 24.3 Release workflow

File: `.github/workflows/release.yml`

- Name: `Upload Python Package`.
- Triggers: published GitHub releases and manual `workflow_dispatch`.
- Job: `pypi-publish`.
- Runner: `ubuntu-latest`.
- Environment: `release`.
- Permission: `id-token: write` for trusted publishing.
- Steps:
  1. Checkout via pinned `actions/checkout` v4 SHA.
  2. Setup Python 3.9 via pinned `actions/setup-python` v5 SHA.
  3. Setup PDM via pinned `pdm-project/setup-pdm` v3 SHA with cache.
  4. Build with `pdm build`.
  5. Publish with pinned `pypa/gh-action-pypi-publish` release/v1 SHA.

## 25. Documentation files

### 25.1 README

The README explains:

- What stacked PRs are.
- Installation via `pipx install stack-pr` or `pipx install .` from source.
- Dependency on `gh` and `gh auth login`.
- Basic workflow: branch from main, make commits, run `view`, run `submit`, amend/re-submit, rebase, land.
- Commands: `submit/export`, `view`, `abandon`, `land`, `config`.
- Commit range customization using `-B`, `-H`, `-T`.
- Full command-line option reference.
- Example config file.

### 25.2 CHANGELOG

Current entries:

- Top of tree.
- Version 0.1.3: fix `$USERNAME` replacement bug in branch names.
- Version 0.1.2: config files, branch name customization, branch deletion bug fix on abandon, suppressed subcommand output.
- Version 0.1.1 heading only.

### 25.3 CONTRIBUTING

Brief contribution guide covering issue filing, pull requests, maintainer review, and maintainer-controlled merges.

### 25.4 LICENSE

Apache License v2.0 with LLVM Exceptions.

## 26. Ignore and attributes files

`.gitignore` ignores common Python build artifacts, caches, virtual environments, coverage outputs, PDM/Pixi local state, `.stack-pr.cfg`, `.vscode`, and `.DS_Store`. `pdm.lock` is intentionally not ignored, despite a commented note.

`.gitattributes` marks `pixi.lock` as YAML and generated for GitHub Linguist:

```gitattributes
pixi.lock linguist-language=YAML linguist-generated=true
```

## 27. Reproduction checklist

To reproduce the project:

1. Create a Python package named `stack-pr` with source root `src/stack_pr`.
2. Provide empty `__init__.py` and `py.typed` files.
3. Implement `__main__.py` to import `main` from `stack_pr.cli` and call it under `if __name__ == "__main__"`.
4. Implement `shell_commands.py` as the subprocess wrapper described above.
5. Implement `git.py` with Git/GH helper functions and `GitConfig` username override.
6. Implement `cli.py` with:
   - regex constants,
   - ANSI output helpers,
   - `CommitHeader`, `StackEntry`, `CommonArgs`,
   - stack discovery, branch naming, metadata management,
   - `submit/export`, `view`, `land`, `abandon`, and `config` commands,
   - argparse and config loading,
   - main dispatch and safety handling.
7. Match `pyproject.toml` build metadata, dependencies, Ruff/Pytest/Mypy config, and console script.
8. Include README, changelog, contribution guide, Apache-2.0-with-LLVM-exceptions license, `.gitignore`, `.gitattributes`, and GitHub workflows.
9. Include tests for branch naming, rebase detection, and shell command quiet/non-quiet behavior.
10. Verify with `pytest -q`.

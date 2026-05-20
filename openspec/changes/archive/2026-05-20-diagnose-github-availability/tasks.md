## 1. Diagnosis Model and Check Ordering

- [ ] 1.1 Add a `github_availability` check to the diagnose check sequence before `online_pr_state`.
- [ ] 1.2 Ensure offline mode emits `github_availability` as `unknown` without invoking any `gh` command or network operation.
- [ ] 1.3 Ensure online mode skips the availability probe when `gh` is not installed and reports the condition without masking the existing `gh_installed` check.

## 2. GitHub Availability Probe

- [ ] 2.1 Implement a read-only GitHub availability probe through the existing diagnose `Runner` abstraction and `internal/shell` path.
- [ ] 2.2 Add classification logic for likely GitHub outage signals such as 5xx responses, service-unavailable messages, timeouts, and connection failures.
- [ ] 2.3 Keep authentication and authorization failures out of outage classification so `github_authentication` remains responsible for them.
- [ ] 2.4 Return a blocking `github_availability` entry for likely outages with `blocks` including `submit`, `land`, and `abandon`, plus a recovery-oriented `suggested_fix`.

## 3. Online PR-State Behavior

- [ ] 3.1 Make `online_pr_state` avoid trusting or claiming live PR synchronization when `github_availability` is blocking.
- [ ] 3.2 Preserve local-only check execution during likely outages, including repository, working tree, rebase, base/head, branch template, and stack discovery checks.
- [ ] 3.3 Preserve individual PR lookup reporting for non-outage repository-specific failures when GitHub is reachable.

## 4. Recommendation Engine

- [ ] 4.1 Add the GitHub-unavailable branch to the recommendation priority after dirty-tree handling and before missing-PR handling.
- [ ] 4.2 Ensure the outage recommendation does not use `stack-pr submit`, `stack-pr land`, or `stack-pr abandon` as the primary command.
- [ ] 4.3 Ensure the outage recommendation explains that live GitHub state cannot currently be trusted for mutating stack-pr operations.

## 5. Output and Documentation

- [ ] 5.1 Ensure JSON output includes the stable `github_availability` check entry in both offline and online modes.
- [ ] 5.2 Ensure Markdown output surfaces the availability check, its blocking status, and suggested fix when an outage is detected.
- [ ] 5.3 Update README/help text as needed to clarify that `--online` includes a GitHub availability probe and that outage findings block mutating recommendations.

## 6. Tests

- [ ] 6.1 Add diagnose tests proving offline mode emits `github_availability` without invoking `gh`.
- [ ] 6.2 Add tests for successful online availability probe behavior.
- [ ] 6.3 Add tests for likely outage classification and blocking check fields.
- [ ] 6.4 Add tests proving authentication/authorization failures are not classified as outages.
- [ ] 6.5 Add tests proving repository-specific PR lookup failures remain `online_pr_state` outcomes when availability succeeds.
- [ ] 6.6 Add recommendation tests for the GitHub-unavailable branch and its priority relative to dirty-tree and missing-PR cases.
- [ ] 6.7 Run `go test ./...`, `go vet ./...`, and gofmt check.

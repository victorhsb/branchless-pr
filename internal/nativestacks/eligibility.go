package nativestacks

import (
	"fmt"
	"strings"
)

// ValidateWritePlan validates repository identity, direct ref chaining, and the
// lifecycle of PRs that would be added by a create or append operation.
func ValidateWritePlan(plan *Result, prs map[int]*PullRequest, stacks StackSet, repoSlug string) error {
	if plan == nil || !plan.IsWriteAction() {
		return nil
	}
	if len(plan.LocalPRs) > 100 {
		return fmt.Errorf("branchless-pr conservatively limits native Stacks to 100 total PRs; GitHub documents 100 per request but not an aggregate limit")
	}
	seen := make(map[int]struct{}, len(plan.LocalPRs))
	for _, number := range plan.LocalPRs {
		if _, ok := seen[number]; ok {
			return fmt.Errorf("local native Stack plan contains duplicate PR #%d", number)
		}
		seen[number] = struct{}{}
		pr := prs[number]
		if pr == nil {
			return fmt.Errorf("native Stack plan is missing pull request #%d", number)
		}
		if pr.Head.Repo == nil || !strings.EqualFold(pr.Head.Repo.FullName, repoSlug) {
			got := "<unknown>"
			if pr.Head.Repo != nil {
				got = pr.Head.Repo.FullName
			}
			return fmt.Errorf("pull request #%d head belongs to %s, expected %s", number, got, repoSlug)
		}
		if pr.Base.Repo != nil && !strings.EqualFold(pr.Base.Repo.FullName, repoSlug) {
			return fmt.Errorf("pull request #%d base belongs to %s, expected %s", number, pr.Base.Repo.FullName, repoSlug)
		}
	}

	for i, number := range plan.LocalPRs {
		pr := prs[number]
		if i == 0 {
			if plan.Kind == ActionAppend {
				s := stacks[plan.StackNumber]
				if s == nil {
					return fmt.Errorf("append plan references missing native Stack #%d", plan.StackNumber)
				}
				if pr.Base.Ref != s.Base.Ref {
					return fmt.Errorf("bottom pull request #%d targets %q, expected Stack base %q", number, pr.Base.Ref, s.Base.Ref)
				}
			}
			continue
		}
		previous := prs[plan.LocalPRs[i-1]]
		if pr.Base.Ref != previous.Head.Ref {
			return fmt.Errorf("pull request #%d targets %q, expected previous head %q from PR #%d", number, pr.Base.Ref, previous.Head.Ref, previous.Number)
		}
	}

	candidates := plan.LocalPRs
	if plan.Kind == ActionAppend {
		candidates = plan.SuffixPRs
	}
	for _, number := range candidates {
		pr := prs[number]
		switch {
		case pr.MergedAt != nil:
			return fmt.Errorf("pull request #%d is already merged and cannot be added to a native Stack", number)
		case pr.State != "open":
			return fmt.Errorf("pull request #%d is %s and cannot be added to a native Stack", number, pr.State)
		case pr.IsQueued():
			return fmt.Errorf("pull request #%d is merge-queued and cannot be added to a native Stack", number)
		case pr.IsAutoMergeEnabled():
			return fmt.Errorf("pull request #%d has auto-merge enabled and cannot be added to a native Stack", number)
		}
	}
	return nil
}

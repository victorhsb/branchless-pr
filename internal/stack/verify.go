package stack

import (
	"fmt"

	"github.com/victorhsb/branchless-pr/internal/pr"
)

// PRInfoProvider returns GitHub state for a PR reference.
type PRInfoProvider func(prRef string) (*pr.Info, error)

// VerifyWithProvider validates every StackEntry using the supplied PR info provider.
func VerifyWithProvider(st Stack, checkBase bool, provider PRInfoProvider) error {
	return verifyWithLookup(st, checkBase, provider)
}

// VerifyWithInfo validates every StackEntry with a caller-provided PR info
// lookup. Missing entries are rejected rather than crossing the GitHub boundary
// from the stack package.
func VerifyWithInfo(st Stack, checkBase bool, lookup func(prRef string) (*pr.Info, bool)) error {
	return verifyWithLookup(st, checkBase, func(prRef string) (*pr.Info, error) {
		info, ok := lookupInfo(lookup, prRef)
		if ok {
			return info, nil
		}
		return nil, fmt.Errorf("PR state for %s was not provided", prRef)
	})
}

func verifyWithLookup(st Stack, checkBase bool, lookup func(prRef string) (*pr.Info, error)) error {
	for i, e := range st {
		if e.HasMissingInfo() {
			return fmt.Errorf("\033[91mERROR: Cannot verify stack: entry %d is missing PR, head, or base info.\033[0m", i)
		}

		prNum, err := e.PRNumber()
		if err != nil {
			return fmt.Errorf("\033[91mERROR: Stack entry %d has malformed PR link: %v\033[0m", i, err)
		}

		info, err := lookup(e.PR())
		if err != nil {
			return fmt.Errorf("\033[91mERROR: Cannot verify stack: unable to query PR #%d: %v\033[0m", prNum, err)
		}

		if info.State != "OPEN" {
			return fmt.Errorf("\033[91mERROR: Associated PR #%d is not open (state=%s).\033[0m", info.Number, info.State)
		}

		if info.Number != prNum {
			return fmt.Errorf(
				"\033[91mERROR: PR number mismatch for entry %d: metadata says #%d, GitHub says #%d.\033[0m",
				i, prNum, info.Number,
			)
		}

		if info.HeadRefName != e.Head() {
			return fmt.Errorf(
				"\033[91mERROR: PR head branch mismatch for #%d: metadata says %q, GitHub says %q.\033[0m",
				info.Number, e.Head(), info.HeadRefName,
			)
		}

		if checkBase {
			if info.BaseRefName != e.Base() {
				return fmt.Errorf(
					"\033[91mERROR: PR base branch mismatch for #%d: metadata says %q, GitHub says %q.\033[0m",
					info.Number, e.Base(), info.BaseRefName,
				)
			}
			if i == 0 {
				ok := info.MergeStateStatus == "CLEAN" || info.MergeStateStatus == "UNKNOWN" || info.MergeStateStatus == "UNSTABLE"
				if !ok {
					return fmt.Errorf(
						"\033[91mERROR: Bottom PR #%d is not mergeable (mergeStateStatus=%s).\033[0m",
						info.Number, info.MergeStateStatus,
					)
				}
			}
		}
	}
	return nil
}

func lookupInfo(lookup func(prRef string) (*pr.Info, bool), prRef string) (*pr.Info, bool) {
	if lookup == nil {
		return nil, false
	}
	info, ok := lookup(prRef)
	return info, ok && info != nil
}

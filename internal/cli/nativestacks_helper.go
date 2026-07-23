package cli

import (
	"fmt"
	"os"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/nativestacks"
	"github.com/victorhsb/branchless-pr/internal/receipts"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

// pushWithLeases pushes branches using force-with-lease when leases are
// provided, otherwise falls back to a plain force push.
func pushWithLeases(remote string, heads []string, leases map[string]string) error {
	if len(leases) == 0 {
		return git.ForcePush(remote, heads...)
	}
	return git.ForcePushWithLease(remote, leases, heads...)
}

// effectiveReceiptDestination resolves the receipt destination from the flag or
// config, defaulting to "off".
func effectiveReceiptDestination(cfg *config.Config, flag string) string {
	if flag != "" {
		return flag
	}
	return cfg.Get("receipt", "submit")
}

// newSubmitReceipt builds a receipt envelope from the invocation context.
func newSubmitReceipt(app *AppContext, command string, st stack.Stack) *receipts.Receipt {
	repoCtx := receipts.RepoContext{
		Root:               app.RepoRoot,
		OriginalBranch:     app.OrigBranch,
		Remote:             app.Args.Remote,
		Target:             app.Args.Target,
		Base:               app.Args.Base,
		Head:               app.Args.Head,
		BranchNameTemplate: app.Args.BranchNameTemplate,
	}
	stackCtx := receipts.StackContext{Size: len(st)}
	for _, e := range st {
		prURL := ""
		if e.HasPR() {
			prURL = e.PR()
		}
		stackCtx.Entries = append(stackCtx.Entries, receipts.StackEntryCtx{
			Commit:     e.Commit.SHA,
			Title:      e.Commit.Title,
			HeadBranch: e.Head(),
			BaseBranch: e.Base(),
			PRURL:      prURL,
		})
	}
	return receipts.NewReceipt(command, repoCtx, stackCtx)
}

// nativeMode returns the configured native-stacks mode, or off on error.
func nativeMode(cfg *config.Config) (config.NativeStacksMode, error) {
	return config.ParseNativeStacksMode(cfg.Get("github", "native_stacks"))
}

// nativeSubmitPreflight probes native Stack availability and the gh-stack
// extension when native mode is enabled. It returns a result describing whether
// native reconciliation should proceed, along with availability/extension state.
// For auto mode it warns and disables native behavior when unavailable.
func nativeSubmitPreflight(app *AppContext, st stack.Stack, mode config.NativeStacksMode) (*nativePreflightResult, error) {
	if mode == config.NativeStacksOff {
		return &nativePreflightResult{enabled: false}, nil
	}

	if len(st) < 2 {
		// Single-PR stacks are standalone.
		return &nativePreflightResult{enabled: false}, nil
	}

	prNumbers := localPRNumbers(st)

	owner, repo, err := git.RepoSlug(app.Args.Remote)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve owner/repo for native submit: %w", err)
	}

	client := nativestacks.NewAPIClient(owner, repo)
	if err := client.ProbeAvailability(); err != nil {
		if nativestacks.IsFeatureUnavailable(err) {
			if mode == config.NativeStacksAuto {
				fmt.Fprintln(os.Stderr, "Warning: native Stacks is unavailable; using legacy submit.")
				return &nativePreflightResult{enabled: false, fallback: "native Stacks unavailable"}, nil
			}
			return nil, &nativestacks.FeatureUnavailable{Msg: "native Stacks is required but unavailable for this repository"}
		}
		return nil, fmt.Errorf("cannot probe native Stacks availability: %w", err)
	}

	// Load membership for existing PRs to decide whether a write is needed.
	var result *nativestacks.Result
	if len(prNumbers) == 0 {
		// All PRs will be created: prospective native Stack create.
		result = &nativestacks.Result{Kind: nativestacks.ActionCreate}
	} else {
		memberships, stacks, err := client.LoadMembership(prNumbers)
		if err != nil {
			if nativestacks.IsFeatureUnavailable(err) {
				if mode == config.NativeStacksAuto {
					fmt.Fprintln(os.Stderr, "Warning: native Stacks is unavailable; using legacy submit.")
					return &nativePreflightResult{enabled: false, fallback: "native Stacks unavailable"}, nil
				}
				return nil, &nativestacks.FeatureUnavailable{Msg: "native Stacks is required but unavailable for this repository"}
			}
			return nil, fmt.Errorf("cannot load native membership: %w", err)
		}
		result = nativestacks.Classify(prNumbers, memberships, stacks)
	}

	// Determine if the extension is required for the planned action.
	needsWriter := result.IsWriteAction()
	if needsWriter {
		ext, err := nativestacks.FindExtension()
		if err != nil {
			return nil, fmt.Errorf("cannot check gh-stack extension: %w", err)
		}
		if !ext.Installed {
			if mode == config.NativeStacksAuto && result.Kind == nativestacks.ActionCreate {
				// Safe fallback: no existing native Stack would be left inconsistent.
				fmt.Fprintln(os.Stderr, "Warning: gh-stack extension is missing; skipping native Stack creation.")
				return &nativePreflightResult{enabled: false, fallback: "gh-stack extension missing"}, nil
			}
			return nil, &nativestacks.ErrExtensionMissing{Min: nativestacks.MinimumExtensionVersion}
		}
		if err := nativestacks.ValidateExtensionVersion(ext.Version, nativestacks.MinimumExtensionVersion); err != nil {
			return nil, err
		}
	}

	if result.Kind == nativestacks.ActionIneligible {
		if mode == config.NativeStacksAuto {
			fmt.Fprintf(os.Stderr, "Warning: native Stack ineligible: %s; using legacy submit.\n", result.Conflict)
			return &nativePreflightResult{enabled: false, fallback: result.Conflict}, nil
		}
		return nil, fmt.Errorf("native Stacks is required but the stack is ineligible: %s", result.Conflict)
	}

	return &nativePreflightResult{
		enabled:   true,
		client:    client,
		owner:     owner,
		repo:      repo,
		prNumbers: prNumbers,
		plan:      result,
	}, nil
}

// reconcileNativeStack applies the planned native Stack action and verifies
// the result through REST. It is called after all ordinary submit effects.
func reconcileNativeStack(pf *nativePreflightResult) (*nativestacks.Result, error) {
	if pf == nil || !pf.enabled {
		return nil, nil
	}
	result := pf.plan
	switch result.Kind {
	case nativestacks.ActionNoop:
		return result, nil
	case nativestacks.ActionCreate:
		if err := nativestacks.LinkCreate(result.LocalPRs); err != nil {
			return result, fmt.Errorf("native Stack create failed: %w", err)
		}
	case nativestacks.ActionAppend:
		if err := nativestacks.LinkAppend(result.StackNumber, result.SuffixPRs); err != nil {
			return result, fmt.Errorf("native Stack append failed: %w", err)
		}
	case nativestacks.ActionConflict:
		return result, fmt.Errorf("native membership conflict: %s", result.Conflict)
	default:
		return result, fmt.Errorf("unexpected native action %q", result.Kind)
	}

	// Verify the resulting Stack matches the planned sequence.
	if result.IsWriteAction() {
		s, err := pf.client.GetStack(result.StackNumber)
		if err != nil {
			return result, fmt.Errorf("cannot verify native Stack after write: %w", err)
		}
		got := make([]int, len(s.PRs))
		for i, pr := range s.PRs {
			got[i] = pr.Number
		}
		if !intSliceEqual(got, result.LocalPRs) {
			return result, fmt.Errorf("native Stack verification failed: remote sequence %v != planned %v", got, result.LocalPRs)
		}
	}

	return result, nil
}

type nativePreflightResult struct {
	enabled     bool
	client      *nativestacks.APIClient
	owner       string
	repo        string
	prNumbers   []int
	memberships map[int]*nativestacks.Membership
	stacks      nativestacks.StackSet
	plan        *nativestacks.Result
	fallback    string
}

func printNativeDryRunPlan(pf *nativePreflightResult) {
	if pf == nil || !pf.enabled {
		fmt.Println("Native Stack integration: disabled or skipped")
		return
	}
	fmt.Printf("Native Stack action: %s\n", pf.plan.Kind)
	if pf.plan.StackNumber != 0 {
		fmt.Printf("  stack number: %d\n", pf.plan.StackNumber)
	}
	if len(pf.plan.LocalPRs) > 0 {
		fmt.Printf("  local PRs: %s\n", nativestacks.FormatPRList(pf.plan.LocalPRs))
	}
	if len(pf.plan.SuffixPRs) > 0 {
		fmt.Printf("  suffix PRs: %s\n", nativestacks.FormatPRList(pf.plan.SuffixPRs))
	}
	if pf.plan.Conflict != "" {
		fmt.Printf("  conflict: %s\n", pf.plan.Conflict)
	}
	if pf.fallback != "" {
		fmt.Printf("  fallback: %s\n", pf.fallback)
	}
}

func localPRNumbers(st stack.Stack) []int {
	var nums []int
	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		n, err := e.PRNumber()
		if err != nil {
			continue
		}
		nums = append(nums, n)
	}
	return nums
}

func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

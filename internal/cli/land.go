package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

func landCmd() *cobra.Command {
	var wholeStackFlag bool
	cmd := &cobra.Command{
		Use:   "land",
		Short: "Land the bottom-most PR in the stack.",
		Long: `Land stacked PRs into the target branch.

The default "bottom-only" style squash-merges the bottom PR and rebases the
rest of the stack. The "whole-stack" style (set via land.style or the
--whole-stack flag) retargets the tip PR to the target branch and queues a
GitHub rebase auto-merge so the entire stack lands in a single operation.
Whole-stack requires that the repository target branch has GitHub merge queue
enabled.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			style := effectiveLandStyle(app, wholeStackFlag)
			return WithRecovery(app, func() error { return landImpl(app, style) })
		},
	}
	cmd.Flags().BoolVar(&wholeStackFlag, "whole-stack", false, "Land the entire stack via rebase-merge of the tip PR (overrides land.style)")
	return cmd
}

// effectiveLandStyle resolves the active land style for a single invocation.
// The --whole-stack flag wins; otherwise the configured land.style is used,
// defaulting to bottom-only when unset or unrecognised.
func effectiveLandStyle(app *AppContext, wholeStackFlag bool) string {
	if wholeStackFlag {
		return "whole-stack"
	}
	switch app.Config.Get("land", "style") {
	case "whole-stack":
		return "whole-stack"
	default:
		return "bottom-only"
	}
}

func landImpl(app *AppContext, style string) error {
	// 3. Optionally fast-forward local base.
	if err := maybeRebaseBase(app); err != nil {
		return err
	}

	// 4. Discover stack.
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
	if err != nil {
		return err
	}

	// 5. Empty stack.
	if st.IsEmpty() {
		fmt.Println("Empty stack!")
		return nil
	}

	// 6. Read metadata, assign bases, print stack.
	for _, e := range st {
		e.ReadMetadata()
	}
	st.AssignBases(app.Args.Target)
	fmt.Println("Stack:")
	st.PrintStack(app.Args.Hyperlinks, true)
	fmt.Println()

	// 7. Verify the stack against GitHub with check_base=true.
	if err := stack.Verify(st, true); err != nil {
		return err
	}

	// 8. Dispatch to the selected landing strategy.
	if style == "whole-stack" {
		return landWholeStack(app, st)
	}
	return landBottomOnly(app, st)
}

func landBottomOnly(app *AppContext, st stack.Stack) error {
	bottom := st.Bottom()
	if err := git.Fetch(app.Args.Remote); err != nil {
		return err
	}
	if err := git.Checkout(app.Args.Remote+"/"+bottom.Head(), bottom.Head()); err != nil {
		return fmt.Errorf("ERROR: Cannot checkout remote branch while landing: %w", err)
	}
	if err := pr.EditBase(bottom.PR(), app.Args.Target); err != nil {
		return fmt.Errorf("ERROR: Cannot set base on bottom PR: %w", err)
	}

	prNum, err := bottom.PRNumber()
	if err != nil {
		return err
	}
	titleLine := strings.SplitN(bottom.Commit.Title, "\n", 2)[0]
	squashTitle := fmt.Sprintf("%s (#%d)", titleLine, prNum)
	squashBody := stripStackInfo(bottom.Commit.Body)
	squashBody = strings.TrimSpace(squashBody)
	if squashBody == "" {
		squashBody = " "
	}
	if err := pr.MergeSquash(bottom.PR(), squashTitle, []byte(squashBody)); err != nil {
		return fmt.Errorf("ERROR: Cannot merge bottom PR: %w", err)
	}

	// 9. Rebase remaining stack entries.
	if len(st) > 1 {
		remaining := st[1:]
		fmt.Println("Rebasing the rest of the stack")
		for _, e := range remaining.Reverse() {
			fmt.Println(e.PrettyLine(app.Args.Hyperlinks, true))
		}
		fmt.Println()

		remoteTarget := app.Args.Remote + "/" + app.Args.Target
		for _, e := range remaining {
			if err := git.Fetch(app.Args.Remote); err != nil {
				return err
			}
			if err := git.Checkout(app.Args.Remote+"/"+e.Head(), e.Head()); err != nil {
				return fmt.Errorf("ERROR: Cannot checkout remote branch %q while landing: %w", e.Head(), err)
			}
			if err := git.RebaseWithAuthorDate(remoteTarget, e.Head()); err != nil {
				return fmt.Errorf("ERROR: Cannot rebase %q onto %s: %w", e.Head(), remoteTarget, err)
			}
			if err := git.ForcePush(app.Args.Remote, e.Head()); err != nil {
				return fmt.Errorf("ERROR: Cannot push %q: %w", e.Head(), err)
			}
		}

		// Set the new bottom PR base to target.
		newBottom := remaining[0]
		if err := pr.EditBase(newBottom.PR(), app.Args.Target); err != nil {
			return fmt.Errorf("ERROR: Cannot update new bottom PR base: %w", err)
		}
	}

	// 10-13. Cleanup: restore original branch, delete locals, rebase target+orig.
	return landCleanup(app, st)
}

// landWholeStack lands every PR in the stack atomically by retargeting the
// tip PR to the target branch and performing a GitHub rebase merge.
// Pre-flight steps 1-7 in landImpl have already discovered/verified the stack.
var landWholeStack = landWholeStackImpl

func landWholeStackImpl(app *AppContext, st stack.Stack) error {
	owner, repo, err := git.RepoSlug(app.Args.Remote)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot resolve owner/repo from remote %q: %w", app.Args.Remote, err)
	}
	allowed, err := pr.RebaseMergeAllowed(owner, repo)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot query repository merge settings: %w", err)
	}
	if !allowed {
		return fmt.Errorf("ERROR: Repository %s/%s does not allow rebase merges. Enable rebase merges in repository settings or use land.style = bottom-only.", owner, repo)
	}

	mqStatus, err := pr.MergeQueueEnabled(owner, repo, app.Args.Target)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot query repository merge queue settings: %w", err)
	}
	if mqStatus == pr.MergeQueueStatusDisabled {
		return fmt.Errorf("ERROR: --whole-stack only works for repositories with merge queue enabled")
	}

	if err := git.Fetch(app.Args.Remote); err != nil {
		return err
	}

	tip := st.Top()
	if err := pr.EditBase(tip.PR(), app.Args.Target); err != nil {
		return fmt.Errorf("ERROR: Cannot set base on tip PR: %w", err)
	}
	if err := pr.MergeRebaseAuto(tip.PR()); err != nil {
		return err
	}

	if err := git.CheckoutBranch(app.OrigBranch); err != nil {
		return fmt.Errorf("ERROR: Cannot checkout original branch: %w", err)
	}
	fmt.Printf("Whole-stack landing has been queued for %s\n", tip.PR())
	return nil
}

// landCleanup restores the original branch, deletes local stack branches, and
// rebases the local target plus original branch onto REMOTE/TARGET. Shared
// between bottom-only and whole-stack landing.
func landCleanup(app *AppContext, st stack.Stack) error {
	if err := git.CheckoutBranch(app.OrigBranch); err != nil {
		return fmt.Errorf("ERROR: Cannot checkout original branch: %w", err)
	}

	heads := make([]string, 0, len(st))
	for _, e := range st {
		heads = append(heads, e.Head())
	}
	git.DeleteLocalBranches(heads...)

	remoteTarget := app.Args.Remote + "/" + app.Args.Target
	if exists, _ := git.BranchExists(app.Args.Target); exists {
		if err := git.Rebase(remoteTarget, app.Args.Target); err != nil {
			return fmt.Errorf("ERROR: Cannot rebase local target %q: %w", app.Args.Target, err)
		}
	}
	if err := git.Rebase(remoteTarget, app.OrigBranch); err != nil {
		return fmt.Errorf("ERROR: Cannot rebase original branch %q: %w", app.OrigBranch, err)
	}
	return nil
}

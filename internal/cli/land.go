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
	return &cobra.Command{
		Use:   "land",
		Short: "Land the bottom-most PR in the stack.",
		Long:  `Squash-merges the bottom PR and rebases the rest of the stack.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			return WithRecovery(app, func() error { return landImpl(app) })
		},
	}
}

func landImpl(app *AppContext) error {
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

	// 8. Land the bottom-most PR.
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

	// 10. Checkout original branch.
	if err := git.CheckoutBranch(app.OrigBranch); err != nil {
		return fmt.Errorf("ERROR: Cannot checkout original branch: %w", err)
	}

	// 11. Delete local stack branches.
	heads := make([]string, 0, len(st))
	for _, e := range st {
		heads = append(heads, e.Head())
	}
	git.DeleteLocalBranches(heads...)

	// 12. If a local target branch exists, rebase it onto REMOTE/TARGET.
	if exists, _ := git.BranchExists(app.Args.Target); exists {
		if err := git.Rebase(app.Args.Remote+"/"+app.Args.Target, app.Args.Target); err != nil {
			return fmt.Errorf("ERROR: Cannot rebase local target %q: %w", app.Args.Target, err)
		}
	}

	// 13. Rebase the original branch onto REMOTE/TARGET.
	if err := git.Rebase(app.Args.Remote+"/"+app.Args.Target, app.OrigBranch); err != nil {
		return fmt.Errorf("ERROR: Cannot rebase original branch %q: %w", app.OrigBranch, err)
	}

	return nil
}

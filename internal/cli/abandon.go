package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

func abandonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abandon",
		Short: "Remove stack metadata and delete generated branches.",
		Long:  `Strips stack metadata from commits and deletes local/remote generated branches.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			return WithRecovery(app, func() error { return abandonImpl(app) })
		},
	}
}

func abandonImpl(app *AppContext) error {
	fmt.Println(stack.Headerf("ABANDON"))

	// 2. Discover stack.
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
	if err != nil {
		return err
	}

	// 3. Empty stack.
	if st.IsEmpty() {
		fmt.Println("Empty stack!")
		fmt.Println(stack.Greenf("SUCCESS!"))
		return nil
	}

	// 5. Read metadata; for entries lacking heads, assign new ones from the template.
	for _, e := range st {
		e.ReadMetadata()
	}
	if err := git.Fetch(app.Args.Remote); err != nil {
		return err
	}
	tmpl := stack.ParseTemplate(app.Args.BranchNameTemplate)
	if err := st.AssignHeads(tmpl, app.Username, app.OrigBranch, app.Args.Remote); err != nil {
		return err
	}
	// Materialize local branches for each entry pointing at its commit.
	for _, e := range st {
		if err := git.Checkout(e.Commit.SHA, e.Head()); err != nil {
			return err
		}
	}

	// 6. Set base branches.
	st.AssignBases(app.Args.Target)

	// 7. Print stack.
	fmt.Println("Stack:")
	st.PrintStack(app.Args.Hyperlinks, true)
	fmt.Println()

	// 8. Strip metadata from each commit, rebasing each on top of the previous as needed.
	var newTopSHA string
	for i, e := range st {
		stripped := stripStackInfo(e.Commit.CommitMsg())
		stripped = strings.TrimRight(stripped, "\n") + "\n"

		if i == 0 {
			if err := git.CheckoutBranch(e.Head()); err != nil {
				return err
			}
		} else {
			if err := git.RebaseWithAuthorDate(e.Base(), e.Head()); err != nil {
				return fmt.Errorf("ERROR: Cannot rebase %q during abandon: %w", e.Head(), err)
			}
		}
		if err := git.CommitAmend([]byte(stripped)); err != nil {
			return fmt.Errorf("ERROR: Cannot strip stack metadata from %q: %w", e.Head(), err)
		}
		sha, err := git.RevParse(e.Head())
		if err != nil {
			return err
		}
		newTopSHA = sha
	}

	// 9. Rebase the original branch on top of the new top.
	if newTopSHA != "" {
		if err := git.RebaseWithAuthorDate(newTopSHA, app.OrigBranch); err != nil {
			return fmt.Errorf("ERROR: Cannot rebase original branch onto stripped stack: %w", err)
		}
	}

	// 10. Delete local generated branches (best-effort).
	heads := make([]string, 0, len(st))
	for _, e := range st {
		heads = append(heads, e.Head())
	}
	git.DeleteLocalBranches(heads...)

	// 11. Delete remote branches that both match the template AND are heads of stack entries.
	remoteDel := make([]string, 0, len(heads))
	for _, h := range heads {
		if tmpl.Match(h, app.Username, app.OrigBranch) {
			remoteDel = append(remoteDel, h)
		}
	}
	if len(remoteDel) > 0 {
		if err := git.DeleteRemoteBranches(app.Args.Remote, remoteDel...); err != nil {
			fmt.Printf("Warning: failed to delete some remote branches: %v\n", err)
		}
	}

	fmt.Println(stack.Greenf("SUCCESS!"))
	return nil
}

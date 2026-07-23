package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/nativestacks"
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

func nativeAbandonPreflight(app *AppContext, st stack.Stack) error {
	mode, err := config.ParseNativeStacksMode(app.Config.Get("github", "native_stacks"))
	if err != nil {
		return err
	}
	if mode == config.NativeStacksOff {
		return nil
	}

	prNumbers := make([]int, 0, len(st))
	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		n, err := e.PRNumber()
		if err != nil {
			return err
		}
		prNumbers = append(prNumbers, n)
	}
	if len(prNumbers) < 2 {
		// Single PR stacks are standalone; legacy cleanup is safe.
		return nil
	}

	owner, repo, err := git.RepoSlug(app.Args.Remote)
	if err != nil {
		return fmt.Errorf("cannot resolve owner/repo for native abandon check: %w", err)
	}

	client := nativestacks.NewAPIClient(owner, repo)
	memberships, stacks, err := client.LoadMembership(prNumbers)
	if err != nil {
		if nativestacks.IsFeatureUnavailable(err) {
			if mode == config.NativeStacksAuto {
				fmt.Fprintln(os.Stderr, "Warning: native Stacks is unavailable; using legacy abandon.")
				return nil
			}
		}
		return fmt.Errorf("cannot load native membership for abandon: %w", err)
	}

	result := nativestacks.Classify(prNumbers, memberships, stacks)
	switch result.Kind {
	case nativestacks.ActionNoop:
		// Exact native membership: unstack before deleting remote branches.
		ext, err := nativestacks.FindExtension()
		if err != nil {
			return fmt.Errorf("cannot check gh-stack extension: %w", err)
		}
		if !ext.Installed {
			return &nativestacks.ErrExtensionMissing{Min: nativestacks.MinimumExtensionVersion}
		}
		if err := nativestacks.ValidateExtensionVersion(ext.Version, nativestacks.MinimumExtensionVersion); err != nil {
			return err
		}
		if err := nativestacks.Unstack(result.StackNumber); err != nil {
			return fmt.Errorf("failed to unstack native Stack #%d: %w", result.StackNumber, err)
		}
		// Verify unstack result.
		s, err := client.GetStack(result.StackNumber)
		if err != nil && !nativestacks.IsFeatureUnavailable(err) {
			return fmt.Errorf("cannot verify native unstack result: %w", err)
		}
		if s != nil && len(s.PRs) > 0 {
			for _, pr := range s.PRs {
				// If an unmerged local PR remains stacked, stop before branch deletion.
				if pr.State != "MERGED" {
					for _, n := range prNumbers {
						if pr.Number == n {
							return fmt.Errorf("native unstack left unmerged PR #%d stacked; stopping before remote branch deletion", n)
						}
					}
				}
			}
		}
		return nil
	case nativestacks.ActionCreate:
		if mode == config.NativeStacksAuto {
			return nil // legacy cleanup for unstacked PRs
		}
		return fmt.Errorf("native Stacks is required but the stack is not linked; cannot abandon safely")
	case nativestacks.ActionConflict:
		return fmt.Errorf("native membership conflict blocks abandon: %s", result.Conflict)
	case nativestacks.ActionAppend:
		// Partial membership means an existing Stack would be left inconsistent.
		return fmt.Errorf("native membership conflict blocks abandon: %s", result.Conflict)
	default:
		return nil
	}
}

func abandonImpl(app *AppContext) error {
	// 2. Discover stack.
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
	if err != nil {
		return err
	}

	// 3. Empty stack.
	if st.IsEmpty() {
		fmt.Println("Empty stack!")
		return nil
	}

	// 4. Native unstack preflight before any local mutation.
	if err := nativeAbandonPreflight(app, st); err != nil {
		return err
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

	return nil
}

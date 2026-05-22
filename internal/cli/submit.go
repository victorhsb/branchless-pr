package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

func submitCmd() *cobra.Command {
	var (
		draft     bool
		keepBody  bool
		draftMask string
		reviewer  string
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Create or update the stack of PRs.",
		Long: `Creates or updates stacked PRs for commits in BASE..HEAD.

Use --dry-run to preview the planned actions without applying any local Git or GitHub changes.`,
		Aliases: []string{"export"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSubmit(cmd, dryRun, draft, keepBody, draftMask, reviewer)
		},
	}

	cmd.Flags().BoolVar(&keepBody, "keep-body", false, "Preserve current PR body after stack cross-link section")
	cmd.Flags().BoolVarP(&draft, "draft", "d", false, "Create all new PRs as draft")
	cmd.Flags().StringVar(&draftMask, "draft-bitmask", "", "Per-PR draft bitmask; chars must be 0 or 1")
	cmd.Flags().StringVar(&reviewer, "reviewer", "", "Reviewer list; default from STACK_PR_DEFAULT_REVIEWER or config repo.reviewer")
	cmd.Flags().BoolVarP(&flagStash, "stash", "s", false, "Stash uncommitted changes before submitting and pop afterward")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview submit/export actions without applying local Git or GitHub changes")

	return cmd
}

func runSubmit(cmd *cobra.Command, dryRun, draft, keepBody bool, draftBitmask, reviewer string) error {
	app, ok := FromContext(cmd.Context())
	if !ok {
		return fmt.Errorf("missing app context")
	}

	// Resolve reviewer with precedence: arg > env > config.
	if reviewer == "" {
		reviewer = DefaultReviewer(app.Config, reviewer)
	}

	return WithRecovery(app, func() error {
		return submitImpl(app, dryRun, draft, keepBody, draftBitmask, reviewer)
	})
}

func submitImpl(app *AppContext, dryRun, draft, keepBody bool, draftBitmask, reviewer string) (err error) {
	// SPEC §8 step 21: pop stash on success (recovery handles error path).
	defer func() {
		if err == nil && app.StashCreated {
			if perr := git.StashPop(); perr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to pop stash: %v\n", perr)
			}
		}
	}()

	// 2. Rebase guard.
	if git.IsRebaseInProgress() {
		return fmt.Errorf("ERROR: Rebase in progress. Finish or abort it before submitting")
	}

	// 4. Optionally fast-forward local base (skipped in dry-run; mutating).
	if !dryRun {
		if err := maybeRebaseBase(app); err != nil {
			return err
		}
	}

	// 5. Discover stack.
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
	if err != nil {
		return err
	}

	// 6. Empty stack.
	if st.IsEmpty() {
		fmt.Println("Empty stack!")
		if dryRun {
			fmt.Println(dryRunNoChangesNote)
		}
		return nil
	}

	// 7. Read metadata and track which entries need amendment.
	needsMeta := make([]bool, len(st))
	for i, e := range st {
		if !e.ReadMetadata() {
			needsMeta[i] = true
		}
	}

	// 7. Validate draft bitmask.
	isDraft, ok := resolveDraftFlags(draft, draftBitmask, len(st))
	if !ok {
		return nil
	}

	// 8. Initialize local branches.
	if err := git.Fetch(app.Args.Remote); err != nil {
		return err
	}
	tmpl := stack.ParseTemplate(app.Args.BranchNameTemplate)
	if err := st.AssignHeads(tmpl, app.Username, app.OrigBranch, app.Args.Remote); err != nil {
		return err
	}

	// 9. Compute base branches (pure assignment, no git ops).
	st.AssignBases(app.Args.Target)

	// Dry-run: print plan and exit before any mutating operation.
	if dryRun {
		printDryRunPlan(st, needsMeta, isDraft)
		return nil
	}

	for _, e := range st {
		if err := git.Checkout(e.Commit.SHA, e.Head()); err != nil {
			return err
		}
	}

	// 10. Does the original branch need rebasing on top later?
	needsBranchRebase := false
	if top := st.Top(); top != nil {
		needsBranchRebase, _ = git.IsAncestor(top.Head(), app.OrigBranch)
	}

	// 11. Temporarily draft existing PRs and reset bases to target.
	var tmpDraftPRs []string
	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		info, err := pr.View(e.PR())
		if err != nil {
			return fmt.Errorf("ERROR: Cannot verify stack: %w", err)
		}
		if !info.IsDraft {
			if err := pr.ReadyUndo(e.PR()); err != nil {
				return fmt.Errorf("ERROR: Cannot update PR draft state: %w", err)
			}
			e.IsTmpDraft = true
			tmpDraftPRs = append(tmpDraftPRs, e.PR())
		}
		if err := pr.EditBase(e.PR(), app.Args.Target); err != nil {
			return fmt.Errorf("ERROR: Cannot reset PR base: %w", err)
		}
	}

	// 12. Force-push all heads.
	heads := make([]string, len(st))
	for i, e := range st {
		heads[i] = e.Head()
	}
	if err := git.ForcePush(app.Args.Remote, heads...); err != nil {
		return fmt.Errorf("ERROR: Cannot push branches: %w", err)
	}

	// 13. Create missing PRs.
	for i, e := range st {
		if e.HasPR() {
			continue
		}
		body := stack.BuildPRBody(e, st, false, "")
		opts := pr.CreateOptions{
			Base:     e.Base(),
			Head:     e.Head(),
			Title:    e.Commit.Title,
			Body:     body,
			Reviewer: reviewer,
			Draft:    isDraft[i],
		}
		prURL, err := pr.Create(opts)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot create a PR: %w", err)
		}
		e.SetPR(prURL)
	}

	// 14. Verify.
	if err := stack.Verify(st, false); err != nil {
		return err
	}

	// 15. Print stack.
	fmt.Println("Stack:")
	st.PrintStack(app.Args.Hyperlinks, true)
	fmt.Println()

	// 16. Add metadata to commit messages.
	metaModified := false
	for i, e := range st {
		if needsMeta[i] {
			if !metaModified {
				if err := git.CheckoutBranch(e.Head()); err != nil {
					return err
				}
			} else {
				if err := git.RebaseWithAuthorDate(e.Base(), e.Head()); err != nil {
					return err
				}
			}
			msg := []byte(e.Commit.CommitMsg() + e.MetadataLine())
			if err := git.CommitAmend(msg); err != nil {
				return fmt.Errorf("ERROR: Cannot update stack metadata: %w", err)
			}
			metaModified = true
		} else if metaModified {
			if err := git.RebaseWithAuthorDate(e.Base(), e.Head()); err != nil {
				return err
			}
		}
	}

	// 17. Force-push amended branches.
	if err := git.ForcePush(app.Args.Remote, heads...); err != nil {
		return fmt.Errorf("ERROR: Cannot push amended branches: %w", err)
	}

	// 18. Update PRs with cross-links, titles, bodies, and correct bases.
	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		existingBody := ""
		if keepBody {
			info, err := pr.View(e.PR())
			if err != nil {
				return fmt.Errorf("ERROR: Cannot fetch PR body: %w", err)
			}
			existingBody = info.Body
		}
		body := stack.BuildPRBody(e, st, keepBody, existingBody)
		if err := pr.Edit(e.PR(), e.Commit.Title, e.Base(), body); err != nil {
			return fmt.Errorf("ERROR: Cannot update PR: %w", err)
		}
	}

	// 19. Restore temporary drafts.
	for _, prRef := range tmpDraftPRs {
		if err := pr.Ready(prRef); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to mark PR %s ready: %v\n", prRef, err)
		}
	}

	// 20. Restore original branch.
	if needsBranchRebase {
		if err := git.RebaseWithAuthorDate(st.Top().Head(), app.OrigBranch); err != nil {
			return fmt.Errorf("ERROR: Cannot rebase original branch: %w", err)
		}
	} else {
		if err := git.CheckoutBranch(app.OrigBranch); err != nil {
			return fmt.Errorf("ERROR: Cannot checkout original branch: %w", err)
		}
	}

	// 21. Delete local generated branches (best-effort).
	git.DeleteLocalBranches(heads...)

	// 22. Tips.
	if app.Args.ShowTips {
		printSubmitTips(st)
	}

	return nil
}

// maybeRebaseBase fast-forwards the local base if it is strictly behind
// REMOTE/TARGET while HEAD already contains the target.
func maybeRebaseBase(app *AppContext) error {
	base := app.Args.Base
	remoteTarget := app.Args.Remote + "/" + app.Args.Target

	baseAncRemote, err := git.IsAncestor(base, remoteTarget)
	if err != nil || !baseAncRemote {
		return nil
	}
	remoteAncHead, err := git.IsAncestor(remoteTarget, app.Args.Head)
	if err != nil || !remoteAncHead {
		return nil
	}
	baseHash, _ := git.RevParse(base)
	targetHash, _ := git.RevParse(remoteTarget)
	if baseHash == targetHash {
		return nil
	}

	if err := git.Rebase(remoteTarget, base); err != nil {
		return fmt.Errorf("ERROR: Cannot rebase base: %w", err)
	}
	if err := git.CheckoutBranch(app.OrigBranch); err != nil {
		return fmt.Errorf("ERROR: Cannot checkout original branch after base rebase: %w", err)
	}
	newBase, _ := git.RevParse(base)
	app.Args.Base = newBase
	return nil
}

func printSubmitTips(st stack.Stack) {
	fmt.Println()
	hasAllPRs := true
	for _, e := range st {
		if !e.HasPR() {
			hasAllPRs = false
			break
		}
	}
	if hasAllPRs {
		fmt.Println("Your stack has been submitted successfully.")
		fmt.Println("You can view it with: stack-pr view")
		fmt.Println("Once reviewed, land it with: stack-pr land")
	} else {
		fmt.Println("Some PRs could not be created. Check the output above for errors.")
	}
}

const dryRunNoChangesNote = "No local Git changes, remote pushes, or GitHub PR changes were made."

// resolveDraftFlags computes the per-entry draft array from the --draft and
// --draft-bitmask flags. It mirrors the real-submit validation: on invalid
// input it prints an error to stderr and returns ok=false so the caller
// returns nil (matching the existing non-fatal exit behavior).
func resolveDraftFlags(draft bool, draftBitmask string, n int) ([]bool, bool) {
	isDraft := make([]bool, n)
	if draftBitmask != "" {
		if len(draftBitmask) != n {
			fmt.Fprintf(os.Stderr, "draft bitmask length (%d) does not match stack length (%d)\n", len(draftBitmask), n)
			return nil, false
		}
		for i, c := range draftBitmask {
			switch c {
			case '1':
				isDraft[i] = true
			case '0':
				isDraft[i] = false
			default:
				fmt.Fprintf(os.Stderr, "draft bitmask must contain only 0 or 1, got %q at position %d\n", string(c), i)
				return nil, false
			}
		}
	}
	if draft {
		for i := range isDraft {
			isDraft[i] = true
		}
	}
	return isDraft, true
}

// printDryRunPlan emits a human-readable plan of the submit/export actions
// that would be performed for the given stack. The stack is printed in
// bottom-to-top order to match real submit ordering.
func printDryRunPlan(st stack.Stack, needsMeta, isDraft []bool) {
	fmt.Println("Planned actions:")
	for i, e := range st {
		action := "create PR"
		if e.HasPR() {
			action = "update PR"
		}
		fmt.Printf("  %d. %s\n", i+1, action)
		fmt.Printf("     title: %s\n", e.Commit.Title)
		fmt.Printf("     head:  %s\n", e.Head())
		fmt.Printf("     base:  %s\n", e.Base())
		if e.HasPR() {
			fmt.Printf("     PR:    %s\n", e.PR())
		} else {
			draftLabel := "ready"
			if isDraft[i] {
				draftLabel = "draft"
			}
			fmt.Printf("     draft: %s\n", draftLabel)
		}
		if needsMeta[i] {
			fmt.Println("     metadata: would add stack-info commit metadata")
		}
	}
	fmt.Println()
	fmt.Println(dryRunNoChangesNote)
}

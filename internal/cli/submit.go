package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

type submitOptions struct {
	DryRun       bool
	Draft        bool
	KeepBody     bool
	DraftBitmask string
	Reviewer     string
}

func submitCmd() *cobra.Command {
	opts := submitOptions{}

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Create or update the stack of PRs.",
		Long: `Creates or updates stacked PRs for commits in BASE..HEAD.

Use --dry-run to preview the planned actions without applying any local Git or GitHub changes.`,
		Aliases: []string{"export"},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			opts.Reviewer = DefaultReviewer(app.Config, opts.Reviewer)
			return WithRecovery(app, func() error {
				return submitImpl(app, opts)
			})
		},
	}

	cmd.Flags().BoolVar(&opts.KeepBody, "keep-body", false, "Preserve current PR body after stack cross-link section")
	cmd.Flags().BoolVarP(&opts.Draft, "draft", "d", false, "Create all new PRs as draft")
	cmd.Flags().StringVar(&opts.DraftBitmask, "draft-bitmask", "", "Per-PR draft bitmask; chars must be 0 or 1")
	cmd.Flags().StringVar(&opts.Reviewer, "reviewer", "", "Reviewer list; default from STACK_PR_DEFAULT_REVIEWER or config repo.reviewer")
	cmd.Flags().BoolVarP(&flagStash, "stash", "s", false, "Stash uncommitted changes before submitting and pop afterward")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview submit/export actions without applying local Git or GitHub changes")

	return cmd
}

func submitImpl(app *AppContext, opts submitOptions) (err error) {
	defer func() {
		if err == nil && app.StashCreated {
			if perr := git.StashPop(); perr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to pop stash: %v\n", perr)
			}
		}
	}()

	if err := validateSubmitPreconditions(app, opts); err != nil {
		return err
	}

	st, needsMeta, isDraft, err := discoverAndPrepareStack(app, opts)
	if err != nil {
		return err
	}
	if st == nil {
		return nil
	}

	experimentalEngine := useExperimentalSubmitEngine(app)
	if opts.DryRun {
		printDryRunPlan(st, needsMeta, isDraft)
		return nil
	}

	if experimentalEngine {
		return applyMutationsOptimized(app, st, needsMeta, isDraft, opts)
	}
	return applyMutations(app, st, needsMeta, isDraft, opts)
}

func useExperimentalSubmitEngine(app *AppContext) bool {
	if os.Getenv("STACK_PR_EXPERIMENTAL_SUBMIT_ENGINE") == "1" {
		return true
	}
	enabled, err := app.Config.GetBool("submit", "experimental_engine")
	return err == nil && enabled
}

func validateSubmitPreconditions(app *AppContext, opts submitOptions) error {
	if git.IsRebaseInProgress() {
		return fmt.Errorf("ERROR: Rebase in progress. Finish or abort it before submitting")
	}
	if !opts.DryRun {
		if err := maybeRebaseBase(app); err != nil {
			return err
		}
	}
	return nil
}

func discoverAndPrepareStack(app *AppContext, opts submitOptions) (stack.Stack, []bool, []bool, error) {
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
	if err != nil {
		return nil, nil, nil, err
	}

	if st.IsEmpty() {
		fmt.Println("Empty stack!")
		if opts.DryRun {
			fmt.Println(dryRunNoChangesNote)
		}
		return nil, nil, nil, nil
	}

	needsMeta := make([]bool, len(st))
	for i, e := range st {
		if !e.ReadMetadata() {
			needsMeta[i] = true
		}
	}

	isDraft, ok := resolveDraftFlags(opts.Draft, opts.DraftBitmask, len(st))
	if !ok {
		return nil, nil, nil, nil
	}

	if err := git.Fetch(app.Args.Remote); err != nil {
		return nil, nil, nil, err
	}
	tmpl := stack.ParseTemplate(app.Args.BranchNameTemplate)
	if err := st.AssignHeads(tmpl, app.Username, app.OrigBranch, app.Args.Remote); err != nil {
		return nil, nil, nil, err
	}

	st.AssignBases(app.Args.Target)

	return st, needsMeta, isDraft, nil
}

func applyMutations(app *AppContext, st stack.Stack, needsMeta, isDraft []bool, opts submitOptions) error {
	if err := initializeStackBranches(st); err != nil {
		return err
	}

	needsBranchRebase := false
	if top := st.Top(); top != nil {
		needsBranchRebase, _ = git.IsAncestor(top.Head(), app.OrigBranch)
	}

	tmpDraftPRs, err := tempDraftAndResetBases(st, app.Args.Target)
	if err != nil {
		return err
	}

	heads := make([]string, len(st))
	for i, e := range st {
		heads[i] = e.Head()
	}
	if err := git.ForcePush(app.Args.Remote, heads...); err != nil {
		return fmt.Errorf("ERROR: Cannot push branches: %w", err)
	}

	for i, e := range st {
		if e.HasPR() {
			continue
		}
		body := stack.BuildPRBody(e, st, false, "")
		prOpts := pr.CreateOptions{
			Base:     e.Base(),
			Head:     e.Head(),
			Title:    e.Commit.Title,
			Body:     body,
			Reviewer: opts.Reviewer,
			Draft:    isDraft[i],
		}
		prURL, err := pr.Create(prOpts)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot create a PR: %w", err)
		}
		e.SetPR(prURL)
	}

	if err := stack.Verify(st, false); err != nil {
		return err
	}

	fmt.Println("Stack:")
	st.PrintStack(app.Args.Hyperlinks, true)
	fmt.Println()

	if err := amendCommitMetadata(st, needsMeta); err != nil {
		return err
	}

	if err := git.ForcePush(app.Args.Remote, heads...); err != nil {
		return fmt.Errorf("ERROR: Cannot push amended branches: %w", err)
	}

	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		existingBody := ""
		if opts.KeepBody {
			info, err := pr.View(e.PR())
			if err != nil {
				return fmt.Errorf("ERROR: Cannot fetch PR body: %w", err)
			}
			existingBody = info.Body
		}
		body := stack.BuildPRBody(e, st, opts.KeepBody, existingBody)
		if err := pr.Edit(e.PR(), e.Commit.Title, e.Base(), body); err != nil {
			return fmt.Errorf("ERROR: Cannot update PR: %w", err)
		}
	}

	for _, prRef := range tmpDraftPRs {
		if err := pr.Ready(prRef); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to mark PR %s ready: %v\n", prRef, err)
		}
	}

	if needsBranchRebase {
		if err := git.RebaseWithAuthorDate(st.Top().Head(), app.OrigBranch); err != nil {
			return fmt.Errorf("ERROR: Cannot rebase original branch: %w", err)
		}
	} else {
		if err := git.CheckoutBranch(app.OrigBranch); err != nil {
			return fmt.Errorf("ERROR: Cannot checkout original branch: %w", err)
		}
	}

	git.DeleteLocalBranches(heads...)

	if app.Args.ShowTips {
		printSubmitTips(st)
	}

	return nil
}

func applyMutationsOptimized(app *AppContext, st stack.Stack, needsMeta, isDraft []bool, opts submitOptions) error {
	if err := initializeStackBranches(st); err != nil {
		return err
	}

	needsBranchRebase := false
	if top := st.Top(); top != nil {
		needsBranchRebase, _ = git.IsAncestor(top.Head(), app.OrigBranch)
	}

	cache, err := newSubmitPRStateCache(st)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot fetch PR state: %w", err)
	}
	tmpDraftPRs, err := tempDraftAndResetBasesOptimized(st, app.Args.Target, cache)
	if err != nil {
		return err
	}

	heads := make([]string, len(st))
	for i, e := range st {
		heads[i] = e.Head()
	}
	if err := git.ForcePush(app.Args.Remote, heads...); err != nil {
		return fmt.Errorf("ERROR: Cannot push branches: %w", err)
	}

	for i, e := range st {
		if e.HasPR() {
			continue
		}
		body := stack.BuildPRBody(e, st, false, "")
		prOpts := pr.CreateOptions{
			Base:     e.Base(),
			Head:     e.Head(),
			Title:    e.Commit.Title,
			Body:     body,
			Reviewer: opts.Reviewer,
			Draft:    isDraft[i],
		}
		prURL, err := pr.Create(prOpts)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot create a PR: %w", err)
		}
		e.SetPR(prURL)
		prNum, _ := e.PRNumber()
		cache.set(prURL, &pr.Info{
			BaseRefName: e.Base(),
			HeadRefName: e.Head(),
			Number:      prNum,
			State:       "OPEN",
			Body:        string(body),
			Title:       e.Commit.Title,
			URL:         prURL,
			IsDraft:     isDraft[i],
		})
	}

	if err := stack.VerifyWithProvider(st, false, cache.get); err != nil {
		return err
	}

	fmt.Println("Stack:")
	st.PrintStack(app.Args.Hyperlinks, true)
	fmt.Println()

	changedTips, err := amendCommitMetadataChanged(st, needsMeta)
	if err != nil {
		return err
	}
	if changedTips {
		if err := git.ForcePush(app.Args.Remote, heads...); err != nil {
			return fmt.Errorf("ERROR: Cannot push amended branches: %w", err)
		}
	}

	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		existingBody := ""
		if opts.KeepBody {
			info, err := cache.get(e.PR())
			if err != nil {
				return fmt.Errorf("ERROR: Cannot fetch PR body: %w", err)
			}
			existingBody = info.Body
		}
		body := stack.BuildPRBody(e, st, opts.KeepBody, existingBody)
		info, err := cache.get(e.PR())
		if err != nil {
			return fmt.Errorf("ERROR: Cannot fetch PR state: %w", err)
		}
		if !submitPREditNeeded(info, e.Commit.Title, e.Base(), body) {
			continue
		}
		if err := pr.Edit(e.PR(), e.Commit.Title, e.Base(), body); err != nil {
			return fmt.Errorf("ERROR: Cannot update PR: %w", err)
		}
		cache.updateEdit(e.PR(), e.Commit.Title, e.Base(), string(body))
	}

	for _, prRef := range tmpDraftPRs {
		if err := pr.Ready(prRef); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to mark PR %s ready: %v\n", prRef, err)
			continue
		}
		cache.updateDraft(prRef, false)
	}

	if needsBranchRebase {
		if err := git.RebaseWithAuthorDate(st.Top().Head(), app.OrigBranch); err != nil {
			return fmt.Errorf("ERROR: Cannot rebase original branch: %w", err)
		}
	} else {
		if err := git.CheckoutBranch(app.OrigBranch); err != nil {
			return fmt.Errorf("ERROR: Cannot checkout original branch: %w", err)
		}
	}

	git.DeleteLocalBranches(heads...)

	if app.Args.ShowTips {
		printSubmitTips(st)
	}

	return nil
}

func initializeStackBranches(st stack.Stack) error {
	for _, e := range st {
		if err := git.ForceUpdateBranch(e.Head(), e.Commit.SHA); err != nil {
			return err
		}
	}
	return nil
}

func tempDraftAndResetBases(st stack.Stack, target string) ([]string, error) {
	var tmpDraftPRs []string
	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		info, err := pr.View(e.PR())
		if err != nil {
			return nil, fmt.Errorf("ERROR: Cannot verify stack: %w", err)
		}
		if !info.IsDraft {
			if err := pr.ReadyUndo(e.PR()); err != nil {
				return nil, fmt.Errorf("ERROR: Cannot update PR draft state: %w", err)
			}
			e.IsTmpDraft = true
			tmpDraftPRs = append(tmpDraftPRs, e.PR())
		}
		if err := pr.EditBase(e.PR(), target); err != nil {
			return nil, fmt.Errorf("ERROR: Cannot reset PR base: %w", err)
		}
	}
	return tmpDraftPRs, nil
}

func tempDraftAndResetBasesOptimized(st stack.Stack, target string, cache *submitPRStateCache) ([]string, error) {
	var tmpDraftPRs []string
	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		info, err := cache.get(e.PR())
		if err != nil {
			return nil, fmt.Errorf("ERROR: Cannot verify stack: %w", err)
		}
		if !info.IsDraft {
			if err := pr.ReadyUndo(e.PR()); err != nil {
				return nil, fmt.Errorf("ERROR: Cannot update PR draft state: %w", err)
			}
			e.IsTmpDraft = true
			tmpDraftPRs = append(tmpDraftPRs, e.PR())
			cache.updateDraft(e.PR(), true)
		}
		if info.BaseRefName != target {
			if err := pr.EditBase(e.PR(), target); err != nil {
				return nil, fmt.Errorf("ERROR: Cannot reset PR base: %w", err)
			}
			cache.updateBase(e.PR(), target)
		}
	}
	return tmpDraftPRs, nil
}

func submitPREditNeeded(info *pr.Info, title, base string, body []byte) bool {
	return info.Title != title || info.Body != string(body) || info.BaseRefName != base
}

func amendCommitMetadata(st stack.Stack, needsMeta []bool) error {
	_, err := amendCommitMetadataChanged(st, needsMeta)
	return err
}

func amendCommitMetadataChanged(st stack.Stack, needsMeta []bool) (bool, error) {
	metaModified := false
	for i, e := range st {
		if needsMeta[i] {
			if !metaModified {
				if err := git.CheckoutBranch(e.Head()); err != nil {
					return false, err
				}
			} else {
				if err := git.RebaseWithAuthorDate(e.Base(), e.Head()); err != nil {
					return false, err
				}
			}
			msg := []byte(e.Commit.CommitMsg() + e.MetadataLine())
			if err := git.CommitAmend(msg); err != nil {
				return false, fmt.Errorf("ERROR: Cannot update stack metadata: %w", err)
			}
			metaModified = true
		} else if metaModified {
			if err := git.RebaseWithAuthorDate(e.Base(), e.Head()); err != nil {
				return false, err
			}
		}
	}
	return metaModified, nil
}

type submitPRStateCache struct {
	infos map[string]*pr.Info
}

func newSubmitPRStateCache(st stack.Stack) (*submitPRStateCache, error) {
	refs := make([]string, 0, len(st))
	for _, e := range st {
		if e.HasPR() {
			refs = append(refs, e.PR())
		}
	}
	infos, err := pr.LoadForSubmit(refs)
	if err != nil {
		return nil, err
	}
	return &submitPRStateCache{infos: infos}, nil
}

func (c *submitPRStateCache) get(prRef string) (*pr.Info, error) {
	info, ok := c.infos[prRef]
	if !ok {
		info, err := pr.View(prRef)
		if err != nil {
			return nil, err
		}
		c.infos[prRef] = info
		return info, nil
	}
	return info, nil
}

func (c *submitPRStateCache) set(prRef string, info *pr.Info) {
	c.infos[prRef] = info
}

func (c *submitPRStateCache) updateDraft(prRef string, isDraft bool) {
	if info, ok := c.infos[prRef]; ok {
		info.IsDraft = isDraft
	}
}

func (c *submitPRStateCache) updateBase(prRef, base string) {
	if info, ok := c.infos[prRef]; ok {
		info.BaseRefName = base
	}
}

func (c *submitPRStateCache) updateEdit(prRef, title, base, body string) {
	if info, ok := c.infos[prRef]; ok {
		info.Title = title
		info.BaseRefName = base
		info.Body = body
	}
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

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/pr"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

type fixOptions struct {
	PRNumber int
	Replace  bool
	DryRun   bool
}

func fixCmd() *cobra.Command {
	opts := fixOptions{}

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Repair stack metadata on HEAD from an existing PR.",
		Long: `Attach an existing PR to the current HEAD commit by adding or replacing stack-info metadata.

This command only amends the local HEAD commit. It does not create branches, push branches, or modify PRs on GitHub.

After fixing metadata, run 'bpr submit' to push and update PRs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			return fixImpl(app, opts)
		},
	}

	cmd.Flags().IntVar(&opts.PRNumber, "pr", 0, "PR number to attach to HEAD (required)")
	cmd.Flags().BoolVar(&opts.Replace, "replace", false, "Replace existing stack-info metadata if it differs")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview the planned metadata repair without amending HEAD")

	_ = cmd.MarkFlagRequired("pr")

	return cmd
}

func fixImpl(app *AppContext, opts fixOptions) error {
	if git.AnySequencerInProgress() {
		return fmt.Errorf("ERROR: A Git operation (rebase, merge, or cherry-pick) is in progress. Finish or abort it before running fix")
	}

	prInfo, err := pr.ViewByNumber(opts.PRNumber)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot load PR %d: %w", opts.PRNumber, err)
	}

	headSHA, err := git.RevParse("HEAD")
	if err != nil {
		return fmt.Errorf("ERROR: Cannot determine HEAD: %w", err)
	}

	commitMsg, err := git.CommitMsg()
	if err != nil {
		return fmt.Errorf("ERROR: Cannot read HEAD commit message: %w", err)
	}

	existingPR, existingBranch := stack.ExtractStackInfo(commitMsg)

	if opts.DryRun {
		printFixDryRun(prInfo, headSHA, existingPR, existingBranch)
		printFixAdvisoryWarnings(app)
		return nil
	}

	if prInfo.HeadRefOid != "" && prInfo.HeadRefOid != headSHA {
		fmt.Fprintf(os.Stderr, "Warning: PR head SHA (%s) differs from local HEAD (%s)\n", prInfo.HeadRefOid, headSHA)
	}

	if existingPR != "" {
		if existingPR == prInfo.URL && existingBranch == prInfo.HeadRefName {
			fmt.Println("HEAD is already fixed with matching metadata.")
			printFixAdvisoryWarnings(app)
			return nil
		}
		if !opts.Replace {
			return fmt.Errorf("ERROR: HEAD already has different stack metadata (PR: %s, branch: %s). Use --replace to overwrite", existingPR, existingBranch)
		}
	}

	newMsg := buildFixedMessage(commitMsg, prInfo.URL, prInfo.HeadRefName)
	if err := git.CommitAmend([]byte(newMsg)); err != nil {
		return fmt.Errorf("ERROR: Cannot amend HEAD with stack metadata: %w", err)
	}

	fmt.Println("Fixed stack metadata on HEAD.")
	fmt.Println("Run 'bpr submit' to push the amended commit and update PRs.")

	printFixAdvisoryWarnings(app)

	return nil
}

func printFixDryRun(prInfo *pr.Info, headSHA, existingPR, existingBranch string) {
	fmt.Printf("PR URL:        %s\n", prInfo.URL)
	fmt.Printf("PR head branch: %s\n", prInfo.HeadRefName)
	fmt.Printf("Local HEAD:    %s\n", headSHA)
	if existingPR != "" {
		fmt.Printf("Existing metadata: PR: %s, branch: %s\n", existingPR, existingBranch)
		if existingPR == prInfo.URL && existingBranch == prInfo.HeadRefName {
			fmt.Println("Planned action: none (HEAD already has matching metadata)")
		} else {
			fmt.Printf("Planned action: replace metadata with 'stack-info: PR: %s, branch: %s'\n", prInfo.URL, prInfo.HeadRefName)
		}
	} else {
		fmt.Println("Existing metadata: none")
		fmt.Printf("Planned action: append 'stack-info: PR: %s, branch: %s'\n", prInfo.URL, prInfo.HeadRefName)
	}
	fmt.Println("No commit was changed.")
}

func buildFixedMessage(currentMsg, prURL, headBranch string) string {
	stripped := stripStackInfo(currentMsg)
	stripped = strings.TrimRight(stripped, "\n")
	if stripped == "" {
		return fmt.Sprintf("stack-info: PR: %s, branch: %s\n", prURL, headBranch)
	}
	return stripped + "\n\nstack-info: PR: " + prURL + ", branch: " + headBranch + "\n"
}

func printFixAdvisoryWarnings(app *AppContext) {
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not determine stack readiness: %v\n", err)
		return
	}

	if st.IsEmpty() {
		return
	}

	missingCount := 0
	malformedCount := 0
	for _, e := range st {
		if !e.ReadMetadata() {
			missingCount++
		} else {
			if _, err := e.PRNumber(); err != nil {
				malformedCount++
			}
		}
	}

	if missingCount > 0 {
		fmt.Fprintf(os.Stderr, "Warning: stack is not fully ready to submit — %d entr%s missing PR metadata\n",
			missingCount, pluralize(missingCount, "y is", "ies are"))
	}
	if malformedCount > 0 {
		fmt.Fprintf(os.Stderr, "Warning: stack has %d entr%s with malformed PR metadata\n",
			malformedCount, pluralize(malformedCount, "y", "ies"))
	}
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/git"
	"github.com/victorhsb/branchless-pr/internal/nativestacks"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

func viewCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "view",
		Short: "Safely inspect the current stack.",
		Long:  `Does not modify commits or push branches. May fetch/prune the remote.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, ok := FromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("missing app context")
			}
			return runView(app, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", `Output format: "text" or "json"`)
	return cmd
}

func runView(app *AppContext, format string) error {
	// 2. Warn if base is auto-updatable.
	remoteTarget := app.Args.Remote + "/" + app.Args.Target
	if warn, err := maybeWarnBaseBehind(app.Git, app.Args.Base, remoteTarget, app.Args.Head); err != nil {
		return err
	} else if warn != "" {
		fmt.Println(warn)
		fmt.Println()
	}

	// 3. Discover stack.
	st, err := stack.Discover(app.Git, app.Args.Base, app.Args.Head)
	if err != nil {
		return err
	}

	// 4. Empty stack.
	if st.IsEmpty() {
		if format == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("Empty stack!")
		}
		return nil
	}

	// 5. Read metadata for each entry.
	for _, e := range st {
		e.ReadMetadata()
	}

	// 6. Assign heads for entries missing metadata by scanning remote.
	// Unlike submit, we only compute names; we don't create branches.
	tmpl := stack.ParseTemplate(app.Args.BranchNameTemplate)
	if err := st.AssignHeads(app.Git, tmpl, app.Username, app.OrigBranch, app.Args.Remote); err != nil {
		return err
	}

	// 7. Set base branches.
	st.AssignBases(app.Args.Target)

	// 8. Load native-stack membership in auto/required mode.
	mode, _ := config.ParseNativeStacksMode(app.Config.Get("github", "native_stacks"))
	if mode != config.NativeStacksOff {
		if err := loadNativeMembership(app, st, mode); err != nil {
			// In auto mode, surface as a warning but still render the stack.
			if mode == config.NativeStacksAuto {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			} else {
				return err
			}
		}
	}

	// 9. Print stack newest-to-oldest.
	if err := writeViewStack(os.Stdout, st, format, app.Args.Hyperlinks); err != nil {
		return err
	}
	fmt.Println()

	// 10. Print tips.
	if app.Args.ShowTips {
		printViewTips(st)
	}

	return nil
}

func writeViewStack(w io.Writer, st stack.Stack, format string, links bool) error {
	switch format {
	case "text":
		fmt.Fprintln(w, "Stack:")
		for _, e := range st.Reverse() {
			fmt.Fprintln(w, e.PrettyLine(links, true))
		}
		return nil
	case "json":
		payload, err := st.ToJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(payload))
		return nil
	default:
		return fmt.Errorf("unknown view format %q: expected \"text\" or \"json\"", format)
	}
}

// maybeWarnBaseBehind returns a non-empty warning string when the local base is
// strictly behind REMOTE/TARGET while HEAD already contains it (same condition
// that submit would auto-rebase).
func maybeWarnBaseBehind(repo *git.Repo, base, remoteTarget, head string) (string, error) {
	baseAncRemote, err := repo.IsAncestor(base, remoteTarget)
	if err != nil || !baseAncRemote {
		return "", nil
	}
	remoteAncHead, err := repo.IsAncestor(remoteTarget, head)
	if err != nil || !remoteAncHead {
		return "", nil
	}
	baseHash, _ := repo.RevParse(base)
	targetHash, _ := repo.RevParse(remoteTarget)
	if baseHash == targetHash {
		return "", nil
	}
	return fmt.Sprintf("Warning: local base is behind %s.\n"+
		"Consider updating it before exporting with:\n"+
		"  git rebase %s %s", remoteTarget, remoteTarget, base), nil
}

func loadNativeMembership(app *AppContext, st stack.Stack, mode config.NativeStacksMode) error {
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
	if len(prNumbers) == 0 {
		return nil
	}

	owner, repo, err := app.Git.RepoSlug(app.Args.Remote)
	if err != nil {
		return fmt.Errorf("cannot resolve owner/repo for native membership: %w", err)
	}

	client := nativestacks.NewAPIClient(owner, repo, app.PR)
	memberships, _, err := client.LoadMembership(prNumbers)
	if err != nil {
		if nativestacks.IsFeatureUnavailable(err) {
			return &nativestacks.FeatureUnavailable{Msg: "native Stacks is unavailable for this repository"}
		}
		return fmt.Errorf("cannot load native membership: %w", err)
	}

	// Track stacks seen for drift reporting.
	seenStacks := make(map[int]bool)
	for _, e := range st {
		if !e.HasPR() {
			continue
		}
		n, _ := e.PRNumber()
		m, ok := memberships[n]
		if !ok || m == nil || !m.IsStacked() {
			continue
		}
		e.NativeStackNumber = m.StackNumber
		e.NativeStackPosition = m.Position
		e.NativeStackSize = m.StackSize
		e.NativeStackBase = m.StackBase
		seenStacks[*m.StackNumber] = true
	}

	// Drift detection: warn if PRs are split across multiple stacks or if order
	// does not match a single contiguous stack prefix.
	if len(seenStacks) > 1 {
		fmt.Fprintf(os.Stderr, "Warning: native membership drift detected: stack entries belong to multiple native Stacks\n")
	}
	return nil
}

func printViewTips(st stack.Stack) {
	allReady := true
	for _, e := range st {
		if e.HasMissingInfo() {
			allReady = false
			break
		}
	}
	if allReady {
		fmt.Println("Your stack is ready to land.")
		fmt.Println("To update the stack, run: stack-pr submit")
		fmt.Println("To land the stack, run: stack-pr land")
	} else {
		fmt.Println("Your stack has not been submitted yet.")
		fmt.Println("To submit the stack, run: stack-pr submit")
	}
	fmt.Println()
}

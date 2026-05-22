package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/victorhsb/branchless-pr/internal/git"
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
	if warn, err := maybeWarnBaseBehind(app.Args.Base, remoteTarget, app.Args.Head); err != nil {
		return err
	} else if warn != "" {
		fmt.Println(warn)
		fmt.Println()
	}

	// 3. Discover stack.
	st, err := stack.Discover(app.Args.Base, app.Args.Head)
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
	if err := st.AssignHeads(tmpl, app.Username, app.OrigBranch, app.Args.Remote); err != nil {
		return err
	}

	// 7. Set base branches.
	st.AssignBases(app.Args.Target)

	// 8. Print stack newest-to-oldest.
	if err := writeViewStack(os.Stdout, st, format, app.Args.Hyperlinks); err != nil {
		return err
	}
	fmt.Println()

	// 9. Print tips.
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
func maybeWarnBaseBehind(base, remoteTarget, head string) (string, error) {
	baseAncRemote, err := git.IsAncestor(base, remoteTarget)
	if err != nil || !baseAncRemote {
		return "", nil
	}
	remoteAncHead, err := git.IsAncestor(remoteTarget, head)
	if err != nil || !remoteAncHead {
		return "", nil
	}
	baseHash, _ := git.RevParse(base)
	targetHash, _ := git.RevParse(remoteTarget)
	if baseHash == targetHash {
		return "", nil
	}
	return fmt.Sprintf("Warning: local base is behind %s.\n"+
		"Consider updating it before exporting with:\n"+
		"  git rebase %s %s", remoteTarget, remoteTarget, base), nil
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

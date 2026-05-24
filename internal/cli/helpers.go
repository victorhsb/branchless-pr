package cli

import (
	"fmt"
	"regexp"

	"github.com/victorhsb/branchless-pr/internal/git"
)

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

var stackInfoLine = regexp.MustCompile(`(?m)^stack-info: PR: .+, branch: .+\n?`)

func stripStackInfo(body string) string {
	return stackInfoLine.ReplaceAllString(body, "")
}

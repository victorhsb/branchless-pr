package cli

import (
	"fmt"
	"regexp"
)

func maybeRebaseBase(app *AppContext) error {
	base := app.Args.Base
	remoteTarget := app.Args.Remote + "/" + app.Args.Target

	baseAncRemote, err := app.Git.IsAncestor(base, remoteTarget)
	if err != nil || !baseAncRemote {
		return nil
	}
	remoteAncHead, err := app.Git.IsAncestor(remoteTarget, app.Args.Head)
	if err != nil || !remoteAncHead {
		return nil
	}
	baseHash, _ := app.Git.RevParse(base)
	targetHash, _ := app.Git.RevParse(remoteTarget)
	if baseHash == targetHash {
		return nil
	}

	if err := app.Git.Rebase(remoteTarget, base); err != nil {
		return fmt.Errorf("ERROR: Cannot rebase base: %w", err)
	}
	if err := app.Git.CheckoutBranch(app.OrigBranch); err != nil {
		return fmt.Errorf("ERROR: Cannot checkout original branch after base rebase: %w", err)
	}
	newBase, _ := app.Git.RevParse(base)
	app.Args.Base = newBase
	return nil
}

var stackInfoLine = regexp.MustCompile(`(?m)^stack-info: PR: .+, branch: .+\n?`)

func stripStackInfo(body string) string {
	return stackInfoLine.ReplaceAllString(body, "")
}

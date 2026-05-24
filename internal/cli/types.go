package cli

import (
	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/invocation"
)

type CommonArgs = invocation.CommonArgs
type AppContext = invocation.AppContext

func ResolveSharedArgs(cfg *config.Config, base, head, remote, target string, hyperlinks, verbose *bool, tmpl string, showTips *bool) CommonArgs {
	return invocation.ResolveSharedArgs(cfg, base, head, remote, target, hyperlinks, verbose, tmpl, showTips)
}

func DefaultReviewer(cfg *config.Config, arg string) string {
	return invocation.DefaultReviewer(cfg, arg)
}

func RequireCleanRepo() error {
	return invocation.RequireCleanRepo()
}

func WithRecovery(app *AppContext, fn func() error) error {
	return invocation.WithRecovery(app, fn)
}

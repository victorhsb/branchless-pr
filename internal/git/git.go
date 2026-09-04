package git

import "github.com/victorhsb/branchless-pr/internal/shell"

// Repo runs Git commands in one working directory through an injectable runner.
// An empty Dir uses the process working directory.
type Repo struct {
	Dir string
	run shell.Runner
}

// New returns a Git repository command boundary.
func New(dir string, run shell.Runner) *Repo {
	if run == nil {
		run = shell.Default{}
	}
	return &Repo{Dir: dir, run: run}
}

func (r *Repo) runner() shell.Runner {
	return r.run
}

func (r *Repo) opts(opts shell.RunOpts) shell.RunOpts {
	if r != nil {
		opts.Dir = r.Dir
	}
	return opts
}

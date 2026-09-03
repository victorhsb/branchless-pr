package diagnose

import (
	"os/exec"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

type Runner interface {
	Output(args []string, opts shell.RunOpts) (string, error)
	Run(args []string, opts shell.RunOpts) ([]byte, []byte, error)
	LookPath(file string) (string, error)
}

type DefaultRunner struct{ shell.Default }

func (DefaultRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

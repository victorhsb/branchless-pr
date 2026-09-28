package diagnose

import "github.com/victorhsb/branchless-pr/internal/pr"

// Git is the read-only repository information needed by diagnosis.
// It is implemented by *git.Repo.
type Git interface {
	RepoRoot() (string, error)
	CurrentBranchName() (string, error)
	TrackedChangeCount() (int, error)
	RebaseInProgress() (bool, error)
	RevParse(string) (string, error)
	MergeBase(string, string) (string, error)
	RevListHeaders(string, string) (string, error)
	IsAncestor(string, string) (bool, error)
}

// GitHub is the read-only GitHub boundary implemented by *pr.Client.
// Only LookPath is used without --online.
type GitHub interface {
	LookPath() error
	AuthStatus() error
	ProbeAvailability() (string, error)
	InspectPR(string) (*pr.Info, error)
}

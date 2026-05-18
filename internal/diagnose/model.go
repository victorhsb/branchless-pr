// Package diagnose implements the read-only agent diagnosis report.
package diagnose

const SchemaVersion = "1"

type Status string

const (
	StatusOK       Status = "ok"
	StatusWarning  Status = "warning"
	StatusBlocking Status = "blocking"
	StatusUnknown  Status = "unknown"
)

type Report struct {
	SchemaVersion  string         `json:"schema_version"`
	Status         Status         `json:"status"`
	Repo           RepoContext    `json:"repo"`
	Stack          StackSummary   `json:"stack"`
	Checks         []CheckEntry   `json:"checks"`
	Recommendation Recommendation `json:"recommendation"`
}

type RepoContext struct {
	Root               string `json:"root,omitempty"`
	CurrentBranch      string `json:"current_branch,omitempty"`
	Remote             string `json:"remote"`
	Target             string `json:"target"`
	Base               string `json:"base,omitempty"`
	Head               string `json:"head"`
	BranchNameTemplate string `json:"branch_name_template"`
	Online             bool   `json:"online"`
}

type StackSummary struct {
	Size             int `json:"size"`
	EntriesWithPR    int `json:"entries_with_pr"`
	EntriesMissingPR int `json:"entries_missing_pr"`
}

type CheckEntry struct {
	ID           string   `json:"id"`
	Status       Status   `json:"status"`
	Message      string   `json:"message"`
	Blocks       []string `json:"blocks,omitempty"`
	SuggestedFix string   `json:"suggested_fix,omitempty"`
}

type Recommendation struct {
	Command              string           `json:"command"`
	Reason               string           `json:"reason"`
	SideEffects          bool             `json:"side_effects"`
	RequiresConfirmation bool             `json:"requires_confirmation"`
	PotentialNextActions []Recommendation `json:"potential_next_actions,omitempty"`
}

func (r *Report) finalize() {
	r.Status = overallStatus(r.Checks)
	r.Recommendation = BuildRecommendation(*r)
}

func overallStatus(checks []CheckEntry) Status {
	hasUnknown := false
	hasWarning := false
	for _, c := range checks {
		switch c.Status {
		case StatusBlocking:
			return StatusBlocking
		case StatusUnknown:
			hasUnknown = true
		case StatusWarning:
			hasWarning = true
		}
	}
	if hasUnknown {
		return StatusUnknown
	}
	if hasWarning {
		return StatusWarning
	}
	return StatusOK
}

func findCheck(checks []CheckEntry, id string) (CheckEntry, bool) {
	for _, c := range checks {
		if c.ID == id {
			return c, true
		}
	}
	return CheckEntry{}, false
}

package diagnose

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func RenderJSON(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func RenderText(report Report) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# stack-pr agent diagnose\n\n")
	fmt.Fprintf(&b, "Status: `%s`  \n", report.Status)
	fmt.Fprintf(&b, "Schema version: `%s`\n\n", report.SchemaVersion)

	fmt.Fprintf(&b, "## Repository\n\n")
	fmt.Fprintf(&b, "- Root: `%s`\n", display(report.Repo.Root))
	fmt.Fprintf(&b, "- Current branch: `%s`\n", display(report.Repo.CurrentBranch))
	fmt.Fprintf(&b, "- Remote: `%s`\n", report.Repo.Remote)
	fmt.Fprintf(&b, "- Target: `%s`\n", report.Repo.Target)
	fmt.Fprintf(&b, "- Base: `%s`\n", display(report.Repo.Base))
	fmt.Fprintf(&b, "- Head: `%s`\n", report.Repo.Head)
	fmt.Fprintf(&b, "- Branch-name template: `%s`\n", report.Repo.BranchNameTemplate)
	fmt.Fprintf(&b, "- Online checks: `%t`\n\n", report.Repo.Online)

	fmt.Fprintf(&b, "## Stack\n\n")
	fmt.Fprintf(&b, "- Size: `%d`\n", report.Stack.Size)
	fmt.Fprintf(&b, "- Entries with PR metadata: `%d`\n", report.Stack.EntriesWithPR)
	fmt.Fprintf(&b, "- Entries missing PR metadata: `%d`\n\n", report.Stack.EntriesMissingPR)

	fmt.Fprintf(&b, "## Checks\n\n")
	for _, c := range report.Checks {
		fmt.Fprintf(&b, "- `%s` **%s**: %s\n", c.ID, c.Status, c.Message)
		if len(c.Blocks) > 0 {
			fmt.Fprintf(&b, "  - Blocks: `%v`\n", c.Blocks)
		}
		if c.SuggestedFix != "" {
			fmt.Fprintf(&b, "  - Suggested fix: %s\n", c.SuggestedFix)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "## Recommendation\n\n")
	renderRecommendation(&b, report.Recommendation, "")
	if len(report.Recommendation.PotentialNextActions) > 0 {
		fmt.Fprintf(&b, "\n### Potential next actions\n\n")
		for _, next := range report.Recommendation.PotentialNextActions {
			renderRecommendation(&b, next, "- ")
		}
	}
	return b.String()
}

func renderRecommendation(b *bytes.Buffer, r Recommendation, prefix string) {
	fmt.Fprintf(b, "%sCommand: `%s`\n", prefix, r.Command)
	fmt.Fprintf(b, "%sReason: %s\n", prefix, r.Reason)
	fmt.Fprintf(b, "%sSide effects: `%t`\n", prefix, r.SideEffects)
	fmt.Fprintf(b, "%sRequires confirmation: `%t`\n", prefix, r.RequiresConfirmation)
}

func display(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

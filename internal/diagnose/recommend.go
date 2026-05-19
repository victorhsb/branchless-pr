package diagnose

import agentmeta "github.com/victorhsb/branchless-pr/internal/agent"

func BuildRecommendation(r Report) Recommendation {
	if c, ok := findCheck(r.Checks, "git_repository"); ok && c.Status == StatusBlocking {
		return Recommendation{
			Command:              "cd <git-repository>",
			Reason:               "The current working directory is not inside a Git repository, so stack-pr cannot inspect a stack.",
			SideEffects:          false,
			RequiresConfirmation: false,
		}
	}
	if c, ok := findCheck(r.Checks, "rebase_in_progress"); ok && c.Status == StatusBlocking {
		return Recommendation{
			Command:              "git rebase --continue | git rebase --abort",
			Reason:               "A rebase is in progress and should be resolved before running stack-pr operations.",
			SideEffects:          true,
			RequiresConfirmation: true,
		}
	}
	if r.Stack.Size == 0 {
		return Recommendation{
			Command:              "create commits on top of the target branch",
			Reason:               "No commits were found in the configured BASE..HEAD stack range.",
			SideEffects:          true,
			RequiresConfirmation: true,
		}
	}
	if c, ok := findCheck(r.Checks, "working_tree_clean"); ok && c.Status == StatusBlocking {
		return Recommendation{
			Command:              "clean the working tree (commit, stash, or revert changes)",
			Reason:               "Mutating stack-pr commands require a clean working tree.",
			SideEffects:          true,
			RequiresConfirmation: true,
		}
	}
	if c, ok := findCheck(r.Checks, "github_availability"); ok && c.Status == StatusBlocking {
		return Recommendation{
			Command:              "wait for GitHub availability or inspect local state only",
			Reason:               "GitHub appears unavailable, so live GitHub state cannot currently be trusted for mutating stack-pr operations.",
			SideEffects:          false,
			RequiresConfirmation: false,
		}
	}
	if r.Stack.EntriesMissingPR > 0 {
		return recommendationFromCommand("submit --dry-run", "One or more commits are missing PR metadata; dry-run previews the create-or-update plan without mutating local Git or GitHub.")
	}

	rec := recommendationFromCommand("view", "The stack appears fully submitted; inspect the stack JSON before deciding on any mutating operation.")
	land := recommendationFromCommand("land", "Potential next action only after explicit human approval; landing merges the bottom PR and mutates local and remote state.")
	land.SideEffects = true
	land.RequiresConfirmation = true
	rec.PotentialNextActions = []Recommendation{land}
	return rec
}

func recommendationFromCommand(key, reason string) Recommendation {
	if spec, ok := agentmeta.CommandSpec(key); ok {
		return Recommendation{
			Command:              spec.Name,
			Reason:               reason,
			SideEffects:          spec.SideEffects,
			RequiresConfirmation: spec.RequiresExplicitConfirmation || spec.SideEffects,
		}
	}
	return Recommendation{Command: "stack-pr " + key, Reason: reason}
}

// Package agent contains static metadata and prompt rendering helpers for
// agent-facing stack-pr commands.
package agent

// AgentCommandSpec describes a stack-pr command in terms useful to LLM agents.
type AgentCommandSpec struct {
	Name                         string
	Purpose                      string
	SideEffects                  bool
	RequiresExplicitConfirmation bool
	Effects                      []string
	SafeBefore                   []string
	Never                        []string
}

// CommandKeys is the canonical order for command metadata.
var CommandKeys = []string{
	"view",
	"comments",
	"checks",
	"submit --dry-run",
	"fix --dry-run",
	"submit",
	"fix",
	"land",
	"abandon",
	"config",
}

// Commands is the shared registry of user-facing stack-pr command metadata.
var Commands = map[string]AgentCommandSpec{
	"view": {
		Name:        "stack-pr view",
		Purpose:     "Inspect the local stack and PR metadata without changing commits or PRs.",
		SideEffects: false,
		SafeBefore: []string{
			"Planning a submit, land, or abandon operation.",
			"Checking whether stack metadata is present.",
		},
		Never: []string{
			"Treat view output as proof that a mutating operation has already happened.",
		},
	},
	"comments": {
		Name:        "stack-pr comments",
		Purpose:     "Collect PR review comments across the stack without changing commits or PRs.",
		SideEffects: false,
		SafeBefore: []string{
			"Summarizing outstanding review feedback.",
			"Planning code changes based on stack-wide PR comments.",
		},
		Never: []string{
			"Treat comment output as approval to resolve, edit, merge, or delete anything.",
		},
	},
	"checks": {
		Name:        "stack-pr checks",
		Purpose:     "Report CI and review-attention state across the stack without changing commits or PRs.",
		SideEffects: false,
		SafeBefore: []string{
			"Inspecting failed CI checks across every PR in the stack.",
			"Planning code changes from stable failed-check IDs.",
			"Checking lightweight review/comment pressure before full comment inspection.",
		},
		Never: []string{
			"Treat check output as approval to rerun, resolve, merge, or delete anything.",
			"Use checks as a replacement for stack-pr comments when full comment details are needed.",
		},
	},
	"submit --dry-run": {
		Name:        "stack-pr submit --dry-run",
		Purpose:     "Preview the PR create/update plan without local Git mutations, pushes, or GitHub writes.",
		SideEffects: false,
		SafeBefore: []string{
			"Asking the user to approve a real submit.",
			"Explaining what branches, PRs, and metadata would be touched.",
		},
		Never: []string{
			"Assume the dry-run applied any of the displayed changes.",
		},
	},
	"fix --dry-run": {
		Name:        "stack-pr fix --dry-run",
		Purpose:     "Preview metadata repair on HEAD without amending the commit or writing to GitHub.",
		SideEffects: false,
		SafeBefore: []string{
			"Asking the user to approve a real fix before amending HEAD.",
			"Inspecting what PR metadata would be attached to HEAD.",
		},
		Never: []string{
			"Assume the dry-run changed any local commit.",
		},
	},
	"submit": {
		Name:                         "stack-pr submit",
		Purpose:                      "Create or update GitHub PRs for each commit in the stack.",
		SideEffects:                  true,
		RequiresExplicitConfirmation: true,
		Effects: []string{
			"May rebase local commits when updating the base.",
			"Creates or updates generated local branches.",
			"Force-pushes generated stack branches.",
			"Creates or edits GitHub pull requests.",
			"May amend commits to add stack-info metadata.",
		},
		SafeBefore: []string{
			"The user has reviewed the dry-run plan or explicitly requested submission.",
			"The working tree is clean, unless using --stash intentionally.",
		},
		Never: []string{
			"Run without explicit user intent to publish or update PRs.",
			"Use as a read-only inspection command.",
		},
	},
	"fix": {
		Name:                         "stack-pr fix",
		Purpose:                      "Repair stack-info metadata on HEAD from an existing PR.",
		SideEffects:                  true,
		RequiresExplicitConfirmation: true,
		Effects: []string{
			"Amends HEAD to add or replace stack-info metadata.",
			"Does not create branches, push branches, or modify PRs on GitHub.",
		},
		SafeBefore: []string{
			"The commit is missing stack-info metadata for an existing PR.",
			"The user wants to repair HEAD metadata before running submit.",
		},
		Never: []string{
			"Use fix as a substitute for submit when the user wants to publish or update PRs.",
			"Run fix when the user only wants read-only inspection.",
		},
	},
	"land": {
		Name:                         "stack-pr land",
		Purpose:                      "Squash-merge the bottom PR and rebase the remaining stack.",
		SideEffects:                  true,
		RequiresExplicitConfirmation: true,
		Effects: []string{
			"Merges the bottom pull request on GitHub.",
			"Rebases and force-pushes remaining stack branches.",
			"Deletes local generated branches for landed entries.",
			"Rebases the original branch and local target branch when present.",
		},
		SafeBefore: []string{
			"The user explicitly confirms they want the bottom PR landed.",
			"stack-pr view shows the stack is ready.",
		},
		Never: []string{
			"Run merely to inspect merge readiness.",
			"Run when the user asked only to update PR descriptions or branches.",
		},
	},
	"abandon": {
		Name:                         "stack-pr abandon",
		Purpose:                      "Remove stack metadata and delete generated stack branches.",
		SideEffects:                  true,
		RequiresExplicitConfirmation: true,
		Effects: []string{
			"Amends commits to strip stack-info metadata.",
			"Rebases commits and the original branch.",
			"Deletes generated local branches.",
			"Deletes matching generated remote branches when present.",
		},
		SafeBefore: []string{
			"The user explicitly confirms they want to abandon stack-pr management for the stack.",
		},
		Never: []string{
			"Run to close or merge pull requests.",
			"Run as a cleanup step after an unrelated failure unless the user confirms.",
		},
	},
	"config": {
		Name:        "stack-pr config",
		Purpose:     "Read or write the .stack-pr.cfg configuration file.",
		SideEffects: true,
		Effects: []string{
			"Writes to .stack-pr.cfg in the repository root.",
			"May create the file if it does not exist.",
		},
		SafeBefore: []string{
			"The user explicitly requests a configuration change.",
			"The working directory is at the repository root.",
		},
		Never: []string{
			"Modify configuration as part of normal stack operations.",
			"Assume the repository root is known without verification.",
		},
	},
}

// CommandSpec returns metadata for key.
func CommandSpec(key string) (AgentCommandSpec, bool) {
	spec, ok := Commands[key]
	return spec, ok
}

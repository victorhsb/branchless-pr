package invocation

type CommandPolicy struct {
	AgentOnly      bool
	ConfigOnly     bool
	AllowsDirty    bool
	UsesStash      bool
	RequiresTarget bool
}

func PolicyFor(command string, inAgentSubtree, inConfigSubtree bool) CommandPolicy {
	if inAgentSubtree {
		return CommandPolicy{AgentOnly: true}
	}
	if inConfigSubtree {
		return CommandPolicy{ConfigOnly: true, AllowsDirty: true}
	}

	policy := CommandPolicy{RequiresTarget: true}
	switch command {
	case "view", "comments", "checks":
		policy.AllowsDirty = true
	case "submit", "export":
		policy.UsesStash = true
	case "fix":
		// fix requires clean tree (no dirty allowed), no stash, requires target branch
		// Does not allow dirty, does not use stash, requires target
	}
	return policy
}

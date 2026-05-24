package invocation

import "testing"

func TestPolicyForReadOnlyCommandsAllowDirtyAndRequireTarget(t *testing.T) {
	for _, command := range []string{"view", "comments", "checks"} {
		policy := PolicyFor(command, false, false)
		if !policy.AllowsDirty {
			t.Fatalf("%s AllowsDirty = false, want true", command)
		}
		if !policy.RequiresTarget {
			t.Fatalf("%s RequiresTarget = false, want true", command)
		}
	}
}

func TestPolicyForAgentAndConfigSubtrees(t *testing.T) {
	agent := PolicyFor("prompt", true, false)
	if !agent.AgentOnly || agent.RequiresTarget {
		t.Fatalf("agent policy = %+v", agent)
	}

	config := PolicyFor("config", false, true)
	if !config.ConfigOnly || !config.AllowsDirty || config.RequiresTarget {
		t.Fatalf("config policy = %+v", config)
	}
}

func TestPolicyForSubmitUsesStash(t *testing.T) {
	policy := PolicyFor("submit", false, false)
	if !policy.UsesStash || !policy.RequiresTarget || policy.AllowsDirty {
		t.Fatalf("submit policy = %+v", policy)
	}
}

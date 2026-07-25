package nativestacks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// APIClient wraps read-only `gh api` calls for native Stacks.
type APIClient struct {
	owner string
	repo  string
}

// NewAPIClient returns a client scoped to owner/repo.
func NewAPIClient(owner, repo string) *APIClient {
	return &APIClient{owner: owner, repo: repo}
}

// ProbeAvailability checks whether the repository supports native Stacks.
// It first confirms ordinary repository access, then probes the Stacks endpoint.
// A documented unsupported-feature response is classified as FeatureUnavailable.
func (c *APIClient) ProbeAvailability() error {
	if _, err := c.repoInfo(); err != nil {
		return err
	}
	_, err := c.listStacks()
	if err == nil {
		return nil
	}
	if IsFeatureUnavailable(err) {
		return &FeatureUnavailable{Msg: "GitHub native Stacks is unavailable for this repository"}
	}
	return err
}

func (c *APIClient) repoInfo() ([]byte, error) {
	path := fmt.Sprintf("repos/%s/%s", c.owner, c.repo)
	args := []string{"gh", "api", path}
	out, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return nil, fmt.Errorf("repository access: %w", err)
	}
	return []byte(out), nil
}

func (c *APIClient) listStacks() ([]byte, error) {
	path := fmt.Sprintf("repos/%s/%s/stacks", c.owner, c.repo)
	args := []string{"gh", "api", path}
	out, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return nil, classifyAPIError(err)
	}
	return []byte(out), nil
}

// GetStack loads a single native Stack by number.
func (c *APIClient) GetStack(number int) (*Stack, error) {
	path := fmt.Sprintf("repos/%s/%s/stacks/%d", c.owner, c.repo, number)
	args := []string{"gh", "api", path}
	out, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return nil, classifyAPIError(err)
	}
	var s Stack
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		return nil, fmt.Errorf("parse stack %d: %w", number, err)
	}
	return &s, nil
}

// LoadMembership reads native-stack membership for the given PR numbers.
// It returns a map from PR number to membership and the full set of Stacks
// that contain any of those PRs. Unstacked PRs have a nil StackNumber.
// If the Stacks feature is unavailable, it returns a FeatureUnavailable error.
func (c *APIClient) LoadMembership(prNumbers []int) (map[int]*Membership, StackSet, error) {
	out, err := c.listStacks()
	if err != nil {
		return nil, nil, err
	}
	var stacks []Stack
	if err := json.Unmarshal(out, &stacks); err != nil {
		return nil, nil, fmt.Errorf("parse stacks list: %w", err)
	}
	result := make(map[int]*Membership, len(prNumbers))
	for _, n := range prNumbers {
		result[n] = &Membership{PRNumber: n}
	}
	stackSet := make(StackSet)
	for _, s := range stacks {
		containsLocal := false
		for i, pr := range s.PRs {
			m, ok := result[pr.Number]
			if ok {
				containsLocal = true
				n := s.Number
				m.StackNumber = &n
				m.Position = i + 1
				m.StackBase = s.Base.Ref
				m.StackSize = len(s.PRs)
			}
		}
		if containsLocal {
			cp := s
			stackSet[s.Number] = &cp
		}
	}
	return result, stackSet, nil
}

// classifyAPIError converts gh api failures into stable error types.
// It treats a documented Stacks endpoint 404 or disabled message as
// FeatureUnavailable only after ordinary repository access succeeds.
func classifyAPIError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "disabled") ||
		strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "preview") {
		return &FeatureUnavailable{Msg: "GitHub native Stacks endpoint is unavailable"}
	}
	return err
}

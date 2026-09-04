package nativestacks

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/pr"
)

// Transport supplies GitHub API responses without exposing subprocess details
// to the native Stacks domain package.
type Transport interface {
	Request(method, endpoint string, body []byte, write bool) (*pr.APIResponse, error)
	Paginate(endpoint string) ([]byte, error)
	GraphQL(query string, fields map[string]string) (*pr.APIResponse, error)
}

var _ Transport = (*pr.Client)(nil)

// APIClient implements native Stacks domain operations over a GitHub transport.
type APIClient struct {
	owner              string
	repo               string
	availabilityProbed bool
	transport          Transport
}

// NewAPIClient returns a client scoped to owner/repo.
func NewAPIClient(owner, repo string, transport Transport) *APIClient {
	return &APIClient{owner: owner, repo: repo, transport: transport}
}

func (c *APIClient) repoPath(suffix string) string {
	base := fmt.Sprintf("repos/%s/%s", c.owner, c.repo)
	if suffix == "" {
		return base
	}
	return base + "/" + strings.TrimPrefix(suffix, "/")
}

// ProbeAvailability confirms ordinary repository access before interpreting a
// repository-level Stack-list 404 as feature unavailability.
func (c *APIClient) ProbeAvailability() error {
	if _, err := c.request("GET", c.repoPath(""), nil, false); err != nil {
		return fmt.Errorf("repository access: %w", err)
	}
	_, err := c.request("GET", c.repoPath("stacks?per_page=1"), nil, false)
	if err == nil {
		c.availabilityProbed = true
		return nil
	}
	if pr.IsAPIStatus(err, 404) {
		return &FeatureUnavailable{Msg: "GitHub native Stacks is unavailable for this repository"}
	}
	return fmt.Errorf("probe native Stacks: %w", err)
}

func (c *APIClient) ensureAvailability() error {
	if c.availabilityProbed {
		return nil
	}
	return c.ProbeAvailability()
}

// GetPullRequest loads and validates one REST pull-request resource.
func (c *APIClient) GetPullRequest(number int) (*PullRequest, error) {
	if number <= 0 {
		return nil, fmt.Errorf("pull request number must be positive")
	}
	resp, err := c.request("GET", c.repoPath(fmt.Sprintf("pulls/%d", number)), nil, false)
	if err != nil {
		return nil, err
	}
	if resp.Status != 200 {
		return nil, unexpectedStatus("GET", c.repoPath(fmt.Sprintf("pulls/%d", number)), resp, 200, false)
	}
	var pr PullRequest
	if err := json.Unmarshal(resp.Body, &pr); err != nil {
		return nil, fmt.Errorf("parse pull request #%d: %w", number, err)
	}
	if err := pr.Validate(); err != nil {
		return nil, fmt.Errorf("validate pull request #%d: %w", number, err)
	}
	if pr.Number != number {
		return nil, fmt.Errorf("pull request endpoint #%d returned #%d", number, pr.Number)
	}
	return &pr, nil
}

// GetStack loads and validates a native Stack by repository-scoped number.
func (c *APIClient) GetStack(number int) (*Stack, error) {
	if number <= 0 {
		return nil, fmt.Errorf("stack number must be positive")
	}
	path := c.repoPath(fmt.Sprintf("stacks/%d", number))
	resp, err := c.request("GET", path, nil, false)
	if err != nil {
		if pr.IsAPIStatus(err, 404) {
			return nil, &StackNotFound{Number: number, Err: err}
		}
		return nil, err
	}
	if resp.Status != 200 {
		return nil, unexpectedStatus("GET", path, resp, 200, false)
	}
	s, err := decodeStack(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse stack #%d: %w", number, err)
	}
	if s.Number != number {
		return nil, fmt.Errorf("stack endpoint #%d returned stack #%d", number, s.Number)
	}
	return s, nil
}

// FindStackForPR uses the filtered Stack list endpoint. Zero results mean
// unstacked; multiple results violate the singular-membership contract.
func (c *APIClient) FindStackForPR(prNumber int) (*Stack, error) {
	path := c.repoPath("stacks?pull_request=" + strconv.Itoa(prNumber))
	resp, err := c.request("GET", path, nil, false)
	if err != nil {
		return nil, err
	}
	if resp.Status != 200 {
		return nil, unexpectedStatus("GET", path, resp, 200, false)
	}
	stacks, err := decodeStackArray(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse filtered stacks for PR #%d: %w", prNumber, err)
	}
	switch len(stacks) {
	case 0:
		return nil, nil
	case 1:
		return &stacks[0], nil
	default:
		return nil, fmt.Errorf("PR #%d belongs to %d native Stacks; membership is ambiguous", prNumber, len(stacks))
	}
}

// ListStacks enumerates all repository Stacks with GitHub CLI pagination.
func (c *APIClient) ListStacks() ([]Stack, error) {
	path := c.repoPath("stacks?per_page=100")
	stdout, err := c.transport.Paginate(path)
	if err != nil {
		return nil, err
	}
	var pages [][]Stack
	if err := json.Unmarshal(stdout, &pages); err != nil {
		return nil, fmt.Errorf("parse paginated stacks: %w", err)
	}
	var stacks []Stack
	for _, page := range pages {
		for i := range page {
			if err := page[i].Validate(); err != nil {
				return nil, fmt.Errorf("validate paginated stack: %w", err)
			}
			stacks = append(stacks, page[i])
		}
	}
	return stacks, nil
}

// LoadState reads each candidate PR and every complete Stack referenced by
// those PR membership summaries.
func (c *APIClient) LoadState(prNumbers []int) (map[int]*PullRequest, map[int]*Membership, StackSet, error) {
	if err := c.ensureAvailability(); err != nil {
		return nil, nil, nil, err
	}
	prs := make(map[int]*PullRequest, len(prNumbers))
	stackNumbers := make(map[int]struct{})
	for _, number := range prNumbers {
		pr, err := c.GetPullRequest(number)
		if err != nil {
			return nil, nil, nil, err
		}
		prs[number] = pr
		if pr.Stack != nil {
			stackNumbers[pr.Stack.Number] = struct{}{}
		}
	}

	stacks := make(StackSet, len(stackNumbers))
	for number := range stackNumbers {
		s, err := c.GetStack(number)
		if err != nil {
			return nil, nil, nil, err
		}
		stacks[number] = s
	}

	memberships := make(map[int]*Membership, len(prNumbers))
	for _, number := range prNumbers {
		pr := prs[number]
		m := &Membership{PRNumber: number}
		if pr.Stack != nil {
			s := stacks[pr.Stack.Number]
			if s == nil {
				return nil, nil, nil, fmt.Errorf("PR #%d references missing Stack #%d", number, pr.Stack.Number)
			}
			if pr.Stack.Size != len(s.PRs) {
				return nil, nil, nil, fmt.Errorf("PR #%d reports Stack size %d but Stack #%d contains %d members", number, pr.Stack.Size, s.Number, len(s.PRs))
			}
			if pr.Stack.ID != s.ID {
				return nil, nil, nil, fmt.Errorf("PR #%d reports Stack id %d but Stack #%d has id %d", number, pr.Stack.ID, s.Number, s.ID)
			}
			if pr.Stack.Position > len(s.PRs) || s.PRs[pr.Stack.Position-1].Number != number {
				return nil, nil, nil, fmt.Errorf("PR #%d membership position does not match Stack #%d order", number, s.Number)
			}
			if pr.Stack.Base.Ref != s.Base.Ref {
				return nil, nil, nil, fmt.Errorf("PR #%d ultimate base %q does not match Stack #%d base %q", number, pr.Stack.Base.Ref, s.Number, s.Base.Ref)
			}
			stackNumber := pr.Stack.Number
			m.StackNumber = &stackNumber
			m.Position = pr.Stack.Position
			m.StackBase = pr.Stack.Base.Ref
			m.StackSize = pr.Stack.Size
			if len(s.PRs) > 0 {
				m.StackHead = s.PRs[len(s.PRs)-1].Head.Ref
			}
		}
		memberships[number] = m
	}
	return prs, memberships, stacks, nil
}

// LoadMembership retains the existing planner-facing API.
func (c *APIClient) LoadMembership(prNumbers []int) (map[int]*Membership, StackSet, error) {
	_, memberships, stacks, err := c.LoadState(prNumbers)
	return memberships, stacks, err
}

// LoadWriteLifecycle enriches only the PRs that would be newly added with the
// GraphQL lifecycle fields not exposed reliably by the REST pull-request
// resource, notably merge-queue membership.
func (c *APIClient) LoadWriteLifecycle(plan *Result, prs map[int]*PullRequest) error {
	if plan == nil || !plan.IsWriteAction() {
		return nil
	}
	candidates := plan.LocalPRs
	if plan.Kind == ActionAppend {
		candidates = plan.SuffixPRs
	}
	for _, number := range candidates {
		pr := prs[number]
		if pr == nil {
			return fmt.Errorf("cannot load lifecycle for missing pull request #%d", number)
		}
		if err := c.loadPullRequestLifecycle(pr); err != nil {
			return err
		}
	}
	return nil
}

func (c *APIClient) loadPullRequestLifecycle(pr *PullRequest) error {
	const query = `query($owner:String!,$repo:String!,$number:Int!){repository(owner:$owner,name:$repo){pullRequest(number:$number){number state isDraft merged mergedAt mergeQueueEntry{id} autoMergeRequest{enabledAt}}}}`
	resp, err := c.transport.GraphQL(query, map[string]string{
		"owner":  c.owner,
		"repo":   c.repo,
		"number": strconv.Itoa(pr.Number),
	})
	if err != nil {
		return err
	}
	if resp.Status != 200 {
		return unexpectedStatus("POST", "graphql", resp, 200, false)
	}
	var envelope struct {
		Data struct {
			Repository struct {
				PullRequest *struct {
					Number          int     `json:"number"`
					State           string  `json:"state"`
					IsDraft         bool    `json:"isDraft"`
					Merged          bool    `json:"merged"`
					MergedAt        *string `json:"mergedAt"`
					MergeQueueEntry *struct {
						ID string `json:"id"`
					} `json:"mergeQueueEntry"`
					AutoMergeRequest *struct {
						EnabledAt string `json:"enabledAt"`
					} `json:"autoMergeRequest"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(resp.Body, &envelope); err != nil {
		return fmt.Errorf("decode lifecycle response for PR #%d: %w", pr.Number, err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("query lifecycle for PR #%d: %s", pr.Number, envelope.Errors[0].Message)
	}
	state := envelope.Data.Repository.PullRequest
	if state == nil || state.Number != pr.Number {
		return fmt.Errorf("query lifecycle for PR #%d returned no matching pull request", pr.Number)
	}
	pr.State = strings.ToLower(state.State)
	pr.Draft = state.IsDraft
	pr.MergedAt = state.MergedAt
	pr.MergeQueueEntry = nil
	if state.MergeQueueEntry != nil && state.MergeQueueEntry.ID != "" {
		pr.MergeQueueEntry = json.RawMessage(`{"id":true}`)
	}
	pr.AutoMerge = nil
	if state.AutoMergeRequest != nil {
		pr.AutoMerge = json.RawMessage(`{"enabled":true}`)
	}
	return nil
}

// CreateStack creates a Stack and validates the exact returned sequence.
func (c *APIClient) CreateStack(prNumbers []int) (*Stack, error) {
	if len(prNumbers) < 2 || len(prNumbers) > 100 {
		return nil, fmt.Errorf("native Stack create requires 2-100 PR numbers, got %d", len(prNumbers))
	}
	body, err := json.Marshal(struct {
		PullRequests []int `json:"pull_requests"`
	}{PullRequests: prNumbers})
	if err != nil {
		return nil, err
	}
	path := c.repoPath("stacks")
	resp, err := c.request("POST", path, body, true)
	if err != nil {
		return c.reconcileCreateFailure(prNumbers, err)
	}
	if resp.Status != 201 {
		return c.reconcileCreateFailure(prNumbers, unexpectedStatus("POST", path, resp, 201, true))
	}
	s, err := decodeStack(resp.Body)
	if err != nil {
		return c.reconcileCreateFailure(prNumbers, &pr.APIError{
			Method:         "POST",
			Endpoint:       path,
			Status:         resp.Status,
			Message:        fmt.Sprintf("parse created Stack: %v", err),
			OutcomeUnknown: true,
		})
	}
	if !intSlicesEqual(prSequence(s), prNumbers) {
		return nil, fmt.Errorf("native Stack create verification failed: remote sequence [%s] != planned [%s]", formatSequence(prSequence(s)), formatSequence(prNumbers))
	}
	return s, nil
}

// AppendStack appends only a new suffix and validates the complete intended
// sequence in the returned Stack.
func (c *APIClient) AppendStack(stackNumber int, suffix, intended []int) (*Stack, error) {
	if stackNumber <= 0 {
		return nil, fmt.Errorf("stack number must be positive")
	}
	if len(suffix) < 1 || len(suffix) > 100 {
		return nil, fmt.Errorf("native Stack append requires 1-100 suffix PR numbers, got %d", len(suffix))
	}
	body, err := json.Marshal(struct {
		PullRequests []int `json:"pull_requests"`
	}{PullRequests: suffix})
	if err != nil {
		return nil, err
	}
	path := c.repoPath(fmt.Sprintf("stacks/%d/add", stackNumber))
	resp, err := c.request("POST", path, body, true)
	if err != nil {
		return c.reconcileAppendFailure(stackNumber, intended, err)
	}
	if resp.Status != 200 {
		return c.reconcileAppendFailure(stackNumber, intended, unexpectedStatus("POST", path, resp, 200, true))
	}
	s, err := decodeStack(resp.Body)
	if err != nil {
		return c.reconcileAppendFailure(stackNumber, intended, &pr.APIError{
			Method:         "POST",
			Endpoint:       path,
			Status:         resp.Status,
			Message:        fmt.Sprintf("parse appended Stack: %v", err),
			OutcomeUnknown: true,
		})
	}
	if !intSlicesEqual(prSequence(s), intended) {
		return nil, fmt.Errorf("native Stack append verification failed: remote sequence [%s] != planned [%s]", formatSequence(prSequence(s)), formatSequence(intended))
	}
	return s, nil
}

// UnstackResult distinguishes a dissolved 204 response from a partial 200
// response containing the surviving Stack.
type UnstackResult struct {
	Dissolved bool
	Stack     *Stack
	Recovered bool
}

// Unstack removes every eligible member through the documented REST endpoint.
func (c *APIClient) Unstack(stackNumber int) (*UnstackResult, error) {
	if stackNumber <= 0 {
		return nil, fmt.Errorf("stack number must be positive")
	}
	path := c.repoPath(fmt.Sprintf("stacks/%d/unstack", stackNumber))
	resp, err := c.request("POST", path, nil, true)
	if err != nil {
		return c.reconcileUnstackFailure(stackNumber, err)
	}
	switch resp.Status {
	case 204:
		if len(strings.TrimSpace(string(resp.Body))) != 0 {
			return nil, fmt.Errorf("native Stack unstack returned 204 with an unexpected body")
		}
		return &UnstackResult{Dissolved: true}, nil
	case 200:
		s, err := decodeStack(resp.Body)
		if err != nil {
			return c.reconcileUnstackFailure(stackNumber, &pr.APIError{
				Method:         "POST",
				Endpoint:       path,
				Status:         resp.Status,
				Message:        fmt.Sprintf("parse partial unstack result: %v", err),
				OutcomeUnknown: true,
			})
		}
		return &UnstackResult{Stack: s}, nil
	default:
		return c.reconcileUnstackFailure(stackNumber, unexpectedStatus("POST", path, resp, 200, true))
	}
}

func (c *APIClient) reconcileCreateFailure(intended []int, writeErr error) (*Stack, error) {
	var apiErr *pr.APIError
	if !errors.As(writeErr, &apiErr) || !apiErr.OutcomeUnknown {
		return nil, writeErr
	}
	s, readErr := c.FindStackForPR(intended[0])
	if readErr != nil {
		return nil, fmt.Errorf("%w; create outcome remains unverified: %v", writeErr, readErr)
	}
	if s == nil {
		return nil, writeErr
	}
	if intSlicesEqual(prSequence(s), intended) {
		return s, nil
	}
	return nil, fmt.Errorf("%w; create outcome conflicts with Stack #%d sequence [%s]", writeErr, s.Number, formatSequence(prSequence(s)))
}

func (c *APIClient) reconcileAppendFailure(stackNumber int, intended []int, writeErr error) (*Stack, error) {
	var apiErr *pr.APIError
	if !errors.As(writeErr, &apiErr) || !apiErr.OutcomeUnknown {
		return nil, writeErr
	}
	s, readErr := c.GetStack(stackNumber)
	if readErr != nil {
		return nil, fmt.Errorf("%w; append outcome remains unverified: %v", writeErr, readErr)
	}
	if intSlicesEqual(prSequence(s), intended) {
		return s, nil
	}
	return nil, fmt.Errorf("%w; append reconciliation found Stack #%d sequence [%s], expected [%s]", writeErr, stackNumber, formatSequence(prSequence(s)), formatSequence(intended))
}

func (c *APIClient) reconcileUnstackFailure(stackNumber int, writeErr error) (*UnstackResult, error) {
	var apiErr *pr.APIError
	if !errors.As(writeErr, &apiErr) || !apiErr.OutcomeUnknown {
		return nil, writeErr
	}
	s, readErr := c.GetStack(stackNumber)
	if IsStackNotFound(readErr) {
		return &UnstackResult{Dissolved: true, Recovered: true}, nil
	}
	if readErr != nil {
		return nil, fmt.Errorf("%w; unstack outcome remains unverified: %v", writeErr, readErr)
	}
	return &UnstackResult{Stack: s, Recovered: true}, nil
}

func (c *APIClient) request(method, endpoint string, body []byte, write bool) (*pr.APIResponse, error) {
	return c.transport.Request(method, endpoint, body, write)
}

func unexpectedStatus(method, endpoint string, resp *pr.APIResponse, expected int, write bool) error {
	return &pr.APIError{
		Method:         method,
		Endpoint:       endpoint,
		Status:         resp.Status,
		Message:        fmt.Sprintf("unexpected success status, expected %d: %s", expected, strings.TrimSpace(string(resp.Body))),
		Headers:        resp.Headers,
		OutcomeUnknown: write,
	}
}

func decodeStack(data []byte) (*Stack, error) {
	var s Stack
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func decodeStackArray(data []byte) ([]Stack, error) {
	var stacks []Stack
	if err := json.Unmarshal(data, &stacks); err != nil {
		return nil, err
	}
	for i := range stacks {
		if err := stacks[i].Validate(); err != nil {
			return nil, err
		}
	}
	return stacks, nil
}

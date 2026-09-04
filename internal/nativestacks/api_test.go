package nativestacks

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type runResult struct {
	stdout []byte
	stderr []byte
	err    error
}

type recordedCall struct {
	args  []string
	stdin []byte
}

type queuedTransport struct {
	t       *testing.T
	results []runResult
	next    int
	calls   []recordedCall
}

func queuedClient(t *testing.T, results ...runResult) (*APIClient, *[]recordedCall) {
	t.Helper()
	transport := &queuedTransport{t: t, results: results}
	client := NewAPIClient("octocat", "hello-world", transport)
	client.availabilityProbed = true
	t.Cleanup(func() {
		if transport.next != len(results) {
			t.Errorf("used %d/%d queued transport results", transport.next, len(results))
		}
	})
	return client, &transport.calls
}

func (t *queuedTransport) Request(method, endpoint string, body []byte, write bool) (*Response, error) {
	t.calls = append(t.calls, recordedCall{
		args:  []string{"REQUEST", method, endpoint},
		stdin: append([]byte(nil), body...),
	})
	result := t.nextResult()
	return testResponse(method, endpoint, write, result)
}

func (t *queuedTransport) Paginate(endpoint string) ([]byte, error) {
	t.calls = append(t.calls, recordedCall{args: []string{"PAGINATE", "--paginate", "--slurp", endpoint}})
	result := t.nextResult()
	if result.err != nil {
		return nil, result.err
	}
	return result.stdout, nil
}

func (t *queuedTransport) GraphQL(query string, fields map[string]string) (*Response, error) {
	t.calls = append(t.calls, recordedCall{args: []string{"GRAPHQL", query}})
	result := t.nextResult()
	return testResponse("POST", "graphql", false, result)
}

func (t *queuedTransport) nextResult() runResult {
	t.t.Helper()
	if t.next >= len(t.results) {
		t.t.Fatalf("unexpected transport call")
	}
	result := t.results[t.next]
	t.next++
	return result
}

func testResponse(method, endpoint string, write bool, result runResult) (*Response, error) {
	resp, parseErr := testParseIncludedResponse(result.stdout)
	if result.err == nil {
		return resp, parseErr
	}
	status := 0
	headers := ""
	message := strings.TrimSpace(strings.Join([]string{string(result.stdout), string(result.stderr)}, "\n"))
	if parseErr == nil {
		status = resp.Status
		headers = resp.Headers
		if body := strings.TrimSpace(string(resp.Body)); body != "" {
			message = body
		}
	}
	return nil, &APIError{
		Method:         method,
		Endpoint:       endpoint,
		Status:         status,
		Message:        message,
		Headers:        headers,
		OutcomeUnknown: write && (status == 0 || status >= 500),
		Err:            result.err,
	}
}

func testParseIncludedResponse(out []byte) (*Response, error) {
	normalized := strings.ReplaceAll(string(out), "\r\n", "\n")
	statusLineEnd := strings.IndexByte(normalized, '\n')
	if statusLineEnd < 0 {
		return nil, fmt.Errorf("response omitted HTTP status headers")
	}
	statusFields := strings.Fields(normalized[:statusLineEnd])
	if len(statusFields) < 2 {
		return nil, fmt.Errorf("response omitted HTTP status")
	}
	status, err := strconv.Atoi(statusFields[1])
	if err != nil {
		return nil, err
	}
	blank := strings.Index(normalized, "\n\n")
	if blank < 0 {
		return &Response{Status: status, Headers: normalized}, nil
	}
	return &Response{
		Status:  status,
		Headers: normalized[:blank+2],
		Body:    []byte(normalized[blank+2:]),
	}, nil
}

func included(status int, body string) []byte {
	return []byte(fmt.Sprintf("HTTP/2.0 %d Test\ncontent-type: application/json\n\n%s", status, body))
}

func stackFixture(number int, prs ...int) string {
	members := make([]string, len(prs))
	for i, number := range prs {
		members[i] = fmt.Sprintf(`{
			"number": %d,
			"state": "open",
			"draft": false,
			"merged_at": null,
			"head": {"ref": "branch-%d", "sha": "sha-%d"},
			"future_member_field": true
		}`, number, number, number)
	}
	return fmt.Sprintf(`{
		"id": 9876543210,
		"number": %d,
		"node_id": "S_node",
		"url": "https://api.github.com/repos/octocat/hello-world/stacks/%d",
		"base": {"ref": "main"},
		"open": true,
		"created_at": "2026-07-25T10:00:00Z",
		"pull_requests": [%s],
		"future_stack_field": {"ignored": true}
	}`, number, number, strings.Join(members, ","))
}

func prFixture(number int, directBase string, membership string) string {
	return fmt.Sprintf(`{
		"number": %d,
		"state": "open",
		"draft": false,
		"merged_at": null,
		"head": {
			"ref": "branch-%d",
			"sha": "sha-%d",
			"repo": {"full_name": "octocat/hello-world"}
		},
		"base": {
			"ref": %q,
			"sha": "base-sha",
			"repo": {"full_name": "octocat/hello-world"}
		},
		"stack": %s,
		"auto_merge": null,
		"merge_queue_entry": null,
		"future_pr_field": 123
	}`, number, number, number, directBase, membership)
}

func membershipFixture(stackNumber, size, position int) string {
	return fmt.Sprintf(`{
		"id": 9876543210,
		"number": %d,
		"size": %d,
		"position": %d,
		"base": {"ref": "main", "sha": "main-sha"}
	}`, stackNumber, size, position)
}

func TestRESTModelsDecodePublishedShapesAndUnknownFields(t *testing.T) {
	var pr PullRequest
	if err := json.Unmarshal([]byte(prFixture(20, "branch-10", membershipFixture(7, 2, 2))), &pr); err != nil {
		t.Fatal(err)
	}
	if err := pr.Validate(); err != nil {
		t.Fatal(err)
	}
	if pr.Stack.ID != 9876543210 || pr.Stack.Number != 7 || pr.Stack.Position != 2 {
		t.Fatalf("membership = %+v", pr.Stack)
	}
	if pr.Base.Ref != "branch-10" || pr.Stack.Base.Ref != "main" {
		t.Fatalf("direct base = %q, ultimate base = %q", pr.Base.Ref, pr.Stack.Base.Ref)
	}

	var standalone PullRequest
	if err := json.Unmarshal([]byte(prFixture(30, "main", "null")), &standalone); err != nil {
		t.Fatal(err)
	}
	if err := standalone.Validate(); err != nil {
		t.Fatal(err)
	}
	if standalone.Stack != nil {
		t.Fatalf("standalone stack = %+v, want nil", standalone.Stack)
	}

	s, err := decodeStack([]byte(stackFixture(7, 10, 20)))
	if err != nil {
		t.Fatal(err)
	}
	if got := prSequence(s); !reflect.DeepEqual(got, []int{10, 20}) {
		t.Fatalf("sequence = %v", got)
	}
	if s.Base.SHA != "" {
		t.Fatalf("Stack base.sha = %q, want optional/empty", s.Base.SHA)
	}
}

func TestPullRequestWithoutStackFieldIsUnstacked(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "stack field absent (pre-native-stacks PR)",
			body: strings.Replace(prFixture(10, "main", "null"), "\n\t\t\"stack\": null,", "", 1),
		},
		{
			name: "stack field null",
			body: prFixture(10, "main", "null"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var pr PullRequest
			if err := json.Unmarshal([]byte(tc.body), &pr); err != nil {
				t.Fatal(err)
			}
			if err := pr.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want unstacked PR to validate", err)
			}
			if pr.Stack != nil {
				t.Fatalf("Stack = %+v, want nil", pr.Stack)
			}
		})
	}

	t.Run("present but invalid membership still errors", func(t *testing.T) {
		var pr PullRequest
		if err := json.Unmarshal([]byte(prFixture(10, "main", membershipFixture(7, 2, 3))), &pr); err != nil {
			t.Fatal(err)
		}
		if err := pr.Validate(); err == nil {
			t.Fatal("expected impossible position error")
		}
	})

	t.Run("other required fields still enforced when absent", func(t *testing.T) {
		for _, field := range []string{`"state": "open",`, `"merged_at": null,`} {
			body := strings.Replace(prFixture(10, "main", "null"), field, "", 1)
			var pr PullRequest
			if err := json.Unmarshal([]byte(body), &pr); err != nil {
				t.Fatal(err)
			}
			if err := pr.Validate(); err == nil {
				t.Fatalf("expected validation error for missing %s", field)
			}
		}
	})
}

func TestRESTModelsRejectMalformedAmbiguousAndDuplicateData(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "missing required member field",
			body: strings.Replace(stackFixture(7, 10), `"merged_at": null,`, "", 1),
		},
		{
			name: "duplicate member",
			body: stackFixture(7, 10, 10),
		},
		{
			name: "missing open",
			body: strings.Replace(stackFixture(7, 10), `"open": true,`, "", 1),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeStack([]byte(tc.body)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	badPosition := prFixture(10, "main", membershipFixture(7, 2, 3))
	var pr PullRequest
	if err := json.Unmarshal([]byte(badPosition), &pr); err != nil {
		t.Fatal(err)
	}
	if err := pr.Validate(); err == nil {
		t.Fatal("expected impossible position error")
	}
}

func TestStackMemberMergedAtDistinguishesClosedOutcomes(t *testing.T) {
	mergedAt := "2026-07-25T12:00:00Z"
	merged := StackPR{State: "closed", MergedAt: &mergedAt}
	closed := StackPR{State: "closed", MergedAt: nil}
	if !merged.IsMerged() {
		t.Fatal("merged_at should classify member as merged")
	}
	if closed.IsMerged() {
		t.Fatal("closed with merged_at null must remain unmerged")
	}
}

func TestProbeAvailabilityDisambiguatesRepositoryLevel404(t *testing.T) {
	client, _ := queuedClient(t,
		runResult{stdout: included(200, `{}`)},
		runResult{
			stdout: included(404, `{"message":"Not Found"}`),
			err:    errors.New("exit status 1"),
		},
	)
	client.availabilityProbed = false
	err := client.ProbeAvailability()
	if !IsFeatureUnavailable(err) {
		t.Fatalf("error = %v, want FeatureUnavailable", err)
	}
}

func TestNumberedStack404IsNotFeatureUnavailable(t *testing.T) {
	client, _ := queuedClient(t, runResult{
		stdout: included(404, `{"message":"Not Found"}`),
		err:    errors.New("exit status 1"),
	})
	_, err := client.GetStack(7)
	if !IsStackNotFound(err) {
		t.Fatalf("error = %v, want StackNotFound", err)
	}
	if IsFeatureUnavailable(err) {
		t.Fatalf("numbered 404 was incorrectly classified unavailable: %v", err)
	}
}

func TestFindStackForPRAcceptsZeroOneAndRejectsAmbiguous(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		wantNil bool
		wantErr bool
	}{
		{name: "unstacked", body: `[]`, wantNil: true},
		{name: "one", body: `[` + stackFixture(7, 10) + `]`},
		{name: "ambiguous", body: `[` + stackFixture(7, 10) + `,` + stackFixture(8, 10) + `]`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := queuedClient(t, runResult{stdout: included(200, tc.body)})
			s, err := client.FindStackForPR(10)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && (s == nil) != tc.wantNil {
				t.Fatalf("stack = %+v, wantNil %v", s, tc.wantNil)
			}
		})
	}
}

func TestListStacksUsesPaginationAndFlattensPages(t *testing.T) {
	client, calls := queuedClient(t, runResult{
		stdout: []byte(`[[` + stackFixture(8, 20) + `],[` + stackFixture(7, 10) + `]]`),
	})
	stacks, err := client.ListStacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 2 || stacks[0].Number != 8 || stacks[1].Number != 7 {
		t.Fatalf("stacks = %+v", stacks)
	}
	got := strings.Join((*calls)[0].args, " ")
	if !strings.Contains(got, "--paginate") || !strings.Contains(got, "--slurp") {
		t.Fatalf("pagination args = %q", got)
	}
}

func TestLoadStateReadsPRsAndEachUniqueCompleteStack(t *testing.T) {
	client, calls := queuedClient(t,
		runResult{stdout: included(200, prFixture(10, "main", membershipFixture(7, 2, 1)))},
		runResult{stdout: included(200, prFixture(20, "branch-10", membershipFixture(7, 2, 2)))},
		runResult{stdout: included(200, stackFixture(7, 10, 20))},
	)
	prs, memberships, stacks, err := client.LoadState([]int{10, 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 2 || len(stacks) != 1 || memberships[20].Position != 2 {
		t.Fatalf("prs=%d memberships=%+v stacks=%d", len(prs), memberships, len(stacks))
	}
	if len(*calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(*calls))
	}
}

func TestCreateAndAppendUseDocumentedPayloadsAndStatuses(t *testing.T) {
	createClient, createCalls := queuedClient(t,
		runResult{stdout: included(201, stackFixture(7, 10, 20))},
	)
	created, err := createClient.CreateStack([]int{10, 20})
	if err != nil {
		t.Fatal(err)
	}
	if created.Number != 7 {
		t.Fatalf("created Stack number = %d", created.Number)
	}
	var createBody struct {
		PullRequests []int `json:"pull_requests"`
	}
	if err := json.Unmarshal((*createCalls)[0].stdin, &createBody); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(createBody.PullRequests, []int{10, 20}) {
		t.Fatalf("create payload = %v", createBody.PullRequests)
	}

	appendClient, appendCalls := queuedClient(t,
		runResult{stdout: included(200, stackFixture(7, 10, 20, 30))},
	)
	if _, err := appendClient.AppendStack(7, []int{30}, []int{10, 20, 30}); err != nil {
		t.Fatal(err)
	}
	var appendBody struct {
		PullRequests []int `json:"pull_requests"`
	}
	if err := json.Unmarshal((*appendCalls)[0].stdin, &appendBody); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(appendBody.PullRequests, []int{30}) {
		t.Fatalf("append payload = %v", appendBody.PullRequests)
	}
	if got := strings.Join((*appendCalls)[0].args, " "); !strings.Contains(got, "stacks/7/add") {
		t.Fatalf("append args = %q", got)
	}
}

func TestCreateAndAppendRequestBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*APIClient) error
	}{
		{name: "create one", run: func(c *APIClient) error { _, err := c.CreateStack([]int{1}); return err }},
		{name: "create 101", run: func(c *APIClient) error { _, err := c.CreateStack(make([]int, 101)); return err }},
		{name: "append zero", run: func(c *APIClient) error { _, err := c.AppendStack(7, nil, nil); return err }},
		{name: "append 101", run: func(c *APIClient) error { _, err := c.AppendStack(7, make([]int, 101), nil); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := queuedClient(t)
			if err := tc.run(client); err == nil {
				t.Fatal("expected boundary error")
			}
		})
	}
}

func TestCreateAndAppendAcceptMaximumRequestSize(t *testing.T) {
	createPRs := make([]int, 100)
	for i := range createPRs {
		createPRs[i] = i + 1
	}
	createClient, _ := queuedClient(t,
		runResult{stdout: included(201, stackFixture(7, createPRs...))},
	)
	if _, err := createClient.CreateStack(createPRs); err != nil {
		t.Fatal(err)
	}

	suffix := make([]int, 100)
	for i := range suffix {
		suffix[i] = i + 101
	}
	appendClient, _ := queuedClient(t,
		runResult{stdout: included(200, stackFixture(7, suffix...))},
	)
	if _, err := appendClient.AppendStack(7, suffix, suffix); err != nil {
		t.Fatal(err)
	}
}

func TestCompletedStackRemainsDecodableAndDiscoverable(t *testing.T) {
	completed := strings.Replace(stackFixture(7, 10), `"open": true`, `"open": false`, 1)
	completed = strings.Replace(completed, `"state": "open"`, `"state": "closed"`, 1)
	completed = strings.Replace(completed, `"merged_at": null`, `"merged_at": "2026-07-25T12:00:00Z"`, 1)
	client, _ := queuedClient(t, runResult{stdout: included(200, `[`+completed+`]`)})
	s, err := client.FindStackForPR(10)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil || s.Open {
		t.Fatalf("Stack = %+v, want completed resource", s)
	}
}

func TestUnstackDistinguishesPartialAndDissolved(t *testing.T) {
	partialClient, partialCalls := queuedClient(t,
		runResult{stdout: included(200, stackFixture(7, 10))},
	)
	partial, err := partialClient.Unstack(7)
	if err != nil {
		t.Fatal(err)
	}
	if partial.Dissolved || partial.Stack == nil {
		t.Fatalf("partial result = %+v", partial)
	}
	if len((*partialCalls)[0].stdin) != 0 {
		t.Fatalf("unstack sent body %q", (*partialCalls)[0].stdin)
	}

	dissolvedClient, _ := queuedClient(t, runResult{stdout: included(204, "")})
	dissolved, err := dissolvedClient.Unstack(7)
	if err != nil {
		t.Fatal(err)
	}
	if !dissolved.Dissolved || dissolved.Stack != nil {
		t.Fatalf("dissolved result = %+v", dissolved)
	}
}

func TestUncertainWritesReconcileBeforeReturning(t *testing.T) {
	t.Run("create exact after transport failure", func(t *testing.T) {
		client, calls := queuedClient(t,
			runResult{stderr: []byte("connection reset"), err: errors.New("exit status 1")},
			runResult{stdout: included(200, `[`+stackFixture(7, 10, 20)+`]`)},
		)
		s, err := client.CreateStack([]int{10, 20})
		if err != nil {
			t.Fatal(err)
		}
		if s.Number != 7 || len(*calls) != 2 {
			t.Fatalf("stack=%+v calls=%d", s, len(*calls))
		}
	})

	t.Run("append exact after 503", func(t *testing.T) {
		client, _ := queuedClient(t,
			runResult{stdout: included(503, `{"message":"unavailable"}`), err: errors.New("exit status 1")},
			runResult{stdout: included(200, stackFixture(7, 10, 20, 30))},
		)
		if _, err := client.AppendStack(7, []int{30}, []int{10, 20, 30}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unstack dissolved after transport failure", func(t *testing.T) {
		client, _ := queuedClient(t,
			runResult{stderr: []byte("EOF"), err: errors.New("exit status 1")},
			runResult{stdout: included(404, `{"message":"Not Found"}`), err: errors.New("exit status 1")},
		)
		result, err := client.Unstack(7)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Dissolved || !result.Recovered {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("create conflict after transport failure", func(t *testing.T) {
		client, _ := queuedClient(t,
			runResult{stderr: []byte("connection reset"), err: errors.New("exit status 1")},
			runResult{stdout: included(200, `[`+stackFixture(7, 10, 30)+`]`)},
		)
		_, err := client.CreateStack([]int{10, 20})
		if err == nil || !strings.Contains(err.Error(), "conflicts") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestValidationFailureIsNotRetried(t *testing.T) {
	client, calls := queuedClient(t, runResult{
		stdout: included(422, `{"message":"Validation Failed"}`),
		err:    errors.New("exit status 1"),
	})
	_, err := client.CreateStack([]int{10, 20})
	if !IsAPIStatus(err, 422) {
		t.Fatalf("error = %v, want HTTP 422", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("calls = %d, want no retry or reconciliation for definite validation failure", len(*calls))
	}
}

func TestValidateWritePlanChecksChainRepositoryAndLifecycle(t *testing.T) {
	valid := map[int]*PullRequest{
		10: testPR(10, "main", "branch-10"),
		20: testPR(20, "branch-10", "branch-20"),
	}
	plan := &Result{Kind: ActionCreate, LocalPRs: []int{10, 20}}
	if err := ValidateWritePlan(plan, valid, nil, "octocat/hello-world"); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		mutate func(map[int]*PullRequest)
	}{
		{"broken chain", func(prs map[int]*PullRequest) { prs[20].Base.Ref = "wrong" }},
		{"cross repository", func(prs map[int]*PullRequest) { prs[20].Head.Repo.FullName = "other/repo" }},
		{"closed unmerged", func(prs map[int]*PullRequest) { prs[20].State = "closed" }},
		{"merged", func(prs map[int]*PullRequest) { merged := "now"; prs[20].MergedAt = &merged }},
		{"queued", func(prs map[int]*PullRequest) { prs[20].MergeQueueEntry = json.RawMessage(`{"id":"q"}`) }},
		{"auto merge", func(prs map[int]*PullRequest) { prs[20].AutoMerge = json.RawMessage(`{"enabled_at":"now"}`) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prs := map[int]*PullRequest{
				10: testPR(10, "main", "branch-10"),
				20: testPR(20, "branch-10", "branch-20"),
			}
			tc.mutate(prs)
			if err := ValidateWritePlan(plan, prs, nil, "octocat/hello-world"); err == nil {
				t.Fatal("expected eligibility error")
			}
		})
	}
}

func TestLoadWriteLifecycleQueriesMergeQueueAndAutoMergeState(t *testing.T) {
	body := `{
		"data": {
			"repository": {
				"pullRequest": {
					"number": 20,
					"state": "OPEN",
					"isDraft": true,
					"merged": false,
					"mergedAt": null,
					"mergeQueueEntry": {"id": "MQ_1"},
					"autoMergeRequest": {"enabledAt": "2026-07-25T12:00:00Z"}
				}
			}
		}
	}`
	client, calls := queuedClient(t, runResult{stdout: included(200, body)})
	prs := map[int]*PullRequest{20: testPR(20, "main", "branch-20")}
	plan := &Result{Kind: ActionCreate, LocalPRs: []int{20}}
	if err := client.LoadWriteLifecycle(plan, prs); err != nil {
		t.Fatal(err)
	}
	if !prs[20].IsQueued() || !prs[20].IsAutoMergeEnabled() || !prs[20].Draft {
		t.Fatalf("lifecycle = %+v", prs[20])
	}
	args := strings.Join((*calls)[0].args, " ")
	if !strings.Contains(args, "mergeQueueEntry") || !strings.Contains(args, "autoMergeRequest") {
		t.Fatalf("GraphQL args = %q", args)
	}
}

func testPR(number int, base, head string) *PullRequest {
	return &PullRequest{
		Number:   number,
		State:    "open",
		MergedAt: nil,
		Head: PullRequestRef{
			Ref:  head,
			SHA:  "head-sha",
			Repo: &Repository{FullName: "octocat/hello-world"},
		},
		Base: PullRequestRef{
			Ref:  base,
			SHA:  "base-sha",
			Repo: &Repository{FullName: "octocat/hello-world"},
		},
	}
}

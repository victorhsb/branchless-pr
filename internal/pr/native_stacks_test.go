package pr

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/nativestacks"
	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func includedResponse(status int, body string) string {
	return fmt.Sprintf("HTTP/2.0 %d Test\ncontent-type: application/json\n\n%s", status, body)
}

func TestNativeStacksRequestParsesIncludedResponse(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{status: 200, body: `{"ok":true}`},
		{status: 201, body: `{"created":true}`},
		{status: 204},
	} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			run := shelltest.New(t, shelltest.Response{
				Match:  shelltest.Exact("gh", "api", "--include", "--method", "GET", "repos/acme/widget/stacks"),
				Stdout: includedResponse(tc.status, tc.body),
			})
			resp, err := NewClient(run).Request("GET", "repos/acme/widget/stacks", nil, false)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Status != tc.status || strings.TrimSpace(string(resp.Body)) != tc.body {
				t.Fatalf("response = %+v", resp)
			}
		})
	}
}

func TestNativeStacksRequestPreservesStatusAndWriteUncertainty(t *testing.T) {
	for _, tc := range []struct {
		status         int
		write          bool
		outcomeUnknown bool
	}{
		{status: 401},
		{status: 403},
		{status: 422, write: true},
		{status: 429, write: true},
		{status: 503, write: true, outcomeUnknown: true},
	} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			run := shelltest.New(t, shelltest.Response{
				Match:    shelltest.Exact("gh", "api", "--include", "--method", "POST", "repos/acme/widget/stacks"),
				Stdout:   includedResponse(tc.status, `{"message":"failure"}`),
				ExitCode: 1,
			})
			_, err := NewClient(run).Request("POST", "repos/acme/widget/stacks", nil, tc.write)
			var apiErr *nativestacks.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %T, want *nativestacks.APIError", err)
			}
			if apiErr.Status != tc.status || apiErr.OutcomeUnknown != tc.outcomeUnknown {
				t.Fatalf("APIError = %+v", apiErr)
			}
		})
	}
}

func TestNativeStacksPaginateUsesSlurp(t *testing.T) {
	run := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Exact("gh", "api", "--paginate", "--slurp", "repos/acme/widget/stacks?per_page=100"),
		Stdout: `[[{"number":1}]]`,
	})
	out, err := NewClient(run).Paginate("repos/acme/widget/stacks?per_page=100")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `[[{"number":1}]]` {
		t.Fatalf("output = %q", out)
	}
}

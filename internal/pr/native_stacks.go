package pr

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/nativestacks"
	"github.com/victorhsb/branchless-pr/internal/shell"
)

var (
	includedStatusRE = regexp.MustCompile(`(?m)^HTTP/\S+[ \t]+([0-9]{3})[^\n]*\n`)
	stderrStatusRE   = regexp.MustCompile(`(?i)\bHTTP[ /]+([0-9]{3})\b`)
)

var _ nativestacks.Transport = (*Client)(nil)

// NativeStacks returns native Stack domain operations scoped to owner/repo and
// backed by this client's GitHub transport.
func (c *Client) NativeStacks(owner, repo string) *nativestacks.APIClient {
	return nativestacks.NewAPIClient(owner, repo, c)
}

// Request performs one status-preserving GitHub REST request.
func (c *Client) Request(method, endpoint string, body []byte, write bool) (*nativestacks.Response, error) {
	args := []string{"gh", "api", "--include", "--method", method, endpoint}
	opts := shell.RunOpts{Quiet: true}
	if body != nil {
		args = append(args, "--input", "-")
		opts.Stdin = body
	}
	stdout, stderr, err := c.runner().Run(args, opts)
	if err != nil {
		return nil, classifyNativeRequestError(method, endpoint, write, stdout, stderr, err)
	}
	resp, parseErr := parseIncludedResponse(stdout)
	if parseErr != nil {
		return nil, &nativestacks.APIError{
			Method:         method,
			Endpoint:       endpoint,
			Message:        parseErr.Error(),
			OutcomeUnknown: write,
		}
	}
	return resp, nil
}

// Paginate fetches all pages for a native Stacks collection.
func (c *Client) Paginate(endpoint string) ([]byte, error) {
	args := []string{"gh", "api", "--paginate", "--slurp", endpoint}
	stdout, stderr, err := c.runner().Run(args, shell.RunOpts{Quiet: true})
	if err != nil {
		return nil, classifyNativeRequestError("GET", endpoint, false, stdout, stderr, err)
	}
	return stdout, nil
}

// GraphQL performs one status-preserving GraphQL request for native Stacks.
func (c *Client) GraphQL(query string, fields map[string]string) (*nativestacks.Response, error) {
	args := []string{"gh", "api", "--include", "--method", "POST", "graphql", "-f", "query=" + query}
	for _, key := range []string{"owner", "repo"} {
		if value, ok := fields[key]; ok {
			args = append(args, "-f", key+"="+value)
		}
	}
	if number, ok := fields["number"]; ok {
		args = append(args, "-F", "number="+number)
	}
	var extra []string
	for key := range fields {
		if key != "owner" && key != "repo" && key != "number" {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	for _, key := range extra {
		args = append(args, "-f", key+"="+fields[key])
	}

	stdout, stderr, err := c.runner().Run(args, shell.RunOpts{Quiet: true})
	if err != nil {
		return nil, classifyNativeRequestError("POST", "graphql", false, stdout, stderr, err)
	}
	resp, parseErr := parseIncludedResponse(stdout)
	if parseErr != nil {
		return nil, &nativestacks.APIError{
			Method:   "POST",
			Endpoint: "graphql",
			Message:  parseErr.Error(),
		}
	}
	return resp, nil
}

func parseIncludedResponse(out []byte) (*nativestacks.Response, error) {
	normalized := strings.ReplaceAll(string(out), "\r\n", "\n")
	matches := includedStatusRE.FindAllStringSubmatchIndex(normalized, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("gh api response omitted HTTP status headers")
	}
	match := matches[len(matches)-1]
	status, err := strconv.Atoi(normalized[match[2]:match[3]])
	if err != nil {
		return nil, fmt.Errorf("parse HTTP status: %w", err)
	}
	headerStart := match[0]
	blank := strings.Index(normalized[headerStart:], "\n\n")
	if blank < 0 {
		return &nativestacks.Response{Status: status, Headers: normalized[headerStart:]}, nil
	}
	bodyStart := headerStart + blank + 2
	return &nativestacks.Response{
		Status:  status,
		Headers: normalized[headerStart:bodyStart],
		Body:    []byte(normalized[bodyStart:]),
	}, nil
}

func classifyNativeRequestError(method, endpoint string, write bool, stdout, stderr []byte, err error) error {
	status := 0
	headers := ""
	message := strings.TrimSpace(strings.Join([]string{string(stdout), string(stderr)}, "\n"))
	if resp, parseErr := parseIncludedResponse(stdout); parseErr == nil {
		status = resp.Status
		headers = resp.Headers
		if body := strings.TrimSpace(string(resp.Body)); body != "" {
			message = body
		}
	}
	if status == 0 {
		if match := stderrStatusRE.FindStringSubmatch(string(stderr)); len(match) == 2 {
			status, _ = strconv.Atoi(match[1])
		}
	}
	return &nativestacks.APIError{
		Method:         method,
		Endpoint:       endpoint,
		Status:         status,
		Message:        message,
		Headers:        headers,
		OutcomeUnknown: write && (status == 0 || status >= 500),
		Err:            err,
	}
}

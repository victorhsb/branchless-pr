// Package pr provides thin wrappers around gh CLI commands for PR operations.
// See SPEC §12–§15 for the expected behaviour of each command.
package pr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// Info holds the JSON response from `gh pr view --json`.
type Info struct {
	BaseRefName      string `json:"baseRefName"`
	HeadRefName      string `json:"headRefName"`
	Number           int    `json:"number"`
	State            string `json:"state"`
	Body             string `json:"body"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	MergeStateStatus string `json:"mergeStateStatus"`
	IsDraft          bool   `json:"isDraft"`
}

// View queries PR metadata from GitHub.
func View(prRef string) (*Info, error) {
	args := []string{
		"gh", "pr", "view", prRef,
		"--json", "baseRefName,headRefName,number,state,body,title,url,mergeStateStatus,isDraft",
	}
	out, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return nil, fmt.Errorf("gh pr view %s: %w", prRef, err)
	}
	var info Info
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return nil, fmt.Errorf("parse pr view response: %w", err)
	}
	return &info, nil
}

// LoadForSubmit fetches the PR state submit/export needs for existing PRs.
func LoadForSubmit(prRefs []string) (map[string]*Info, error) {
	infos := make(map[string]*Info, len(prRefs))
	for _, prRef := range prRefs {
		if prRef == "" {
			continue
		}
		info, err := View(prRef)
		if err != nil {
			return nil, err
		}
		infos[prRef] = info
	}
	return infos, nil
}

// EditBase updates the base branch of a PR.
func EditBase(prRef, base string) error {
	args := []string{"gh", "pr", "edit", prRef, "-B", base}
	_, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return fmt.Errorf("gh pr edit -B %s %s: %w", base, prRef, err)
	}
	return nil
}

// Edit updates title, body (from stdin), and base of a PR.
func Edit(prRef, title, base string, body []byte) error {
	args := []string{"gh", "pr", "edit", prRef, "-t", title, "-F", "-", "-B", base}
	_, _, err := shell.Run(args, shell.RunOpts{Stdin: body})
	if err != nil {
		return fmt.Errorf("gh pr edit %s: %w", prRef, err)
	}
	return nil
}

// CreateOptions configures a new PR creation.
type CreateOptions struct {
	Base     string
	Head     string
	Title    string
	Body     []byte
	Reviewer string // comma-separated list; empty means none
	Draft    bool
}

// Create opens a new PR and returns its reference (URL).
func Create(opts CreateOptions) (string, error) {
	args := []string{
		"gh", "pr", "create",
		"-B", opts.Base,
		"-H", opts.Head,
		"-t", opts.Title,
		"-F", "-",
	}
	if opts.Reviewer != "" {
		for _, r := range strings.Split(opts.Reviewer, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				args = append(args, "--reviewer", r)
			}
		}
	}
	if opts.Draft {
		args = append(args, "--draft")
	}
	out, errOut, err := shell.Run(args, shell.RunOpts{Quiet: true, Check: true, Stdin: opts.Body})
	if err != nil {
		stderr := strings.TrimSpace(string(errOut))
		if stderr != "" {
			return "", fmt.Errorf("gh pr create: %w: %s", err, stderr)
		}
		return "", fmt.Errorf("gh pr create: %w", err)
	}
	combined := append(append([]byte{}, out...), '\n')
	combined = append(combined, errOut...)
	return parseCreateOutput(combined)
}

func parseCreateOutput(out []byte) (string, error) {
	fields := strings.Fields(string(out))
	for i := len(fields) - 1; i >= 0; i-- {
		field := strings.Trim(fields[i], "()[]<>,.")
		isURL := strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://")
		if isURL && strings.Contains(field, "/pull/") {
			return field, nil
		}
	}
	if strings.TrimSpace(string(out)) == "" {
		return "", fmt.Errorf("gh pr create: unexpected empty output")
	}
	return "", fmt.Errorf("gh pr create: could not parse PR URL from output")
}

// Ready marks a PR as ready for review.
func Ready(prRef string) error {
	args := []string{"gh", "pr", "ready", prRef}
	_, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return fmt.Errorf("gh pr ready %s: %w", prRef, err)
	}
	return nil
}

// ReadyUndo marks a PR as draft again.
func ReadyUndo(prRef string) error {
	args := []string{"gh", "pr", "ready", prRef, "--undo"}
	_, err := shell.Output(args, shell.RunOpts{})
	if err != nil {
		return fmt.Errorf("gh pr ready --undo %s: %w", prRef, err)
	}
	return nil
}

// MergeSquash performs a squash merge on a PR.
func MergeSquash(prRef, title string, body []byte) error {
	args := []string{"gh", "pr", "merge", prRef, "--squash", "-t", title, "-F", "-"}
	_, _, err := shell.Run(args, shell.RunOpts{Stdin: body})
	if err != nil {
		return fmt.Errorf("gh pr merge --squash %s: %w", prRef, err)
	}
	return nil
}

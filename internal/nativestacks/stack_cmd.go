package nativestacks

import (
	"fmt"
	"strconv"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// LinkCreate runs `gh stack link <pr1> <pr2> ...` to create a native Stack.
// It passes only PR numbers and never --open or lifecycle flags.
func LinkCreate(prNumbers []int) error {
	if len(prNumbers) < 2 {
		return fmt.Errorf("link create requires at least 2 PRs, got %d", len(prNumbers))
	}
	args := []string{"gh", "stack", "link"}
	for _, n := range prNumbers {
		args = append(args, strconv.Itoa(n))
	}
	_, stderr, err := shell.Run(args, shell.RunOpts{Quiet: true})
	if err != nil {
		return ClassifyExtensionError(err, stderr)
	}
	return nil
}

// LinkAppend runs `gh stack link <stack-number> <suffix-pr...>` to append
// unstacked PRs to an existing native Stack.
func LinkAppend(stackNumber int, suffixPRs []int) error {
	if len(suffixPRs) == 0 {
		return fmt.Errorf("link append requires at least one suffix PR")
	}
	args := []string{"gh", "stack", "link", strconv.Itoa(stackNumber)}
	for _, n := range suffixPRs {
		args = append(args, strconv.Itoa(n))
	}
	_, stderr, err := shell.Run(args, shell.RunOpts{Quiet: true})
	if err != nil {
		return ClassifyExtensionError(err, stderr)
	}
	return nil
}

// Unstack runs `gh stack unstack <stack-number>` to dissolve a native Stack.
func Unstack(stackNumber int) error {
	args := []string{"gh", "stack", "unstack", strconv.Itoa(stackNumber)}
	_, stderr, err := shell.Run(args, shell.RunOpts{Quiet: true})
	if err != nil {
		return ClassifyExtensionError(err, stderr)
	}
	return nil
}

package stackstate

import (
	"github.com/victorhsb/branchless-pr/internal/stack"
)

type Args struct {
	Base               string
	Head               string
	Remote             string
	Target             string
	BranchNameTemplate string
	Username           string
	OrigBranch         string
}

func Load(args Args) (stack.Stack, error) {
	st, err := stack.Discover(args.Base, args.Head)
	if err != nil {
		return nil, err
	}
	for _, e := range st {
		e.ReadMetadata()
	}
	if st.IsEmpty() {
		return st, nil
	}
	tmpl := stack.ParseTemplate(args.BranchNameTemplate)
	if err := st.AssignHeads(tmpl, args.Username, args.OrigBranch, args.Remote); err != nil {
		return nil, err
	}
	st.AssignBases(args.Target)
	return st, nil
}

func SafeHead(e *stack.Entry) string {
	if e.HasHead() {
		return e.Head()
	}
	return ""
}

func Index(st stack.Stack, entry *stack.Entry) int {
	for i, e := range st {
		if e == entry {
			return i + 1
		}
	}
	return 0
}

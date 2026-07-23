package nativestacks

import (
	"testing"
)

func TestValidateExtensionVersion(t *testing.T) {
	cases := []struct {
		version string
		min     string
		wantErr bool
	}{
		{"0.0.8", "0.0.8", false},
		{"0.0.9", "0.0.8", false},
		{"0.1.0", "0.0.8", false},
		{"1.0.0", "0.0.8", false},
		{"0.0.7", "0.0.8", true},
		{"0.0.8", "0.0.9", true},
		{"", "0.0.8", true},
		{"v0.0.8", "0.0.8", false},
		{"0.0.8-alpha", "0.0.8", true},
		{"not-semver", "0.0.8", true},
	}
	for _, c := range cases {
		err := ValidateExtensionVersion(c.version, c.min)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateExtensionVersion(%q, %q) err = %v, wantErr=%v", c.version, c.min, err, c.wantErr)
		}
	}
}

func ptr(n int) *int { return &n }

func TestClassify(t *testing.T) {
	emptyStacks := StackSet{}
	cases := []struct {
		name        string
		local       []int
		memberships map[int]*Membership
		stacks      StackSet
		wantKind    ActionKind
		wantStack   int
		wantSuffix  []int
	}{
		{
			name:     "single PR is standalone",
			local:    []int{1},
			wantKind: ActionIneligible,
		},
		{
			name:     "empty stack",
			local:    []int{},
			wantKind: ActionIneligible,
		},
		{
			name:     "oversized stack",
			local:    make([]int, 101),
			wantKind: ActionIneligible,
		},
		{
			name:     "all unstacked creates",
			local:    []int{1, 2, 3},
			stacks:   emptyStacks,
			wantKind: ActionCreate,
		},
		{
			name:  "exact match noop",
			local: []int{10, 20, 30},
			memberships: map[int]*Membership{
				10: {PRNumber: 10, StackNumber: ptr(5), Position: 1},
				20: {PRNumber: 20, StackNumber: ptr(5), Position: 2},
				30: {PRNumber: 30, StackNumber: ptr(5), Position: 3},
			},
			stacks: StackSet{
				5: {Number: 5, Base: "main", Size: 3, PRs: []StackPR{
					{Number: 10, Position: 1},
					{Number: 20, Position: 2},
					{Number: 30, Position: 3},
				}},
			},
			wantKind:  ActionNoop,
			wantStack: 5,
		},
		{
			name:  "append unstacked suffix",
			local: []int{10, 20, 30, 40},
			memberships: map[int]*Membership{
				10: {PRNumber: 10, StackNumber: ptr(5), Position: 1},
				20: {PRNumber: 20, StackNumber: ptr(5), Position: 2},
				30: {PRNumber: 30, StackNumber: ptr(5), Position: 3},
			},
			stacks: StackSet{
				5: {Number: 5, Base: "main", Size: 3, PRs: []StackPR{
					{Number: 10, Position: 1},
					{Number: 20, Position: 2},
					{Number: 30, Position: 3},
				}},
			},
			wantKind:   ActionAppend,
			wantStack:  5,
			wantSuffix: []int{40},
		},
		{
			name:  "remote extra PR conflicts",
			local: []int{10, 20},
			memberships: map[int]*Membership{
				10: {PRNumber: 10, StackNumber: ptr(5), Position: 1},
				20: {PRNumber: 20, StackNumber: ptr(5), Position: 2},
			},
			stacks: StackSet{
				5: {Number: 5, Base: "main", Size: 3, PRs: []StackPR{
					{Number: 10, Position: 1},
					{Number: 20, Position: 2},
					{Number: 30, Position: 3},
				}},
			},
			wantKind: ActionConflict,
		},
		{
			name:  "reordered membership conflicts",
			local: []int{20, 10},
			memberships: map[int]*Membership{
				20: {PRNumber: 20, StackNumber: ptr(5), Position: 2},
				10: {PRNumber: 10, StackNumber: ptr(5), Position: 1},
			},
			stacks: StackSet{
				5: {Number: 5, Base: "main", Size: 2, PRs: []StackPR{
					{Number: 10, Position: 1},
					{Number: 20, Position: 2},
				}},
			},
			wantKind: ActionConflict,
		},
		{
			name:  "mixed stacks conflict",
			local: []int{10, 20, 30},
			memberships: map[int]*Membership{
				10: {PRNumber: 10, StackNumber: ptr(5), Position: 1},
				20: {PRNumber: 20, StackNumber: ptr(5), Position: 2},
				30: {PRNumber: 30, StackNumber: ptr(6), Position: 1},
			},
			stacks: StackSet{
				5: {Number: 5, Base: "main", Size: 2, PRs: []StackPR{
					{Number: 10, Position: 1},
					{Number: 20, Position: 2},
				}},
				6: {Number: 6, Base: "main", Size: 1, PRs: []StackPR{
					{Number: 30, Position: 1},
				}},
			},
			wantKind: ActionConflict,
		},
		{
			name:  "suffix in another stack conflicts",
			local: []int{10, 20, 30, 40},
			memberships: map[int]*Membership{
				10: {PRNumber: 10, StackNumber: ptr(5), Position: 1},
				20: {PRNumber: 20, StackNumber: ptr(5), Position: 2},
				30: {PRNumber: 30, StackNumber: ptr(5), Position: 3},
				40: {PRNumber: 40, StackNumber: ptr(6), Position: 1},
			},
			stacks: StackSet{
				5: {Number: 5, Base: "main", Size: 3, PRs: []StackPR{
					{Number: 10, Position: 1},
					{Number: 20, Position: 2},
					{Number: 30, Position: 3},
				}},
				6: {Number: 6, Base: "main", Size: 1, PRs: []StackPR{
					{Number: 40, Position: 1},
				}},
			},
			wantKind: ActionConflict,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.memberships == nil {
				c.memberships = map[int]*Membership{}
			}
			got := Classify(c.local, c.memberships, c.stacks)
			if got.Kind != c.wantKind {
				t.Errorf("Kind = %q, want %q", got.Kind, c.wantKind)
			}
			if c.wantStack != 0 && got.StackNumber != c.wantStack {
				t.Errorf("StackNumber = %d, want %d", got.StackNumber, c.wantStack)
			}
			if len(c.wantSuffix) > 0 && !intSlicesEqual(got.SuffixPRs, c.wantSuffix) {
				t.Errorf("SuffixPRs = %v, want %v", got.SuffixPRs, c.wantSuffix)
			}
		})
	}
}

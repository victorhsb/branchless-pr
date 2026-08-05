package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/config"
	"github.com/victorhsb/branchless-pr/internal/nativestacks"
	"github.com/victorhsb/branchless-pr/internal/stack"
)

type fakeNativeSubmitClient struct {
	created     []int
	appended    []int
	stack       *nativestacks.Stack
	err         error
	probeErr    error
	prs         map[int]*nativestacks.PullRequest
	memberships map[int]*nativestacks.Membership
	stacks      nativestacks.StackSet
}

func (f *fakeNativeSubmitClient) ProbeAvailability() error { return f.probeErr }

func (f *fakeNativeSubmitClient) LoadState([]int) (map[int]*nativestacks.PullRequest, map[int]*nativestacks.Membership, nativestacks.StackSet, error) {
	return f.prs, f.memberships, f.stacks, f.err
}

func (f *fakeNativeSubmitClient) LoadWriteLifecycle(*nativestacks.Result, map[int]*nativestacks.PullRequest) error {
	return nil
}

func (f *fakeNativeSubmitClient) CreateStack(prs []int) (*nativestacks.Stack, error) {
	f.created = append([]int(nil), prs...)
	return f.stack, f.err
}

func (f *fakeNativeSubmitClient) AppendStack(_ int, suffix, _ []int) (*nativestacks.Stack, error) {
	f.appended = append([]int(nil), suffix...)
	return f.stack, f.err
}

func TestReconcileNativeStackUsesDirectRESTClient(t *testing.T) {
	t.Run("create records returned Stack number", func(t *testing.T) {
		client := &fakeNativeSubmitClient{stack: &nativestacks.Stack{Number: 17}}
		plan := &nativestacks.Result{Kind: nativestacks.ActionCreate, LocalPRs: []int{10, 20}}
		got, err := reconcileNativeStack(&nativePreflightResult{enabled: true, client: client, plan: plan})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(client.created, []int{10, 20}) || got.StackNumber != 17 {
			t.Fatalf("created=%v result=%+v", client.created, got)
		}
	})

	t.Run("append sends only suffix", func(t *testing.T) {
		client := &fakeNativeSubmitClient{stack: &nativestacks.Stack{Number: 17}}
		plan := &nativestacks.Result{
			Kind:        nativestacks.ActionAppend,
			LocalPRs:    []int{10, 20, 30},
			StackNumber: 17,
			SuffixPRs:   []int{30},
		}
		if _, err := reconcileNativeStack(&nativePreflightResult{enabled: true, client: client, plan: plan}); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(client.appended, []int{30}) {
			t.Fatalf("appended = %v", client.appended)
		}
	})

	t.Run("write failure is reported", func(t *testing.T) {
		client := &fakeNativeSubmitClient{err: errors.New("REST failure")}
		plan := &nativestacks.Result{Kind: nativestacks.ActionCreate, LocalPRs: []int{10, 20}}
		_, err := reconcileNativeStack(&nativePreflightResult{enabled: true, client: client, plan: plan})
		if err == nil || !strings.Contains(err.Error(), "native Stack create failed") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestProspectiveNativePlanWithMissingPRs(t *testing.T) {
	cases := []struct {
		name     string
		observed *nativestacks.Result
		want     nativestacks.ActionKind
	}{
		{
			name:     "single existing unstacked becomes prospective create",
			observed: &nativestacks.Result{Kind: nativestacks.ActionCreate},
			want:     nativestacks.ActionCreate,
		},
		{
			name: "exact existing Stack becomes prospective append",
			observed: &nativestacks.Result{
				Kind:        nativestacks.ActionNoop,
				StackNumber: 7,
				RemotePRs:   []int{10, 20},
			},
			want: nativestacks.ActionAppend,
		},
		{
			name:     "conflict remains conflict",
			observed: &nativestacks.Result{Kind: nativestacks.ActionConflict},
			want:     nativestacks.ActionConflict,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := prospectiveNativePlan(tc.observed, []int{10})
			if got.Kind != tc.want {
				t.Fatalf("kind = %q, want %q", got.Kind, tc.want)
			}
		})
	}
}

func TestNativeSubmitPreflightModesAndProspectiveActions(t *testing.T) {
	t.Run("auto unavailable falls back", func(t *testing.T) {
		client := &fakeNativeSubmitClient{probeErr: &nativestacks.FeatureUnavailable{}}
		result, err := nativeSubmitPreflightWithClient(twoEntryNativeStack(false), config.NativeStacksAuto, client, "octocat", "hello-world")
		if err != nil {
			t.Fatal(err)
		}
		if result.enabled || result.fallback == "" {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("required unavailable fails", func(t *testing.T) {
		client := &fakeNativeSubmitClient{probeErr: &nativestacks.FeatureUnavailable{}}
		_, err := nativeSubmitPreflightWithClient(twoEntryNativeStack(false), config.NativeStacksRequired, client, "octocat", "hello-world")
		if err == nil {
			t.Fatal("expected required-mode failure")
		}
	})

	t.Run("one existing unstacked PR plans create", func(t *testing.T) {
		client := &fakeNativeSubmitClient{
			memberships: map[int]*nativestacks.Membership{10: {PRNumber: 10}},
			stacks:      nativestacks.StackSet{},
		}
		result, err := nativeSubmitPreflightWithClient(twoEntryNativeStack(true), config.NativeStacksAuto, client, "octocat", "hello-world")
		if err != nil {
			t.Fatal(err)
		}
		if !result.enabled || result.plan.Kind != nativestacks.ActionCreate {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("legacy PRs without membership plan create instead of failing", func(t *testing.T) {
		// Pre-native-stacks PRs carry no stack membership field server-side;
		// LoadState reports them as unstacked and auto mode must backfill them
		// through a create rather than aborting.
		client := &fakeNativeSubmitClient{
			memberships: map[int]*nativestacks.Membership{
				10: {PRNumber: 10},
				20: {PRNumber: 20},
			},
			stacks: nativestacks.StackSet{},
		}
		result, err := nativeSubmitPreflightWithClient(stackForNativeLandRefusalTest(), config.NativeStacksAuto, client, "octocat", "hello-world")
		if err != nil {
			t.Fatal(err)
		}
		if !result.enabled || result.plan.Kind != nativestacks.ActionCreate {
			t.Fatalf("result = %+v", result)
		}
		if !reflect.DeepEqual(result.plan.LocalPRs, []int{10, 20}) {
			t.Fatalf("plan.LocalPRs = %v, want [10 20]", result.plan.LocalPRs)
		}
	})
}

func twoEntryNativeStack(bottomHasPR bool) stack.Stack {
	bottom := entryForLandTest("alice/stack/1", "")
	if bottomHasPR {
		bottom.SetPR("https://github.com/octocat/hello-world/pull/10")
	}
	top := entryForLandTest("alice/stack/2", "")
	return stack.Stack{bottom, top}
}

func TestEnsureUnstackAllowsCleanup(t *testing.T) {
	mergedAt := "2026-07-25T12:00:00Z"
	cases := []struct {
		name    string
		result  *nativestacks.UnstackResult
		wantErr bool
	}{
		{name: "dissolved", result: &nativestacks.UnstackResult{Dissolved: true}},
		{
			name: "only merged local survivor",
			result: &nativestacks.UnstackResult{Stack: &nativestacks.Stack{PRs: []nativestacks.StackPR{
				{Number: 10, State: "closed", MergedAt: &mergedAt},
			}}},
		},
		{
			name: "closed unmerged local survivor",
			result: &nativestacks.UnstackResult{Stack: &nativestacks.Stack{PRs: []nativestacks.StackPR{
				{Number: 10, State: "closed", MergedAt: nil},
			}}},
			wantErr: true,
		},
		{
			name: "open local survivor",
			result: &nativestacks.UnstackResult{Stack: &nativestacks.Stack{PRs: []nativestacks.StackPR{
				{Number: 20, State: "open", MergedAt: nil},
			}}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureUnstackAllowsCleanup(tc.result, []int{10, 20})
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestNativeLandRefusalHasNoMutationPath(t *testing.T) {
	st := stackForNativeLandRefusalTest()
	err := nativeLandRefusal(st, "whole-stack")
	if err == nil {
		t.Fatal("expected native landing refusal")
	}
	if !strings.Contains(err.Error(), st.Top().PR()) || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("error = %v", err)
	}
}

func stackForNativeLandRefusalTest() stack.Stack {
	bottom := entryForLandTest("alice/stack/1", "https://github.com/octocat/hello-world/pull/10")
	top := entryForLandTest("alice/stack/2", "https://github.com/octocat/hello-world/pull/20")
	return stack.Stack{bottom, top}
}

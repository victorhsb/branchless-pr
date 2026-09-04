package pr

import (
	"strings"
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestCreateCapturesPRURL(t *testing.T) {
	runner := shelltest.New(t, shelltest.Response{
		Match:  shelltest.Prefix("gh", "pr", "create"),
		Stdout: "https://github.com/acme/repo/pull/123\n",
	})
	got, err := NewClient(runner).Create(CreateOptions{
		Base:  "main",
		Head:  "alice/stack/1",
		Title: "Test PR",
		Body:  []byte("body"),
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	want := "https://github.com/acme/repo/pull/123"
	if got != want {
		t.Fatalf("Create URL = %q, want %q", got, want)
	}
}

func TestClientsKeepRunnerStateIsolated(t *testing.T) {
	for _, login := range []string{"octocat", "mona"} {
		login := login
		t.Run(login, func(t *testing.T) {
			t.Parallel()
			runner := shelltest.New(t, shelltest.Response{
				Match:  shelltest.Exact("gh", "api", "graphql", "-f", "query=query{viewer{login}}"),
				Stdout: `{"data":{"viewer":{"login":"` + login + `"}}}`,
			})

			got, err := NewClient(runner).GetGHUsername()
			if err != nil {
				t.Fatal(err)
			}
			if got != login {
				t.Fatalf("GetGHUsername = %q, want %q", got, login)
			}
		})
	}
}

func TestLoadForSubmitFetchesEachPR(t *testing.T) {
	response := `{"baseRefName":"main","headRefName":"alice/stack/1","number":42,"state":"OPEN","body":"body","title":"Title","url":"https://github.com/acme/repo/pull/42","mergeStateStatus":"CLEAN","isDraft":false}`
	runner := shelltest.New(t,
		shelltest.Response{Match: shelltest.Prefix("gh", "pr", "view", "1"), Stdout: response},
		shelltest.Response{Match: shelltest.Prefix("gh", "pr", "view", "2"), Stdout: response},
	)
	infos, err := NewClient(runner).LoadForSubmit([]string{"1", "2"})
	if err != nil {
		t.Fatalf("LoadForSubmit returned error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("infos len = %d, want 2", len(infos))
	}
	if got := len(runner.Calls()); got != 2 {
		t.Fatalf("pr view calls = %d, want 2", got)
	}
}

func TestParseCreateOutputFindsLastPullURL(t *testing.T) {
	out := []byte("Creating pull request\nhttps://github.com/acme/repo/pull/122\n✓ Created https://github.com/acme/repo/pull/123\nLearn more at https://docs.github.com/\n")
	got, err := parseCreateOutput(out)
	if err != nil {
		t.Fatalf("parseCreateOutput returned error: %v", err)
	}
	want := "https://github.com/acme/repo/pull/123"
	if got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}

func TestParseCreateOutputRejectsEmptyOutput(t *testing.T) {
	_, err := parseCreateOutput([]byte("\n\t "))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unexpected empty output") {
		t.Fatalf("error = %q, want unexpected empty output", err.Error())
	}
}

func TestViewManyFetchesEachPRRef(t *testing.T) {
	runner := shelltest.New(t,
		shelltest.Response{
			Match:  shelltest.Prefix("gh", "pr", "view", "41"),
			Stdout: `{"baseRefName":"main","headRefName":"alice/stack/1","number":41,"state":"OPEN","body":"body 41","title":"Title 41","url":"https://github.com/acme/repo/pull/41","mergeStateStatus":"CLEAN","isDraft":false}`,
		},
		shelltest.Response{
			Match:  shelltest.Prefix("gh", "pr", "view", "42"),
			Stdout: `{"baseRefName":"alice/stack/1","headRefName":"alice/stack/2","number":42,"state":"OPEN","body":"body 42","title":"Title 42","url":"https://github.com/acme/repo/pull/42","mergeStateStatus":"CLEAN","isDraft":true}`,
		},
	)
	got, err := NewClient(runner).ViewMany([]string{"41", "42"})
	if err != nil {
		t.Fatalf("ViewMany returned error: %v", err)
	}
	if got["41"].Title != "Title 41" || got["42"].Title != "Title 42" || !got["42"].IsDraft {
		t.Fatalf("ViewMany result = %+v", got)
	}
	if count := len(runner.Calls()); count != 2 {
		t.Fatalf("gh view calls = %d, want 2", count)
	}
}

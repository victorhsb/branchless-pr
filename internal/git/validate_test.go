package git

import (
	"testing"

	"github.com/victorhsb/branchless-pr/internal/shell/shelltest"
)

func TestValidateRemoteNameRejectsInjection(t *testing.T) {
	cases := []struct {
		name   string
		remote string
		ok     bool
	}{
		{"plain", "origin", true},
		{"upstream", "upstream", true},
		{"fork with dash inside", "my-fork", true},
		{"dots inside", "co.worker", true},

		// Reproduced as local code execution before this validation existed:
		// `git fetch --prune '--upload-pack=touch PWNED'` created PWNED.
		{"upload-pack option injection", "--upload-pack=touch /tmp/pwned", false},
		{"single dash option", "-c", false},
		{"exec option", "--exec=touch /tmp/pwned", false},

		// git accepts a transport URL wherever it accepts a remote name, so a
		// `--` terminator alone does not stop these.
		{"ext transport url", "ext::sh -c touch% /tmp/pwned", false},
		{"file url", "file:///tmp/evil.git", false},
		{"https url", "https://example.com/a/b.git", false},
		{"scp style", "git@example.com:a/b.git", false},

		{"absolute path", "/tmp/evil.git", false},
		{"relative path", "./evil.git", false},
		{"parent path", "../evil.git", false},

		{"empty", "", false},
		{"embedded space", "ori gin", false},
		{"newline", "origin\nfetch", false},
		{"escape char", "origin\x1b[31m", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRemoteName(tc.remote)
			if tc.ok && err != nil {
				t.Fatalf("ValidateRemoteName(%q) = %v, want nil", tc.remote, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("ValidateRemoteName(%q) = nil, want rejection", tc.remote)
			}
		})
	}
}

func TestValidateRefNameRejectsInjection(t *testing.T) {
	cases := []struct {
		name string
		ref  string
		ok   bool
	}{
		{"plain", "main", true},
		{"slashed", "alice/stack/3", true},
		{"leading dash", "--upload-pack=touch /tmp/pwned", false},
		{"empty", "", false},
		{"space", "my branch", false},
		{"control char", "main\x00", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRefName("branch name", tc.ref)
			if tc.ok != (err == nil) {
				t.Fatalf("ValidateRefName(%q) = %v, ok=%v", tc.ref, err, tc.ok)
			}
		})
	}
}

// The remote wrappers must reject a hostile value before it ever reaches a git
// argument vector, regardless of the `--` terminators they also pass.
func TestRemoteWrappersRejectHostileRemote(t *testing.T) {
	const hostile = "--upload-pack=touch /tmp/pwned"
	repo := New("", shelltest.New(t))

	if err := repo.Fetch(hostile); err == nil {
		t.Error("Fetch accepted a hostile remote")
	}
	if err := repo.ForcePush(hostile, "foo"); err == nil {
		t.Error("ForcePush accepted a hostile remote")
	}
	if _, err := repo.ResolveRemoteRefs(hostile, "foo"); err == nil {
		t.Error("ResolveRemoteRefs accepted a hostile remote")
	}
	if err := repo.ForcePushWithLease(hostile, nil, "foo"); err == nil {
		t.Error("ForcePushWithLease accepted a hostile remote")
	}
	if err := repo.DeleteRemoteBranches(hostile, "foo"); err == nil {
		t.Error("DeleteRemoteBranches accepted a hostile remote")
	}
	if _, _, err := repo.RepoSlug(hostile); err == nil {
		t.Error("RepoSlug accepted a hostile remote")
	}
	if err := repo.TargetExists(hostile, "main"); err == nil {
		t.Error("TargetExists accepted a hostile remote")
	}
	if err := repo.TargetExists("origin", "--upload-pack=touch /tmp/pwned"); err == nil {
		t.Error("TargetExists accepted a hostile target")
	}
}

func TestRemoteWrappersRejectHostileBranchName(t *testing.T) {
	const hostile = "--upload-pack=touch /tmp/pwned"
	repo := New("", shelltest.New(t))

	if err := repo.ForcePush("origin", hostile); err == nil {
		t.Error("ForcePush accepted a hostile branch name")
	}
	if err := repo.DeleteRemoteBranches("origin", hostile); err == nil {
		t.Error("DeleteRemoteBranches accepted a hostile branch name")
	}
	if err := repo.ForcePushWithLease("origin", nil, hostile); err == nil {
		t.Error("ForcePushWithLease accepted a hostile branch name")
	}
}

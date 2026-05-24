package git

import "testing"

func TestParseRepoSlug(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{name: "https with .git", url: "https://github.com/acme/widget.git", wantOwner: "acme", wantRepo: "widget"},
		{name: "https without .git", url: "https://github.com/acme/widget", wantOwner: "acme", wantRepo: "widget"},
		{name: "https trailing slash", url: "https://github.com/acme/widget/", wantOwner: "acme", wantRepo: "widget"},
		{name: "ssh with .git", url: "git@github.com:acme/widget.git", wantOwner: "acme", wantRepo: "widget"},
		{name: "ssh without .git", url: "git@github.com:acme/widget", wantOwner: "acme", wantRepo: "widget"},
		{name: "ssh:// scheme", url: "ssh://git@github.com/acme/widget.git", wantOwner: "acme", wantRepo: "widget"},
		{name: "whitespace tolerated", url: "  https://github.com/acme/widget.git\n", wantOwner: "acme", wantRepo: "widget"},
		{name: "empty", url: "", wantErr: true},
		{name: "unsupported scheme", url: "file:///tmp/repo", wantErr: true},
		{name: "missing repo", url: "https://github.com/acme", wantErr: true},
		{name: "missing owner", url: "https://github.com//repo.git", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, err := parseRepoSlug(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseRepoSlug(%q) = (%q, %q, nil); want error", tc.url, owner, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRepoSlug(%q) returned error: %v", tc.url, err)
			}
			if owner != tc.wantOwner || repo != tc.wantRepo {
				t.Fatalf("parseRepoSlug(%q) = (%q, %q); want (%q, %q)", tc.url, owner, repo, tc.wantOwner, tc.wantRepo)
			}
		})
	}
}

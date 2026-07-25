package nativestacks

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/shell"
)

// MinimumExtensionVersion is the minimum supported github/gh-stack extension
// version. The value is provisional until preview testing confirms a stable
// release for the external-tool link/unstack workflows.
const MinimumExtensionVersion = "0.0.8"

// ExtensionStatus reports whether gh-stack is installed and its version.
type ExtensionStatus struct {
	Installed bool
	Version   string
}

// ErrExtensionMissing is returned when the gh-stack extension is not installed
// or does not satisfy the minimum supported version.
type ErrExtensionMissing struct {
	Version string
	Min     string
}

func (e *ErrExtensionMissing) Error() string {
	if e.Version == "" {
		return fmt.Sprintf("gh-stack extension is not installed; install it with: gh extension install github/gh-stack (minimum version %s)", e.Min)
	}
	return fmt.Sprintf("gh-stack extension version %s is below the minimum supported version %s; upgrade with: gh extension upgrade github/gh-stack", e.Version, e.Min)
}

// FindExtension probes `gh extension list` for github/gh-stack.
func FindExtension() (*ExtensionStatus, error) {
	out, err := shell.Output([]string{"gh", "extension", "list"}, shell.RunOpts{})
	if err != nil {
		return nil, fmt.Errorf("list gh extensions: %w", err)
	}
	return parseExtensionList(out), nil
}

// parseExtensionList scans `gh extension list` output for the gh-stack
// extension and returns its installed version. The output is tab-separated
// with columns: <extension-name>\t<repo>\t<ref>. The extension name (e.g.
// "gh stack") may contain spaces, so we search every whitespace-delimited
// field for the repo identifier rather than assuming a fixed column index.
func parseExtensionList(out string) *ExtensionStatus {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "github/gh-stack" || strings.HasSuffix(f, "/gh-stack") {
				status := &ExtensionStatus{Installed: true}
				if i+1 < len(fields) {
					status.Version = strings.TrimPrefix(fields[i+1], "v")
				}
				return status
			}
		}
	}
	return &ExtensionStatus{Installed: false}
}

var versionRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:[+-].*)?$`)

// ValidateExtensionVersion reports whether version is at least minVersion.
func ValidateExtensionVersion(version, minVersion string) error {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	minVersion = strings.TrimPrefix(strings.TrimSpace(minVersion), "v")
	if version == "" {
		return &ErrExtensionMissing{Version: version, Min: minVersion}
	}
	if !versionRe.MatchString(version) {
		return &ErrExtensionMissing{Version: version, Min: minVersion}
	}
	if !versionRe.MatchString(minVersion) {
		return fmt.Errorf("invalid minimum version %q", minVersion)
	}
	if cmp, err := compareSemVer(version, minVersion); err != nil {
		return err
	} else if cmp < 0 {
		return &ErrExtensionMissing{Version: version, Min: minVersion}
	}
	return nil
}

// compareSemVer compares two semver strings without prerelease handling.
// It returns -1/0/1 for less/equal/greater.
func compareSemVer(a, b string) (int, error) {
	parse := func(s string) ([]int, error) {
		parts := strings.Split(s, ".")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid semver %q", s)
		}
		out := make([]int, 3)
		for i, p := range parts {
			n, err := parseUint(p)
			if err != nil {
				return nil, fmt.Errorf("invalid semver %q: %w", s, err)
			}
			out[i] = n
		}
		return out, nil
	}
	av, err := parse(a)
	if err != nil {
		return 0, err
	}
	bv, err := parse(b)
	if err != nil {
		return 0, err
	}
	for i := range av {
		if av[i] < bv[i] {
			return -1, nil
		}
		if av[i] > bv[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseUint(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-digit in version segment %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

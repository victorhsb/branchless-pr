package cli

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestPrimaryCommandsDoNotPrintBannersOrSuccessMarkers(t *testing.T) {
	files := []string{"view.go", "submit.go", "land.go", "abandon.go"}
	disallowed := []struct {
		name string
		re   *regexp.Regexp
	}{
		{name: "SUCCESS marker", re: regexp.MustCompile(`SUCCESS!`)},
		{name: "colored command header", re: regexp.MustCompile(`Headerf\(`)},
		{name: "dry-run submit banner", re: regexp.MustCompile(`DRY RUN: SUBMIT`)},
		{name: "plain command banner", re: regexp.MustCompile(`fmt\.(?:F)?Print(?:f|ln)?\([^\n]*(?:SUBMIT|VIEW|LAND|ABANDON)`)},
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, rule := range disallowed {
			if rule.re.Match(data) {
				t.Fatalf("%s contains disallowed %s output", file, rule.name)
			}
		}
	}
}

func TestRootCommandSilencesCobraErrorPreambles(t *testing.T) {
	cmd, err := newRootCommand([]string{"view"})
	if err != nil {
		t.Fatalf("newRootCommand: %v", err)
	}
	if !cmd.SilenceUsage {
		t.Fatal("root command should silence usage on runtime errors")
	}
	if !cmd.SilenceErrors {
		t.Fatal("root command should silence Cobra error preambles")
	}
}

func TestOutputContractDocumented(t *testing.T) {
	data, err := os.ReadFile("../../SPEC.md")
	if err != nil {
		t.Fatalf("read SPEC.md: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"Commands do not print command banners",
		"do not print generic success/failure markers",
		"without Cobra's extra `Error:` or usage preambles",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("SPEC.md missing output contract text %q", want)
		}
	}
}

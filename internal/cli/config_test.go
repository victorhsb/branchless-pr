package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigInitCreatesFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("STACKPR_CONFIG", filepath.Join(tmp, ".stack-pr.cfg"))

	out, err := executeRootForTest([]string{"config", "init"})
	if err != nil {
		t.Fatalf("config init failed: %v\nstderr: %s", err, out)
	}
	if !strings.Contains(out, "Created") {
		t.Fatalf("expected 'Created' in output, got: %s", out)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".stack-pr.cfg"))
	if err != nil {
		t.Fatalf("reading created config: %v", err)
	}
	if !strings.Contains(string(data), "[common]") {
		t.Fatal("generated config missing [common] section")
	}
}

func TestConfigInitGuardOverwrite(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("STACKPR_CONFIG", filepath.Join(tmp, ".stack-pr.cfg"))

	_, err := executeRootForTest([]string{"config", "init"})
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	_, err = executeRootForTest([]string{"config", "init"})
	if err == nil {
		t.Fatal("expected second init to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestConfigSetWorksAfterRefactor(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("STACKPR_CONFIG", filepath.Join(tmp, ".stack-pr.cfg"))

	// Create an empty config so set can modify it.
	if err := os.WriteFile(filepath.Join(tmp, ".stack-pr.cfg"), []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeRootForTest([]string{"config", "set", "repo.target=develop"})
	if err != nil {
		t.Fatalf("config set failed: %v\nstderr: %s", err, out)
	}
	if !strings.Contains(out, "repo.target = develop") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestConfigSetBackwardCompatibility(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("STACKPR_CONFIG", filepath.Join(tmp, ".stack-pr.cfg"))

	if err := os.WriteFile(filepath.Join(tmp, ".stack-pr.cfg"), []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeRootForTest([]string{"config", "repo.target=develop"})
	if err != nil {
		t.Fatalf("legacy config syntax failed: %v\nstderr: %s", err, out)
	}
	if !strings.Contains(out, "repo.target = develop") {
		t.Fatalf("unexpected output: %s", out)
	}
}

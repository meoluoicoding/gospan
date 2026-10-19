package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_E2E_RegexTrigger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfig(t, path, "matches:\n  - trigger: \"^;d(\\\\d+)$\"\n    trigger_type: regex\n    replace: \"Section $1\"\n")

	code, stdout, stderr := runCLI(t, "match", "test", "--config", path, "--json", ";d42")
	if code != 0 {
		t.Fatalf("exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "Section 42") {
		t.Fatalf("stdout = %q, want Section 42", stdout)
	}

	// Invalid regex must be rejected with exit code 3.
	bad := filepath.Join(dir, "bad.yaml")
	os.WriteFile(bad, []byte("matches:\n  - trigger: '^;d(\\d+$'\n    trigger_type: regex\n    replace: \"x\"\n"), 0o644)
	code, _, stderr = runCLI(t, "config", "validate", "--config", bad)
	if code != 3 {
		t.Fatalf("invalid regex exit = %d, want 3 (stderr=%s)", code, stderr)
	}
}

// Package integration contains end-to-end tests. cli_e2e_test.go exercises the
// real CLI binary through `go run ./cmd/gospan`, treating it as a black box.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("module root not found")
		}
		dir = parent
	}
}

var (
	binaryOnce sync.Once
	binaryPath string
	binaryErr  error
)

func ensureBinary(t *testing.T) string {
	t.Helper()
	binaryOnce.Do(func() {
		root := moduleRoot(t)
		candidate := filepath.Join(root, "dist", "gospan.exe")
		if _, err := os.Stat(candidate); err == nil {
			binaryPath = candidate
			return
		}
		tmp, err := os.MkdirTemp("", "gospan-e2e-*")
		if err != nil {
			binaryErr = err
			return
		}
		out := filepath.Join(tmp, "gospan.exe")
		cmd := exec.Command("go", "build", "-o", out, "./cmd/gospan")
		cmd.Dir = root
		var buf bytes.Buffer
		cmd.Stderr = &buf
		if err := cmd.Run(); err != nil {
			binaryErr = fmt.Errorf("%v: %s", err, buf.String())
			return
		}
		binaryPath = out
	})
	if binaryErr != nil {
		t.Fatal(binaryErr)
	}
	return binaryPath
}

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(ensureBinary(t), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("exec failed: %v\nstderr: %s", err, stderr.String())
		}
	}
	return code, stdout.String(), stderr.String()
}

func TestCLI_E2E_Version(t *testing.T) {
	code, stdout, _ := runCLI(t, "version")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("version output is empty")
	}
}

func TestCLI_E2E_ConfigInitAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	code, _, stderr := runCLI(t, "config", "init", "--config", path)
	if code != 0 {
		t.Fatalf("init exit = %d (stderr=%s)", code, stderr)
	}
	code, stdout, stderr := runCLI(t, "config", "validate", "--config", path, "--json")
	if code != 0 {
		t.Fatalf("validate exit = %d (stderr=%s)", code, stderr)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout)
	}
	if res["ok"] != true {
		t.Errorf("ok = %v, want true", res["ok"])
	}
}

func TestCLI_E2E_MatchAddTestRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	runCLI(t, "config", "init", "--config", path)

	code, _, stderr := runCLI(t, "match", "add", "--config", path, ";e2e", "end to end value")
	if code != 0 {
		t.Fatalf("add exit = %d (stderr=%s)", code, stderr)
	}

	code, stdout, stderr := runCLI(t, "match", "test", "--config", path, ";e2e")
	if code != 0 || !strings.Contains(stdout, "end to end value") {
		t.Fatalf("test after add: code=%d stdout=%q stderr=%s", code, stdout, stderr)
	}

	code, _, stderr = runCLI(t, "match", "remove", "--config", path, ";e2e")
	if code != 0 {
		t.Fatalf("remove exit = %d (stderr=%s)", code, stderr)
	}

	code, stdout, _ = runCLI(t, "match", "test", "--config", path, ";e2e")
	if code != 0 || !strings.Contains(stdout, "no match") {
		t.Fatalf("after remove: code=%d stdout=%q", code, stdout)
	}
}

func TestCLI_E2E_InvalidConfigExit3(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("matches:\n  - trigger: \"\"\n    replace: \"x\"\n"), 0o644)

	code, _, stderr := runCLI(t, "config", "validate", "--config", path)
	if code != 3 {
		t.Fatalf("exit = %d, want 3 (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stderr, "must not be empty") {
		t.Errorf("stderr = %q, want validation message", stderr)
	}
}

func TestCLI_E2E_VocabRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	runCLI(t, "config", "init", "--config", path)

	code, _, stderr := runCLI(t, "vocab", "add", "--config", path, ";x", "example")
	if code != 0 {
		t.Fatalf("add exit = %d (stderr=%s)", code, stderr)
	}

	code, stdout, stderr := runCLI(t, "vocab", "list", "--config", path)
	if code != 0 || !strings.Contains(stdout, ";x") {
		t.Fatalf("list: code=%d stdout=%q stderr=%s", code, stdout, stderr)
	}

	out := filepath.Join(dir, "export.txt")
	code, _, stderr = runCLI(t, "vocab", "export", "--config", path, out)
	if code != 0 {
		t.Fatalf("export exit = %d (stderr=%s)", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !strings.Contains(string(data), ";x") {
		t.Errorf("export = %q, want ;x", string(data))
	}
}

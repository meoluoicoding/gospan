package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snzpet/gospan/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr, "test-version")
	return code, stdout.String(), stderr.String()
}

const validConfig = `app:
  name: gospan
  log_level: info
matches:
  - trigger: ";kt"
    replace: "kiểm tra"
`

func TestVersion(t *testing.T) {
	code, stdout, _ := run(t, "version")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.TrimSpace(stdout) != "test-version" {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestNoArgsIsUsageError(t *testing.T) {
	code, _, _ := run(t)
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	code, _, _ := run(t, "frobnicate")
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestConfigValidateOK(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, stdout, stderr := run(t, "config", "validate", "--config", path)
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "ok:") {
		t.Errorf("stdout = %q, want ok", stdout)
	}
}

func TestConfigValidateInvalid(t *testing.T) {
	path := writeConfig(t, `matches:
  - trigger: ""
    replace: "x"
`)
	code, _, stderr := run(t, "config", "validate", "--config", path)
	if code != ExitValidation {
		t.Fatalf("exit = %d, want %d (stderr=%s)", code, ExitValidation, stderr)
	}
	if !strings.Contains(stderr, "must not be empty") {
		t.Errorf("stderr = %q, want validation message", stderr)
	}
}

func TestMatchTestMatches(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, stdout, stderr := run(t, "match", "test", "--config", path, ";kt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "kiểm tra") {
		t.Errorf("stdout = %q, want expansion", stdout)
	}
}

func TestMatchTestNotFound(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, stdout, _ := run(t, "match", "test", "--config", path, "nothing")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no match") {
		t.Errorf("stdout = %q, want no-match message", stdout)
	}
}

func TestMatchTestJSON(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, stdout, _ := run(t, "match", "test", "--config", path, "--json", ";kt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "\"matched\": true") {
		t.Errorf("stdout = %q, want matched true", stdout)
	}
}

func TestConfigPath(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, stdout, stderr := run(t, "config", "path", "--config", path, "--json")
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "config.yaml") {
		t.Errorf("stdout = %q, want config.yaml", stdout)
	}
}

func TestDoctorJSON(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, stdout, stderr := run(t, "doctor", "--config", path, "--json")
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "\"config_ok\": true") {
		t.Errorf("stdout = %q, want config_ok true", stdout)
	}
	if !strings.Contains(stdout, "\"triggers\": 1") {
		t.Errorf("stdout = %q, want triggers 1", stdout)
	}
}

func TestStatus(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, stdout, stderr := run(t, "status", "--config", path)
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "triggers") || !strings.Contains(stdout, "1") {
		t.Errorf("stdout = %q, want triggers 1", stdout)
	}
}

func TestHelp(t *testing.T) {
	code, stdout, _ := run(t, "help")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "gospan") {
		t.Errorf("stdout = %q, want usage", stdout)
	}
}

func TestVocabExport(t *testing.T) {
	path := writeConfig(t, validConfig)
	dir := t.TempDir()
	out := filepath.Join(dir, "export.txt")

	code, _, stderr := run(t, "vocab", "add", "--config", path, ";x", "example")
	if code != ExitOK {
		t.Fatalf("add exit = %d (stderr=%s)", code, stderr)
	}

	code, _, stderr = run(t, "vocab", "export", "--config", path, out)
	if code != ExitOK {
		t.Fatalf("export exit = %d (stderr=%s)", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !strings.Contains(string(data), ";x") {
		t.Errorf("export file = %q, want ;x", string(data))
	}
	_ = stderr
}

func TestConfigInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	code, stdout, stderr := run(t, "config", "init", "--config", path)
	if code != ExitOK {
		t.Fatalf("init exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "created") {
		t.Errorf("stdout = %q, want created", stdout)
	}

	// A second init must not overwrite.
	code, _, _ = run(t, "config", "init", "--config", path)
	if code == ExitOK {
		t.Error("second config init should fail")
	}

	// The created configuration must validate.
	code, _, stderr = run(t, "config", "validate", "--config", path)
	if code != ExitOK {
		t.Fatalf("created config does not validate: %s", stderr)
	}
}

func TestMatchAddAndRemoveYAML(t *testing.T) {
	path := writeConfig(t, validConfig)

	code, _, stderr := run(t, "match", "add", "--config", path, ";new", "brand new")
	if code != ExitOK {
		t.Fatalf("add exit = %d (stderr=%s)", code, stderr)
	}

	code, stdout, _ := run(t, "match", "test", "--config", path, ";new")
	if code != ExitOK || !strings.Contains(stdout, "brand new") {
		t.Fatalf("added rule not active: stdout=%q", stdout)
	}

	code, _, stderr = run(t, "match", "remove", "--config", path, ";new")
	if code != ExitOK {
		t.Fatalf("remove exit = %d (stderr=%s)", code, stderr)
	}

	code, stdout, _ = run(t, "match", "test", "--config", path, ";new")
	if code != ExitOK || !strings.Contains(stdout, "no match") {
		t.Fatalf("removed rule still active: stdout=%q", stdout)
	}
}

func TestMatchAddDuplicateFails(t *testing.T) {
	path := writeConfig(t, validConfig)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	code, _, stderr := run(t, "match", "add", "--config", path, ";kt", "duplicate")
	if code != ExitValidation {
		t.Fatalf("exit = %d, want %d (stderr=%s)", code, ExitValidation, stderr)
	}
	if !strings.Contains(stderr, "duplicate") {
		t.Errorf("stderr = %q, want duplicate error", stderr)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Error("configuration file changed despite rejected add")
	}
}

func TestMatchRemoveMissing(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, _, _ := run(t, "match", "remove", "--config", path, ";zzz")
	if code != ExitError {
		t.Fatalf("exit = %d, want %d", code, ExitError)
	}
}

func TestMatchAddFlagsAfterPositional(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, _, stderr := run(t, "match", "add", "--config", path, ";x", "value", "--description", "desc")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr=%s)", code, stderr)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, m := range cfg.Matches {
		if m.Trigger == ";x" {
			if m.Description != "desc" {
				t.Errorf("description = %q, want desc", m.Description)
			}
			return
		}
	}
	t.Error(";x was not added")
}

func TestMatchAddTooManyArgs(t *testing.T) {
	path := writeConfig(t, validConfig)
	code, _, _ := run(t, "match", "add", "--config", path, ";x", "y", "z")
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestMatchRemoveSearchesIncludes(t *testing.T) {
	dir := t.TempDir()
	extra := filepath.Join(dir, "extra.yaml")
	if err := os.WriteFile(extra, []byte("matches:\n  - trigger: \";inc\"\n    replace: \"from include\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("include:\n  - extra.yaml\nmatches: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := run(t, "match", "remove", "--config", path, ";inc")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr=%s)", code, stderr)
	}
	data, err := os.ReadFile(extra)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), ";inc") {
		t.Errorf("included rule not removed: %s", data)
	}
}

func TestMatchTestRegex(t *testing.T) {
	cfg := "matches:\n  - trigger: \"^;d(\\\\d+)$\"\n    trigger_type: regex\n    replace: \"Section $1\"\n"
	path := writeConfig(t, cfg)
	code, stdout, stderr := run(t, "match", "test", "--config", path, "--json", ";d42")
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "Section 42") {
		t.Errorf("stdout = %q, want Section 42", stdout)
	}
}

func TestMatchListAppFilter(t *testing.T) {
	cfg := "matches:\n  - trigger: \";code\"\n    replace: \"console.log()\"\n    apps:\n      include: [\"code.exe\"]\n  - trigger: \";all\"\n    replace: \"everywhere\"\n    apps:\n      exclude: [\"code.exe\"]\n"
	path := writeConfig(t, cfg)
	code, stdout, stderr := run(t, "match", "list", "--config", path, "--app", "code.exe", "--json")
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, ";code") {
		t.Errorf("stdout = %q, want ;code", stdout)
	}
	if strings.Contains(stdout, ";all") {
		t.Errorf("stdout = %q, should not contain ;all", stdout)
	}
}

func TestMatchTestLongMultiline(t *testing.T) {
	path := writeConfig(t, "matches:\n  - trigger: \";sig\"\n    replace: |\n      Trân trọng,\n      Khoa\n")
	code, stdout, stderr := run(t, "match", "test", "--config", path, "--json", ";sig")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, "Trân trọng,") {
		t.Errorf("stdout = %q, want multiline replacement", stdout)
	}
}

func TestConfigValidateRejectsOversizedReplacement(t *testing.T) {
	long := strings.Repeat("b", 200)
	cfg := "engine:\n  max_expansion_length: 100\nmatches:\n  - trigger: \";x\"\n    replace: \"" + long + "\"\n"
	path := writeConfig(t, cfg)

	code, _, stderr := run(t, "config", "validate", "--config", path)
	if code != ExitValidation {
		t.Fatalf("exit = %d, want %d", code, ExitValidation)
	}
	if !strings.Contains(stderr, "exceeds engine.max_expansion_length") {
		t.Errorf("stderr = %q, want length error", stderr)
	}
}

func TestVocabAddAndList(t *testing.T) {
	path := writeConfig(t, validConfig)

	code, _, stderr := run(t, "vocab", "add", "--config", path, ";x", "example")
	if code != ExitOK {
		t.Fatalf("add exit = %d (stderr=%s)", code, stderr)
	}

	code, stdout, stderr := run(t, "vocab", "list", "--config", path)
	if code != ExitOK {
		t.Fatalf("list exit = %d (stderr=%s)", code, stderr)
	}
	if !strings.Contains(stdout, ";x") {
		t.Errorf("stdout = %q, want ;x", stdout)
	}
}

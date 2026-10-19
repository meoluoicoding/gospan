package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteStarterCreatesValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")

	if err := WriteStarter(path); err != nil {
		t.Fatalf("WriteStarter: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("starter configuration does not validate: %v", err)
	}
	if len(cfg.Matches) == 0 {
		t.Error("starter configuration has no matches")
	}
	if cfg.Engine.MaxTriggerLength == 0 {
		t.Error("starter configuration missing engine defaults")
	}
}

func TestWriteStarterRefusesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	const original = "matches: []\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteStarter(path); err == nil {
		t.Fatal("WriteStarter should refuse to overwrite an existing file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Error("existing configuration was modified")
	}
}

func TestIsNotExist(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsNotExist(err) {
		t.Errorf("IsNotExist(%v) = false, want true", err)
	}
}

package config

import (
	"errors"
	"strings"
	"testing"
)

func TestParseAppliesDefaults(t *testing.T) {
	cfg, err := Parse([]byte("matches: []\n"), "test.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.App.Name != "gospan" {
		t.Errorf("App.Name = %q, want gospan", cfg.App.Name)
	}
	if cfg.App.LogLevel != "info" {
		t.Errorf("App.LogLevel = %q, want info", cfg.App.LogLevel)
	}
	if cfg.Engine.MaxTriggerLength != 64 {
		t.Errorf("MaxTriggerLength = %d, want 64", cfg.Engine.MaxTriggerLength)
	}
	if cfg.Suggestions.LearnFromTyping {
		t.Error("learning should be disabled by default")
	}
	if cfg.Input.IgnoreInjectedEvents != true {
		t.Error("injected events should be ignored by default")
	}
}

func TestParseRecordsMatchSourceAndLine(t *testing.T) {
	doc := "app:\n  name: gospan\nmatches:\n  - trigger: \";a\"\n    replace: \"1\"\n  - trigger: \";b\"\n    replace: \"2\"\n"
	cfg, err := Parse([]byte(doc), "f.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(cfg.Matches))
	}
	if cfg.Matches[0].Source != "f.yaml" {
		t.Errorf("source = %q, want f.yaml", cfg.Matches[0].Source)
	}
	if cfg.Matches[0].Line != 4 {
		t.Errorf("first line = %d, want 4", cfg.Matches[0].Line)
	}
	if cfg.Matches[1].Line != 6 {
		t.Errorf("second line = %d, want 6", cfg.Matches[1].Line)
	}
}

func TestValidateDuplicateTriggerReportsLocation(t *testing.T) {
	doc := "\nmatches:\n  - trigger: \";kt\"\n    replace: \"a\"\n  - trigger: \";kt\"\n    replace: \"b\"\n"
	cfg, err := Parse([]byte(doc), "dupes.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected a validation error")
	}

	var verrs ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("error type = %T, want ValidationErrors", err)
	}

	var found *ValidationError
	for _, e := range verrs {
		if strings.Contains(e.Msg, "duplicate trigger") {
			found = e
		}
	}
	if found == nil {
		t.Fatalf("no duplicate-trigger error in: %v", err)
	}
	if found.Line != 5 {
		t.Errorf("duplicate line = %d, want 5", found.Line)
	}
	if !strings.Contains(found.Error(), "dupes.yaml:5") {
		t.Errorf("error %q missing file:line prefix", found.Error())
	}
}

func TestValidateCaseInsensitiveDuplicate(t *testing.T) {
	doc := "matches:\n  - trigger: \";KT\"\n    replace: \"a\"\n  - trigger: \";kt\"\n    replace: \"b\"\n"
	cfg, err := Parse([]byte(doc), "c.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected duplicate error for case-insensitive triggers")
	}

	cfg.Engine.CaseSensitive = true
	if err := Validate(cfg); err != nil {
		t.Errorf("case-sensitive config should accept \";KT\" and \";kt\": %v", err)
	}
}

func TestValidateRejectsEmptyTrigger(t *testing.T) {
	doc := "matches:\n  - trigger: \"\"\n    replace: \"a\"\n"
	cfg, err := Parse([]byte(doc), "e.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	err = Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("Validate = %v, want empty-trigger error", err)
	}
}

func TestValidateRejectsBadLogLevel(t *testing.T) {
	doc := "app:\n  log_level: loud\nmatches: []\n"
	cfg, err := Parse([]byte(doc), "l.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "unknown log level") {
		t.Fatalf("Validate = %v, want log-level error", err)
	}
}

func TestValidateRejectsOversizedTrigger(t *testing.T) {
	doc := "engine:\n  max_trigger_length: 3\nmatches:\n  - trigger: \";toolong\"\n    replace: \"x\"\n"
	cfg, err := Parse([]byte(doc), "o.yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "exceeds engine.max_trigger_length") {
		t.Fatalf("Validate = %v, want length error", err)
	}
}

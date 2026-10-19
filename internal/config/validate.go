package config

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Absolute validation limits. These bound the work any configuration can
// request, protecting the engine from malicious or accidental oversized input.
const (
	HardMaxTriggerLength   = 1024
	HardMaxExpansionLength = 1 << 20 // 1 MiB
)

// ValidationError describes a single configuration problem with location
// information so the user can find and fix it.
type ValidationError struct {
	File  string
	Line  int
	Field string
	Msg   string
}

// Error renders the error in the form:
//
//	configs/matches/vietnamese.yaml:12: duplicate trigger ";kt"
func (e *ValidationError) Error() string {
	var b strings.Builder
	if e.File != "" {
		b.WriteString(e.File)
		if e.Line > 0 {
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(e.Line))
		}
		if e.Field != "" || e.Msg != "" {
			b.WriteString(": ")
		}
	}
	if e.Field != "" {
		b.WriteString(e.Field)
		if e.Msg != "" {
			b.WriteString(": ")
		}
	}
	b.WriteString(e.Msg)
	return b.String()
}

// ValidationErrors aggregates every problem found in a configuration.
type ValidationErrors []*ValidationError

// Error joins all validation problems with newlines.
func (es ValidationErrors) Error() string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "\n")
}

// Validate enforces the semantic rules described in architect.md §6.1. It
// returns a ValidationErrors describing every problem found, or nil.
func Validate(cfg *Config) error {
	var errs ValidationErrors
	file := cfg.App.ConfigPath

	add := func(field, msg string) {
		errs = append(errs, &ValidationError{File: file, Field: field, Msg: msg})
	}

	// App.
	if strings.TrimSpace(cfg.App.Name) == "" {
		add("app.name", "must not be empty")
	}
	switch cfg.App.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		add("app.log_level", fmt.Sprintf("unknown log level %q (want debug, info, warn, or error)", cfg.App.LogLevel))
	}

	// Engine.
	if cfg.Engine.MaxTriggerLength <= 0 {
		add("engine.max_trigger_length", "must be greater than zero")
	} else if cfg.Engine.MaxTriggerLength > HardMaxTriggerLength {
		add("engine.max_trigger_length", fmt.Sprintf("must not exceed %d", HardMaxTriggerLength))
	}
	if cfg.Engine.MaxExpansionLength <= 0 {
		add("engine.max_expansion_length", "must be greater than zero")
	} else if cfg.Engine.MaxExpansionLength > HardMaxExpansionLength {
		add("engine.max_expansion_length", fmt.Sprintf("must not exceed %d", HardMaxExpansionLength))
	}
	if utf8.RuneCountInString(cfg.Engine.EscapeChar) > 1 {
		add("engine.escape_char", "must be a single character (or empty to disable)")
	}

	// Input.
	if cfg.Input.MinimumIntervalMS < 0 {
		add("input.minimum_interval_ms", "must not be negative")
	}
	if cfg.Input.BufferSize <= 0 {
		add("input.buffer_size", "must be greater than zero")
	} else if cfg.Engine.MaxTriggerLength > 0 && cfg.Input.BufferSize < cfg.Engine.MaxTriggerLength {
		add("input.buffer_size", fmt.Sprintf("must be at least engine.max_trigger_length (%d)", cfg.Engine.MaxTriggerLength))
	}

	// Suggestions.
	if cfg.Suggestions.MinimumOccurrences <= 0 {
		add("suggestions.minimum_occurrences", "must be greater than zero")
	}
	if cfg.Suggestions.MinimumPhraseLength <= 0 {
		add("suggestions.minimum_phrase_length", "must be greater than zero")
	}
	if cfg.Suggestions.MaximumCandidates <= 0 {
		add("suggestions.maximum_candidates", "must be greater than zero")
	}

	// Storage.
	switch cfg.Storage.Backend {
	case "jsonl", "sqlite":
	default:
		add("storage.backend", fmt.Sprintf("unknown backend %q (want jsonl or sqlite)", cfg.Storage.Backend))
	}
	if strings.TrimSpace(cfg.Storage.Path) == "" {
		add("storage.path", "must not be empty")
	}

	// Insertion.
	switch cfg.Insertion.Strategy {
	case "auto", "backspace", "clipboard":
	default:
		add("insertion.strategy", fmt.Sprintf("unknown strategy %q (want auto, backspace, or clipboard)", cfg.Insertion.Strategy))
	}
	if cfg.Insertion.ClipboardThreshold <= 0 {
		add("insertion.clipboard_threshold", "must be greater than zero")
	}

	// Clipboard.
	if cfg.Clipboard.MaxLength <= 0 {
		add("clipboard.max_length", "must be greater than zero")
	}

	// Matches.
	errs = append(errs, validateMatches(cfg)...)

	if len(errs) == 0 {
		return nil
	}
	return errs
}

func validateMatches(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	seen := make(map[string]int, len(cfg.Matches)) // normalized trigger -> index of first definition

	for i := range cfg.Matches {
		m := &cfg.Matches[i]

		trigger := m.Trigger
		if trigger == "" {
			errs = append(errs, &ValidationError{File: m.Source, Line: m.Line, Field: "trigger", Msg: "must not be empty"})
			continue
		}

		triggerType := m.TriggerType
		if triggerType == "" {
			triggerType = "exact"
		}
		switch triggerType {
		case "exact", "regex":
		default:
			errs = append(errs, &ValidationError{File: m.Source, Line: m.Line, Field: "trigger_type",
				Msg: fmt.Sprintf("unknown trigger_type %q (want exact or regex)", m.TriggerType)})
			continue
		}
		if triggerType == "regex" {
			if _, err := regexp.Compile(trigger); err != nil {
				errs = append(errs, &ValidationError{File: m.Source, Line: m.Line, Field: "trigger",
					Msg: fmt.Sprintf("invalid regex: %v", err)})
				continue
			}
		}

		// App filters: validate glob patterns.
		for _, pat := range append(append([]string{}, m.Apps.Include...), m.Apps.Exclude...) {
			if _, err := path.Match(strings.ToLower(strings.TrimSpace(pat)), "x"); err != nil {
				errs = append(errs, &ValidationError{File: m.Source, Line: m.Line, Field: "apps",
					Msg: fmt.Sprintf("invalid glob pattern %q: %v", pat, err)})
			}
		}

		if n := utf8.RuneCountInString(trigger); n > cfg.Engine.MaxTriggerLength {
			errs = append(errs, &ValidationError{
				File: m.Source, Line: m.Line, Field: "trigger",
				Msg: fmt.Sprintf("length %d exceeds engine.max_trigger_length (%d)", n, cfg.Engine.MaxTriggerLength),
			})
		}
		if n := utf8.RuneCountInString(m.Replace); n > cfg.Engine.MaxExpansionLength {
			errs = append(errs, &ValidationError{
				File: m.Source, Line: m.Line, Field: "replace",
				Msg: fmt.Sprintf("length %d exceeds engine.max_expansion_length (%d)", n, cfg.Engine.MaxExpansionLength),
			})
		}

		// Duplicate detection only applies to exact triggers.
		if triggerType == "exact" {
			key := trigger
			if !cfg.Engine.CaseSensitive {
				key = strings.ToLower(trigger)
			}
			if prev, ok := seen[key]; ok {
				first := &cfg.Matches[prev]
				errs = append(errs, &ValidationError{
					File: m.Source, Line: m.Line, Field: "trigger",
					Msg: fmt.Sprintf("duplicate trigger %q; first defined at %s",
						trigger, location(first.Source, first.Line)),
				})
				continue
			}
			seen[key] = i
		}
	}
	return errs
}

func location(file string, line int) string {
	switch {
	case file == "" && line == 0:
		return "unknown location"
	case line == 0:
		return file
	default:
		return file + ":" + strconv.Itoa(line)
	}
}

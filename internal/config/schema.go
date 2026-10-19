// Package config defines the Gospan configuration schema, loader, validator,
// and file watcher.
//
// The configuration is the human-editable source of truth for trigger rules.
// Loading is a two-phase process: parse YAML into Go structs, then validate
// semantic rules before the configuration becomes active. A partially
// constructed or invalid configuration is never published.
package config

// Config is the root configuration document.
type Config struct {
	App         AppConfig         `yaml:"app"`
	Engine      EngineConfig      `yaml:"engine"`
	Input       InputConfig       `yaml:"input"`
	Suggestions SuggestionsConfig `yaml:"suggestions"`
	Storage     StorageConfig     `yaml:"storage"`
	Insertion   InsertionConfig   `yaml:"insertion"`
	Clipboard   ClipboardConfig   `yaml:"clipboard"`

	// Include lists additional YAML files (relative to the including file)
	// whose `matches` are merged into this configuration.
	Include []string `yaml:"include"`

	// Matches holds the active trigger rules, in document order.
	Matches []MatchConfig `yaml:"matches"`
}

// AppConfig contains application-level settings.
type AppConfig struct {
	Name             string `yaml:"name"`
	LogLevel         string `yaml:"log_level"`
	StartWithWindows bool   `yaml:"start_with_windows"`

	// ConfigPath is the absolute path the configuration was loaded from.
	// It is populated by the loader and never read from YAML.
	ConfigPath string `yaml:"-"`
}

// EngineConfig controls matching behavior.
type EngineConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CaseSensitive      bool   `yaml:"case_sensitive"`
	WordBoundary       bool   `yaml:"word_boundary"`
	EscapeChar         string `yaml:"escape_char"`
	MaxTriggerLength   int    `yaml:"max_trigger_length"`
	MaxExpansionLength int    `yaml:"max_expansion_length"`
}

// InputConfig controls keyboard event handling and filtering.
type InputConfig struct {
	MinimumIntervalMS    int      `yaml:"minimum_interval_ms"`
	IgnoreInjectedEvents bool     `yaml:"ignore_injected_events"`
	BufferSize           int      `yaml:"buffer_size"`
	ExcludedProcesses    []string `yaml:"excluded_processes"`
}

// SuggestionsConfig controls the optional vocabulary learning subsystem.
type SuggestionsConfig struct {
	Enabled             bool `yaml:"enabled"`
	LearnFromTyping     bool `yaml:"learn_from_typing"`
	MinimumOccurrences  int  `yaml:"minimum_occurrences"`
	MinimumPhraseLength int  `yaml:"minimum_phrase_length"`
	MaximumCandidates   int  `yaml:"maximum_candidates"`
}

// StorageConfig selects the vocabulary persistence backend.
type StorageConfig struct {
	Backend string `yaml:"backend"`
	Path    string `yaml:"path"`
}

// InsertionConfig controls how replacements are inserted into the target app.
type InsertionConfig struct {
	// Strategy selects the insertion method: "auto" (default), "backspace",
	// or "clipboard".
	Strategy string `yaml:"strategy"`
	// ClipboardThreshold is the replacement length (in runes) above which the
	// "auto" strategy switches to clipboard-assisted insertion.
	ClipboardThreshold int `yaml:"clipboard_threshold"`
}

// ClipboardConfig controls the {{clipboard}} template variable.
type ClipboardConfig struct {
	// Enabled opts in to reading the clipboard. Disabled by default for privacy.
	Enabled bool `yaml:"enabled"`
	// MaxLength caps the amount of clipboard text inserted (in runes).
	MaxLength int `yaml:"max_length"`
}

// MatchConfig is a single trigger rule.
type MatchConfig struct {
	Trigger     string    `yaml:"trigger"`
	TriggerType string    `yaml:"trigger_type"`
	Replace     string    `yaml:"replace"`
	Description string    `yaml:"description"`
	Apps        MatchApps `yaml:"apps"`

	// Enabled allows a rule to be disabled without deleting it. A nil value
	// means enabled (the default).
	Enabled *bool `yaml:"enabled"`

	// Source and Line record where the rule was defined. They are populated
	// by the loader and never read from YAML.
	Source string `yaml:"-"`
	Line   int    `yaml:"-"`
}

// MatchApps restricts a rule to certain foreground applications. An empty
// Include means "all applications"; Exclude always takes precedence.
type MatchApps struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

// IsEnabled reports whether the rule is active. Rules are enabled by default.
func (m MatchConfig) IsEnabled() bool {
	return m.Enabled == nil || *m.Enabled
}

// Defaults returns a configuration with every field set to a safe default.
// The loader unmarshals the user's document on top of these values, so absent
// fields inherit the defaults here.
func Defaults() *Config {
	return &Config{
		App: AppConfig{
			Name:     "gospan",
			LogLevel: "info",
		},
		Engine: EngineConfig{
			Enabled:            true,
			CaseSensitive:      false,
			WordBoundary:       true,
			EscapeChar:         `\`,
			MaxTriggerLength:   64,
			MaxExpansionLength: 16384,
		},
		Input: InputConfig{
			MinimumIntervalMS:    0,
			IgnoreInjectedEvents: true,
			BufferSize:           256,
			ExcludedProcesses:    []string{},
		},
		Suggestions: SuggestionsConfig{
			Enabled:             true,
			LearnFromTyping:     false,
			MinimumOccurrences:  5,
			MinimumPhraseLength: 2,
			MaximumCandidates:   5,
		},
		Storage: StorageConfig{
			Backend: "jsonl",
			Path:    "data/vocabulary.jsonl",
		},
		Insertion: InsertionConfig{
			Strategy:           "auto",
			ClipboardThreshold: 256,
		},
		Clipboard: ClipboardConfig{
			Enabled:   false,
			MaxLength: 4096,
		},
	}
}

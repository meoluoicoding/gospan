package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads the configuration at path, resolves includes, and validates it.
// It returns the last-validated configuration or an error. The returned error
// is a ValidationErrors when semantic rules fail, which callers can render with
// file and line information.
func Load(path string) (*Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg, err := Parse(data, abs)
	if err != nil {
		return nil, err
	}

	visited := map[string]bool{abs: true}
	if err := resolveIncludes(cfg, filepath.Dir(abs), visited); err != nil {
		return nil, err
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	cfg.App.ConfigPath = abs
	return cfg, nil
}

// Parse decodes a YAML document into a Config using sensible defaults. It does
// not validate; call Validate afterwards. The name argument is recorded as the
// source of each match rule so validation errors can reference it.
func Parse(data []byte, name string) (*Config, error) {
	cfg := Defaults()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	lines := matchLineNumbers(&root)

	for i := range cfg.Matches {
		cfg.Matches[i].Source = name
		if i < len(lines) {
			cfg.Matches[i].Line = lines[i]
		}
	}
	return cfg, nil
}

// resolveIncludes merges match rules from every included file into cfg.
func resolveIncludes(cfg *Config, dir string, visited map[string]bool) error {
	for _, inc := range cfg.Include {
		p := inc
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, inc)
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = filepath.Clean(p)
		}
		if visited[abs] {
			return ValidationErrors{{
				File: abs,
				Msg:  "include cycle detected",
			}}
		}
		visited[abs] = true

		data, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("read include %q: %w", inc, err)
		}

		sub, err := Parse(data, abs)
		if err != nil {
			return err
		}
		if err := resolveIncludes(sub, filepath.Dir(abs), visited); err != nil {
			return err
		}
		cfg.Matches = append(cfg.Matches, sub.Matches...)
	}
	return nil
}

// matchLineNumbers returns the source line of each entry in the top-level
// `matches` sequence, or nil when there is none.
func matchLineNumbers(root *yaml.Node) []int {
	seq := sequenceFor(root, "matches")
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil
	}
	lines := make([]int, len(seq.Content))
	for i, n := range seq.Content {
		lines[i] = n.Line
	}
	return lines
}

// sequenceFor returns the top-level mapping value for key, or nil.
func sequenceFor(root *yaml.Node, key string) *yaml.Node {
	doc := root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == key {
			return doc.Content[i+1]
		}
	}
	return nil
}

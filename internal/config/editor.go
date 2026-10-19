package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Editor performs safe, validated edits on a YAML configuration file. It
// operates on the document tree so unrelated sections and comments are
// preserved, validates the result before committing, writes to a temporary file
// in the same directory, keeps a .bak backup, and only then replaces the
// original atomically.
type Editor struct {
	path         string // absolute path of the file to edit
	validateRoot string // absolute path of the root configuration, if known
}

// NewEditor returns an Editor for path. When path is the root configuration,
// pass the same value as validateRoot so the whole configuration (including
// includes) is validated before committing. Pass an empty validateRoot for a
// standalone dictionary file.
func NewEditor(path, validateRoot string) *Editor {
	e := &Editor{path: absPath(path)}
	if strings.TrimSpace(validateRoot) != "" {
		e.validateRoot = absPath(validateRoot)
	}
	return e
}

// Path returns the absolute path being edited.
func (e *Editor) Path() string { return e.path }

// AddMatch appends a trigger rule. Duplicates (including those in included
// files) are rejected and the file is left untouched.
func (e *Editor) AddMatch(trigger, replacement, description string) error {
	if strings.TrimSpace(trigger) == "" {
		return fmt.Errorf("trigger must not be empty")
	}
	if replacement == "" {
		return fmt.Errorf("replacement must not be empty")
	}

	return e.edit(func(root *yaml.Node) error {
		seq := ensureMatchesSeq(root)
		rule := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		rule.Content = append(rule.Content, scalarKey("trigger"), quotedValue(trigger))
		rule.Content = append(rule.Content, scalarKey("replace"), replacementValue(replacement))
		if description != "" {
			rule.Content = append(rule.Content, scalarKey("description"), quotedValue(description))
		}
		seq.Content = append(seq.Content, rule)
		return nil
	})
}

// RemoveMatch deletes the first rule whose trigger matches. It reports whether
// a rule was removed. When caseInsensitive is true, triggers are compared
// case-insensitively.
func (e *Editor) RemoveMatch(trigger string, caseInsensitive bool) (bool, error) {
	removed := false
	err := e.edit(func(root *yaml.Node) error {
		seq := findMatchesSeq(root)
		if seq == nil {
			return nil
		}
		kept := make([]*yaml.Node, 0, len(seq.Content))
		for _, item := range seq.Content {
			if !removed && triggerEqual(nodeTrigger(item), trigger, caseInsensitive) {
				removed = true
				continue
			}
			kept = append(kept, item)
		}
		seq.Content = kept
		return nil
	})
	return removed, err
}

func triggerEqual(a, b string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// edit loads the document, applies mutate, validates, and commits atomically.
func (e *Editor) edit(mutate func(root *yaml.Node) error) error {
	data, err := os.ReadFile(e.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", e.path, err)
	}

	doc, err := documentNode(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", e.path, err)
	}
	if err := mutate(doc.Content[0]); err != nil {
		return err
	}

	out, err := marshal(doc)
	if err != nil {
		return fmt.Errorf("encode %s: %w", e.path, err)
	}

	return e.commit(out)
}

// commit validates the new content and replaces the file atomically.
func (e *Editor) commit(out []byte) error {
	// First validate in-file so errors point at the real path and line.
	cfg, err := Parse(out, e.path)
	if err != nil {
		return err
	}
	if err := Validate(cfg); err != nil {
		return err
	}

	dir := filepath.Dir(e.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".gospan-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	// When editing the root configuration, validate the whole document
	// (including includes) from the temporary file, which lives in the same
	// directory so relative includes resolve correctly.
	if e.validateRoot != "" && sameFile(e.path, e.validateRoot) {
		if _, err := Load(tmpName); err != nil {
			cleanup()
			return relabel(err, tmpName, e.path)
		}
	}

	if _, err := os.Stat(e.path); err == nil {
		if err := backup(e.path); err != nil {
			cleanup()
			return err
		}
	}

	if err := os.Rename(tmpName, e.path); err != nil {
		cleanup()
		return fmt.Errorf("replace %s: %w", e.path, err)
	}
	return nil
}

// documentNode returns a document node wrapping a mapping, creating an empty
// document when data is empty.
func documentNode(data []byte) (*yaml.Node, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}},
		}, nil
	}

	doc := &yaml.Node{}
	if err := yaml.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("unexpected empty document")
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("configuration root must be a mapping")
	}
	return doc, nil
}

func marshal(doc *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ensureMatchesSeq(root *yaml.Node) *yaml.Node {
	if i := findKey(root, "matches"); i >= 0 {
		v := root.Content[i+1]
		if v.Kind != yaml.SequenceNode {
			v.Kind = yaml.SequenceNode
			v.Tag = "!!seq"
			v.Content = nil
		}
		return v
	}
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	root.Content = append(root.Content, scalarKey("matches"), seq)
	return seq
}

func findMatchesSeq(root *yaml.Node) *yaml.Node {
	i := findKey(root, "matches")
	if i < 0 {
		return nil
	}
	v := root.Content[i+1]
	if v.Kind != yaml.SequenceNode {
		return nil
	}
	return v
}

// findKey returns the index of a key in a mapping node, or -1.
func findKey(mapping *yaml.Node, key string) int {
	if mapping.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func nodeTrigger(item *yaml.Node) string {
	i := findKey(item, "trigger")
	if i < 0 {
		return ""
	}
	return item.Content[i+1].Value
}

func scalarKey(k string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}
}

// quotedValue returns a double-quoted string scalar. Quoting is mandatory for
// triggers and descriptions because they may start with YAML indicator
// characters ("#", "-", "*", "!", "&", ...) or resemble other types ("123",
// "true", "null"), which would otherwise change meaning or fail to round-trip.
func quotedValue(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v, Style: yaml.DoubleQuotedStyle}
}

// replacementValue uses a literal block scalar for multi-line replacements so
// they stay readable, and a double-quoted scalar otherwise.
func replacementValue(v string) *yaml.Node {
	if strings.Contains(v, "\n") {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v, Style: yaml.LiteralStyle}
	}
	return quotedValue(v)
}

func backup(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read for backup: %w", err)
	}
	if err := os.WriteFile(path+".bak", data, 0o600); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	return nil
}

// relabel rewrites references to a temporary validation path back to the real
// path so user-facing errors never mention the temp file. It preserves the
// ValidationErrors type and therefore the CLI exit code.
func relabel(err error, from, to string) error {
	var verrs ValidationErrors
	if errors.As(err, &verrs) {
		for _, e := range verrs {
			if e.File == from {
				e.File = to
			}
			e.Msg = strings.ReplaceAll(e.Msg, from, to)
		}
		return verrs
	}
	return errors.New(strings.ReplaceAll(err.Error(), from, to))
}

func sameFile(a, b string) bool {
	return strings.EqualFold(filepath.Clean(absPath(a)), filepath.Clean(absPath(b)))
}

func absPath(p string) string {
	if p == "" {
		return ""
	}
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return filepath.Clean(p)
}

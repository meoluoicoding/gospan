// Package engine implements the platform-independent text expansion matching
// engine. It must not import any operating-system specific package.
package engine

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/snzpet/gospan/internal/config"
)

// Match is a resolved trigger and its replacement.
type Match struct {
	Trigger     string
	Replacement string
	Description string
	Apps        config.MatchApps

	// Regex-only fields.
	IsRegex      bool
	Regex        *regexp.Regexp
	MatchedText  string
	MatchIndices []int
}

// MatchContext carries ambient information about the current input, such as the
// foreground application, used for app-specific filtering.
type MatchContext struct {
	ProcessName string
}

// regexRule is a compiled regex trigger.
type regexRule struct {
	pattern *regexp.Regexp
	raw     string
	match   Match
}

// entry is an immutable indexed rule (exact triggers only).
type entry struct {
	match  Match
	key    string
	defer_ bool
}

// Index maps normalized triggers to their rules for direct lookup.
type Index struct {
	byTrigger     map[string]entry
	regexRules    []regexRule
	caseSensitive bool
	wordBoundary  bool
	maxTrigger    int
}

// BuildIndex constructs an immutable index from the configured rules.
func BuildIndex(matches []config.MatchConfig, caseSensitive, wordBoundary bool) (*Index, error) {
	idx := &Index{
		byTrigger:     make(map[string]entry, len(matches)),
		caseSensitive: caseSensitive,
		wordBoundary:  wordBoundary,
	}

	for _, m := range matches {
		if !m.IsEnabled() {
			continue
		}
		tt := m.TriggerType
		if tt == "" {
			tt = "exact"
		}

		if tt == "regex" {
			re, err := regexp.Compile(m.Trigger)
			if err != nil {
				return nil, fmt.Errorf("invalid regex %q: %w", m.Trigger, err)
			}
			for _, r := range idx.regexRules {
				if r.raw == m.Trigger {
					return nil, fmt.Errorf("duplicate regex trigger %q", m.Trigger)
				}
			}
			idx.regexRules = append(idx.regexRules, regexRule{
				pattern: re,
				raw:     m.Trigger,
				match: Match{
					Trigger:     m.Trigger,
					Replacement: m.Replace,
					Description: m.Description,
					Apps:        m.Apps,
					IsRegex:     true,
					Regex:       re,
				},
			})
			continue
		}

		key := normalize(m.Trigger, caseSensitive)
		if _, exists := idx.byTrigger[key]; exists {
			return nil, fmt.Errorf("duplicate trigger %q", m.Trigger)
		}
		idx.byTrigger[key] = entry{
			match: Match{
				Trigger:     m.Trigger,
				Replacement: m.Replace,
				Description: m.Description,
				Apps:        m.Apps,
			},
			key: key,
		}
		if n := len([]rune(m.Trigger)); n > idx.maxTrigger {
			idx.maxTrigger = n
		}
	}
	markDeferred(idx.byTrigger)
	return idx, nil
}

func markDeferred(byTrigger map[string]entry) {
	keys := make([]string, 0, len(byTrigger))
	for k := range byTrigger {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := 0; i+1 < len(keys); i++ {
		if strings.HasPrefix(keys[i+1], keys[i]) {
			e := byTrigger[keys[i]]
			e.defer_ = true
			byTrigger[keys[i]] = e
		}
	}
}

func (ix *Index) Len() int {
	return len(ix.byTrigger) + len(ix.regexRules)
}

func normalize(s string, caseSensitive bool) string {
	if caseSensitive {
		return s
	}
	return strings.ToLower(s)
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func allowedByApps(processName string, apps config.MatchApps) bool {
	if len(apps.Include) == 0 && len(apps.Exclude) == 0 {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(processName))
	for _, pat := range apps.Exclude {
		if globMatch(pat, name) {
			return false
		}
	}
	if len(apps.Include) == 0 {
		return true
	}
	for _, pat := range apps.Include {
		if globMatch(pat, name) {
			return true
		}
	}
	return false
}

func globMatch(pattern, name string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	ok, err := path.Match(pattern, name)
	if err != nil {
		return false
	}
	return ok
}

// matchAt finds the longest exact trigger ending at the end of buf.
func (ix *Index) matchAt(buf []rune, ctx MatchContext) (entry, int, bool) {
	max := ix.maxTrigger
	if max > len(buf) {
		max = len(buf)
	}
	for n := max; n >= 1; n-- {
		start := len(buf) - n
		if !ix.boundaryOK(buf, start) {
			continue
		}
		key := normalize(string(buf[start:]), ix.caseSensitive)
		e, ok := ix.byTrigger[key]
		if !ok {
			continue
		}
		if !allowedByApps(ctx.ProcessName, e.match.Apps) {
			continue
		}
		return e, start, true
	}
	return entry{}, 0, false
}

func (ix *Index) boundaryOK(buf []rune, start int) bool {
	if !ix.wordBoundary {
		return true
	}
	if start == 0 {
		return true
	}
	if !isWordRune(buf[start]) {
		return true
	}
	return !isWordRune(buf[start-1])
}

// matchRegex tries all regex rules and returns the best suffix match.
func (ix *Index) matchRegex(buf []rune, ctx MatchContext) (regexRule, string, []int, bool) {
	if len(ix.regexRules) == 0 {
		return regexRule{}, "", nil, false
	}
	s := string(buf)
	bestIdx := -1
	var bestMatch string
	var bestLoc []int
	for i, rule := range ix.regexRules {
		if !allowedByApps(ctx.ProcessName, rule.match.Apps) {
			continue
		}
		loc := runRegexWithTimeout(rule.pattern, s, 100*time.Millisecond)
		if loc == nil {
			continue
		}
		if loc[1] != len(s) {
			continue
		}
		matched := s[loc[0]:loc[1]]
		startInBuf := loc[0]
		if !ix.boundaryOKRegex(buf, startInBuf, matched) {
			continue
		}
		if bestIdx < 0 || len(matched) > len(bestMatch) ||
			(len(matched) == len(bestMatch) && i < bestIdx) {
			bestIdx = i
			bestMatch = matched
			bestLoc = loc
		}
	}
	if bestIdx < 0 {
		return regexRule{}, "", nil, false
	}
	rule := ix.regexRules[bestIdx]
	start := bestLoc[0]
	rel := make([]int, len(bestLoc))
	for i, v := range bestLoc {
		if v >= 0 {
			rel[i] = v - start
		} else {
			rel[i] = -1
		}
	}
	return rule, bestMatch, rel, true
}

func (ix *Index) boundaryOKRegex(buf []rune, start int, matched string) bool {
	if !ix.wordBoundary || len(matched) == 0 {
		return true
	}
	if start == 0 {
		return true
	}
	first := []rune(matched)[0]
	if !isWordRune(first) {
		return true
	}
	return !isWordRune(buf[start-1])
}

func runRegexWithTimeout(re *regexp.Regexp, s string, timeout time.Duration) []int {
	done := make(chan []int, 1)
	go func() {
		done <- re.FindStringSubmatchIndex(s)
	}()
	select {
	case loc := <-done:
		return loc
	case <-time.After(timeout):
		return nil
	}
}

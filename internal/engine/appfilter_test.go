package engine

import (
	"testing"

	"github.com/snzpet/gospan/internal/config"
)

func appRule(trigger, replace string, apps config.MatchApps) config.MatchConfig {
	m := rule(trigger, replace)
	m.Apps = apps
	return m
}

func TestAppSpecificInclude(t *testing.T) {
	e2 := newTestEngine(t, Config{}, appRule(";code", "console.log()", config.MatchApps{Include: []string{"code.exe"}}))
	if _, count := e2.FeedStringContext(";code", MatchContext{ProcessName: "code.exe"}); count != 1 {
		t.Errorf("should expand in code.exe, got count=%d", count)
	}

	e3 := newTestEngine(t, Config{}, appRule(";code", "console.log()", config.MatchApps{Include: []string{"code.exe"}}))
	if _, count := e3.FeedStringContext(";code", MatchContext{ProcessName: "notepad.exe"}); count != 0 {
		t.Errorf("should not expand in notepad.exe, got count=%d", count)
	}
}

func (e *Engine) FeedStringContext(s string, ctx MatchContext) (Result, int) {
	// helper for tests
	var last Result
	count := 0
	for _, r := range s {
		if res, ok := e.FeedWithContext(r, ctx); ok {
			last = res
			count++
		}
	}
	return last, count
}

func TestAppSpecificExclude(t *testing.T) {
	e := newTestEngine(t, Config{}, appRule(";x", "secret", config.MatchApps{Exclude: []string{"keepass.exe"}}))

	if _, count := e.FeedStringContext(";x", MatchContext{ProcessName: "notepad.exe"}); count != 1 {
		t.Errorf("should expand in notepad, got %d", count)
	}
	if _, count := e.FeedStringContext(";x", MatchContext{ProcessName: "keepass.exe"}); count != 0 {
		t.Errorf("should not expand in keepass, got %d", count)
	}
}

func TestAppSpecificGlob(t *testing.T) {
	e := newTestEngine(t, Config{}, appRule(";x", "v", config.MatchApps{Include: []string{"code*.exe"}}))
	if _, count := e.FeedStringContext(";x", MatchContext{ProcessName: "code-insiders.exe"}); count != 1 {
		t.Errorf("glob include should match code-insiders.exe, got %d", count)
	}
	if _, count := e.FeedStringContext(";x", MatchContext{ProcessName: "notepad.exe"}); count != 0 {
		t.Errorf("glob include should not match notepad.exe, got %d", count)
	}
}

func TestAppSpecificEmptyAllowsAll(t *testing.T) {
	e := newTestEngine(t, Config{}, rule(";x", "v"))
	if _, count := e.FeedStringContext(";x", MatchContext{ProcessName: "anything.exe"}); count != 1 {
		t.Errorf("no apps filter should allow all, got %d", count)
	}
	if _, count := e.FeedStringContext(";x", MatchContext{}); count != 1 {
		t.Errorf("empty process name should allow all, got %d", count)
	}
}

func TestAppSpecificRegex(t *testing.T) {
	m := regexMatch(`^;x(\d+)$`, "X$1")
	m.Apps = config.MatchApps{Include: []string{"code.exe"}}
	e := newTestEngine(t, Config{}, m)
	if _, count := e.FeedStringContext(";x42", MatchContext{ProcessName: "code.exe"}); count != 1 {
		t.Errorf("regex should expand in code.exe, got %d", count)
	}
	if _, count := e.FeedStringContext(";x42", MatchContext{ProcessName: "notepad.exe"}); count != 0 {
		t.Errorf("regex should not expand in notepad.exe, got %d", count)
	}
}

func TestAllowedByAppsExported(t *testing.T) {
	if !AllowedByApps("code.exe", config.MatchApps{Include: []string{"code.exe"}}) {
		t.Error("AllowedByApps should return true")
	}
	if AllowedByApps("notepad.exe", config.MatchApps{Include: []string{"code.exe"}}) {
		t.Error("AllowedByApps should return false")
	}
}

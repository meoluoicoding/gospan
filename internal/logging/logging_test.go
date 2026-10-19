package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := []struct {
		input    string
		debugMsg bool
		infoMsg  bool
	}{
		{"debug", true, true},
		{"info", false, true},
		{"", false, true},
		{"bogus", false, true},
		{"warn", false, false},
		{"error", false, false},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		l := New(tc.input, &buf)
		l.Debug("debugmsg")
		l.Info("infomsg")
		out := buf.String()
		if tc.debugMsg && !strings.Contains(out, "debugmsg") {
			t.Errorf("New(%q): debug message missing", tc.input)
		}
		if !tc.debugMsg && strings.Contains(out, "debugmsg") {
			t.Errorf("New(%q): debug message leaked", tc.input)
		}
		if tc.infoMsg && !strings.Contains(out, "infomsg") {
			t.Errorf("New(%q): info message missing", tc.input)
		}
		if !tc.infoMsg && strings.Contains(out, "infomsg") {
			t.Errorf("New(%q): info message leaked", tc.input)
		}
	}
}

func TestNewDefaultsToStderr(t *testing.T) {
	l := New("", nil)
	if l == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestNewWritesToProvidedWriter(t *testing.T) {
	var buf bytes.Buffer
	l := New("debug", &buf)
	l.Info("hello", "k", "v")
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("output = %q, want hello", buf.String())
	}
	if !strings.Contains(buf.String(), "k=v") {
		t.Errorf("output = %q, want k=v", buf.String())
	}
}

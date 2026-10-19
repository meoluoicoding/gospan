package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.1.0", "0.1.1", true},
		{"0.1.0", "0.2.0", true},
		{"0.1.0", "1.0.0", true},
		{"1.0.0", "0.9.9", false},
		{"1.0.0", "1.0.0", false},
		{"v1.2.3", "v1.2.3", false},
		{"1.0.0-rc1", "1.0.0", true},
		{"dev", "0.1.0", true},
	}
	for _, tc := range cases {
		got := IsNewer(tc.current, tc.latest)
		if got != tc.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestCheckLatestOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Release{TagName: "v0.2.0", HTMLURL: "https://example.com/v0.2.0", Name: "v0.2.0"})
	}))
	defer srv.Close()

	old := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = old }()

	rel, err := CheckLatest("meoluoicoding/gospan", 5*time.Second)
	if err != nil {
		t.Fatalf("CheckLatest: %v", err)
	}
	if rel.TagName != "v0.2.0" {
		t.Errorf("TagName = %q, want v0.2.0", rel.TagName)
	}
	if !IsNewer("0.1.0", rel.TagName) {
		t.Error("v0.2.0 should be newer than 0.1.0")
	}
}

func TestCheckLatestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	old := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = old }()

	if _, err := CheckLatest("meoluoicoding/gospan", 5*time.Second); err == nil {
		t.Error("expected error for 404")
	}
}

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want [3]int
	}{
		{"0.1.0", [3]int{0, 1, 0}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.0.0-rc1", [3]int{1, 0, 0}},
		{"dev", [3]int{0, 0, 0}},
		{"", [3]int{0, 0, 0}},
	}
	for _, tc := range cases {
		got := parseVersion(tc.in)
		if got != tc.want {
			t.Errorf("parseVersion(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

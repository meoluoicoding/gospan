// Package update checks for new Gospan releases on GitHub.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Release is the subset of a GitHub release we care about.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

// apiBaseURL is the GitHub API base URL. It is a variable so tests can point it
// at a local server.
var apiBaseURL = "https://api.github.com"

// CheckLatest queries the GitHub releases API for the latest Gospan release.
func CheckLatest(repo string, timeout time.Duration) (Release, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBaseURL, repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gospan-update-check")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Release{}, fmt.Errorf("no releases published yet")
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return Release{}, fmt.Errorf("decode response: %w", err)
	}
	if rel.TagName == "" {
		return Release{}, fmt.Errorf("no release found")
	}
	return rel, nil
}

// IsNewer reports whether latest is strictly newer than current using a simple
// numeric comparison of dot-separated version components. Pre-release suffixes
// are ignored.
func IsNewer(current, latest string) bool {
	cur := parseVersion(current)
	lat := parseVersion(latest)
	for i := 0; i < 3; i++ {
		if lat[i] > cur[i] {
			return true
		}
		if lat[i] < cur[i] {
			return false
		}
	}
	// Numeric parts are equal: a pre-release is older than the release.
	if hasPrerelease(current) && !hasPrerelease(latest) {
		return true
	}
	return false
}

func hasPrerelease(v string) bool {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimSpace(v)
	return strings.ContainsAny(v, "-+")
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimSpace(v)
	// strip pre-release suffix
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out [3]int
	parts := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		n := 0
		for _, c := range parts[i] {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		out[i] = n
	}
	return out
}

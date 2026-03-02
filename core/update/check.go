// Package update checks for newer versions of the codectx CLI.
//
// It queries the GitHub releases API in the background and caches
// the result to avoid repeated network calls. A check runs at most
// once every 24 hours, never blocks the main command, and is
// disabled when CODECTX_NO_UPDATE_CHECK=1 is set.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// releaseURL is the GitHub API endpoint for the latest release.
// It is a var so tests can override it with a local httptest server.
var releaseURL = "https://api.github.com/repos/securacore/codectx/releases/latest"

const (
	// checkInterval is the minimum time between network checks.
	checkInterval = 24 * time.Hour

	// requestTimeout caps how long the HTTP request can take.
	requestTimeout = 3 * time.Second

	// cacheFile is stored relative to the user config directory.
	cacheName = "codectx/update-check.json"
)

// Result holds the outcome of an update check.
type Result struct {
	// Latest is the newest available version (e.g. "0.4.0", no "v" prefix).
	Latest string
	// Current is the running version.
	Current string
	// Available is true when Latest is newer than Current.
	Available bool
}

// cache is the on-disk format for the last check result.
type cache struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// Check returns an update Result or nil if no check was performed.
// It is designed to be called from a goroutine and collected with
// a short timeout so it never blocks the main command.
//
// current is the running version (e.g. "0.3.0" or "dev").
// If current is "dev", no check is performed.
func Check(current string) *Result {
	if os.Getenv("CODECTX_NO_UPDATE_CHECK") == "1" {
		return nil
	}
	if current == "dev" {
		return nil
	}

	current = strings.TrimPrefix(current, "v")
	cached := loadCache()

	// Use cached result if fresh enough.
	if cached != nil && time.Since(cached.CheckedAt) < checkInterval {
		return buildResult(current, cached.LatestVersion)
	}

	// Fetch latest release from GitHub.
	latest, err := FetchLatest()
	if err != nil {
		return nil
	}

	latest = strings.TrimPrefix(latest, "v")
	saveCache(latest)

	return buildResult(current, latest)
}

// buildResult compares versions and returns a Result.
// Uses simple string comparison — both are expected to be semver.
func buildResult(current, latest string) *Result {
	if latest == "" || latest == current {
		return nil
	}

	if !isNewer(latest, current) {
		return nil
	}

	return &Result{
		Latest:    latest,
		Current:   current,
		Available: true,
	}
}

// isNewer returns true if a is a newer semver than b.
// Compares major.minor.patch numerically.
func isNewer(a, b string) bool {
	aParts := splitVersion(a)
	bParts := splitVersion(b)

	for i := 0; i < 3; i++ {
		if aParts[i] > bParts[i] {
			return true
		}
		if aParts[i] < bParts[i] {
			return false
		}
	}
	return false
}

// splitVersion parses "1.2.3" into [1, 2, 3]. Returns [0,0,0] on failure.
func splitVersion(v string) [3]int {
	var parts [3]int
	_, _ = fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}

// cachePath returns the full path to the cache file.
// It respects XDG_CONFIG_HOME when set, falling back to the
// platform-specific user config directory.
func cachePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, cacheName)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, cacheName)
}

// loadCache reads the cached check result from disk.
func loadCache() *cache {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil
	}
	var c cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return &c
}

// saveCache writes the check result to disk.
func saveCache(latest string) {
	c := cache{
		LatestVersion: latest,
		CheckedAt:     time.Now(),
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	path := cachePath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o644)
}

// releaseResponse is the minimal GitHub API response we need.
type releaseResponse struct {
	TagName string `json:"tag_name"`
}

// FetchLatest queries the GitHub releases API for the latest release tag.
func FetchLatest() (string, error) {
	return fetchFromURL(releaseURL)
}

// fetchFromURL queries a given URL for a GitHub release response.
// Separated from fetchLatest so tests can point at an httptest server.
func fetchFromURL(url string) (string, error) {
	client := &http.Client{Timeout: requestTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck_devVersionSkips(t *testing.T) {
	result := Check("dev")
	assert.Nil(t, result)
}

func TestCheck_envDisabled(t *testing.T) {
	t.Setenv("CODECTX_NO_UPDATE_CHECK", "1")
	result := Check("0.1.0")
	assert.Nil(t, result)
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected bool
	}{
		{"patch bump", "0.1.1", "0.1.0", true},
		{"minor bump", "0.2.0", "0.1.5", true},
		{"major bump", "1.0.0", "0.9.9", true},
		{"same version", "0.1.0", "0.1.0", false},
		{"older", "0.1.0", "0.2.0", false},
		{"major older", "0.9.0", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isNewer(tt.a, tt.b))
		})
	}
}

func TestSplitVersion(t *testing.T) {
	assert.Equal(t, [3]int{1, 2, 3}, splitVersion("1.2.3"))
	assert.Equal(t, [3]int{0, 0, 0}, splitVersion("invalid"))
	assert.Equal(t, [3]int{10, 20, 30}, splitVersion("10.20.30"))
}

func TestBuildResult_sameVersion(t *testing.T) {
	result := buildResult("0.1.0", "0.1.0")
	assert.Nil(t, result)
}

func TestBuildResult_newerAvailable(t *testing.T) {
	result := buildResult("0.1.0", "0.2.0")
	require.NotNil(t, result)
	assert.True(t, result.Available)
	assert.Equal(t, "0.1.0", result.Current)
	assert.Equal(t, "0.2.0", result.Latest)
}

func TestBuildResult_olderAvailable(t *testing.T) {
	result := buildResult("0.2.0", "0.1.0")
	assert.Nil(t, result)
}

func TestBuildResult_emptyLatest(t *testing.T) {
	result := buildResult("0.1.0", "")
	assert.Nil(t, result)
}

func TestCacheRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	saveCache("0.5.0")

	c := loadCache()
	require.NotNil(t, c)
	assert.Equal(t, "0.5.0", c.LatestVersion)
	assert.WithinDuration(t, time.Now(), c.CheckedAt, 2*time.Second)
}

func TestLoadCache_missing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c := loadCache()
	assert.Nil(t, c)
}

func TestLoadCache_corrupt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cacheDir := filepath.Join(dir, "codectx")
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(cacheDir, "update-check.json"),
		[]byte("{corrupt"),
		0o644,
	))

	c := loadCache()
	assert.Nil(t, c)
}

func TestFetchLatest_success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
		resp := releaseResponse{TagName: "v0.5.0"}
		data, _ := json.Marshal(resp)
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	// Override releaseURL for test by calling the server directly.
	tag, err := fetchFromURL(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, "v0.5.0", tag)
}

func TestFetchLatest_nonOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	_, err := fetchFromURL(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestFetchLatest_invalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer ts.Close()

	_, err := fetchFromURL(ts.URL)
	assert.Error(t, err)
}

func TestFetchLatest_networkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	_, err := fetchFromURL(ts.URL)
	assert.Error(t, err)
}

func TestCheck_usesCachedWhenFresh(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CODECTX_NO_UPDATE_CHECK", "")

	// Write a fresh cache entry.
	cacheDir := filepath.Join(dir, "codectx")
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))
	c := cache{
		LatestVersion: "0.5.0",
		CheckedAt:     time.Now(),
	}
	data, _ := json.Marshal(c)
	require.NoError(t, os.WriteFile(
		filepath.Join(cacheDir, "update-check.json"),
		data,
		0o644,
	))

	result := Check("0.1.0")
	require.NotNil(t, result)
	assert.True(t, result.Available)
	assert.Equal(t, "0.5.0", result.Latest)
}

func TestCheck_staleCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CODECTX_NO_UPDATE_CHECK", "")

	// Point releaseURL at a closed server so FetchLatest always fails.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()
	old := releaseURL
	releaseURL = ts.URL
	t.Cleanup(func() { releaseURL = old })

	// Write a stale cache entry (25 hours old).
	cacheDir := filepath.Join(dir, "codectx")
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))
	c := cache{
		LatestVersion: "0.5.0",
		CheckedAt:     time.Now().Add(-25 * time.Hour),
	}
	data, _ := json.Marshal(c)
	require.NoError(t, os.WriteFile(
		filepath.Join(cacheDir, "update-check.json"),
		data,
		0o644,
	))

	// FetchLatest fails (closed server), so Check returns nil.
	result := Check("0.1.0")
	assert.Nil(t, result)
}

func TestCheck_stripsVPrefix(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CODECTX_NO_UPDATE_CHECK", "")

	cacheDir := filepath.Join(dir, "codectx")
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))
	c := cache{
		LatestVersion: "0.5.0",
		CheckedAt:     time.Now(),
	}
	data, _ := json.Marshal(c)
	require.NoError(t, os.WriteFile(
		filepath.Join(cacheDir, "update-check.json"),
		data,
		0o644,
	))

	// Pass version with v prefix.
	result := Check("v0.1.0")
	require.NotNil(t, result)
	assert.Equal(t, "0.1.0", result.Current)
	assert.Equal(t, "0.5.0", result.Latest)
}

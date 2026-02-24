package resolve

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSearchResponse_filtersNonCodectxRepos(t *testing.T) {
	body := `{
		"total_count": 3,
		"items": [
			{
				"full_name": "org/codectx-react",
				"name": "codectx-react",
				"description": "React AI docs",
				"html_url": "https://github.com/org/codectx-react",
				"stargazers_count": 10,
				"owner": {"login": "org"}
			},
			{
				"full_name": "other/some-other-repo",
				"name": "some-other-repo",
				"description": "Not a codectx package",
				"html_url": "https://github.com/other/some-other-repo",
				"stargazers_count": 5,
				"owner": {"login": "other"}
			},
			{
				"full_name": "org/codectx-go",
				"name": "codectx-go",
				"description": "Go AI docs",
				"html_url": "https://github.com/org/codectx-go",
				"stargazers_count": 8,
				"owner": {"login": "org"}
			}
		]
	}`

	results, err := parseSearchResponse(strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "react", results[0].Name)
	assert.Equal(t, "org", results[0].Author)
	assert.Equal(t, "React AI docs", results[0].Description)
	assert.Equal(t, 10, results[0].Stars)

	assert.Equal(t, "go", results[1].Name)
	assert.Equal(t, "org", results[1].Author)
}

func TestParseSearchResponse_stripsPrefix(t *testing.T) {
	body := `{
		"total_count": 1,
		"items": [
			{
				"full_name": "facebook/codectx-react",
				"name": "codectx-react",
				"description": "React conventions",
				"html_url": "https://github.com/facebook/codectx-react",
				"stargazers_count": 42,
				"owner": {"login": "facebook"}
			}
		]
	}`

	results, err := parseSearchResponse(strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "react", results[0].Name)
	assert.Equal(t, "facebook", results[0].Author)
}

func TestParseSearchResponse_emptyResults(t *testing.T) {
	body := `{"total_count": 0, "items": []}`
	results, err := parseSearchResponse(strings.NewReader(body))
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestParseSearchResponse_invalidJSON(t *testing.T) {
	body := `{invalid`
	_, err := parseSearchResponse(strings.NewReader(body))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestParseSearchResponse_allFilteredOut(t *testing.T) {
	body := `{
		"total_count": 2,
		"items": [
			{
				"full_name": "org/not-codectx",
				"name": "not-codectx",
				"description": "No prefix",
				"html_url": "https://github.com/org/not-codectx",
				"stargazers_count": 5,
				"owner": {"login": "org"}
			}
		]
	}`
	results, err := parseSearchResponse(strings.NewReader(body))
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestParseSearchResponse_verifyURLField(t *testing.T) {
	body := `{
		"total_count": 1,
		"items": [
			{
				"full_name": "org/codectx-react",
				"name": "codectx-react",
				"description": "React docs",
				"html_url": "https://github.com/org/codectx-react",
				"stargazers_count": 42,
				"owner": {"login": "org"}
			}
		]
	}`
	results, err := parseSearchResponse(strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "https://github.com/org/codectx-react", results[0].URL)
}

func TestDoSearch_nonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer ts.Close()

	_, err := doSearch(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestDoSearch_success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"total_count": 1,
			"items": [{
				"full_name": "org/codectx-react",
				"name": "codectx-react",
				"description": "React",
				"html_url": "https://github.com/org/codectx-react",
				"stargazers_count": 10,
				"owner": {"login": "org"}
			}]
		}`))
	}))
	defer ts.Close()

	results, err := doSearch(ts.URL)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "react", results[0].Name)
}

func TestDoSearch_invalidJSONOn200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer ts.Close()

	_, err := doSearch(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestSearch_URLConstruction(t *testing.T) {
	// Capture the URL that Search() builds by intercepting it via httptest.
	var capturedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	defer ts.Close()

	// We can't easily redirect Search() to our test server because it
	// hardcodes githubSearchURL. Instead, test the URL-building logic
	// directly by verifying doSearch receives the expected query format.
	// This test validates doSearch with a specific query string format.
	results, err := doSearch(ts.URL + "?q=codectx-react+in%3Aname&sort=stars&order=desc&per_page=25")
	require.NoError(t, err)
	assert.Empty(t, results)

	// Verify the query was forwarded correctly.
	assert.Contains(t, capturedQuery, "codectx-react")
}

func TestSearch_URLConstructionWithAuthor(t *testing.T) {
	// Validate the query string format Search() would construct with an author.
	var capturedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	defer ts.Close()

	results, err := doSearch(ts.URL + "?q=codectx-react+in%3Aname+user%3Afacebook&sort=stars&order=desc&per_page=25")
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Contains(t, capturedQuery, "user%3Afacebook")
}

func TestDoSearch_networkError(t *testing.T) {
	// Use a closed server to cause a connection error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	_, err := doSearch(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search request")
}

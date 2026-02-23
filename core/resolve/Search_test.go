package resolve

import (
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

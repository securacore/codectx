package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURL_basic(t *testing.T) {
	ref, source, err := ParseURL("https://github.com/org/codectx-react")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Equal(t, "org", ref.Author)
	assert.Empty(t, ref.Version)
	assert.Equal(t, "https://github.com/org/codectx-react.git", source)
}

func TestParseURL_withGitSuffix(t *testing.T) {
	ref, source, err := ParseURL("https://github.com/org/codectx-go.git")
	require.NoError(t, err)
	assert.Equal(t, "go", ref.Name)
	assert.Equal(t, "org", ref.Author)
	assert.Equal(t, "https://github.com/org/codectx-go.git", source)
}

func TestParseURL_withTreeVersion(t *testing.T) {
	ref, _, err := ParseURL("https://github.com/org/codectx-react/tree/v1.2.0")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Equal(t, "org", ref.Author)
	assert.Equal(t, "1.2.0", ref.Version)
}

func TestParseURL_withTreeVersionNoPrefix(t *testing.T) {
	ref, _, err := ParseURL("https://github.com/org/codectx-react/tree/1.2.0")
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", ref.Version)
}

func TestParseURL_nonGitHub(t *testing.T) {
	_, _, err := ParseURL("https://gitlab.com/org/codectx-react")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only GitHub URLs")
}

func TestParseURL_noCodectxPrefix(t *testing.T) {
	_, _, err := ParseURL("https://github.com/org/some-repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codectx- naming convention")
}

func TestParseURL_missingRepo(t *testing.T) {
	_, _, err := ParseURL("https://github.com/org")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner and repository")
}

func TestParseURL_emptyNameAfterPrefix(t *testing.T) {
	_, _, err := ParseURL("https://github.com/org/codectx-")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty package name")
}

func TestParseURL_wwwGitHub(t *testing.T) {
	ref, _, err := ParseURL("https://www.github.com/org/codectx-react")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Equal(t, "org", ref.Author)
}

func TestParseURL_hyphenatedPackageName(t *testing.T) {
	ref, source, err := ParseURL("https://github.com/org/codectx-my-lib")
	require.NoError(t, err)
	assert.Equal(t, "my-lib", ref.Name)
	assert.Equal(t, "https://github.com/org/codectx-my-lib.git", source)
}

func TestParseURL_trailingSlash(t *testing.T) {
	ref, _, err := ParseURL("https://github.com/org/codectx-react/")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Equal(t, "org", ref.Author)
}

func TestParseURL_nonTreeThirdSegment(t *testing.T) {
	// /blob/main should NOT extract a version.
	ref, _, err := ParseURL("https://github.com/org/codectx-react/blob/main")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Empty(t, ref.Version)
}

func TestParseURL_httpScheme(t *testing.T) {
	ref, source, err := ParseURL("http://github.com/org/codectx-react")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	// Clone URL should still be https.
	assert.Equal(t, "https://github.com/org/codectx-react.git", source)
}

func TestParseURL_treeVersionSourceURL(t *testing.T) {
	// Verify that the source URL for /tree/ URLs is the base repo, not the tree path.
	_, source, err := ParseURL("https://github.com/org/codectx-react/tree/v1.2.0")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/org/codectx-react.git", source)
}

func TestParseURL_extraSegmentsBeyondTree(t *testing.T) {
	// /tree/v1.0.0/src/foo -- version extracted from segments[3], rest ignored.
	ref, _, err := ParseURL("https://github.com/org/codectx-react/tree/v1.0.0/src/foo")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", ref.Version)
}

func TestParseURL_rootURL(t *testing.T) {
	_, _, err := ParseURL("https://github.com/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner and repository")
}

// --- IsURL ---

func TestIsURL_true(t *testing.T) {
	assert.True(t, IsURL("https://github.com/org/codectx-react"))
	assert.True(t, IsURL("http://github.com/org/codectx-react"))
}

func TestIsURL_false(t *testing.T) {
	assert.False(t, IsURL("react@org"))
	assert.False(t, IsURL("react@org:1.0.0"))
	assert.False(t, IsURL("react"))
}

func TestIsURL_emptyString(t *testing.T) {
	assert.False(t, IsURL(""))
}

func TestIsURL_nonHTTPScheme(t *testing.T) {
	assert.True(t, IsURL("ftp://example.com"))
	assert.True(t, IsURL("ssh://git@github.com"))
}

package resolve

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustVersion(s string) *semver.Version {
	v, err := semver.NewVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestMatchVersion_latestWhenNoConstraint(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
		mustVersion("2.0.0"),
		mustVersion("1.5.0"),
	}

	matched, err := matchVersion(versions, "")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", matched.String())
}

func TestMatchVersion_latestWithSingleVersion(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
	}

	matched, err := matchVersion(versions, "")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", matched.String())
}

func TestMatchVersion_caretConstraint(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
		mustVersion("1.2.0"),
		mustVersion("1.9.9"),
		mustVersion("2.0.0"),
	}

	matched, err := matchVersion(versions, "^1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "1.9.9", matched.String())
}

func TestMatchVersion_tildeConstraint(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
		mustVersion("1.0.5"),
		mustVersion("1.1.0"),
		mustVersion("1.2.0"),
	}

	matched, err := matchVersion(versions, "~1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "1.0.5", matched.String())
}

func TestMatchVersion_exactConstraint(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
		mustVersion("1.1.0"),
		mustVersion("2.0.0"),
	}

	matched, err := matchVersion(versions, "1.1.0")
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", matched.String())
}

func TestMatchVersion_noMatch(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
		mustVersion("2.0.0"),
	}

	_, err := matchVersion(versions, "^3.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no version matching")
}

func TestMatchVersion_invalidConstraint(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
	}

	_, err := matchVersion(versions, "not-a-version")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version constraint")
}

func TestMatchVersion_emptyVersions(t *testing.T) {
	_, err := matchVersion([]*semver.Version{}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no versions provided")
}

func TestMatchVersion_preReleaseNotMatchedByCaret(t *testing.T) {
	versions := []*semver.Version{
		mustVersion("1.0.0"),
		mustVersion("1.1.0-beta.1"),
		mustVersion("1.1.0"),
	}

	matched, err := matchVersion(versions, "^1.0.0")
	require.NoError(t, err)
	// Semver caret should match stable releases.
	assert.Equal(t, "1.1.0", matched.String())
}

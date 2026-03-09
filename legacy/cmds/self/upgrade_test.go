package self

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command metadata ---

func TestSelfCommand_metadata(t *testing.T) {
	assert.Equal(t, "self", Command.Name)
	assert.NotEmpty(t, Command.Usage)
	require.Len(t, Command.Commands, 1)
	assert.Equal(t, "upgrade", Command.Commands[0].Name)
}

func TestUpgradeCommand_metadata(t *testing.T) {
	assert.Equal(t, "upgrade", upgradeCommand.Name)
	assert.NotEmpty(t, upgradeCommand.Usage)
	assert.NotNil(t, upgradeCommand.Action)
}

// --- runUpgrade ---

func TestRunUpgrade_devBuild(t *testing.T) {
	neverCalled := func() (string, error) {
		t.Fatal("fetchLatest should not be called for dev builds")
		return "", nil
	}
	neverUpgraded := func(string) error {
		t.Fatal("upgrade should not be called for dev builds")
		return nil
	}

	err := runUpgrade("dev", neverCalled, neverUpgraded)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dev build")
}

func TestRunUpgrade_alreadyOnLatest(t *testing.T) {
	fetchLatest := func() (string, error) {
		return "v0.5.0", nil
	}
	neverUpgraded := func(string) error {
		t.Fatal("upgrade should not be called when already on latest")
		return nil
	}

	err := runUpgrade("0.5.0", fetchLatest, neverUpgraded)
	assert.NoError(t, err)
}

func TestRunUpgrade_alreadyOnLatest_withVPrefix(t *testing.T) {
	fetchLatest := func() (string, error) {
		return "v0.5.0", nil
	}
	neverUpgraded := func(string) error {
		t.Fatal("upgrade should not be called when already on latest")
		return nil
	}

	// Current version has "v" prefix — should still match.
	err := runUpgrade("v0.5.0", fetchLatest, neverUpgraded)
	assert.NoError(t, err)
}

func TestRunUpgrade_fetchError(t *testing.T) {
	fetchLatest := func() (string, error) {
		return "", fmt.Errorf("network timeout")
	}
	neverUpgraded := func(string) error {
		t.Fatal("upgrade should not be called on fetch error")
		return nil
	}

	err := runUpgrade("0.4.0", fetchLatest, neverUpgraded)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check latest version")
	assert.Contains(t, err.Error(), "network timeout")
}

func TestRunUpgrade_upgradeSuccess(t *testing.T) {
	fetchLatest := func() (string, error) {
		return "v0.6.0", nil
	}
	var upgradedTag string
	upgrade := func(tag string) error {
		upgradedTag = tag
		return nil
	}

	err := runUpgrade("0.5.0", fetchLatest, upgrade)
	assert.NoError(t, err)
	assert.Equal(t, "v0.6.0", upgradedTag)
}

func TestRunUpgrade_upgradeError(t *testing.T) {
	fetchLatest := func() (string, error) {
		return "v0.6.0", nil
	}
	upgrade := func(tag string) error {
		return fmt.Errorf("checksum mismatch")
	}

	err := runUpgrade("0.5.0", fetchLatest, upgrade)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade")
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestRunUpgrade_passesCorrectTag(t *testing.T) {
	fetchLatest := func() (string, error) {
		return "v1.2.3", nil
	}
	var receivedTag string
	upgrade := func(tag string) error {
		receivedTag = tag
		return nil
	}

	err := runUpgrade("1.0.0", fetchLatest, upgrade)
	assert.NoError(t, err)
	// The full tag (with "v" prefix) should be passed to upgrade.
	assert.Equal(t, "v1.2.3", receivedTag)
}

func TestRunUpgrade_fetchReturnsTagWithoutVPrefix(t *testing.T) {
	// Edge case: API returns tag without "v" prefix.
	fetchLatest := func() (string, error) {
		return "0.6.0", nil
	}
	var upgradedTag string
	upgrade := func(tag string) error {
		upgradedTag = tag
		return nil
	}

	err := runUpgrade("0.5.0", fetchLatest, upgrade)
	assert.NoError(t, err)
	assert.Equal(t, "0.6.0", upgradedTag)
}

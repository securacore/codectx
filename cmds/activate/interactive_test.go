package activate

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.Config {
	return &config.Config{
		Name: "test-project",
		Packages: []config.PackageDep{
			{Name: "react", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
			{Name: "go", Author: "org", Version: "^2.0.0", Active: config.Activation{Mode: "none"}},
			{Name: "ts", Author: "org", Version: "^3.0.0", Active: config.Activation{Map: &config.ActivationMap{
				Topics: []string{"conventions"},
			}}},
		},
	}
}

// --- Model initialization ---

func TestNewActivateModel_defaults(t *testing.T) {
	cfg := testConfig()
	m := newActivateModel(cfg)

	assert.Equal(t, viewPackages, m.view)
	assert.Len(t, m.packages, 3)
	assert.Equal(t, 0, m.cursor)
	assert.False(t, m.modified)
	assert.False(t, m.saved)
	assert.False(t, m.quitting)
}

func TestNewActivateModel_packageData(t *testing.T) {
	cfg := testConfig()
	m := newActivateModel(cfg)

	assert.Equal(t, "react", m.packages[0].name)
	assert.Equal(t, "org", m.packages[0].author)
	assert.True(t, m.packages[0].activation.IsAll())

	assert.Equal(t, "go", m.packages[1].name)
	assert.True(t, m.packages[1].activation.IsNone())

	assert.Equal(t, "ts", m.packages[2].name)
	assert.True(t, m.packages[2].activation.IsGranular())
}

// --- Window size ---

func TestActivateModel_windowSize(t *testing.T) {
	m := newActivateModel(testConfig())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(activateModel)
	assert.Equal(t, 120, result.width)
	assert.Equal(t, 40, result.height)
}

// --- Global quit ---

func TestActivateModel_quitCtrlC(t *testing.T) {
	m := newActivateModel(testConfig())
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(activateModel)
	assert.True(t, result.quitting)
	assert.False(t, result.saved)
	assert.NotNil(t, cmd)
}

// --- Package view navigation ---

func TestActivateModel_navigatePackages(t *testing.T) {
	m := newActivateModel(testConfig())
	m.width = 80
	m.height = 30

	// Move down.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := updated.(activateModel)
	assert.Equal(t, 1, result.cursor)

	// Move down again.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result = updated.(activateModel)
	assert.Equal(t, 2, result.cursor)

	// At bottom, should stay.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result = updated.(activateModel)
	assert.Equal(t, 2, result.cursor)

	// Move up.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result = updated.(activateModel)
	assert.Equal(t, 1, result.cursor)
}

func TestActivateModel_navigateArrowKeys(t *testing.T) {
	m := newActivateModel(testConfig())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	result := updated.(activateModel)
	assert.Equal(t, 1, result.cursor)

	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyUp})
	result = updated.(activateModel)
	assert.Equal(t, 0, result.cursor)

	// Up at top stays at 0.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyUp})
	result = updated.(activateModel)
	assert.Equal(t, 0, result.cursor)
}

// --- Toggle activation ---

func TestActivateModel_toggleNoneToAll(t *testing.T) {
	m := newActivateModel(testConfig())
	m.cursor = 1 // "go" package, currently none

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := updated.(activateModel)
	assert.True(t, result.packages[1].activation.IsAll())
	assert.True(t, result.modified)
}

func TestActivateModel_toggleAllToNone(t *testing.T) {
	m := newActivateModel(testConfig())
	m.cursor = 0 // "react" package, currently all

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := updated.(activateModel)
	assert.True(t, result.packages[0].activation.IsNone())
	assert.True(t, result.modified)
}

func TestActivateModel_toggleGranularToNone(t *testing.T) {
	m := newActivateModel(testConfig())
	m.cursor = 2 // "ts" package, currently granular

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := updated.(activateModel)
	assert.True(t, result.packages[2].activation.IsNone())
	assert.True(t, result.modified)
}

// --- Save ---

func TestActivateModel_saveWithChanges(t *testing.T) {
	m := newActivateModel(testConfig())
	// Make a change first.
	m.cursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := updated.(activateModel)

	// Now save.
	updated, cmd := result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	result = updated.(activateModel)
	assert.True(t, result.saved)
	assert.True(t, result.quitting)
	assert.NotNil(t, cmd)
}

func TestActivateModel_saveWithoutChanges(t *testing.T) {
	m := newActivateModel(testConfig())

	// Save without changes should not quit.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	result := updated.(activateModel)
	assert.False(t, result.saved)
	assert.False(t, result.quitting)
	assert.Nil(t, cmd)
}

// --- Quit ---

func TestActivateModel_quitQ(t *testing.T) {
	m := newActivateModel(testConfig())
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	result := updated.(activateModel)
	assert.True(t, result.quitting)
	assert.False(t, result.saved)
	assert.NotNil(t, cmd)
}

func TestActivateModel_quitEsc(t *testing.T) {
	m := newActivateModel(testConfig())
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(activateModel)
	assert.True(t, result.quitting)
	assert.NotNil(t, cmd)
}

// --- Drill-in to entries ---

func TestActivateModel_drillIn(t *testing.T) {
	cfg := testConfig()
	m := newActivateModel(cfg)

	// Populate a manifest for drill-in.
	m.packages[0].manifest = &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "core", Description: "Core principles"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "conventions", Description: "React conventions"},
			{ID: "patterns", Description: "React patterns"},
		},
	}

	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	assert.Equal(t, viewEntries, result.view)
	assert.Equal(t, 0, result.drillIndex)
	assert.Equal(t, 0, result.cursor)
	assert.Len(t, result.entries, 3) // 1 foundation + 2 topics

	// All should be active since package is "all".
	for _, e := range result.entries {
		assert.True(t, e.active, "entry %s should be active", e.id)
	}
}

// --- Entry view navigation and toggles ---

func TestActivateModel_entryNavigation(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[0].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "a", Description: "A"},
			{ID: "b", Description: "B"},
			{ID: "c", Description: "C"},
		},
	}
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // drill in
	result := updated.(activateModel)

	// Navigate down.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result = updated.(activateModel)
	assert.Equal(t, 1, result.cursor)

	// Navigate down.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result = updated.(activateModel)
	assert.Equal(t, 2, result.cursor)

	// At bottom.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result = updated.(activateModel)
	assert.Equal(t, 2, result.cursor)
}

func TestActivateModel_entryToggle(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[0].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "a", Description: "A"},
			{ID: "b", Description: "B"},
		},
	}
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // drill in
	result := updated.(activateModel)

	// Entry 0 should be active (package is "all").
	assert.True(t, result.entries[0].active)

	// Toggle off.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result = updated.(activateModel)
	assert.False(t, result.entries[0].active)

	// Toggle on again.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result = updated.(activateModel)
	assert.True(t, result.entries[0].active)
}

func TestActivateModel_entrySelectAll(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[1].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "a", Description: "A"},
			{ID: "b", Description: "B"},
		},
	}
	m.cursor = 1                                           // "go" package, none
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // drill in
	result := updated.(activateModel)

	// Entries should be inactive (package is "none").
	assert.False(t, result.entries[0].active)
	assert.False(t, result.entries[1].active)

	// Select all.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	result = updated.(activateModel)
	assert.True(t, result.entries[0].active)
	assert.True(t, result.entries[1].active)
}

func TestActivateModel_entryDeselectAll(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[0].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "a", Description: "A"},
		},
	}
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	assert.True(t, result.entries[0].active)

	// Deselect all.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	result = updated.(activateModel)
	assert.False(t, result.entries[0].active)
}

// --- Apply entry changes ---

func TestActivateModel_applyAllSelected(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[1].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "a", Description: "A"},
			{ID: "b", Description: "B"},
		},
	}
	m.cursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // drill in
	result := updated.(activateModel)

	// Select all.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	result = updated.(activateModel)

	// Go back.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result = updated.(activateModel)

	assert.Equal(t, viewPackages, result.view)
	assert.True(t, result.packages[1].activation.IsAll())
}

func TestActivateModel_applyNoneSelected(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[0].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "a", Description: "A"},
		},
	}
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	// Deselect all.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	result = updated.(activateModel)

	// Go back.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result = updated.(activateModel)

	assert.True(t, result.packages[0].activation.IsNone())
}

func TestActivateModel_applyGranular(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[0].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "a", Description: "A"},
			{ID: "b", Description: "B"},
		},
	}
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	// Deselect second entry.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // cursor to 1
	result = updated.(activateModel)
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}) // toggle off
	result = updated.(activateModel)

	// Go back.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result = updated.(activateModel)

	assert.True(t, result.packages[0].activation.IsGranular())
	require.NotNil(t, result.packages[0].activation.Map)
	assert.Equal(t, []string{"a"}, result.packages[0].activation.Map.Topics)
}

// --- View rendering ---

func TestActivateModel_viewPackagesRender(t *testing.T) {
	m := newActivateModel(testConfig())
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "Package Activation")
	assert.Contains(t, view, "react@org")
	assert.Contains(t, view, "go@org")
	assert.Contains(t, view, "ts@org")
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "toggle")
}

func TestActivateModel_viewEntriesRender(t *testing.T) {
	m := newActivateModel(testConfig())
	m.packages[0].manifest = &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "conventions", Description: "React conventions"},
		},
	}
	m.width = 80
	m.height = 24
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	view := result.View()
	assert.Contains(t, view, "react@org")
	assert.Contains(t, view, ui.SymbolDone) // active entry checkbox
	assert.Contains(t, view, "conventions")
	assert.Contains(t, view, "esc back")
}

func TestActivateModel_viewQuitting(t *testing.T) {
	m := newActivateModel(testConfig())
	m.quitting = true
	assert.Empty(t, m.View())
}

func TestActivateModel_viewEmpty(t *testing.T) {
	cfg := &config.Config{Name: "empty", Packages: nil}
	m := newActivateModel(cfg)
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "No packages installed")
}

// --- Helper functions ---

func TestActivationsEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b config.Activation
		want bool
	}{
		{"both none", config.Activation{}, config.Activation{}, true},
		{"both all", config.Activation{Mode: "all"}, config.Activation{Mode: "all"}, true},
		{"none vs all", config.Activation{}, config.Activation{Mode: "all"}, false},
		{"none explicit vs implicit", config.Activation{Mode: "none"}, config.Activation{}, true},
		{"same granular", config.Activation{Map: &config.ActivationMap{Topics: []string{"a"}}}, config.Activation{Map: &config.ActivationMap{Topics: []string{"a"}}}, true},
		{"diff granular", config.Activation{Map: &config.ActivationMap{Topics: []string{"a"}}}, config.Activation{Map: &config.ActivationMap{Topics: []string{"b"}}}, false},
		{"map vs string", config.Activation{Map: &config.ActivationMap{Topics: []string{"a"}}}, config.Activation{Mode: "all"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, activationsEqual(tt.a, tt.b))
		})
	}
}

func TestBuildActiveIDSet_all(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "core"}},
		Topics:     []manifest.TopicEntry{{ID: "react"}},
	}
	ids := buildActiveIDSet(config.Activation{Mode: "all"}, m)
	assert.True(t, ids["foundation:core"])
	assert.True(t, ids["topics:react"])
}

func TestBuildActiveIDSet_none(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "core"}},
	}
	ids := buildActiveIDSet(config.Activation{Mode: "none"}, m)
	assert.Nil(t, ids)
}

func TestBuildActiveIDSet_granular(t *testing.T) {
	m := &manifest.Manifest{
		Topics: []manifest.TopicEntry{{ID: "react"}, {ID: "go"}},
	}
	activation := config.Activation{Map: &config.ActivationMap{Topics: []string{"react"}}}
	ids := buildActiveIDSet(activation, m)
	assert.True(t, ids["topics:react"])
	assert.False(t, ids["topics:go"])
}

func TestActivationLabel(t *testing.T) {
	assert.Equal(t, "all", activationLabel(config.Activation{Mode: "all"}))
	assert.Equal(t, "none", activationLabel(config.Activation{Mode: "none"}))
	assert.Contains(t, activationLabel(config.Activation{Map: &config.ActivationMap{Topics: []string{"react"}}}), "topics: react")
}

// --- Package help text ---

func TestActivateModel_helpTextNoChanges(t *testing.T) {
	m := newActivateModel(testConfig())
	help := m.packageHelpText()
	assert.Contains(t, help, "navigate")
	assert.Contains(t, help, "toggle")
	assert.NotContains(t, help, "save")
}

func TestActivateModel_helpTextWithChanges(t *testing.T) {
	m := newActivateModel(testConfig())
	m.modified = true
	help := m.packageHelpText()
	assert.Contains(t, help, "s save")
}

func TestActivateModel_entryHelpText(t *testing.T) {
	m := newActivateModel(testConfig())
	help := m.entryHelpText()
	assert.Contains(t, help, "navigate")
	assert.Contains(t, help, "toggle")
	assert.Contains(t, help, "a all")
	assert.Contains(t, help, "n none")
	assert.Contains(t, help, "esc back")
}

// --- Activation status display ---

func TestActivationStatus(t *testing.T) {
	m := newActivateModel(testConfig())

	s := m.activationStatus(packageItem{activation: config.Activation{Mode: "all"}})
	assert.Contains(t, s, "all")

	s = m.activationStatus(packageItem{activation: config.Activation{Mode: "none"}})
	assert.Contains(t, s, "none")

	s = m.activationStatus(packageItem{
		activation: config.Activation{Map: &config.ActivationMap{Topics: []string{"a", "b"}}},
	})
	assert.Contains(t, s, "2 entries")
}

// --- Entry view scroll indicators ---

func TestActivateModel_entryViewScroll(t *testing.T) {
	m := newActivateModel(testConfig())

	var topics []manifest.TopicEntry
	for i := 0; i < 30; i++ {
		topics = append(topics, manifest.TopicEntry{ID: strings.Repeat("x", 1) + string(rune('a'+i%26)), Description: "desc"})
	}
	m.packages[0].manifest = &manifest.Manifest{Topics: topics}
	m.width = 80
	m.height = 15 // small height to force scrolling
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	view := result.View()
	assert.Contains(t, view, "more")
}

// --- Activation icon tests ---

func TestActivateModel_activationIcon_all(t *testing.T) {
	m := newActivateModel(testConfig())
	icon := m.activationIcon(config.Activation{Mode: "all"}, nil)
	assert.Contains(t, icon, ui.SymbolActive)
}

func TestActivateModel_activationIcon_granular(t *testing.T) {
	m := newActivateModel(testConfig())
	icon := m.activationIcon(config.Activation{Map: &config.ActivationMap{Topics: []string{"a"}}}, nil)
	assert.Contains(t, icon, ui.SymbolPartial)
}

func TestActivateModel_activationIcon_none(t *testing.T) {
	m := newActivateModel(testConfig())
	icon := m.activationIcon(config.Activation{Mode: "none"}, nil)
	assert.Contains(t, icon, ui.SymbolInactive)
}

// --- Activation status tests ---

func TestActivationStatus_granularWithManifest(t *testing.T) {
	m := newActivateModel(testConfig())
	pkg := packageItem{
		activation: config.Activation{Map: &config.ActivationMap{Topics: []string{"a", "b"}}},
		manifest: &manifest.Manifest{
			Topics: []manifest.TopicEntry{
				{ID: "a"}, {ID: "b"}, {ID: "c"},
			},
		},
	}
	s := m.activationStatus(pkg)
	assert.Contains(t, s, "2 of 3 entries")
}

func TestActivationStatus_granularWithoutManifest(t *testing.T) {
	m := newActivateModel(testConfig())
	pkg := packageItem{
		activation: config.Activation{Map: &config.ActivationMap{Foundation: []string{"x"}, Topics: []string{"y"}}},
	}
	s := m.activationStatus(pkg)
	assert.Contains(t, s, "2 entries")
}

// --- Drill-in error path ---

func TestActivateModel_drillInWithLoadError(t *testing.T) {
	cfg := testConfig()
	m := newActivateModel(cfg)
	m.width = 80
	m.height = 24

	// Pre-set error on first package.
	m.packages[0].loadErr = assert.AnError

	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	assert.Equal(t, viewEntries, result.view)
	assert.Nil(t, result.entries)

	// View should show the error.
	view := result.View()
	assert.Contains(t, view, assert.AnError.Error())
}

func TestActivateModel_drillInManifestAlreadyLoaded(t *testing.T) {
	cfg := testConfig()
	m := newActivateModel(cfg)

	// Pre-load manifest.
	m.packages[0].manifest = &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "cached", Description: "Cached entry"},
		},
	}

	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	assert.Equal(t, viewEntries, result.view)
	require.Len(t, result.entries, 1)
	assert.Equal(t, "cached", result.entries[0].id)
}

// --- Entry view: no entries ---

func TestActivateModel_viewEntries_noEntries(t *testing.T) {
	cfg := testConfig()
	m := newActivateModel(cfg)
	m.packages[0].manifest = &manifest.Manifest{} // empty manifest
	m.width = 80
	m.height = 24
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	view := result.View()
	assert.Contains(t, view, "No entries in this package")
}

// --- Entry view: scroll above indicator ---

func TestActivateModel_entryViewScrollAboveIndicator(t *testing.T) {
	m := newActivateModel(testConfig())

	var topics []manifest.TopicEntry
	for i := 0; i < 30; i++ {
		topics = append(topics, manifest.TopicEntry{
			ID:          string(rune('a' + i%26)),
			Description: "desc",
		})
	}
	m.packages[0].manifest = &manifest.Manifest{Topics: topics}
	m.width = 80
	m.height = 15
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(activateModel)

	// Move cursor to near the bottom.
	result.cursor = 20
	view := result.View()
	assert.Contains(t, view, "above")
}

// --- applyEntryChanges: nil manifest guard ---

func TestActivateModel_applyEntryChanges_nilManifest(t *testing.T) {
	m := newActivateModel(testConfig())
	m.drillIndex = 0
	// manifest is nil by default — applyEntryChanges should not panic.
	m.applyEntryChanges()
	// Activation should be unchanged.
	assert.True(t, m.packages[0].activation.IsAll())
}

// --- Init returns nil ---

func TestActivateModel_Init(t *testing.T) {
	m := newActivateModel(testConfig())
	cmd := m.Init()
	assert.Nil(t, cmd)
}

// --- Unknown msg type is ignored ---

func TestActivateModel_Update_unknownMsg(t *testing.T) {
	m := newActivateModel(testConfig())
	type customMsg struct{}
	updated, cmd := m.Update(customMsg{})
	result := updated.(activateModel)
	assert.Equal(t, viewPackages, result.view)
	assert.Nil(t, cmd)
}

// --- Package view padding ---

func TestActivateModel_viewPackages_smallHeight(t *testing.T) {
	m := newActivateModel(testConfig())
	m.width = 80
	m.height = 5 // very small

	// Should render without panic.
	view := m.View()
	assert.Contains(t, view, "Package Activation")
}

// --- activationLabel edge cases ---

func TestActivationLabel_emptyGranular(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{}}
	label := activationLabel(a)
	assert.Empty(t, label)
}

func TestActivationLabel_promptsAndPlans(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{
		Prompts: []string{"lint"},
		Plans:   []string{"migration"},
	}}
	label := activationLabel(a)
	assert.Contains(t, label, "prompts: lint")
	assert.Contains(t, label, "plans: migration")
}

// --- activationEntryCount: nil map ---

func TestActivationEntryCount_nilMap(t *testing.T) {
	a := config.Activation{Map: nil}
	assert.Equal(t, 0, activationEntryCount(a))
}

// --- findPackage edge cases ---

func TestFindPackage_emptyPackageList(t *testing.T) {
	cfg := &config.Config{Packages: nil}
	assert.Equal(t, -1, findPackage(cfg, "react@org"))
}

func TestFindPackage_multipleAtSigns(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "react", Author: "org@extra"},
		},
	}
	// SplitN with 2 means "react" and "org@extra".
	assert.Equal(t, 0, findPackage(cfg, "react@org@extra"))
}

// --- filterManifestForIDs: prompts and plans ---

func TestFilterManifestForIDs_promptsAndPlans(t *testing.T) {
	m := &manifest.Manifest{
		Prompts: []manifest.PromptEntry{
			{ID: "lint"}, {ID: "review"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migration"}, {ID: "refactor"},
		},
	}
	a := config.Activation{Map: &config.ActivationMap{
		Prompts: []string{"review"},
		Plans:   []string{"migration"},
	}}
	filtered := filterManifestForIDs(m, a)
	require.Len(t, filtered.Prompts, 1)
	assert.Equal(t, "review", filtered.Prompts[0].ID)
	require.Len(t, filtered.Plans, 1)
	assert.Equal(t, "migration", filtered.Plans[0].ID)
}

func TestFilterManifestForIDs_nilSectionSlices(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "a"}},
		Topics:     []manifest.TopicEntry{{ID: "b"}},
	}
	// Map is non-nil but Foundation slice is nil — should skip foundation.
	a := config.Activation{Map: &config.ActivationMap{
		Topics: []string{"b"},
	}}
	filtered := filterManifestForIDs(m, a)
	assert.Empty(t, filtered.Foundation)
	require.Len(t, filtered.Topics, 1)
}

// --- toSet ---

func TestToSet_duplicates(t *testing.T) {
	s := toSet([]string{"a", "a", "b"})
	assert.Len(t, s, 2)
	assert.True(t, s["a"])
	assert.True(t, s["b"])
}

package ide

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/core/ide/launcher"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command structure ---

func TestCommand_structure(t *testing.T) {
	assert.Equal(t, "ide", Command.Name)
	assert.Equal(t, "Launch an AI documentation authoring session", Command.Usage)
}

func TestCommand_flags(t *testing.T) {
	flags := Command.Flags
	require.Len(t, flags, 2)

	names := make(map[string]bool)
	for _, f := range flags {
		for _, n := range f.Names() {
			names[n] = true
		}
	}
	assert.True(t, names["resume"], "expected --resume flag")
	assert.True(t, names["list"], "expected --list flag")
}

func TestCommand_actionIsWired(t *testing.T) {
	assert.NotNil(t, Command.Action)
}

// --- listSessions ---

func TestListSessions(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, outputDir string)
	}{
		{
			name:  "no sessions",
			setup: func(t *testing.T, outputDir string) {},
		},
		{
			name: "single session with category",
			setup: func(t *testing.T, outputDir string) {
				s := coreide.NewSession("claude")
				s.Title = "Test document"
				s.Category = "topic"
				require.NoError(t, coreide.Save(outputDir, s))
			},
		},
		{
			name: "multiple sessions in different phases",
			setup: func(t *testing.T, outputDir string) {
				s1 := coreide.NewSession("claude")
				s1.Title = "Discover session"
				s1.Phase = coreide.PhaseDiscover
				require.NoError(t, coreide.Save(outputDir, s1))

				s2 := coreide.NewSession("opencode")
				s2.Title = "Draft session"
				s2.Phase = coreide.PhaseDraft
				s2.Category = "foundation"
				require.NoError(t, coreide.Save(outputDir, s2))
			},
		},
		{
			name: "session without category",
			setup: func(t *testing.T, outputDir string) {
				s := coreide.NewSession("claude")
				s.Title = "No category"
				s.Category = ""
				require.NoError(t, coreide.Save(outputDir, s))
			},
		},
		{
			name:  "sessions directory does not exist",
			setup: func(t *testing.T, outputDir string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			outputDir := filepath.Join(dir, ".codectx")
			require.NoError(t, os.MkdirAll(outputDir, 0o755))
			tt.setup(t, outputDir)

			err := listSessions(outputDir)
			assert.NoError(t, err)
		})
	}
}

// --- pickOrCreateSession ---

func TestPickOrCreateSession(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, outputDir string) *coreide.Session
		wantTitle  string
		wantBin    string
		wantPhase  coreide.Phase
		wantResume bool // true = should resume the setup session
	}{
		{
			name: "creates new when no active sessions",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				return nil
			},
			wantTitle:  "New document",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDiscover,
			wantResume: false,
		},
		{
			name: "auto-resumes single active session",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				s := coreide.NewSession("claude")
				s.Title = "Existing session"
				s.Phase = coreide.PhaseDraft
				require.NoError(t, coreide.Save(outputDir, s))
				return s
			},
			wantTitle:  "Existing session",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDraft,
			wantResume: true,
		},
		{
			name: "creates new when only completed sessions exist",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				s := coreide.NewSession("claude")
				s.Title = "Done session"
				s.Phase = coreide.PhaseComplete
				require.NoError(t, coreide.Save(outputDir, s))
				return s
			},
			wantTitle:  "New document",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDiscover,
			wantResume: false,
		},
		{
			name: "persists new session to disk",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				return nil
			},
			wantTitle:  "New document",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDiscover,
			wantResume: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			outputDir := filepath.Join(dir, ".codectx")
			require.NoError(t, os.MkdirAll(outputDir, 0o755))

			setupSession := tt.setup(t, outputDir)

			l := launcher.NewClaude("/usr/bin/claude")
			session, err := pickOrCreateSession(outputDir, l)
			require.NoError(t, err)
			require.NotNil(t, session)

			assert.Equal(t, tt.wantTitle, session.Title)
			assert.Equal(t, tt.wantBin, session.Bin)
			assert.Equal(t, tt.wantPhase, session.Phase)

			if tt.wantResume {
				assert.Equal(t, setupSession.ID, session.ID)
			} else if setupSession != nil {
				assert.NotEqual(t, setupSession.ID, session.ID)
			}

			// Every created session should be loadable from disk.
			loaded, err := coreide.Load(outputDir, session.ID)
			require.NoError(t, err)
			assert.Equal(t, session.ID, loaded.ID)
		})
	}
}

// --- assemblePrompt ---

// setupProject creates a temp directory with a codectx.yml and optional
// preferences for testing assemblePrompt.
func setupProject(t *testing.T, prefs *preferences.Preferences) *config.Config {
	t.Helper()
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	docsDir := cfg.DocsDir()
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	outputDir := cfg.OutputDir()
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	if prefs != nil {
		require.NoError(t, preferences.Write(outputDir, prefs))
	}

	return cfg
}

func TestAssemblePrompt(t *testing.T) {
	compTrue := true
	compFalse := false

	tests := []struct {
		name        string
		prefs       *preferences.Preferences
		contains    []string
		notContains []string
	}{
		{
			name:     "includes embedded directive",
			prefs:    nil,
			contains: []string{"documentation", "codectx"},
		},
		{
			name: "includes compression enabled",
			prefs: &preferences.Preferences{
				Compression: &compTrue,
				AI:          &preferences.AIConfig{Class: "claude-sonnet-class"},
			},
			contains: []string{
				"Compression is **enabled**",
				"claude-sonnet-class",
				"Project Preferences",
			},
		},
		{
			name: "includes compression disabled",
			prefs: &preferences.Preferences{
				Compression: &compFalse,
			},
			contains: []string{"Compression is **disabled**"},
		},
		{
			name:        "omits preferences section when empty",
			prefs:       &preferences.Preferences{},
			notContains: []string{"Project Preferences"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := setupProject(t, tt.prefs)

			prompt, err := assemblePrompt(cfg)
			require.NoError(t, err)
			assert.NotEmpty(t, prompt)

			for _, s := range tt.contains {
				assert.Contains(t, prompt, s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, prompt, s)
			}
		})
	}
}

// --- mergeManifests ---

func TestMergeManifests_mergesNonOverlapping(t *testing.T) {
	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Description: "Guiding principles"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "go", Description: "Go conventions"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "markdown", Description: "Markdown rules"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Description: "React patterns"},
		},
	}

	result := mergeManifests(dst, src)
	require.Len(t, result.Foundation, 2)
	require.Len(t, result.Topics, 2)

	foundationIDs := []string{result.Foundation[0].ID, result.Foundation[1].ID}
	assert.Contains(t, foundationIDs, "philosophy")
	assert.Contains(t, foundationIDs, "markdown")

	topicIDs := []string{result.Topics[0].ID, result.Topics[1].ID}
	assert.Contains(t, topicIDs, "go")
	assert.Contains(t, topicIDs, "react")
}

func TestMergeManifests_deduplicatesOverlapping(t *testing.T) {
	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Description: "Guiding principles"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Description: "Duplicate"},
			{ID: "markdown", Description: "New entry"},
		},
	}

	result := mergeManifests(dst, src)
	require.Len(t, result.Foundation, 2)
	// The original entry should be preserved (not overwritten by src).
	assert.Equal(t, "Guiding principles", result.Foundation[0].Description)
}

func TestMergeManifests_nilSrc(t *testing.T) {
	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Description: "Guiding principles"},
		},
	}

	result := mergeManifests(dst, nil)
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
}

func TestMergeManifests_mergesAllSections(t *testing.T) {
	dst := &manifest.Manifest{}
	src := &manifest.Manifest{
		Foundation:  []manifest.FoundationEntry{{ID: "f1"}},
		Topics:      []manifest.TopicEntry{{ID: "t1"}},
		Application: []manifest.ApplicationEntry{{ID: "a1"}},
		Prompts:     []manifest.PromptEntry{{ID: "p1"}},
		Plans:       []manifest.PlanEntry{{ID: "pl1"}},
	}

	result := mergeManifests(dst, src)
	assert.Len(t, result.Foundation, 1)
	assert.Len(t, result.Topics, 1)
	assert.Len(t, result.Application, 1)
	assert.Len(t, result.Prompts, 1)
	assert.Len(t, result.Plans, 1)
}

func TestMergeManifests_deduplicatesPlans(t *testing.T) {
	dst := &manifest.Manifest{
		Plans: []manifest.PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
		},
	}
	src := &manifest.Manifest{
		Plans: []manifest.PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
			{ID: "upgrade", Path: "plans/upgrade/README.md"},
		},
	}

	result := mergeManifests(dst, src)
	require.Len(t, result.Plans, 2)
	planIDs := []string{result.Plans[0].ID, result.Plans[1].ID}
	assert.Contains(t, planIDs, "migrate")
	assert.Contains(t, planIDs, "upgrade")
}

// --- assemblePrompt: package project ---

func TestAssemblePrompt_packageProject(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name:     "codectx-test-pkg",
		Type:     "package",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	docsDir := cfg.DocsDir()
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	require.NoError(t, os.MkdirAll(cfg.OutputDir(), 0o755))

	// Create package/ with a manifest.
	pkgDir := filepath.Join(dir, "package")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	pkgManifest := &manifest.Manifest{
		Name:    "codectx-test-pkg",
		Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Description: "React conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	prompt, err := assemblePrompt(cfg)
	require.NoError(t, err)

	assert.Contains(t, prompt, "Package Authoring")
	assert.Contains(t, prompt, "package project")
	// Package manifest entries should be included.
	assert.Contains(t, prompt, "react")
}

func TestAssemblePrompt_packageMergesDocsAndPkgManifest(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name:     "codectx-dual",
		Type:     "package",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	docsDir := cfg.DocsDir()
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	require.NoError(t, os.MkdirAll(cfg.OutputDir(), 0o755))

	// docs/manifest.yml has a prompt entry.
	docsManifest := &manifest.Manifest{
		Name:    "codectx-dual",
		Version: "1.0.0",
		Prompts: []manifest.PromptEntry{
			{ID: "save", Description: "Save session state"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), docsManifest))

	// package/manifest.yml has a topic entry.
	pkgDir := filepath.Join(dir, "package")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	pkgManifest := &manifest.Manifest{
		Name:    "codectx-dual",
		Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "go-patterns", Description: "Go conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	prompt, err := assemblePrompt(cfg)
	require.NoError(t, err)

	// Both docs and package entries should appear in the prompt.
	assert.Contains(t, prompt, "save")
	assert.Contains(t, prompt, "go-patterns")
	assert.Contains(t, prompt, "Package Authoring")
}

func TestAssemblePrompt_regularProjectNoPackageSection(t *testing.T) {
	cfg := setupProject(t, nil)

	prompt, err := assemblePrompt(cfg)
	require.NoError(t, err)

	assert.NotContains(t, prompt, "Package Authoring")
}

package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupHeuristicsTest(t *testing.T) (string, *ObjectStore, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	objectsDir := filepath.Join(dir, "objects")
	store := NewObjectStore(objectsDir)

	// Store some objects of known sizes.
	pathToHash := make(map[string]string)

	files := map[string]string{
		"foundation/a.md":            "# Philosophy\n\nThis is the philosophy document with enough content.\n",
		"foundation/b.md":            "# Conventions\n\nStandard conventions.\n",
		"application/arch/README.md": "# Architecture\n\nSystem architecture overview.\n",
		"topics/go/README.md":        "# Go\n\nGo conventions and patterns for the project.\n",
		"topics/go/spec.md":          "# Go Spec\n\nDetailed specification.\n",
		"prompts/lint/README.md":     "# Lint\n\nRun the linter.\n",
		"plans/migrate/README.md":    "# Migration\n\nMigration plan.\n",
	}

	for path, content := range files {
		hash, err := store.Store([]byte(content))
		require.NoError(t, err)
		pathToHash[path] = hash
	}

	return dir, store, pathToHash
}

func TestGenerateHeuristics_basic(t *testing.T) {
	dir, _, pathToHash := setupHeuristicsTest(t)
	objectsDir := filepath.Join(dir, "objects")

	m := &manifest.Manifest{
		Name: "test-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/a.md", Load: "always"},
			{ID: "conventions", Path: "foundation/b.md"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md", Spec: "topics/go/spec.md"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "lint", Path: "prompts/lint/README.md"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
		},
	}

	provenance := map[string]string{
		"foundation:philosophy":  "local",
		"foundation:conventions": "local",
		"application:arch":       "local",
		"topics:go":              "go@org",
		"prompts:lint":           "local",
		"plans:migrate":          "local",
	}

	h := generateHeuristics(m, pathToHash, provenance, objectsDir)

	// Totals.
	assert.Equal(t, 6, h.Totals.Entries)
	assert.Greater(t, h.Totals.Objects, 0)
	assert.Greater(t, h.Totals.SizeBytes, 0)
	assert.Greater(t, h.Totals.EstimatedTokens, 0)
	assert.Equal(t, 1, h.Totals.AlwaysLoad)

	// Sections.
	require.NotNil(t, h.Sections.Foundation)
	assert.Equal(t, 2, h.Sections.Foundation.Entries)
	assert.Equal(t, 1, h.Sections.Foundation.AlwaysLoad)
	assert.Greater(t, h.Sections.Foundation.SizeBytes, 0)
	assert.Greater(t, h.Sections.Foundation.EstimatedTokens, 0)

	require.NotNil(t, h.Sections.Application)
	assert.Equal(t, 1, h.Sections.Application.Entries)
	assert.Greater(t, h.Sections.Application.SizeBytes, 0)

	require.NotNil(t, h.Sections.Topics)
	assert.Equal(t, 1, h.Sections.Topics.Entries)
	// Topics size should include spec file.
	assert.Greater(t, h.Sections.Topics.SizeBytes, 0)

	require.NotNil(t, h.Sections.Prompts)
	assert.Equal(t, 1, h.Sections.Prompts.Entries)

	require.NotNil(t, h.Sections.Plans)
	assert.Equal(t, 1, h.Sections.Plans.Entries)

	// Packages: local first, then go@org.
	require.Len(t, h.Packages, 2)
	assert.Equal(t, "local", h.Packages[0].Name)
	assert.Equal(t, 5, h.Packages[0].Entries) // philosophy, conventions, arch, lint, migrate
	assert.Equal(t, "go@org", h.Packages[1].Name)
	assert.Equal(t, 1, h.Packages[1].Entries)

	// CompiledAt should be populated.
	assert.NotEmpty(t, h.CompiledAt)
}

func TestGenerateHeuristics_emptySections(t *testing.T) {
	dir := t.TempDir()
	objectsDir := filepath.Join(dir, "objects")
	require.NoError(t, os.MkdirAll(objectsDir, 0o755))

	m := &manifest.Manifest{Name: "empty"}
	h := generateHeuristics(m, map[string]string{}, map[string]string{}, objectsDir)

	assert.Equal(t, 0, h.Totals.Entries)
	assert.Equal(t, 0, h.Totals.Objects)
	assert.Equal(t, 0, h.Totals.SizeBytes)
	assert.Nil(t, h.Sections.Foundation)
	assert.Nil(t, h.Sections.Topics)
	assert.Nil(t, h.Sections.Prompts)
	assert.Nil(t, h.Sections.Plans)
	assert.Empty(t, h.Packages)
}

func TestCountTokens_realTokenizer(t *testing.T) {
	// Real o200k_base tokenization; exact counts depend on the BPE
	// vocabulary but must be non-zero for non-empty content and zero
	// for empty content.
	assert.Equal(t, 0, countTokens(nil))
	assert.Equal(t, 0, countTokens([]byte("")))

	tokens := countTokens([]byte("# Heading\n\nSome markdown content.\n"))
	assert.Greater(t, tokens, 0, "non-empty content should produce tokens")
	// Real tokenization yields fewer tokens than bytes for English text.
	assert.Less(t, tokens, 34, "tokens should be less than byte count for short English")
}

func TestWriteAndLoadHeuristics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")

	h := &Heuristics{
		CompiledAt: "2026-02-23T12:00:00Z",
		Totals: HeuristicsTotals{
			Entries:         10,
			Objects:         8,
			SizeBytes:       50000,
			EstimatedTokens: 12500,
			AlwaysLoad:      2,
		},
		Sections: HeuristicsSections{
			Foundation: &SectionStats{
				Entries:         3,
				SizeBytes:       15000,
				EstimatedTokens: 3750,
				AlwaysLoad:      2,
			},
			Topics: &SectionStats{
				Entries:         5,
				SizeBytes:       25000,
				EstimatedTokens: 6250,
			},
		},
		Packages: []PackageStats{
			{Name: "local", Entries: 6, SizeBytes: 30000, EstimatedTokens: 7500},
			{Name: "go@org", Entries: 4, SizeBytes: 20000, EstimatedTokens: 5000},
		},
	}

	err := WriteHeuristics(path, h)
	require.NoError(t, err)

	loaded, err := loadHeuristics(path)
	require.NoError(t, err)

	assert.Equal(t, h.CompiledAt, loaded.CompiledAt)
	assert.Equal(t, h.Totals.Entries, loaded.Totals.Entries)
	assert.Equal(t, h.Totals.Objects, loaded.Totals.Objects)
	assert.Equal(t, h.Totals.SizeBytes, loaded.Totals.SizeBytes)
	assert.Equal(t, h.Totals.EstimatedTokens, loaded.Totals.EstimatedTokens)
	assert.Equal(t, h.Totals.AlwaysLoad, loaded.Totals.AlwaysLoad)

	require.NotNil(t, loaded.Sections.Foundation)
	assert.Equal(t, 3, loaded.Sections.Foundation.Entries)
	assert.Equal(t, 2, loaded.Sections.Foundation.AlwaysLoad)

	require.NotNil(t, loaded.Sections.Topics)
	assert.Equal(t, 5, loaded.Sections.Topics.Entries)

	assert.Nil(t, loaded.Sections.Prompts)
	assert.Nil(t, loaded.Sections.Plans)

	require.Len(t, loaded.Packages, 2)
	assert.Equal(t, "local", loaded.Packages[0].Name)
	assert.Equal(t, "go@org", loaded.Packages[1].Name)
}

func TestLoadHeuristics_missingFile(t *testing.T) {
	_, err := loadHeuristics("/nonexistent/heuristics.yml")
	assert.Error(t, err)
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   int
		expected string
	}{
		{"zero", 0, "0 tokens"},
		{"small", 250, "250 tokens"},
		{"one thousand", 1000, "1k tokens"},
		{"fractional", 1500, "1.5k tokens"},
		{"large", 25000, "25k tokens"},
		{"large fractional", 12300, "12.3k tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatTokens(tt.tokens))
		})
	}
}

func TestSortStrings(t *testing.T) {
	s := []string{"go@org", "alpha@first", "react@facebook"}
	sortStrings(s)
	assert.Equal(t, []string{"alpha@first", "go@org", "react@facebook"}, s)
}

func TestSortStrings_empty(t *testing.T) {
	s := []string{}
	sortStrings(s)
	assert.Empty(t, s)
}

func TestSortStrings_single(t *testing.T) {
	s := []string{"only"}
	sortStrings(s)
	assert.Equal(t, []string{"only"}, s)
}

func TestGenerateHeuristics_topicWithFiles(t *testing.T) {
	dir, store, _ := setupHeuristicsTest(t)
	objectsDir := filepath.Join(dir, "objects")

	// Create additional file objects.
	pathToHash := make(map[string]string)
	files := map[string]string{
		"topics/react/README.md": "# React\n\nReact conventions.\n",
		"topics/react/hooks.md":  "# Hooks\n\nHook patterns.\n",
		"topics/react/forms.md":  "# Forms\n\nForm handling.\n",
	}
	for path, content := range files {
		hash, err := store.Store([]byte(content))
		require.NoError(t, err)
		pathToHash[path] = hash
	}

	m := &manifest.Manifest{
		Name: "files-test",
		Topics: []manifest.TopicEntry{
			{
				ID:   "react",
				Path: "topics/react/README.md",
				Files: []string{
					"topics/react/hooks.md",
					"topics/react/forms.md",
				},
			},
		},
	}

	provenance := map[string]string{
		"topics:react": "react@fb",
	}

	h := generateHeuristics(m, pathToHash, provenance, objectsDir)

	require.NotNil(t, h.Sections.Topics)
	assert.Equal(t, 1, h.Sections.Topics.Entries)
	// Size should include README + hooks + forms.
	assert.Greater(t, h.Sections.Topics.SizeBytes, 0)

	// Package stats should include all file sizes.
	require.Len(t, h.Packages, 1)
	assert.Equal(t, "react@fb", h.Packages[0].Name)
	assert.Equal(t, h.Sections.Topics.SizeBytes, h.Packages[0].SizeBytes)
}

func TestWriteHeuristics_invalidPath(t *testing.T) {
	h := &Heuristics{}
	err := WriteHeuristics("/nonexistent/dir/heuristics.yml", h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write heuristics")
}

func TestLoadHeuristics_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")
	require.NoError(t, os.WriteFile(path, []byte("{{invalid"), 0o644))

	_, err := loadHeuristics(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse heuristics")
}

func TestGenerateHeuristics_applicationWithFiles(t *testing.T) {
	dir, store, _ := setupHeuristicsTest(t)
	objectsDir := filepath.Join(dir, "objects")

	// Create additional file objects for application entry.
	pathToHash := make(map[string]string)
	files := map[string]string{
		"application/arch/README.md":      "# Architecture\n\nSystem architecture.\n",
		"application/arch/spec/README.md": "# Arch Spec\n\nDetailed spec.\n",
		"application/arch/decisions.md":   "# Decisions\n\nDesign decisions.\n",
	}
	for path, content := range files {
		hash, err := store.Store([]byte(content))
		require.NoError(t, err)
		pathToHash[path] = hash
	}

	m := &manifest.Manifest{
		Name: "app-files-test",
		Application: []manifest.ApplicationEntry{
			{
				ID:   "arch",
				Path: "application/arch/README.md",
				Spec: "application/arch/spec/README.md",
				Files: []string{
					"application/arch/decisions.md",
				},
			},
		},
	}

	provenance := map[string]string{
		"application:arch": "local",
	}

	h := generateHeuristics(m, pathToHash, provenance, objectsDir)

	require.NotNil(t, h.Sections.Application)
	assert.Equal(t, 1, h.Sections.Application.Entries)
	// Size should include README + spec + decisions.
	assert.Greater(t, h.Sections.Application.SizeBytes, 0)

	// Package stats should include all file sizes.
	require.Len(t, h.Packages, 1)
	assert.Equal(t, "local", h.Packages[0].Name)
	assert.Equal(t, h.Sections.Application.SizeBytes, h.Packages[0].SizeBytes)
}

func TestGenerateHeuristics_tokensSumToTotal(t *testing.T) {
	dir, _, pathToHash := setupHeuristicsTest(t)
	objectsDir := filepath.Join(dir, "objects")

	m := &manifest.Manifest{
		Name: "token-sum-test",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/a.md", Load: "always"},
			{ID: "conventions", Path: "foundation/b.md"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md", Spec: "topics/go/spec.md"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "lint", Path: "prompts/lint/README.md"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
		},
	}

	provenance := map[string]string{
		"foundation:philosophy":  "local",
		"foundation:conventions": "local",
		"application:arch":       "local",
		"topics:go":              "go@org",
		"prompts:lint":           "local",
		"plans:migrate":          "local",
	}

	h := generateHeuristics(m, pathToHash, provenance, objectsDir)

	// Section token counts should sum to the total.
	sectionTokenSum := 0
	if h.Sections.Foundation != nil {
		sectionTokenSum += h.Sections.Foundation.EstimatedTokens
	}
	if h.Sections.Application != nil {
		sectionTokenSum += h.Sections.Application.EstimatedTokens
	}
	if h.Sections.Topics != nil {
		sectionTokenSum += h.Sections.Topics.EstimatedTokens
	}
	if h.Sections.Prompts != nil {
		sectionTokenSum += h.Sections.Prompts.EstimatedTokens
	}
	if h.Sections.Plans != nil {
		sectionTokenSum += h.Sections.Plans.EstimatedTokens
	}
	assert.Equal(t, h.Totals.EstimatedTokens, sectionTokenSum,
		"section tokens should sum to total tokens")
	assert.Greater(t, sectionTokenSum, 0, "total tokens should be positive")
}

func TestGenerateHeuristics_topicFilesIncludeTokens(t *testing.T) {
	dir, store, _ := setupHeuristicsTest(t)
	objectsDir := filepath.Join(dir, "objects")

	pathToHash := make(map[string]string)
	files := map[string]string{
		"topics/react/README.md": "# React\n\nReact conventions.\n",
		"topics/react/hooks.md":  "# Hooks\n\nHook patterns and usage.\n",
	}
	for path, content := range files {
		hash, err := store.Store([]byte(content))
		require.NoError(t, err)
		pathToHash[path] = hash
	}

	// Without extra files.
	mNoFiles := &manifest.Manifest{
		Name: "no-files",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}
	provenance := map[string]string{"topics:react": "local"}
	hNoFiles := generateHeuristics(mNoFiles, pathToHash, provenance, objectsDir)

	// With extra files.
	mWithFiles := &manifest.Manifest{
		Name: "with-files",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md",
				Files: []string{"topics/react/hooks.md"}},
		},
	}
	hWithFiles := generateHeuristics(mWithFiles, pathToHash, provenance, objectsDir)

	require.NotNil(t, hNoFiles.Sections.Topics)
	require.NotNil(t, hWithFiles.Sections.Topics)

	// Token count should be higher when extra files are included.
	assert.Greater(t, hWithFiles.Sections.Topics.EstimatedTokens,
		hNoFiles.Sections.Topics.EstimatedTokens,
		"tokens should increase when extra files are included")

	// Package token stats should match section stats.
	require.Len(t, hWithFiles.Packages, 1)
	assert.Equal(t, hWithFiles.Sections.Topics.EstimatedTokens,
		hWithFiles.Packages[0].EstimatedTokens,
		"package tokens should match section tokens")
}

func TestGenerateHeuristics_applicationAlwaysLoad(t *testing.T) {
	dir, store, _ := setupHeuristicsTest(t)
	objectsDir := filepath.Join(dir, "objects")

	pathToHash := make(map[string]string)
	files := map[string]string{
		"application/arch/README.md": "# Architecture\n\nSystem architecture.\n",
	}
	for path, content := range files {
		hash, err := store.Store([]byte(content))
		require.NoError(t, err)
		pathToHash[path] = hash
	}

	m := &manifest.Manifest{
		Name: "always-load-app",
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md", Load: "always"},
		},
	}

	provenance := map[string]string{
		"application:arch": "local",
	}

	h := generateHeuristics(m, pathToHash, provenance, objectsDir)

	assert.Equal(t, 1, h.Totals.AlwaysLoad)
	require.NotNil(t, h.Sections.Application)
	assert.Equal(t, 1, h.Sections.Application.AlwaysLoad)
}

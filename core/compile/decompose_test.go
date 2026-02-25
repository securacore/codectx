package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- shouldDecompose ---

func TestShouldDecompose_nil(t *testing.T) {
	assert.False(t, shouldDecompose(nil))
}

func TestShouldDecompose_belowAll(t *testing.T) {
	h := &Heuristics{
		Totals: HeuristicsTotals{
			Entries:         100,
			SizeBytes:       10000,
			EstimatedTokens: 2500,
		},
	}
	assert.False(t, shouldDecompose(h))
}

func TestShouldDecompose_entriesExceeded(t *testing.T) {
	h := &Heuristics{
		Totals: HeuristicsTotals{
			Entries:         501,
			SizeBytes:       10000,
			EstimatedTokens: 2500,
		},
	}
	assert.True(t, shouldDecompose(h))
}

func TestShouldDecompose_bytesExceeded(t *testing.T) {
	h := &Heuristics{
		Totals: HeuristicsTotals{
			Entries:         10,
			SizeBytes:       51 * 1024,
			EstimatedTokens: 2500,
		},
	}
	assert.True(t, shouldDecompose(h))
}

func TestShouldDecompose_tokensExceeded(t *testing.T) {
	h := &Heuristics{
		Totals: HeuristicsTotals{
			Entries:         10,
			SizeBytes:       10000,
			EstimatedTokens: 100001,
		},
	}
	assert.True(t, shouldDecompose(h))
}

func TestShouldDecompose_exactThresholdNotExceeded(t *testing.T) {
	h := &Heuristics{
		Totals: HeuristicsTotals{
			Entries:         500,
			SizeBytes:       50 * 1024,
			EstimatedTokens: 100000,
		},
	}
	assert.False(t, shouldDecompose(h))
}

// --- decompose ---

func TestDecompose_allSections(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name:        "big-project",
		Description: "A large project",
		Foundation: []CompiledFoundationEntry{
			{ID: "philosophy", Object: "objects/aaa.md", Description: "Philosophy", Load: "always", Source: "local"},
			{ID: "conventions", Object: "objects/bbb.md", Description: "Conventions", Source: "local"},
			{ID: "markdown", Object: "objects/ccc.md", Description: "Markdown", Source: "local"},
		},
		Topics: []CompiledTopicEntry{
			{ID: "go", Object: "objects/ddd.md", Description: "Go", Source: "go@org"},
			{ID: "react", Object: "objects/eee.md", Description: "React", Source: "react@fb"},
		},
		Prompts: []CompiledPromptEntry{
			{ID: "lint", Object: "objects/fff.md", Description: "Lint", Source: "local"},
		},
		Plans: []CompiledPlanEntry{
			{ID: "migrate", Object: "objects/ggg.md", Description: "Migration", Source: "local"},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Foundation: &SectionStats{EstimatedTokens: 5000},
			Topics:     &SectionStats{EstimatedTokens: 30000},
			Prompts:    &SectionStats{EstimatedTokens: 2000},
			Plans:      &SectionStats{EstimatedTokens: 1500},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// Root manifest should have only always-load foundation entries.
	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, "philosophy", cm.Foundation[0].ID)
	assert.Equal(t, "always", cm.Foundation[0].Load)

	// Other sections should be nil (moved to sub-manifests).
	assert.Nil(t, cm.Topics)
	assert.Nil(t, cm.Prompts)
	assert.Nil(t, cm.Plans)

	// ManifestRefs should point to 4 sub-manifests.
	require.Len(t, cm.Manifests, 4)

	assert.Equal(t, "foundation", cm.Manifests[0].Section)
	assert.Equal(t, "manifests/foundation.yml", cm.Manifests[0].Path)
	assert.Equal(t, 2, cm.Manifests[0].Entries) // conventions + markdown (not philosophy)
	assert.Equal(t, 5000, cm.Manifests[0].EstimatedTokens)

	assert.Equal(t, "topics", cm.Manifests[1].Section)
	assert.Equal(t, "manifests/topics.yml", cm.Manifests[1].Path)
	assert.Equal(t, 2, cm.Manifests[1].Entries)
	assert.Equal(t, 30000, cm.Manifests[1].EstimatedTokens)

	assert.Equal(t, "prompts", cm.Manifests[2].Section)
	assert.Equal(t, 1, cm.Manifests[2].Entries)

	assert.Equal(t, "plans", cm.Manifests[3].Section)
	assert.Equal(t, 1, cm.Manifests[3].Entries)

	// Verify sub-manifest files exist and are loadable.
	foundationSub, err := LoadCompiledManifest(filepath.Join(dir, "manifests", "foundation.yml"))
	require.NoError(t, err)
	require.Len(t, foundationSub.Foundation, 2)
	assert.Equal(t, "conventions", foundationSub.Foundation[0].ID)
	assert.Equal(t, "markdown", foundationSub.Foundation[1].ID)

	topicsSub, err := LoadCompiledManifest(filepath.Join(dir, "manifests", "topics.yml"))
	require.NoError(t, err)
	require.Len(t, topicsSub.Topics, 2)
	assert.Equal(t, "go", topicsSub.Topics[0].ID)
	assert.Equal(t, "react", topicsSub.Topics[1].ID)

	promptsSub, err := LoadCompiledManifest(filepath.Join(dir, "manifests", "prompts.yml"))
	require.NoError(t, err)
	require.Len(t, promptsSub.Prompts, 1)

	plansSub, err := LoadCompiledManifest(filepath.Join(dir, "manifests", "plans.yml"))
	require.NoError(t, err)
	require.Len(t, plansSub.Plans, 1)
}

func TestDecompose_allFoundationAlwaysLoad(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "all-always",
		Foundation: []CompiledFoundationEntry{
			{ID: "a", Object: "objects/aaa.md", Load: "always", Source: "local"},
			{ID: "b", Object: "objects/bbb.md", Load: "always", Source: "local"},
		},
		Topics: []CompiledTopicEntry{
			{ID: "c", Object: "objects/ccc.md", Source: "local"},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Topics: &SectionStats{EstimatedTokens: 1000},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// Both foundation entries should stay in root (both always-load).
	require.Len(t, cm.Foundation, 2)

	// No foundation sub-manifest ref (all entries are always-load).
	// Only topics ref.
	require.Len(t, cm.Manifests, 1)
	assert.Equal(t, "topics", cm.Manifests[0].Section)

	// No foundation sub-manifest file.
	_, err = os.Stat(filepath.Join(dir, "manifests", "foundation.yml"))
	assert.True(t, os.IsNotExist(err))
}

func TestDecompose_emptySections(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "foundation-only",
		Foundation: []CompiledFoundationEntry{
			{ID: "a", Object: "objects/aaa.md", Load: "always", Source: "local"},
			{ID: "b", Object: "objects/bbb.md", Source: "local"},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Foundation: &SectionStats{EstimatedTokens: 3000},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// Root has always-load only.
	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, "a", cm.Foundation[0].ID)

	// Only foundation ref (no topics/prompts/plans).
	require.Len(t, cm.Manifests, 1)
	assert.Equal(t, "foundation", cm.Manifests[0].Section)
}

func TestDecompose_noAlwaysLoad(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "no-always",
		Foundation: []CompiledFoundationEntry{
			{ID: "a", Object: "objects/aaa.md", Source: "local"},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Foundation: &SectionStats{EstimatedTokens: 1000},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// No foundation entries in root (none are always-load).
	assert.Empty(t, cm.Foundation)

	// Foundation ref should exist.
	require.Len(t, cm.Manifests, 1)
	assert.Equal(t, "foundation", cm.Manifests[0].Section)
	assert.Equal(t, 1, cm.Manifests[0].Entries)
}

func TestDecompose_nilHeuristics(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "nil-h",
		Topics: []CompiledTopicEntry{
			{ID: "a", Object: "objects/aaa.md", Source: "local"},
		},
	}

	err := decompose(cm, nil, dir)
	require.NoError(t, err)

	// Should still decompose with zero token estimates.
	require.Len(t, cm.Manifests, 1)
	assert.Equal(t, "topics", cm.Manifests[0].Section)
	assert.Equal(t, 0, cm.Manifests[0].EstimatedTokens)
}

func TestDecompose_applicationWithAlwaysLoad(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "app-always-load",
		Application: []CompiledApplicationEntry{
			{ID: "architecture", Object: "objects/aaa.md", Description: "Architecture overview", Load: "always", Source: "local"},
			{ID: "api-design", Object: "objects/bbb.md", Description: "API design", Source: "local"},
			{ID: "data-model", Object: "objects/ccc.md", Description: "Data model", Source: "local"},
		},
		Topics: []CompiledTopicEntry{
			{ID: "go", Object: "objects/ddd.md", Description: "Go conventions", Source: "local"},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Application: &SectionStats{EstimatedTokens: 8000},
			Topics:      &SectionStats{EstimatedTokens: 3000},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// Always-load application entry should stay in root.
	require.Len(t, cm.Application, 1)
	assert.Equal(t, "architecture", cm.Application[0].ID)
	assert.Equal(t, "always", cm.Application[0].Load)

	// Non-always-load entries should be in sub-manifest.
	appRef := false
	for _, ref := range cm.Manifests {
		if ref.Section == "application" {
			appRef = true
			assert.Equal(t, "manifests/application.yml", ref.Path)
			assert.Equal(t, 2, ref.Entries) // api-design + data-model
			assert.Equal(t, 8000, ref.EstimatedTokens)
			assert.Equal(t, sectionDescriptions["application"], ref.Description)
		}
	}
	assert.True(t, appRef, "should have application manifest reference")

	// Verify sub-manifest file is loadable with correct entries.
	sub, err := LoadCompiledManifest(filepath.Join(dir, "manifests", "application.yml"))
	require.NoError(t, err)
	require.Len(t, sub.Application, 2)
	assert.Equal(t, "api-design", sub.Application[0].ID)
	assert.Equal(t, "data-model", sub.Application[1].ID)
	assert.Equal(t, "app-always-load - Application", sub.Name)
	assert.Equal(t, sectionDescriptions["application"], sub.Description)
}

func TestDecompose_applicationAllAlwaysLoad(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "all-app-always",
		Application: []CompiledApplicationEntry{
			{ID: "arch", Object: "objects/aaa.md", Load: "always", Source: "local"},
			{ID: "design", Object: "objects/bbb.md", Load: "always", Source: "local"},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Application: &SectionStats{EstimatedTokens: 2000},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// All application entries are always-load: they all stay in root.
	require.Len(t, cm.Application, 2)

	// No application sub-manifest ref.
	for _, ref := range cm.Manifests {
		assert.NotEqual(t, "application", ref.Section, "should not have application ref when all are always-load")
	}

	// No application sub-manifest file.
	_, err = os.Stat(filepath.Join(dir, "manifests", "application.yml"))
	assert.True(t, os.IsNotExist(err))
}

func TestDecompose_applicationNoAlwaysLoad(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "no-app-always",
		Application: []CompiledApplicationEntry{
			{ID: "arch", Object: "objects/aaa.md", Source: "local"},
			{ID: "design", Object: "objects/bbb.md", Source: "local"},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Application: &SectionStats{EstimatedTokens: 4000},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// No always-load entries: application should be empty in root.
	assert.Empty(t, cm.Application)

	// Application ref should exist with all entries.
	require.Len(t, cm.Manifests, 1)
	assert.Equal(t, "application", cm.Manifests[0].Section)
	assert.Equal(t, 2, cm.Manifests[0].Entries)
}

func TestDecompose_applicationNilHeuristics(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name: "app-nil-h",
		Application: []CompiledApplicationEntry{
			{ID: "arch", Object: "objects/aaa.md", Load: "always", Source: "local"},
			{ID: "design", Object: "objects/bbb.md", Source: "local"},
		},
	}

	err := decompose(cm, nil, dir)
	require.NoError(t, err)

	// Always-load stays in root.
	require.Len(t, cm.Application, 1)
	assert.Equal(t, "arch", cm.Application[0].ID)

	// Ref should have zero tokens (nil heuristics).
	appRef := false
	for _, ref := range cm.Manifests {
		if ref.Section == "application" {
			appRef = true
			assert.Equal(t, 0, ref.EstimatedTokens)
			assert.Equal(t, 1, ref.Entries)
		}
	}
	assert.True(t, appRef)
}

func TestDecompose_subManifestContent(t *testing.T) {
	dir := t.TempDir()

	cm := &CompiledManifest{
		Name:        "content-test",
		Description: "Testing sub-manifest content",
		Topics: []CompiledTopicEntry{
			{
				ID:          "go",
				Object:      "objects/aaa.md",
				Description: "Go conventions",
				Spec:        "objects/bbb.md",
				Source:      "go@org",
				DependsOn:   []string{"foundation:philosophy"},
			},
		},
	}

	h := &Heuristics{
		Sections: HeuristicsSections{
			Topics: &SectionStats{EstimatedTokens: 5000},
		},
	}

	err := decompose(cm, h, dir)
	require.NoError(t, err)

	// Load sub-manifest and verify entry details are preserved.
	sub, err := LoadCompiledManifest(filepath.Join(dir, "manifests", "topics.yml"))
	require.NoError(t, err)

	require.Len(t, sub.Topics, 1)
	assert.Equal(t, "go", sub.Topics[0].ID)
	assert.Equal(t, "objects/aaa.md", sub.Topics[0].Object)
	assert.Equal(t, "Go conventions", sub.Topics[0].Description)
	assert.Equal(t, "objects/bbb.md", sub.Topics[0].Spec)
	assert.Equal(t, "go@org", sub.Topics[0].Source)
	assert.Equal(t, []string{"foundation:philosophy"}, sub.Topics[0].DependsOn)

	// Sub-manifest should have name and description.
	assert.Equal(t, "content-test - Topics", sub.Name)
	assert.Equal(t, "Technology and domain conventions", sub.Description)
}

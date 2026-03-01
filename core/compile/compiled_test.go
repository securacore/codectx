package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- toCompiledManifest ---

func TestToCompiledManifest_foundation(t *testing.T) {
	unified := &manifest.Manifest{
		Name:        "test-project",
		Description: "Test",
		Foundation: []manifest.FoundationEntry{
			{
				ID:          "philosophy",
				Path:        "foundation/philosophy/README.md",
				Description: "Core philosophy",
				Load:        "always",
				DependsOn:   nil,
				RequiredBy:  []string{"conventions"},
			},
		},
	}

	pathToHash := map[string]string{
		"foundation/philosophy/README.md": "a1b2c3d4e5f67890",
	}
	provenance := map[string]string{
		"foundation:philosophy": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	assert.Equal(t, "test-project", cm.Name)
	assert.Equal(t, "Test", cm.Description)
	require.Len(t, cm.Foundation, 1)

	e := cm.Foundation[0]
	assert.Equal(t, "philosophy", e.ID)
	assert.Equal(t, "objects/a1b2c3d4e5f67890.md", e.Object)
	assert.Equal(t, "Core philosophy", e.Description)
	assert.Equal(t, "always", e.Load)
	assert.Equal(t, "local", e.Source)
	assert.Nil(t, e.DependsOn)
	assert.Equal(t, []string{"conventions"}, e.RequiredBy)
}

func TestToCompiledManifest_topicWithSpecAndFiles(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Topics: []manifest.TopicEntry{
			{
				ID:          "react",
				Path:        "topics/react/README.md",
				Description: "React conventions",
				Spec:        "topics/react/spec/README.md",
				Files:       []string{"topics/react/hooks.md", "topics/react/state.md"},
				DependsOn:   []string{"conventions"},
			},
		},
	}

	pathToHash := map[string]string{
		"topics/react/README.md":      "1111111111111111",
		"topics/react/spec/README.md": "2222222222222222",
		"topics/react/hooks.md":       "3333333333333333",
		"topics/react/state.md":       "4444444444444444",
	}
	provenance := map[string]string{
		"topics:react": "react@facebook",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Topics, 1)
	e := cm.Topics[0]
	assert.Equal(t, "objects/1111111111111111.md", e.Object)
	assert.Equal(t, "objects/2222222222222222.md", e.Spec)
	require.Len(t, e.Files, 2)
	assert.Equal(t, "objects/3333333333333333.md", e.Files[0])
	assert.Equal(t, "objects/4444444444444444.md", e.Files[1])
	assert.Equal(t, "react@facebook", e.Source)
	assert.Equal(t, []string{"conventions"}, e.DependsOn)
}

func TestToCompiledManifest_topicNoSpecNoFiles(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md", Description: "Go"},
		},
	}

	pathToHash := map[string]string{
		"topics/go/README.md": "5555555555555555",
	}
	provenance := map[string]string{
		"topics:go": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Topics, 1)
	assert.Empty(t, cm.Topics[0].Spec)
	assert.Nil(t, cm.Topics[0].Files)
}

func TestToCompiledManifest_prompt(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Prompts: []manifest.PromptEntry{
			{
				ID:          "review",
				Path:        "prompts/review/README.md",
				Description: "Code review",
				DependsOn:   []string{"conventions"},
			},
		},
	}

	pathToHash := map[string]string{
		"prompts/review/README.md": "6666666666666666",
	}
	provenance := map[string]string{
		"prompts:review": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Prompts, 1)
	e := cm.Prompts[0]
	assert.Equal(t, "review", e.ID)
	assert.Equal(t, "objects/6666666666666666.md", e.Object)
	assert.Equal(t, "local", e.Source)
	assert.Equal(t, []string{"conventions"}, e.DependsOn)
}

func TestToCompiledManifest_planWithState(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Plans: []manifest.PlanEntry{
			{
				ID:          "migration",
				Path:        "plans/migration/README.md",
				PlanState:   "plans/migration/plan.yml",
				Description: "Migration plan",
			},
		},
	}

	pathToHash := map[string]string{
		"plans/migration/README.md": "7777777777777777",
	}
	provenance := map[string]string{
		"plans:migration": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Plans, 1)
	e := cm.Plans[0]
	assert.Equal(t, "migration", e.ID)
	assert.Equal(t, "objects/7777777777777777.md", e.Object)
	assert.Equal(t, "state/migration.yml", e.PlanState)
	assert.Equal(t, "local", e.Source)
}

func TestToCompiledManifest_planNoState(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Plans: []manifest.PlanEntry{
			{
				ID:          "simple",
				Path:        "plans/simple/README.md",
				Description: "Simple plan",
			},
		},
	}

	pathToHash := map[string]string{
		"plans/simple/README.md": "8888888888888888",
	}
	provenance := map[string]string{
		"plans:simple": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Plans, 1)
	assert.Empty(t, cm.Plans[0].PlanState)
}

func TestToCompiledManifest_applicationWithSpecAndFiles(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Application: []manifest.ApplicationEntry{
			{
				ID:          "architecture",
				Path:        "application/architecture/README.md",
				Description: "System architecture",
				Spec:        "application/architecture/spec/README.md",
				Files:       []string{"application/architecture/diagrams.md", "application/architecture/decisions.md"},
				Load:        "always",
				DependsOn:   []string{"philosophy"},
			},
		},
	}

	pathToHash := map[string]string{
		"application/architecture/README.md":      "1111111111111111",
		"application/architecture/spec/README.md": "2222222222222222",
		"application/architecture/diagrams.md":    "3333333333333333",
		"application/architecture/decisions.md":   "4444444444444444",
	}
	provenance := map[string]string{
		"application:architecture": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Application, 1)
	e := cm.Application[0]
	assert.Equal(t, "objects/1111111111111111.md", e.Object)
	assert.Equal(t, "objects/2222222222222222.md", e.Spec)
	require.Len(t, e.Files, 2)
	assert.Equal(t, "objects/3333333333333333.md", e.Files[0])
	assert.Equal(t, "objects/4444444444444444.md", e.Files[1])
	assert.Equal(t, "always", e.Load)
	assert.Equal(t, "local", e.Source)
	assert.Equal(t, []string{"philosophy"}, e.DependsOn)
}

func TestToCompiledManifest_applicationNoSpecNoFiles(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Application: []manifest.ApplicationEntry{
			{ID: "overview", Path: "application/overview/README.md", Description: "Overview"},
		},
	}

	pathToHash := map[string]string{
		"application/overview/README.md": "5555555555555555",
	}
	provenance := map[string]string{
		"application:overview": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Application, 1)
	assert.Empty(t, cm.Application[0].Spec)
	assert.Nil(t, cm.Application[0].Files)
	assert.Empty(t, cm.Application[0].Load)
}

func TestToCompiledManifest_allSections(t *testing.T) {
	unified := &manifest.Manifest{
		Name:        "full-project",
		Description: "A full project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a/README.md", Description: "A"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md", Description: "Arch"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "b", Path: "topics/b/README.md", Description: "B"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "c", Path: "prompts/c/README.md", Description: "C"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "d", Path: "plans/d/README.md", PlanState: "plans/d/plan.yml", Description: "D"},
		},
	}

	pathToHash := map[string]string{
		"foundation/a/README.md":     "aaaaaaaaaaaaaaaa",
		"application/arch/README.md": "abababababababab",
		"topics/b/README.md":         "bbbbbbbbbbbbbbbb",
		"prompts/c/README.md":        "cccccccccccccccc",
		"plans/d/README.md":          "dddddddddddddddd",
	}
	provenance := map[string]string{
		"foundation:a":     "local",
		"application:arch": "local",
		"topics:b":         "pkg@org",
		"prompts:c":        "local",
		"plans:d":          "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	assert.Len(t, cm.Foundation, 1)
	assert.Len(t, cm.Application, 1)
	assert.Len(t, cm.Topics, 1)
	assert.Len(t, cm.Prompts, 1)
	assert.Len(t, cm.Plans, 1)
	assert.Equal(t, "full-project", cm.Name)
	assert.Equal(t, "A full project", cm.Description)
}

func TestToCompiledManifest_empty(t *testing.T) {
	unified := &manifest.Manifest{
		Name:        "empty",
		Description: "Empty project",
	}

	cm := toCompiledManifest(unified, map[string]string{}, map[string]string{})

	assert.Equal(t, "empty", cm.Name)
	assert.Nil(t, cm.Foundation)
	assert.Nil(t, cm.Topics)
	assert.Nil(t, cm.Prompts)
	assert.Nil(t, cm.Plans)
}

func TestToCompiledManifest_missingHash(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Foundation: []manifest.FoundationEntry{
			{ID: "missing", Path: "foundation/missing/README.md", Description: "Missing"},
		},
	}

	// Path not in pathToHash — object will be "objects/.md" (empty hash).
	cm := toCompiledManifest(unified, map[string]string{}, map[string]string{})

	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, "objects/.md", cm.Foundation[0].Object)
	assert.Empty(t, cm.Foundation[0].Source)
}

// --- WriteCompiledManifest / loadCompiledManifest ---

func TestWriteAndLoad_compiledManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	original := &CompiledManifest{
		Name:        "round-trip",
		Description: "Round trip test",
		Foundation: []CompiledFoundationEntry{
			{
				ID:          "philosophy",
				Object:      "objects/a1b2c3d4e5f67890.md",
				Description: "Core philosophy",
				Load:        "always",
				Source:      "local",
				RequiredBy:  []string{"conventions"},
			},
		},
		Application: []CompiledApplicationEntry{
			{
				ID:          "architecture",
				Object:      "objects/aaaa111122223333.md",
				Description: "System architecture",
				Spec:        "objects/aaaa444455556666.md",
				Load:        "always",
				Files:       []string{"objects/aaaa777788889999.md"},
				Source:      "local",
				DependsOn:   []string{"philosophy"},
				RequiredBy:  []string{"react"},
			},
		},
		Topics: []CompiledTopicEntry{
			{
				ID:          "react",
				Object:      "objects/1111111111111111.md",
				Description: "React",
				Spec:        "objects/2222222222222222.md",
				Files:       []string{"objects/3333333333333333.md"},
				Source:      "react@facebook",
				DependsOn:   []string{"conventions"},
			},
		},
		Prompts: []CompiledPromptEntry{
			{
				ID:          "review",
				Object:      "objects/4444444444444444.md",
				Description: "Code review",
				Source:      "local",
			},
		},
		Plans: []CompiledPlanEntry{
			{
				ID:          "migration",
				Object:      "objects/5555555555555555.md",
				Description: "Migration plan",
				PlanState:   "state/migration.yml",
				Source:      "local",
			},
		},
	}

	err := WriteCompiledManifest(path, original)
	require.NoError(t, err)

	loaded, err := loadCompiledManifest(path)
	require.NoError(t, err)

	// Verify round-trip.
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Description, loaded.Description)

	require.Len(t, loaded.Foundation, 1)
	assert.Equal(t, "philosophy", loaded.Foundation[0].ID)
	assert.Equal(t, "objects/a1b2c3d4e5f67890.md", loaded.Foundation[0].Object)
	assert.Equal(t, "always", loaded.Foundation[0].Load)
	assert.Equal(t, "local", loaded.Foundation[0].Source)
	assert.Equal(t, []string{"conventions"}, loaded.Foundation[0].RequiredBy)

	require.Len(t, loaded.Application, 1)
	assert.Equal(t, "architecture", loaded.Application[0].ID)
	assert.Equal(t, "objects/aaaa111122223333.md", loaded.Application[0].Object)
	assert.Equal(t, "objects/aaaa444455556666.md", loaded.Application[0].Spec)
	assert.Equal(t, "always", loaded.Application[0].Load)
	assert.Equal(t, []string{"objects/aaaa777788889999.md"}, loaded.Application[0].Files)
	assert.Equal(t, "local", loaded.Application[0].Source)
	assert.Equal(t, []string{"philosophy"}, loaded.Application[0].DependsOn)
	assert.Equal(t, []string{"react"}, loaded.Application[0].RequiredBy)

	require.Len(t, loaded.Topics, 1)
	assert.Equal(t, "objects/2222222222222222.md", loaded.Topics[0].Spec)
	assert.Equal(t, []string{"objects/3333333333333333.md"}, loaded.Topics[0].Files)
	assert.Equal(t, "react@facebook", loaded.Topics[0].Source)

	require.Len(t, loaded.Prompts, 1)
	assert.Equal(t, "review", loaded.Prompts[0].ID)

	require.Len(t, loaded.Plans, 1)
	assert.Equal(t, "state/migration.yml", loaded.Plans[0].PlanState)
}

func TestWriteCompiledManifest_invalidPath(t *testing.T) {
	m := &CompiledManifest{Name: "test"}
	err := WriteCompiledManifest("/nonexistent/deep/manifest.yml", m)
	assert.Error(t, err)
}

func TestLoadCompiledManifest_nonexistent(t *testing.T) {
	_, err := loadCompiledManifest("/nonexistent/manifest.yml")
	assert.Error(t, err)
}

func TestLoadCompiledManifest_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")
	require.NoError(t, os.WriteFile(path, []byte("{invalid yaml"), 0o644))

	_, err := loadCompiledManifest(path)
	assert.Error(t, err)
}

func TestWriteCompiledManifest_omitsEmptySections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	m := &CompiledManifest{
		Name:        "minimal",
		Description: "No sections",
	}

	err := WriteCompiledManifest(path, m)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "foundation:")
	assert.NotContains(t, content, "topics:")
	assert.NotContains(t, content, "prompts:")
	assert.NotContains(t, content, "plans:")
	assert.NotContains(t, content, "manifests:")
}

// --- ManifestRef ---

func TestWriteAndLoad_manifestRefs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	original := &CompiledManifest{
		Name:        "multi-manifest",
		Description: "Decomposed project",
		Foundation: []CompiledFoundationEntry{
			{
				ID:          "philosophy",
				Object:      "objects/a1b2c3d4e5f67890.md",
				Description: "Core philosophy",
				Load:        "always",
				Source:      "local",
			},
		},
		Manifests: []ManifestRef{
			{
				Section:         "foundation",
				Path:            "manifests/foundation.yml",
				Entries:         15,
				EstimatedTokens: 45000,
				Description:     "Core operational context",
			},
			{
				Section:         "topics",
				Path:            "manifests/topics.yml",
				Entries:         180,
				EstimatedTokens: 198000,
				Description:     "Technology and domain conventions",
			},
			{
				Section:         "prompts",
				Path:            "manifests/prompts.yml",
				Entries:         92,
				EstimatedTokens: 55200,
				Description:     "Automated task definitions",
			},
			{
				Section:         "plans",
				Path:            "manifests/plans.yml",
				Entries:         60,
				EstimatedTokens: 14300,
				Description:     "Implementation plans",
			},
		},
	}

	err := WriteCompiledManifest(path, original)
	require.NoError(t, err)

	loaded, err := loadCompiledManifest(path)
	require.NoError(t, err)

	// Always-load entries are inlined.
	require.Len(t, loaded.Foundation, 1)
	assert.Equal(t, "philosophy", loaded.Foundation[0].ID)
	assert.Equal(t, "always", loaded.Foundation[0].Load)

	// Sub-manifest references round-trip.
	require.Len(t, loaded.Manifests, 4)
	assert.Equal(t, "foundation", loaded.Manifests[0].Section)
	assert.Equal(t, "manifests/foundation.yml", loaded.Manifests[0].Path)
	assert.Equal(t, 15, loaded.Manifests[0].Entries)
	assert.Equal(t, 45000, loaded.Manifests[0].EstimatedTokens)
	assert.Equal(t, "Core operational context", loaded.Manifests[0].Description)
	assert.Empty(t, loaded.Manifests[0].Source)

	assert.Equal(t, "topics", loaded.Manifests[1].Section)
	assert.Equal(t, 180, loaded.Manifests[1].Entries)
	assert.Equal(t, "plans", loaded.Manifests[3].Section)

	// Section arrays (except always-load foundation) should be empty.
	assert.Nil(t, loaded.Topics)
	assert.Nil(t, loaded.Prompts)
	assert.Nil(t, loaded.Plans)
}

func TestWriteAndLoad_manifestRefWithSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	// Level 2 decomposition: section manifest decomposed by source package.
	original := &CompiledManifest{
		Name:        "topics-index",
		Description: "Topics section",
		Manifests: []ManifestRef{
			{
				Section:         "topics",
				Source:          "react@facebook",
				Path:            "manifests/topics/react@facebook.yml",
				Entries:         120,
				EstimatedTokens: 132000,
				Description:     "React documentation topics",
			},
			{
				Section:         "topics",
				Source:          "go@google",
				Path:            "manifests/topics/go@google.yml",
				Entries:         87,
				EstimatedTokens: 66000,
				Description:     "Go documentation topics",
			},
		},
	}

	err := WriteCompiledManifest(path, original)
	require.NoError(t, err)

	loaded, err := loadCompiledManifest(path)
	require.NoError(t, err)

	require.Len(t, loaded.Manifests, 2)
	assert.Equal(t, "react@facebook", loaded.Manifests[0].Source)
	assert.Equal(t, "go@google", loaded.Manifests[1].Source)
}

func TestWriteCompiledManifest_omitsManifestsWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	m := &CompiledManifest{
		Name:        "single-mode",
		Description: "Single manifest",
		Foundation: []CompiledFoundationEntry{
			{ID: "a", Object: "objects/aaa.md", Description: "A", Source: "local"},
		},
	}

	err := WriteCompiledManifest(path, m)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "manifests:")
}

func TestToCompiledManifest_topicSpecNotInPathToHash(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Topics: []manifest.TopicEntry{
			{
				ID:          "react",
				Path:        "topics/react/README.md",
				Description: "React",
				Spec:        "topics/react/spec/README.md",
			},
		},
	}

	// Only the main path is in pathToHash; Spec path is missing.
	pathToHash := map[string]string{
		"topics/react/README.md": "1111111111111111",
	}
	provenance := map[string]string{
		"topics:react": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Topics, 1)
	assert.Equal(t, "objects/1111111111111111.md", cm.Topics[0].Object)
	// Spec silently skipped because its path is not in pathToHash.
	assert.Empty(t, cm.Topics[0].Spec)
}

func TestToCompiledManifest_topicFilesNotInPathToHash(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Topics: []manifest.TopicEntry{
			{
				ID:          "react",
				Path:        "topics/react/README.md",
				Description: "React",
				Files:       []string{"topics/react/hooks.md", "topics/react/state.md"},
			},
		},
	}

	// Main path is present, but only one of two Files entries is in pathToHash.
	pathToHash := map[string]string{
		"topics/react/README.md": "1111111111111111",
		"topics/react/hooks.md":  "2222222222222222",
		// "topics/react/state.md" deliberately missing
	}
	provenance := map[string]string{
		"topics:react": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Topics, 1)
	// Only the file present in pathToHash should appear.
	require.Len(t, cm.Topics[0].Files, 1)
	assert.Equal(t, "objects/2222222222222222.md", cm.Topics[0].Files[0])
}

func TestToCompiledManifest_topicAllFilesSkipped(t *testing.T) {
	unified := &manifest.Manifest{
		Name: "test",
		Topics: []manifest.TopicEntry{
			{
				ID:          "react",
				Path:        "topics/react/README.md",
				Description: "React",
				Spec:        "topics/react/spec/README.md",
				Files:       []string{"topics/react/hooks.md"},
			},
		},
	}

	// Only the main path is present. Both Spec and Files are missing.
	pathToHash := map[string]string{
		"topics/react/README.md": "1111111111111111",
	}
	provenance := map[string]string{
		"topics:react": "local",
	}

	cm := toCompiledManifest(unified, pathToHash, provenance)

	require.Len(t, cm.Topics, 1)
	assert.Empty(t, cm.Topics[0].Spec)
	assert.Nil(t, cm.Topics[0].Files)
}

func TestToCompiledManifest_noManifestRefs(t *testing.T) {
	// toCompiledManifest never sets Manifests — that's a write-time concern.
	unified := &manifest.Manifest{
		Name:        "test",
		Description: "Test",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a/README.md", Description: "A"},
		},
	}

	cm := toCompiledManifest(unified, map[string]string{
		"foundation/a/README.md": "aaaaaaaaaaaaaaaa",
	}, map[string]string{
		"foundation:a": "local",
	})

	assert.Nil(t, cm.Manifests)
	require.Len(t, cm.Foundation, 1)
}

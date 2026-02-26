package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- rewriteLinks unit tests ---

func TestRewriteLinks_siblingFile(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md":     "aaaa111122223333",
		"topics/react/components.md": "bbbb444455556666",
	}
	content := []byte("# React\nSee [components](components.md) for details.\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"# React\nSee [components](bbbb444455556666.md) for details.\n",
		string(result))
}

func TestRewriteLinks_childPath(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md":      "aaaa111122223333",
		"topics/react/spec/README.md": "cccc777788889999",
	}
	content := []byte("See [spec](spec/README.md) for reasoning.\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"See [spec](cccc777788889999.md) for reasoning.\n",
		string(result))
}

func TestRewriteLinks_parentPath(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md":      "aaaa111122223333",
		"topics/react/spec/README.md": "cccc777788889999",
	}
	content := []byte("See [conventions](../README.md).\n")
	result := rewriteLinks(content, "topics/react/spec/README.md", pathToHash)

	assert.Equal(t,
		"See [conventions](aaaa111122223333.md).\n",
		string(result))
}

func TestRewriteLinks_crossTopic(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md":      "aaaa111122223333",
		"topics/typescript/README.md": "dddd000011112222",
	}
	content := []byte("See [TypeScript](../typescript/README.md).\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"See [TypeScript](dddd000011112222.md).\n",
		string(result))
}

func TestRewriteLinks_crossSection(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/spec/README.md":     "cccc777788889999",
		"foundation/philosophy/README.md": "eeee333344445555",
	}
	content := []byte("See [philosophy](../../../foundation/philosophy/README.md).\n")
	result := rewriteLinks(content, "topics/react/spec/README.md", pathToHash)

	assert.Equal(t,
		"See [philosophy](eeee333344445555.md).\n",
		string(result))
}

func TestRewriteLinks_preservesFragment(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md":     "aaaa111122223333",
		"topics/react/components.md": "bbbb444455556666",
	}
	content := []byte("See [props](components.md#props-pattern).\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"See [props](bbbb444455556666.md#props-pattern).\n",
		string(result))
}

func TestRewriteLinks_unresolvedTarget(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md": "aaaa111122223333",
	}
	content := []byte("See [TypeScript](../typescript/README.md).\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"See [TypeScript](unresolved:../typescript/README.md).\n",
		string(result))
}

func TestRewriteLinks_unresolvedEscapesRoot(t *testing.T) {
	pathToHash := map[string]string{
		"foundation/a/README.md": "aaaa111122223333",
	}
	content := []byte("See [outside](../../../outside.md).\n")
	result := rewriteLinks(content, "foundation/a/README.md", pathToHash)

	assert.Equal(t,
		"See [outside](unresolved:../../../outside.md).\n",
		string(result))
}

func TestRewriteLinks_unresolvedPreservesFragment(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md": "aaaa111122223333",
	}
	content := []byte("See [hooks](../typescript/README.md#hooks).\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	// Fragment is included in the unresolved URI.
	assert.Equal(t,
		"See [hooks](unresolved:../typescript/README.md#hooks).\n",
		string(result))
}

func TestRewriteLinks_httpLinksUntouched(t *testing.T) {
	pathToHash := map[string]string{
		"foundation/a/README.md": "aaaa111122223333",
	}
	content := []byte("See [docs](https://example.com/docs.md).\n")
	result := rewriteLinks(content, "foundation/a/README.md", pathToHash)

	assert.Equal(t, string(content), string(result))
}

func TestRewriteLinks_httpLinkUntouched(t *testing.T) {
	pathToHash := map[string]string{
		"foundation/a/README.md": "aaaa111122223333",
	}
	content := []byte("See [docs](http://example.com/docs.md).\n")
	result := rewriteLinks(content, "foundation/a/README.md", pathToHash)

	assert.Equal(t, string(content), string(result))
}

func TestRewriteLinks_noLinks(t *testing.T) {
	pathToHash := map[string]string{
		"foundation/a/README.md": "aaaa111122223333",
	}
	content := []byte("# Philosophy\n\nNo links here.\n")
	result := rewriteLinks(content, "foundation/a/README.md", pathToHash)

	assert.Equal(t, string(content), string(result))
}

func TestRewriteLinks_emptyContent(t *testing.T) {
	pathToHash := map[string]string{}
	result := rewriteLinks([]byte{}, "a.md", pathToHash)
	assert.Empty(t, result)
}

func TestRewriteLinks_multipleLinksOnSameLine(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md": "aaaa111122223333",
		"topics/react/hooks.md":  "bbbb444455556666",
		"topics/react/state.md":  "cccc777788889999",
	}
	content := []byte("See [hooks](hooks.md) and [state](state.md) for details.\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"See [hooks](bbbb444455556666.md) and [state](cccc777788889999.md) for details.\n",
		string(result))
}

func TestRewriteLinks_mixedResolvableAndUnresolvable(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md": "aaaa111122223333",
		"topics/react/hooks.md":  "bbbb444455556666",
	}
	content := []byte(`# React
See [hooks](hooks.md) for hook patterns.
See [TypeScript](../typescript/README.md) for types.
`)
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t, `# React
See [hooks](bbbb444455556666.md) for hook patterns.
See [TypeScript](unresolved:../typescript/README.md) for types.
`, string(result))
}

func TestRewriteLinks_nonMdLinksIgnored(t *testing.T) {
	pathToHash := map[string]string{
		"foundation/a/README.md": "aaaa111122223333",
	}
	// Non-.md links are not matched by the regex, so they pass through.
	content := []byte("See [schema](schema.json) and [config](codectx.yml).\n")
	result := rewriteLinks(content, "foundation/a/README.md", pathToHash)

	assert.Equal(t, string(content), string(result))
}

func TestRewriteLinks_fragmentOnlyLinksIgnored(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md": "aaaa111122223333",
	}
	// Fragment-only links don't match the regex (no .md file).
	content := []byte("See [section](#some-section).\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t, string(content), string(result))
}

func TestRewriteLinks_tableOfLinks(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md":                "aaaa111122223333",
		"topics/react/components.md":            "bbbb444455556666",
		"topics/react/composable-components.md": "cccc777788889999",
		"topics/react/hooks.md":                 "dddd000011112222",
	}
	content := []byte(`| Document | Purpose |
| --- | --- |
| [components.md](components.md) | Component conventions |
| [composable-components.md](composable-components.md) | Namespace pattern |
| [hooks.md](hooks.md) | Hook patterns |
`)
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t, fmt.Sprintf(`| Document | Purpose |
| --- | --- |
| [components.md](%s.md) | Component conventions |
| [composable-components.md](%s.md) | Namespace pattern |
| [hooks.md](%s.md) | Hook patterns |
`, "bbbb444455556666", "cccc777788889999", "dddd000011112222"), string(result))
}

func TestRewriteLinks_selfLinkRewritten(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md": "aaaa111122223333",
	}
	content := []byte("See [self](README.md).\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"See [self](aaaa111122223333.md).\n",
		string(result))
}

func TestRewriteLinks_dotSlashPrefix(t *testing.T) {
	pathToHash := map[string]string{
		"topics/react/README.md":      "aaaa111122223333",
		"topics/react/spec/README.md": "cccc777788889999",
	}
	content := []byte("See [spec](./spec/README.md).\n")
	result := rewriteLinks(content, "topics/react/README.md", pathToHash)

	assert.Equal(t,
		"See [spec](cccc777788889999.md).\n",
		string(result))
}

// --- StoreAs unit tests ---

func TestStoreAs_writesFile(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	content := []byte("rewritten content")
	hash := "abcdef1234567890"
	err := store.StoreAs(hash, content)
	require.NoError(t, err)

	path := filepath.Join(dir, "objects", hash+".md")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestStoreAs_idempotent(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	hash := "abcdef1234567890"
	err := store.StoreAs(hash, []byte("first"))
	require.NoError(t, err)

	// Second call with different content: should NOT overwrite.
	err = store.StoreAs(hash, []byte("second"))
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "objects", hash+".md"))
	require.NoError(t, err)
	assert.Equal(t, "first", string(data))
}

func TestStoreAs_createsDirectory(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, "nested", "objects")
	store := NewObjectStore(objDir)

	err := store.StoreAs("abcdef1234567890", []byte("content"))
	require.NoError(t, err)

	info, err := os.Stat(objDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- Compile integration tests for link rewriting ---

func TestCompile_rewritesIntraEntryLinks(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a topic with a README that links to a sub-file and spec.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react", "spec"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "README.md"),
		[]byte("# React\nSee [hooks](hooks.md) and [spec](spec/README.md).\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "hooks.md"),
		[]byte("# Hooks\nBack to [README](README.md).\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "spec", "README.md"),
		[]byte("# Spec\nSee [conventions](../README.md).\n"), 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{
				ID:    "react",
				Path:  "topics/react/README.md",
				Spec:  "topics/react/spec/README.md",
				Files: []string{"topics/react/hooks.md"},
			},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 3, result.ObjectsStored)

	// Compute hashes from raw content.
	readmeHash := ContentHash([]byte("# React\nSee [hooks](hooks.md) and [spec](spec/README.md).\n"))
	hooksHash := ContentHash([]byte("# Hooks\nBack to [README](README.md).\n"))
	specHash := ContentHash([]byte("# Spec\nSee [conventions](../README.md).\n"))

	// Verify the stored README has rewritten links.
	readmeData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", readmeHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(readmeData), hooksHash+".md")
	assert.Contains(t, string(readmeData), specHash+".md")
	assert.NotContains(t, string(readmeData), "hooks.md)")
	assert.NotContains(t, string(readmeData), "spec/README.md)")

	// Verify hooks.md links back to the README.
	hooksData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", hooksHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(hooksData), readmeHash+".md")

	// Verify spec links back to the README.
	specData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", specHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(specData), readmeHash+".md")
}

func TestCompile_rewritesCrossEntryLinks(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create two topics that link to each other.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "typescript"), 0o755))

	reactContent := []byte("# React\nSee [TypeScript](../typescript/README.md).\n")
	tsContent := []byte("# TypeScript\nUsed by [React](../react/README.md).\n")
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), reactContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "typescript", "README.md"), tsContent, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
			{ID: "typescript", Path: "topics/typescript/README.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	reactHash := ContentHash(reactContent)
	tsHash := ContentHash(tsContent)

	// React file should link to typescript's hash.
	reactData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", reactHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(reactData), tsHash+".md")

	// TypeScript file should link to react's hash.
	tsData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", tsHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(tsData), reactHash+".md")
}

func TestCompile_unresolvedLinksGetMarker(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a topic that links to a non-existent topic.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))
	content := []byte("# React\nSee [TypeScript](../typescript/README.md) and [Go](../go/README.md).\n")
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), content, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	reactHash := ContentHash(content)
	data, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", reactHash+".md"))
	require.NoError(t, err)

	// Both links should use the unresolved: scheme.
	assert.Contains(t, string(data), "unresolved:../typescript/README.md")
	assert.Contains(t, string(data), "unresolved:../go/README.md")
	// Original relative paths should not appear as link targets.
	assert.NotContains(t, string(data), "(../typescript/README.md)")
	assert.NotContains(t, string(data), "(../go/README.md)")
}

func TestCompile_crossSectionLinkRewriting(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Foundation doc and a topic spec that links to it.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react", "spec"), 0o755))
	philContent := []byte("# Philosophy\n")
	specContent := []byte("# Spec\nSee [philosophy](../../../foundation/philosophy/README.md).\n")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy", "README.md"), philContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "spec", "README.md"), specContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), []byte("# React\n"), 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Spec: "topics/react/spec/README.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	philHash := ContentHash(philContent)
	specHash := ContentHash(specContent)

	// Spec should link to philosophy's object hash.
	specData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", specHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(specData), philHash+".md")
	assert.NotContains(t, string(specData), "foundation/philosophy/README.md)")
}

func TestCompile_noLinksContentUnchanged(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// A file with no markdown links.
	content := []byte("# Philosophy\n\nGuiding principles for decision-making.\n")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy", "README.md"), content, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	hash := ContentHash(content)
	data, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", hash+".md"))
	require.NoError(t, err)
	assert.Equal(t, string(content), string(data))
}

func TestCompile_httpLinksPreserved(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	content := []byte("# Docs\nSee [spec](https://example.com/spec.md) for details.\n")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "docs", "README.md"), content, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "docs", Path: "foundation/docs/README.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	hash := ContentHash(content)
	data, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", hash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "https://example.com/spec.md")
}

func TestCompile_fragmentPreservedInRewrittenLinks(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))
	hooksContent := []byte("# Hooks\n\n## Custom Hooks\nContent here.\n")
	readmeContent := []byte("# React\nSee [custom hooks](hooks.md#custom-hooks).\n")
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "hooks.md"), hooksContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), readmeContent, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Files: []string{"topics/react/hooks.md"}},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	hooksHash := ContentHash(hooksContent)
	readmeHash := ContentHash(readmeContent)

	data, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", readmeHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), hooksHash+".md#custom-hooks")
}

func TestCompile_packageLinksRewritten(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package with internal links.
	pkgDir := filepath.Join(docsDir, "packages", "react@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "topics", "react"), 0o755))

	readmeContent := []byte("# React\nSee [hooks](hooks.md).\n")
	hooksContent := []byte("# Hooks\nBack to [main](README.md).\n")
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "react", "README.md"), readmeContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "react", "hooks.md"), hooksContent, 0o644))

	pkgManifest := &manifest.Manifest{
		Name: "react", Author: "org", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Files: []string{"topics/react/hooks.md"}},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	cfg.Packages = []config.PackageDep{
		{Name: "react", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	readmeHash := ContentHash(readmeContent)
	hooksHash := ContentHash(hooksContent)

	// README should link to hooks hash.
	readmeData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", readmeHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(readmeData), hooksHash+".md")

	// Hooks should link back to README hash.
	hooksData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", hooksHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(hooksData), readmeHash+".md")
}

func TestCompile_rewritesApplicationEntryLinks(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an application entry with internal links between path, spec, and files.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "application", "arch", "spec"), 0o755))

	archContent := []byte("# Architecture\nSee [spec](spec/README.md) and [decisions](decisions.md).\n")
	specContent := []byte("# Spec\nBack to [architecture](../README.md).\n")
	decisionsContent := []byte("# Decisions\nBased on [architecture](README.md).\n")

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "application", "arch", "README.md"), archContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "application", "arch", "spec", "README.md"), specContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "application", "arch", "decisions.md"), decisionsContent, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Application: []manifest.ApplicationEntry{
			{
				ID:    "arch",
				Path:  "application/arch/README.md",
				Spec:  "application/arch/spec/README.md",
				Files: []string{"application/arch/decisions.md"},
			},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 3, result.ObjectsStored)

	archHash := ContentHash(archContent)
	specHash := ContentHash(specContent)
	decisionsHash := ContentHash(decisionsContent)

	// Architecture README should have rewritten links to spec and decisions.
	archData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", archHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(archData), specHash+".md")
	assert.Contains(t, string(archData), decisionsHash+".md")
	assert.NotContains(t, string(archData), "spec/README.md)")
	assert.NotContains(t, string(archData), "decisions.md)")

	// Spec should link back to architecture README.
	specData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", specHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(specData), archHash+".md")

	// Decisions should link to architecture README.
	decisionsData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", decisionsHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(decisionsData), archHash+".md")

	// Verify compiled manifest has application entry with object references.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Application, 1)
	assert.Equal(t, "arch", cm.Application[0].ID)
	assert.Equal(t, ObjectPath(archHash), cm.Application[0].Object)
	assert.Contains(t, cm.Application[0].Spec, specHash)
	require.Len(t, cm.Application[0].Files, 1)
	assert.Contains(t, cm.Application[0].Files[0], decisionsHash)
}

func TestCompile_rewritesPromptEntryLinks(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a prompt that links to a foundation doc.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "prompts"), 0o755))

	philContent := []byte("# Philosophy\nCore principles.\n")
	promptContent := []byte("# Code Review\nFollow [philosophy](../foundation/philosophy/README.md) when reviewing.\n")

	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy", "README.md"), philContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "prompts", "code-review.md"), promptContent, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Description: "Core philosophy"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "code-review", Path: "prompts/code-review.md", Description: "Code review prompt"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ObjectsStored)

	philHash := ContentHash(philContent)
	promptHash := ContentHash(promptContent)

	// Prompt should have rewritten link to philosophy's object hash.
	promptData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", promptHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(promptData), philHash+".md")
	assert.NotContains(t, string(promptData), "../foundation/philosophy/README.md)")

	// Verify compiled manifest has prompt entry.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Prompts, 1)
	assert.Equal(t, "code-review", cm.Prompts[0].ID)
	assert.Equal(t, ObjectPath(promptHash), cm.Prompts[0].Object)
}

func TestCompile_rewritesPlanEntryLinks(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a plan that links to a topic.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "plans", "migrate"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "db"), 0o755))

	dbContent := []byte("# Database\nDatabase conventions.\n")
	planContent := []byte("# Migration Plan\nRefer to [database conventions](../../topics/db/README.md).\n")

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "db", "README.md"), dbContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "plans", "migrate", "README.md"), planContent, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "db", Path: "topics/db/README.md", Description: "Database conventions"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md", Description: "Migration plan"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ObjectsStored)

	dbHash := ContentHash(dbContent)
	planHash := ContentHash(planContent)

	// Plan should have rewritten link to database topic's object hash.
	planData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", planHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(planData), dbHash+".md")
	assert.NotContains(t, string(planData), "../../topics/db/README.md)")

	// Verify compiled manifest has plan entry.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Plans, 1)
	assert.Equal(t, "migrate", cm.Plans[0].ID)
	assert.Equal(t, ObjectPath(planHash), cm.Plans[0].Object)
}

func TestCompile_crossSectionLinksAllEntryTypes(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create entries across all five sections, with cross-section links.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "application", "arch"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "plans", "v2"), 0o755))

	foundationContent := []byte("# Foundation\nSee [architecture](../../application/arch/README.md).\n")
	appContent := []byte("# Architecture\nSee [react](../../topics/react/README.md).\n")
	topicContent := []byte("# React\nSee [foundation](../../foundation/principles/README.md).\n")
	promptContent := []byte("# Review\nFollow [react](../topics/react/README.md) conventions.\n")
	planContent := []byte("# Plan v2\nBased on [architecture](../../application/arch/README.md).\n")

	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "principles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "principles", "README.md"), foundationContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "application", "arch", "README.md"), appContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), topicContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "prompts", "review.md"), promptContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "plans", "v2", "README.md"), planContent, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "principles", Path: "foundation/principles/README.md", Description: "Principles"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md", Description: "Architecture"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "review", Path: "prompts/review.md", Description: "Review prompt"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "v2", Path: "plans/v2/README.md", Description: "v2 plan"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 5, result.ObjectsStored)

	foundHash := ContentHash(foundationContent)
	appHash := ContentHash(appContent)
	topicHash := ContentHash(topicContent)
	promptHash := ContentHash(promptContent)
	planHash := ContentHash(planContent)

	// Foundation links to architecture.
	foundData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", foundHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(foundData), appHash+".md")

	// Application links to react topic.
	appData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", appHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(appData), topicHash+".md")

	// Topic links to foundation.
	topicData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", topicHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(topicData), foundHash+".md")

	// Prompt links to react topic.
	promptData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", promptHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(promptData), topicHash+".md")

	// Plan links to architecture.
	planData, err := os.ReadFile(filepath.Join(result.OutputDir, "objects", planHash+".md"))
	require.NoError(t, err)
	assert.Contains(t, string(planData), appHash+".md")
}

func TestCompile_hashFromRawContentStoredContentRewritten(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a file with a link that WILL be rewritten.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))
	rawContent := []byte("# React\nSee [hooks](hooks.md).\n")
	hooksContent := []byte("# Hooks\n")
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), rawContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "hooks.md"), hooksContent, 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Files: []string{"topics/react/hooks.md"}},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	// The FILENAME should be based on the raw source content hash.
	rawHash := ContentHash(rawContent)
	hooksHash := ContentHash(hooksContent)
	objectPath := filepath.Join(result.OutputDir, "objects", rawHash+".md")

	// The file should exist at the raw-content-hash filename.
	data, err := os.ReadFile(objectPath)
	require.NoError(t, err)

	// But the STORED CONTENT should have rewritten links (different from raw).
	assert.NotEqual(t, string(rawContent), string(data), "stored content should differ from raw due to link rewriting")
	assert.Contains(t, string(data), hooksHash+".md", "stored content should contain the rewritten link")
	assert.NotContains(t, string(data), "(hooks.md)", "stored content should not contain the original relative link")

	// The compiled manifest should reference the raw-content hash.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Topics, 1)
	assert.Equal(t, ObjectPath(rawHash), cm.Topics[0].Object)
}

func TestCompile_sourceDocsUnmodified(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create files with links that WILL be rewritten in compiled output.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))
	originalContent := "# React\nSee [hooks](hooks.md).\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "README.md"),
		[]byte(originalContent), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "hooks.md"),
		[]byte("# Hooks\n"), 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Files: []string{"topics/react/hooks.md"}},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	_, err := Compile(cfg)
	require.NoError(t, err)

	// Source file should be COMPLETELY UNCHANGED.
	data, err := os.ReadFile(filepath.Join(docsDir, "topics", "react", "README.md"))
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(data))
}

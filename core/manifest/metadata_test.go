package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/markdown"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// BuildMetadata
// ---------------------------------------------------------------------------

func TestBuildMetadata_GroupsBySource(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)

	// 3 unique source files in testChunks.
	if len(m.Documents) != 3 {
		t.Errorf("expected 3 documents, got %d", len(m.Documents))
	}

	if _, ok := m.Documents["docs/topics/auth/jwt.md"]; !ok {
		t.Error("expected document for docs/topics/auth/jwt.md")
	}
	if _, ok := m.Documents["docs/topics/auth/jwt.spec.md"]; !ok {
		t.Error("expected document for docs/topics/auth/jwt.spec.md")
	}
	if _, ok := m.Documents["system/topics/taxonomy-generation/README.md"]; !ok {
		t.Error("expected document for system/topics/taxonomy-generation/README.md")
	}
}

func TestBuildMetadata_ChunkIDs(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)

	doc := m.Documents["docs/topics/auth/jwt.md"]
	if len(doc.Chunks) != 3 {
		t.Fatalf("expected 3 chunks for jwt.md, got %d", len(doc.Chunks))
	}

	// Chunks should be in sequence order.
	expected := []string{"obj:aaa11111.1", "obj:aaa11111.2", "obj:aaa11111.3"}
	for i, want := range expected {
		if doc.Chunks[i] != want {
			t.Errorf("chunk[%d]: expected %q, got %q", i, want, doc.Chunks[i])
		}
	}
}

func TestBuildMetadata_TotalTokens(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)

	doc := m.Documents["docs/topics/auth/jwt.md"]
	expected := 400 + 450 + 380
	if doc.TotalTokens != expected {
		t.Errorf("expected %d total tokens, got %d", expected, doc.TotalTokens)
	}
}

func TestBuildMetadata_DocumentTypes(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)

	tests := map[string]manifest.DocumentType{
		"docs/topics/auth/jwt.md":                     manifest.DocTopic,
		"docs/topics/auth/jwt.spec.md":                manifest.DocTopic,
		"system/topics/taxonomy-generation/README.md": manifest.DocSystem,
	}

	for source, expected := range tests {
		doc := m.Documents[source]
		if doc == nil {
			t.Errorf("missing document %q", source)
			continue
		}
		if doc.Type != expected {
			t.Errorf("%s: expected type %q, got %q", source, expected, doc.Type)
		}
	}
}

func TestBuildMetadata_TitleFromH1(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID: "obj:test.1", Type: chunk.ChunkObject,
			Source: "docs/topics/auth/jwt.md", Sequence: 1, TotalInFile: 1, Tokens: 100,
		},
	}

	blocks := map[string]*markdown.Document{
		"docs/topics/auth/jwt.md": {
			Blocks: []markdown.Block{
				{Type: markdown.BlockHeading, Level: 1, Content: "JWT Token Management"},
				{Type: markdown.BlockParagraph, Content: "Some content..."},
			},
		},
	}

	m := manifest.BuildMetadata(chunks, blocks)
	doc := m.Documents["docs/topics/auth/jwt.md"]
	if doc.Title != "JWT Token Management" {
		t.Errorf("expected title %q, got %q", "JWT Token Management", doc.Title)
	}
}

func TestBuildMetadata_TitleFallsBackToFilename(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID: "obj:test.1", Type: chunk.ChunkObject,
			Source: "docs/topics/auth/jwt-tokens.md", Sequence: 1, TotalInFile: 1, Tokens: 100,
		},
	}

	// No blocks provided — should fall back to filename stem.
	m := manifest.BuildMetadata(chunks, nil)
	doc := m.Documents["docs/topics/auth/jwt-tokens.md"]
	if doc.Title != "jwt-tokens" {
		t.Errorf("expected title %q, got %q", "jwt-tokens", doc.Title)
	}
}

func TestBuildMetadata_TitleFallsBackWhenNoH1(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID: "obj:test.1", Type: chunk.ChunkObject,
			Source: "docs/topics/auth/jwt.md", Sequence: 1, TotalInFile: 1, Tokens: 100,
		},
	}

	// Document with only H2, no H1.
	blocks := map[string]*markdown.Document{
		"docs/topics/auth/jwt.md": {
			Blocks: []markdown.Block{
				{Type: markdown.BlockHeading, Level: 2, Content: "Subsection"},
			},
		},
	}

	m := manifest.BuildMetadata(chunks, blocks)
	doc := m.Documents["docs/topics/auth/jwt.md"]
	if doc.Title != "jwt" {
		t.Errorf("expected fallback title %q, got %q", "jwt", doc.Title)
	}
}

func TestBuildMetadata_TitleReadme(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID: "obj:test.1", Type: chunk.ChunkObject,
			Source: "docs/foundation/coding-standards/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100,
		},
	}

	m := manifest.BuildMetadata(chunks, nil)
	doc := m.Documents["docs/foundation/coding-standards/README.md"]
	if doc.Title != "README" {
		t.Errorf("expected title %q, got %q", "README", doc.Title)
	}
}

func TestBuildMetadata_NilBlocksNoRefs(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)

	for source, doc := range m.Documents {
		if doc.ReferencesTo != nil {
			t.Errorf("%s: references_to should be nil without blocks, got %v", source, doc.ReferencesTo)
		}
		if doc.ReferencedBy != nil {
			t.Errorf("%s: referenced_by should be nil without blocks, got %v", source, doc.ReferencedBy)
		}
	}
}

func TestBuildMetadata_CompiledAtIsSet(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)
	if m.CompiledAt == "" {
		t.Error("expected compiled_at to be set")
	}
}

func TestBuildMetadata_EmptyChunks(t *testing.T) {
	m := manifest.BuildMetadata(nil, nil)
	if len(m.Documents) != 0 {
		t.Errorf("expected 0 documents for nil chunks, got %d", len(m.Documents))
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func TestMetadata_WriteTo_RoundTrip(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.yml")

	if err := m.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading metadata: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "# codectx compiled metadata") {
		t.Error("expected header comment")
	}

	var loaded manifest.Metadata
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling metadata: %v", err)
	}

	if len(loaded.Documents) != len(m.Documents) {
		t.Errorf("round-trip documents: expected %d, got %d", len(m.Documents), len(loaded.Documents))
	}
}

func TestMetadata_WriteTo_2SpaceIndent(t *testing.T) {
	m := manifest.BuildMetadata(testChunks(), nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.yml")

	if err := m.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "\t") {
		t.Error("metadata should not contain tabs")
	}
}

// ---------------------------------------------------------------------------
// MetadataPath
// ---------------------------------------------------------------------------

func TestMetadataPath(t *testing.T) {
	got := manifest.MetadataPath("/project/.codectx/compiled")
	expected := filepath.Join("/project/.codectx/compiled", "metadata.yml")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// ---------------------------------------------------------------------------
// Cross-reference extraction
// ---------------------------------------------------------------------------

func TestBuildMetadata_CrossReferences(t *testing.T) {
	// Two documents: auth links to jwt.
	chunks := []chunk.Chunk{
		{ID: "obj:auth.1", Type: chunk.ChunkObject, Source: "topics/auth/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
		{ID: "obj:jwt.1", Type: chunk.ChunkObject, Source: "topics/jwt/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
	}

	blocks := map[string]*markdown.Document{
		"topics/auth/README.md": {
			Blocks: []markdown.Block{
				{Type: markdown.BlockParagraph, Content: "See JWT docs."},
			},
			Links: []markdown.LinkRef{
				{Destination: "../jwt/README.md", Text: "JWT docs"},
			},
		},
		"topics/jwt/README.md": {
			Blocks: []markdown.Block{
				{Type: markdown.BlockParagraph, Content: "JWT details."},
			},
		},
	}

	m := manifest.BuildMetadata(chunks, blocks)

	// Auth doc should reference JWT doc.
	authDoc := m.Documents["topics/auth/README.md"]
	if len(authDoc.ReferencesTo) != 1 {
		t.Fatalf("auth: expected 1 ReferencesTo, got %d", len(authDoc.ReferencesTo))
	}
	if authDoc.ReferencesTo[0].Path != "topics/jwt/README.md" {
		t.Errorf("auth: ReferencesTo[0].Path = %q, want topics/jwt/README.md", authDoc.ReferencesTo[0].Path)
	}

	// JWT doc should be referenced by auth doc.
	jwtDoc := m.Documents["topics/jwt/README.md"]
	if len(jwtDoc.ReferencedBy) != 1 {
		t.Fatalf("jwt: expected 1 ReferencedBy, got %d", len(jwtDoc.ReferencedBy))
	}
	if jwtDoc.ReferencedBy[0].Path != "topics/auth/README.md" {
		t.Errorf("jwt: ReferencedBy[0].Path = %q, want topics/auth/README.md", jwtDoc.ReferencedBy[0].Path)
	}
}

func TestBuildMetadata_CrossReferences_SelfReference(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:auth.1", Type: chunk.ChunkObject, Source: "topics/auth/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
	}

	blocks := map[string]*markdown.Document{
		"topics/auth/README.md": {
			Links: []markdown.LinkRef{
				{Destination: "./README.md", Text: "this page"}, // self-reference
			},
		},
	}

	m := manifest.BuildMetadata(chunks, blocks)

	authDoc := m.Documents["topics/auth/README.md"]
	if len(authDoc.ReferencesTo) != 0 {
		t.Errorf("self-references should be excluded, got %d refs", len(authDoc.ReferencesTo))
	}
}

func TestBuildMetadata_CrossReferences_NonExistentTarget(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:auth.1", Type: chunk.ChunkObject, Source: "topics/auth/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
	}

	blocks := map[string]*markdown.Document{
		"topics/auth/README.md": {
			Links: []markdown.LinkRef{
				{Destination: "../nonexistent/README.md", Text: "missing doc"},
			},
		},
	}

	m := manifest.BuildMetadata(chunks, blocks)

	authDoc := m.Documents["topics/auth/README.md"]
	if len(authDoc.ReferencesTo) != 0 {
		t.Errorf("links to non-existent docs should be excluded, got %d refs", len(authDoc.ReferencesTo))
	}
}

func TestBuildMetadata_CrossReferences_Bidirectional(t *testing.T) {
	// Auth links to JWT, JWT links to Auth.
	chunks := []chunk.Chunk{
		{ID: "obj:auth.1", Type: chunk.ChunkObject, Source: "topics/auth/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
		{ID: "obj:jwt.1", Type: chunk.ChunkObject, Source: "topics/jwt/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
	}

	blocks := map[string]*markdown.Document{
		"topics/auth/README.md": {
			Links: []markdown.LinkRef{
				{Destination: "../jwt/README.md", Text: "JWT"},
			},
		},
		"topics/jwt/README.md": {
			Links: []markdown.LinkRef{
				{Destination: "../auth/README.md", Text: "Auth"},
			},
		},
	}

	m := manifest.BuildMetadata(chunks, blocks)

	authDoc := m.Documents["topics/auth/README.md"]
	jwtDoc := m.Documents["topics/jwt/README.md"]

	// Auth references JWT.
	if len(authDoc.ReferencesTo) != 1 || authDoc.ReferencesTo[0].Path != "topics/jwt/README.md" {
		t.Errorf("auth: expected ReferencesTo jwt, got %+v", authDoc.ReferencesTo)
	}

	// JWT references Auth.
	if len(jwtDoc.ReferencesTo) != 1 || jwtDoc.ReferencesTo[0].Path != "topics/auth/README.md" {
		t.Errorf("jwt: expected ReferencesTo auth, got %+v", jwtDoc.ReferencesTo)
	}

	// Auth is referenced by JWT.
	if len(authDoc.ReferencedBy) != 1 || authDoc.ReferencedBy[0].Path != "topics/jwt/README.md" {
		t.Errorf("auth: expected ReferencedBy jwt, got %+v", authDoc.ReferencedBy)
	}

	// JWT is referenced by Auth.
	if len(jwtDoc.ReferencedBy) != 1 || jwtDoc.ReferencedBy[0].Path != "topics/auth/README.md" {
		t.Errorf("jwt: expected ReferencedBy auth, got %+v", jwtDoc.ReferencedBy)
	}
}

func TestBuildMetadata_CrossReferences_MultipleTargets(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:auth.1", Type: chunk.ChunkObject, Source: "topics/auth/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
		{ID: "obj:jwt.1", Type: chunk.ChunkObject, Source: "topics/jwt/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
		{ID: "obj:oauth.1", Type: chunk.ChunkObject, Source: "topics/oauth/README.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
	}

	blocks := map[string]*markdown.Document{
		"topics/auth/README.md": {
			Links: []markdown.LinkRef{
				{Destination: "../jwt/README.md", Text: "JWT"},
				{Destination: "../oauth/README.md", Text: "OAuth"},
			},
		},
		"topics/jwt/README.md":   {},
		"topics/oauth/README.md": {},
	}

	m := manifest.BuildMetadata(chunks, blocks)

	authDoc := m.Documents["topics/auth/README.md"]
	if len(authDoc.ReferencesTo) != 2 {
		t.Fatalf("auth: expected 2 ReferencesTo, got %d", len(authDoc.ReferencesTo))
	}

	// Should be sorted.
	if authDoc.ReferencesTo[0].Path != "topics/jwt/README.md" {
		t.Errorf("auth: ReferencesTo[0].Path = %q, want topics/jwt/README.md", authDoc.ReferencesTo[0].Path)
	}
	if authDoc.ReferencesTo[1].Path != "topics/oauth/README.md" {
		t.Errorf("auth: ReferencesTo[1].Path = %q, want topics/oauth/README.md", authDoc.ReferencesTo[1].Path)
	}
}

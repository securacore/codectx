package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testChunks() []chunk.Chunk {
	return []chunk.Chunk{
		{
			ID:          "obj:aaa11111.1",
			Type:        chunk.ChunkObject,
			Source:      "docs/topics/auth/jwt.md",
			Heading:     "Authentication > JWT Tokens",
			Sequence:    1,
			TotalInFile: 3,
			Tokens:      400,
			Content:     "JWT token validation...",
		},
		{
			ID:          "obj:aaa11111.2",
			Type:        chunk.ChunkObject,
			Source:      "docs/topics/auth/jwt.md",
			Heading:     "Authentication > JWT Tokens > Validation",
			Sequence:    2,
			TotalInFile: 3,
			Tokens:      450,
			Content:     "Signature verification...",
		},
		{
			ID:          "obj:aaa11111.3",
			Type:        chunk.ChunkObject,
			Source:      "docs/topics/auth/jwt.md",
			Heading:     "Authentication > JWT Tokens > Refresh",
			Sequence:    3,
			TotalInFile: 3,
			Tokens:      380,
			Content:     "Token refresh lifecycle...",
		},
		{
			ID:          "spec:bbb22222.1",
			Type:        chunk.ChunkSpec,
			Source:      "docs/topics/auth/jwt.spec.md",
			Heading:     "Authentication > JWT Tokens",
			Sequence:    1,
			TotalInFile: 2,
			Tokens:      350,
			Content:     "The JWT strategy was chosen because...",
		},
		{
			ID:          "spec:bbb22222.2",
			Type:        chunk.ChunkSpec,
			Source:      "docs/topics/auth/jwt.spec.md",
			Heading:     "Authentication > JWT Tokens > Refresh",
			Sequence:    2,
			TotalInFile: 2,
			Tokens:      300,
			Content:     "The refresh flow uses rotating tokens...",
		},
		{
			ID:          "sys:ccc33333.1",
			Type:        chunk.ChunkSystem,
			Source:      "system/topics/taxonomy-generation/README.md",
			Heading:     "Taxonomy Generation > Rules",
			Sequence:    1,
			TotalInFile: 2,
			Tokens:      340,
			Content:     "Alias generation rules...",
		},
		{
			ID:          "sys:ccc33333.2",
			Type:        chunk.ChunkSystem,
			Source:      "system/topics/taxonomy-generation/README.md",
			Heading:     "Taxonomy Generation > Examples",
			Sequence:    2,
			TotalInFile: 2,
			Tokens:      280,
			Content:     "Example alias expansions...",
		},
	}
}

// ---------------------------------------------------------------------------
// BuildManifest
// ---------------------------------------------------------------------------

func TestBuildManifest_Totals(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	if m.TotalChunks != 7 {
		t.Errorf("TotalChunks: expected 7, got %d", m.TotalChunks)
	}
	if m.TotalObjectChunks != 3 {
		t.Errorf("TotalObjectChunks: expected 3, got %d", m.TotalObjectChunks)
	}
	if m.TotalSpecChunks != 2 {
		t.Errorf("TotalSpecChunks: expected 2, got %d", m.TotalSpecChunks)
	}
	if m.TotalSystemChunks != 2 {
		t.Errorf("TotalSystemChunks: expected 2, got %d", m.TotalSystemChunks)
	}
	expectedTokens := 400 + 450 + 380 + 350 + 300 + 340 + 280
	if m.TotalTokens != expectedTokens {
		t.Errorf("TotalTokens: expected %d, got %d", expectedTokens, m.TotalTokens)
	}
	if m.Encoding != "cl100k_base" {
		t.Errorf("Encoding: expected %q, got %q", "cl100k_base", m.Encoding)
	}
}

func TestBuildManifest_BridgeHash(t *testing.T) {
	hash := "sha256:abc123"
	m := manifest.BuildManifest(nil, "cl100k_base", &hash, nil)
	if m.BridgeInstructionsHash == nil || *m.BridgeInstructionsHash != hash {
		t.Errorf("BridgeInstructionsHash: expected %q, got %v", hash, m.BridgeInstructionsHash)
	}
}

func TestBuildManifest_NilBridgeHash(t *testing.T) {
	m := manifest.BuildManifest(nil, "cl100k_base", nil, nil)
	if m.BridgeInstructionsHash != nil {
		t.Errorf("BridgeInstructionsHash: expected nil, got %v", m.BridgeInstructionsHash)
	}
}

func TestBuildManifest_EmptyChunks(t *testing.T) {
	m := manifest.BuildManifest(nil, "cl100k_base", nil, nil)
	if m.TotalChunks != 0 {
		t.Errorf("expected 0 total chunks, got %d", m.TotalChunks)
	}
	if len(m.Objects) != 0 || len(m.Specs) != 0 || len(m.System) != 0 {
		t.Error("expected empty maps for nil input")
	}
}

func TestBuildManifest_GroupsByType(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	if len(m.Objects) != 3 {
		t.Errorf("Objects: expected 3 entries, got %d", len(m.Objects))
	}
	if len(m.Specs) != 2 {
		t.Errorf("Specs: expected 2 entries, got %d", len(m.Specs))
	}
	if len(m.System) != 2 {
		t.Errorf("System: expected 2 entries, got %d", len(m.System))
	}
}

// ---------------------------------------------------------------------------
// Adjacency
// ---------------------------------------------------------------------------

func TestBuildManifest_ObjectAdjacency(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	// First object chunk: no previous, next is second.
	obj1 := m.Objects["obj:aaa11111.1"]
	if obj1 == nil {
		t.Fatal("expected obj:aaa11111.1 in objects")
	}
	if obj1.Adjacent == nil {
		t.Fatal("expected adjacency for object chunk")
	}
	if obj1.Adjacent.Previous != nil {
		t.Errorf("first chunk previous: expected nil, got %v", obj1.Adjacent.Previous)
	}
	if obj1.Adjacent.Next == nil || *obj1.Adjacent.Next != "obj:aaa11111.2" {
		t.Errorf("first chunk next: expected obj:aaa11111.2, got %v", obj1.Adjacent.Next)
	}

	// Second object chunk: previous is first, next is third.
	obj2 := m.Objects["obj:aaa11111.2"]
	if obj2.Adjacent.Previous == nil || *obj2.Adjacent.Previous != "obj:aaa11111.1" {
		t.Errorf("second chunk previous: expected obj:aaa11111.1, got %v", obj2.Adjacent.Previous)
	}
	if obj2.Adjacent.Next == nil || *obj2.Adjacent.Next != "obj:aaa11111.3" {
		t.Errorf("second chunk next: expected obj:aaa11111.3, got %v", obj2.Adjacent.Next)
	}

	// Third object chunk: previous is second, no next.
	obj3 := m.Objects["obj:aaa11111.3"]
	if obj3.Adjacent.Previous == nil || *obj3.Adjacent.Previous != "obj:aaa11111.2" {
		t.Errorf("third chunk previous: expected obj:aaa11111.2, got %v", obj3.Adjacent.Previous)
	}
	if obj3.Adjacent.Next != nil {
		t.Errorf("third chunk next: expected nil, got %v", obj3.Adjacent.Next)
	}
}

func TestBuildManifest_SystemAdjacency(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	sys1 := m.System["sys:ccc33333.1"]
	if sys1 == nil {
		t.Fatal("expected sys:ccc33333.1 in system")
	}
	if sys1.Adjacent.Previous != nil {
		t.Errorf("system first previous: expected nil, got %v", sys1.Adjacent.Previous)
	}
	if sys1.Adjacent.Next == nil || *sys1.Adjacent.Next != "sys:ccc33333.2" {
		t.Errorf("system first next: expected sys:ccc33333.2, got %v", sys1.Adjacent.Next)
	}

	sys2 := m.System["sys:ccc33333.2"]
	if sys2.Adjacent.Previous == nil || *sys2.Adjacent.Previous != "sys:ccc33333.1" {
		t.Errorf("system second previous: expected sys:ccc33333.1, got %v", sys2.Adjacent.Previous)
	}
	if sys2.Adjacent.Next != nil {
		t.Errorf("system second next: expected nil, got %v", sys2.Adjacent.Next)
	}
}

func TestBuildManifest_SpecChunksHaveNoAdjacency(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	for id, entry := range m.Specs {
		if entry.Adjacent != nil {
			t.Errorf("spec chunk %s should not have adjacency, got %+v", id, entry.Adjacent)
		}
	}
}

func TestBuildManifest_SingleChunkAdjacency(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID: "obj:solo.1", Type: chunk.ChunkObject,
			Source: "docs/topics/single.md", Sequence: 1, TotalInFile: 1, Tokens: 100,
		},
	}
	m := manifest.BuildManifest(chunks, "cl100k_base", nil, nil)
	entry := m.Objects["obj:solo.1"]
	if entry.Adjacent.Previous != nil || entry.Adjacent.Next != nil {
		t.Error("single chunk should have nil previous and next")
	}
}

func TestBuildManifest_MultipleFilesAdjacency(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:file1.1", Type: chunk.ChunkObject, Source: "docs/topics/a.md", Sequence: 1, TotalInFile: 2, Tokens: 100},
		{ID: "obj:file1.2", Type: chunk.ChunkObject, Source: "docs/topics/a.md", Sequence: 2, TotalInFile: 2, Tokens: 100},
		{ID: "obj:file2.1", Type: chunk.ChunkObject, Source: "docs/topics/b.md", Sequence: 1, TotalInFile: 1, Tokens: 100},
	}
	m := manifest.BuildManifest(chunks, "cl100k_base", nil, nil)

	// File a.md chunks are linked to each other, not to file b.md.
	a1 := m.Objects["obj:file1.1"]
	a2 := m.Objects["obj:file1.2"]
	b1 := m.Objects["obj:file2.1"]

	if a1.Adjacent.Next == nil || *a1.Adjacent.Next != "obj:file1.2" {
		t.Error("file1.1 next should be file1.2")
	}
	if a2.Adjacent.Previous == nil || *a2.Adjacent.Previous != "obj:file1.1" {
		t.Error("file1.2 previous should be file1.1")
	}
	if b1.Adjacent.Previous != nil || b1.Adjacent.Next != nil {
		t.Error("file2.1 should have no adjacency (only chunk in its file)")
	}
}

// ---------------------------------------------------------------------------
// Spec-Object Linking
// ---------------------------------------------------------------------------

func TestBuildManifest_SpecToObjectLinking(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	// Spec "Authentication > JWT Tokens" should link to object "Authentication > JWT Tokens".
	spec1 := m.Specs["spec:bbb22222.1"]
	if spec1.ParentObject == nil {
		t.Fatal("spec:bbb22222.1 should have a parent_object")
	}
	if *spec1.ParentObject != "obj:aaa11111.1" {
		t.Errorf("spec:bbb22222.1 parent_object: expected obj:aaa11111.1, got %q", *spec1.ParentObject)
	}

	// Spec "Authentication > JWT Tokens > Refresh" should link to
	// object "Authentication > JWT Tokens > Refresh".
	spec2 := m.Specs["spec:bbb22222.2"]
	if spec2.ParentObject == nil {
		t.Fatal("spec:bbb22222.2 should have a parent_object")
	}
	if *spec2.ParentObject != "obj:aaa11111.3" {
		t.Errorf("spec:bbb22222.2 parent_object: expected obj:aaa11111.3, got %q", *spec2.ParentObject)
	}
}

func TestBuildManifest_ObjectToSpecReverseLink(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	// Object "Authentication > JWT Tokens" should have spec_chunk pointing back.
	obj1 := m.Objects["obj:aaa11111.1"]
	if obj1.SpecChunk == nil {
		t.Fatal("obj:aaa11111.1 should have a spec_chunk")
	}
	if *obj1.SpecChunk != "spec:bbb22222.1" {
		t.Errorf("obj:aaa11111.1 spec_chunk: expected spec:bbb22222.1, got %q", *obj1.SpecChunk)
	}

	// Object "Authentication > JWT Tokens > Validation" has no matching spec.
	obj2 := m.Objects["obj:aaa11111.2"]
	if obj2.SpecChunk != nil {
		t.Errorf("obj:aaa11111.2 should have nil spec_chunk, got %q", *obj2.SpecChunk)
	}

	// Object "Authentication > JWT Tokens > Refresh" should link to spec.
	obj3 := m.Objects["obj:aaa11111.3"]
	if obj3.SpecChunk == nil {
		t.Fatal("obj:aaa11111.3 should have a spec_chunk")
	}
	if *obj3.SpecChunk != "spec:bbb22222.2" {
		t.Errorf("obj:aaa11111.3 spec_chunk: expected spec:bbb22222.2, got %q", *obj3.SpecChunk)
	}
}

func TestBuildManifest_SpecWithNoMatchingObject(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID: "spec:orphan.1", Type: chunk.ChunkSpec,
			Source: "docs/topics/orphan.spec.md", Heading: "Orphan Topic",
			Sequence: 1, TotalInFile: 1, Tokens: 200,
		},
	}
	m := manifest.BuildManifest(chunks, "cl100k_base", nil, nil)

	spec := m.Specs["spec:orphan.1"]
	if spec.ParentObject != nil {
		t.Errorf("orphan spec should have nil parent_object, got %q", *spec.ParentObject)
	}
}

// ---------------------------------------------------------------------------
// Placeholder fields
// ---------------------------------------------------------------------------

func TestBuildManifest_TermsNilWhenNoChunkTerms(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)
	for id, e := range m.Objects {
		if e.Terms != nil {
			t.Errorf("object %s terms should be nil when chunkTerms is nil, got %v", id, e.Terms)
		}
	}
}

func TestBuildManifest_TermsPopulatedFromChunkTerms(t *testing.T) {
	chunkTerms := map[string][]string{
		"obj:aaa11111.1": {"authentication", "jwt"},
		"obj:aaa11111.2": {"jwt", "validation"},
	}
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, chunkTerms)

	// Chunk 1 should have terms.
	obj1 := m.Objects["obj:aaa11111.1"]
	if len(obj1.Terms) != 2 {
		t.Errorf("obj1 terms: expected 2, got %d", len(obj1.Terms))
	}

	// Chunk 3 has no entry in chunkTerms -> nil terms.
	obj3 := m.Objects["obj:aaa11111.3"]
	if obj3.Terms != nil {
		t.Errorf("obj3 terms should be nil (not in chunkTerms), got %v", obj3.Terms)
	}
}

func TestBuildManifest_BridgeToNextIsNil(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)
	for id, e := range m.Objects {
		if e.BridgeToNext != nil {
			t.Errorf("object %s bridge_to_next should be nil, got %v", id, e.BridgeToNext)
		}
	}
}

// ---------------------------------------------------------------------------
// Entry fields
// ---------------------------------------------------------------------------

func TestBuildManifest_EntryFields(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)
	entry := m.Objects["obj:aaa11111.1"]

	if entry.Type != "object" {
		t.Errorf("Type: expected %q, got %q", "object", entry.Type)
	}
	if entry.Source != "docs/topics/auth/jwt.md" {
		t.Errorf("Source: expected %q, got %q", "docs/topics/auth/jwt.md", entry.Source)
	}
	if entry.Heading != "Authentication > JWT Tokens" {
		t.Errorf("Heading: expected %q, got %q", "Authentication > JWT Tokens", entry.Heading)
	}
	if entry.Sequence != 1 {
		t.Errorf("Sequence: expected 1, got %d", entry.Sequence)
	}
	if entry.TotalInFile != 3 {
		t.Errorf("TotalInFile: expected 3, got %d", entry.TotalInFile)
	}
	if entry.Tokens != 400 {
		t.Errorf("Tokens: expected 400, got %d", entry.Tokens)
	}
}

// ---------------------------------------------------------------------------
// LookupEntry
// ---------------------------------------------------------------------------

func TestLookupEntry_Object(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)
	e := m.LookupEntry("obj:aaa11111.1")
	if e == nil {
		t.Fatal("expected to find obj:aaa11111.1")
	}
	if e.Type != "object" {
		t.Errorf("expected type %q, got %q", "object", e.Type)
	}
}

func TestLookupEntry_Spec(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)
	e := m.LookupEntry("spec:bbb22222.1")
	if e == nil {
		t.Fatal("expected to find spec:bbb22222.1")
	}
}

func TestLookupEntry_System(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)
	e := m.LookupEntry("sys:ccc33333.1")
	if e == nil {
		t.Fatal("expected to find sys:ccc33333.1")
	}
}

func TestLookupEntry_NotFound(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)
	e := m.LookupEntry("obj:nonexistent.1")
	if e != nil {
		t.Error("expected nil for nonexistent ID")
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func TestManifest_WriteTo_RoundTrip(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	if err := m.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}

	content := string(data)

	// Should have the header comment.
	if !strings.HasPrefix(content, "# codectx compiled manifest") {
		t.Error("expected header comment")
	}

	// Should be valid YAML that can be unmarshaled.
	var loaded manifest.Manifest
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling manifest: %v", err)
	}

	if loaded.TotalChunks != m.TotalChunks {
		t.Errorf("round-trip TotalChunks: expected %d, got %d", m.TotalChunks, loaded.TotalChunks)
	}
	if loaded.TotalTokens != m.TotalTokens {
		t.Errorf("round-trip TotalTokens: expected %d, got %d", m.TotalTokens, loaded.TotalTokens)
	}
	if loaded.Encoding != m.Encoding {
		t.Errorf("round-trip Encoding: expected %q, got %q", m.Encoding, loaded.Encoding)
	}
}

func TestManifest_WriteTo_2SpaceIndent(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	if err := m.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Contains(content, "\t") {
		t.Error("manifest should not contain tabs")
	}

	// Verify nested keys use 2-space indent.
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent > 0 && indent%2 != 0 {
			t.Errorf("line %d: indent %d is not a multiple of 2: %q", i+1, indent, line)
		}
	}
}

func TestManifest_WriteTo_NullFields(t *testing.T) {
	m := manifest.BuildManifest(testChunks(), "cl100k_base", nil, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")
	if err := m.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// bridge_instructions_hash should be null.
	if !strings.Contains(content, "bridge_instructions_hash: null") {
		t.Error("expected bridge_instructions_hash: null in output")
	}

	// terms should be null for object entries.
	if !strings.Contains(content, "terms: null") && !strings.Contains(content, "terms: []") {
		t.Error("expected null or empty terms in output")
	}
}

// ---------------------------------------------------------------------------
// EntryPath
// ---------------------------------------------------------------------------

func TestEntryPath(t *testing.T) {
	got := manifest.EntryPath("/project/.codectx/compiled")
	expected := filepath.Join("/project/.codectx/compiled", "manifest.yml")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

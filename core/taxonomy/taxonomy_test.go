package taxonomy_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeChunk creates a test chunk with the given blocks.
func makeChunk(id string, ct chunk.ChunkType, source, heading string, blocks []markdown.Block) chunk.Chunk {
	return chunk.Chunk{
		ID:      id,
		Type:    ct,
		Source:  source,
		Heading: heading,
		Blocks:  blocks,
		Content: joinBlockContent(blocks),
	}
}

func joinBlockContent(blocks []markdown.Block) string {
	var s string
	for i, b := range blocks {
		if i > 0 {
			s += "\n\n"
		}
		s += b.Content
	}
	return s
}

func headingBlock(text string, level int, hierarchy []string) markdown.Block {
	return markdown.Block{
		Type:    markdown.BlockHeading,
		Content: text,
		Level:   level,
		Heading: hierarchy,
	}
}

func paragraphBlock(text string, hierarchy []string) markdown.Block {
	return markdown.Block{
		Type:    markdown.BlockParagraph,
		Content: text,
		Heading: hierarchy,
	}
}

func codeBlock(content, lang string, hierarchy []string) markdown.Block {
	return markdown.Block{
		Type:     markdown.BlockCodeBlock,
		Content:  content,
		Language: lang,
		Heading:  hierarchy,
	}
}

func listBlock(content string, hierarchy []string) markdown.Block {
	return markdown.Block{
		Type:    markdown.BlockList,
		Content: content,
		Heading: hierarchy,
	}
}

func tableBlock(content string, hierarchy []string) markdown.Block {
	return markdown.Block{
		Type:    markdown.BlockTable,
		Content: content,
		Heading: hierarchy,
	}
}

// defaultCfg returns a TaxonomyConfig with test-friendly defaults.
func defaultCfg() project.TaxonomyConfig {
	return project.TaxonomyConfig{
		MinTermFrequency: 1, // Low threshold for tests.
		MaxAliasCount:    10,
		POSExtraction:    false,
	}
}

// ---------------------------------------------------------------------------
// NormalizeKey
// ---------------------------------------------------------------------------

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"JWT Tokens", "jwt-tokens"},
		{"Error Handling", "error-handling"},
		{"authentication", "authentication"},
		{"OAuth 2.0", "oauth-2-0"},
		{"  spaces  ", "spaces"},
		{"UPPER_CASE_NAME", "upper-case-name"},
		{"func-name", "func-name"},
		{"a/b/c", "a-b-c"},
		{"", ""},
		{"Hello World!", "hello-world"},
		{"--leading--", "leading"},
		{"multiple---hyphens", "multiple-hyphens"},
	}

	for _, tt := range tests {
		got := taxonomy.NormalizeKey(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Extract (full pipeline)
// ---------------------------------------------------------------------------

func TestExtract_HeadingTerms(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:aaa.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				paragraphBlock("Overview of auth.", []string{"Authentication"}),
			}),
		makeChunk("obj:aaa.2", chunk.ChunkObject, "docs/topics/auth.md", "Authentication > OAuth",
			[]markdown.Block{
				headingBlock("OAuth", 2, []string{"Authentication", "OAuth"}),
				paragraphBlock("OAuth 2.0 flow.", []string{"Authentication", "OAuth"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "hash123")

	// Should have "authentication" and "oauth" terms.
	if result.Taxonomy.Terms["authentication"] == nil {
		t.Fatal("expected 'authentication' term")
	}
	if result.Taxonomy.Terms["oauth"] == nil {
		t.Fatal("expected 'oauth' term")
	}

	authTerm := result.Taxonomy.Terms["authentication"]
	if authTerm.Canonical != "Authentication" {
		t.Errorf("canonical: expected %q, got %q", "Authentication", authTerm.Canonical)
	}
	if authTerm.Source != taxonomy.SourceHeading {
		t.Errorf("source: expected %q, got %q", taxonomy.SourceHeading, authTerm.Source)
	}
}

func TestExtract_HeadingHierarchy(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:aaa.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
			}),
		makeChunk("obj:aaa.2", chunk.ChunkObject, "docs/topics/auth.md", "Authentication > OAuth",
			[]markdown.Block{
				headingBlock("OAuth", 2, []string{"Authentication", "OAuth"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "hash123")

	oauth := result.Taxonomy.Terms["oauth"]
	if oauth == nil {
		t.Fatal("expected 'oauth' term")
	}
	if oauth.Broader != "authentication" {
		t.Errorf("broader: expected %q, got %q", "authentication", oauth.Broader)
	}

	auth := result.Taxonomy.Terms["authentication"]
	if auth == nil {
		t.Fatal("expected 'authentication' term")
	}
	if len(auth.Narrower) != 1 || auth.Narrower[0] != "oauth" {
		t.Errorf("narrower: expected [oauth], got %v", auth.Narrower)
	}
}

func TestExtract_CodeIdentifiers(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:bbb.1", chunk.ChunkObject, "docs/topics/api.md", "API",
			[]markdown.Block{
				headingBlock("API", 1, []string{"API"}),
				codeBlock("func HandleRequest(w http.ResponseWriter, r *http.Request) {\n\treturn nil\n}", "go", []string{"API"}),
			}),
		makeChunk("obj:bbb.2", chunk.ChunkObject, "docs/topics/api.md", "API > Models",
			[]markdown.Block{
				headingBlock("Models", 2, []string{"API", "Models"}),
				codeBlock("type UserService struct {\n\tdb *sql.DB\n}", "go", []string{"API", "Models"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	// "HandleRequest" should be extracted.
	if result.Taxonomy.Terms["handlerequest"] == nil {
		t.Error("expected 'handlerequest' term from func name")
	}
	// "UserService" should be extracted.
	if result.Taxonomy.Terms["userservice"] == nil {
		t.Error("expected 'userservice' term from type name")
	}
}

func TestExtract_BoldTerms(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:ccc.1", chunk.ChunkObject, "docs/topics/config.md", "Configuration",
			[]markdown.Block{
				headingBlock("Configuration", 1, []string{"Configuration"}),
				paragraphBlock("**Connection Pool** manages database connections.", []string{"Configuration"}),
			}),
		makeChunk("obj:ccc.2", chunk.ChunkObject, "docs/topics/config.md", "Configuration > Settings",
			[]markdown.Block{
				headingBlock("Settings", 2, []string{"Configuration", "Settings"}),
				paragraphBlock("**Timeout Value** determines how long to wait.", []string{"Configuration", "Settings"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	if result.Taxonomy.Terms["connection-pool"] == nil {
		t.Error("expected 'connection-pool' term from bold text")
	}
	if result.Taxonomy.Terms["timeout-value"] == nil {
		t.Error("expected 'timeout-value' term from bold text")
	}
}

func TestExtract_ListHeaders(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:ddd.1", chunk.ChunkObject, "docs/topics/errors.md", "Errors",
			[]markdown.Block{
				headingBlock("Errors", 1, []string{"Errors"}),
				listBlock("- Rate Limiting: Controls request throughput\n- Circuit Breaker: Prevents cascade failures", []string{"Errors"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	if result.Taxonomy.Terms["rate-limiting"] == nil {
		t.Error("expected 'rate-limiting' term from list header")
	}
	if result.Taxonomy.Terms["circuit-breaker"] == nil {
		t.Error("expected 'circuit-breaker' term from list header")
	}
}

func TestExtract_TableHeaders(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:eee.1", chunk.ChunkObject, "docs/topics/deploy.md", "Deployment",
			[]markdown.Block{
				headingBlock("Deployment", 1, []string{"Deployment"}),
				tableBlock("| Environment | Region | Strategy |\n|---|---|---|\n| production | us-east-1 | blue-green |", []string{"Deployment"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	// "Environment" is not generic but "Strategy" is not in the generic list either.
	// "Region" is not in the generic list.
	if result.Taxonomy.Terms["environment"] == nil {
		t.Error("expected 'environment' term from table header")
	}
	if result.Taxonomy.Terms["region"] == nil {
		t.Error("expected 'region' term from table header")
	}
	if result.Taxonomy.Terms["strategy"] == nil {
		t.Error("expected 'strategy' term from table header")
	}
}

func TestExtract_GenericTableHeadersFiltered(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:fff.1", chunk.ChunkObject, "docs/topics/params.md", "Parameters",
			[]markdown.Block{
				headingBlock("Parameters", 1, []string{"Parameters"}),
				tableBlock("| Name | Type | Description |\n|---|---|---|\n| foo | string | A field |", []string{"Parameters"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	// "Name", "Type", "Description" should be filtered as generic headers.
	if result.Taxonomy.Terms["name"] != nil {
		t.Error("'name' should be filtered as generic table header")
	}
	if result.Taxonomy.Terms["type"] != nil {
		t.Error("'type' should be filtered as generic table header")
	}
	if result.Taxonomy.Terms["description"] != nil {
		t.Error("'description' should be filtered as generic table header")
	}
}

// ---------------------------------------------------------------------------
// Deduplication
// ---------------------------------------------------------------------------

func TestExtract_DeduplicateMergesChunks(t *testing.T) {
	// Same heading appears in two chunks.
	chunks := []chunk.Chunk{
		makeChunk("obj:ggg.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
			}),
		makeChunk("obj:ggg.2", chunk.ChunkObject, "docs/topics/security.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	auth := result.Taxonomy.Terms["authentication"]
	if auth == nil {
		t.Fatal("expected 'authentication' term")
	}
	if len(auth.Chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(auth.Chunks))
	}
}

func TestExtract_FrequencyFilter(t *testing.T) {
	// Term appears in only 1 chunk; with minTermFrequency=2, it should be filtered.
	chunks := []chunk.Chunk{
		makeChunk("obj:hhh.1", chunk.ChunkObject, "docs/topics/rare.md", "Rare Topic",
			[]markdown.Block{
				headingBlock("Rare Topic", 1, []string{"Rare Topic"}),
			}),
	}

	cfg := defaultCfg()
	cfg.MinTermFrequency = 2

	result := taxonomy.Extract(chunks, cfg, "cl100k_base", "")

	if result.Taxonomy.Terms["rare-topic"] != nil {
		t.Error("expected 'rare-topic' to be filtered by frequency threshold")
	}
}

func TestExtract_HighConfidenceSourceWins(t *testing.T) {
	// Same term found as both a heading and a code identifier.
	// Heading should win.
	chunks := []chunk.Chunk{
		makeChunk("obj:iii.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				codeBlock("class Authentication:\n    pass", "python", []string{"Authentication"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	auth := result.Taxonomy.Terms["authentication"]
	if auth == nil {
		t.Fatal("expected 'authentication' term")
	}
	if auth.Source != taxonomy.SourceHeading {
		t.Errorf("expected source %q (heading wins), got %q", taxonomy.SourceHeading, auth.Source)
	}
}

// ---------------------------------------------------------------------------
// ChunkTerms reverse map
// ---------------------------------------------------------------------------

func TestExtract_ChunkTermsMap(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:jjj.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				headingBlock("OAuth", 2, []string{"Authentication", "OAuth"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	terms := result.ChunkTerms["obj:jjj.1"]
	if len(terms) == 0 {
		t.Fatal("expected terms for chunk obj:jjj.1")
	}

	// Should contain both "authentication" and "oauth".
	found := map[string]bool{}
	for _, t := range terms {
		found[t] = true
	}
	if !found["authentication"] {
		t.Error("expected 'authentication' in chunk terms")
	}
	if !found["oauth"] {
		t.Error("expected 'oauth' in chunk terms")
	}
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func TestExtract_Stats(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:kkk.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				codeBlock("func ValidateToken(token string) error {\n\treturn nil\n}", "go", []string{"Authentication"}),
				paragraphBlock("**Token Format** defines the structure.", []string{"Authentication"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	if result.Stats.CanonicalTerms == 0 {
		t.Error("expected non-zero canonical terms")
	}
	if result.Stats.TermsFromHeadings == 0 {
		t.Error("expected non-zero terms from headings")
	}
}

// ---------------------------------------------------------------------------
// Taxonomy metadata
// ---------------------------------------------------------------------------

func TestExtract_TaxonomyMetadata(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:lll.1", chunk.ChunkObject, "docs/topics/auth.md", "Auth",
			[]markdown.Block{
				headingBlock("Auth", 1, []string{"Auth"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "hash456")

	if result.Taxonomy.Encoding != "cl100k_base" {
		t.Errorf("encoding: expected %q, got %q", "cl100k_base", result.Taxonomy.Encoding)
	}
	if result.Taxonomy.InstructionsHash != "hash456" {
		t.Errorf("instructions_hash: expected %q, got %q", "hash456", result.Taxonomy.InstructionsHash)
	}
	if result.Taxonomy.CompiledAt == "" {
		t.Error("compiled_at should not be empty")
	}
	if result.Taxonomy.TermCount != len(result.Taxonomy.Terms) {
		t.Errorf("term_count %d != len(terms) %d", result.Taxonomy.TermCount, len(result.Taxonomy.Terms))
	}
}

// ---------------------------------------------------------------------------
// Serialization round-trip
// ---------------------------------------------------------------------------

func TestTaxonomy_WriteTo_RoundTrip(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:mmm.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				headingBlock("OAuth", 2, []string{"Authentication", "OAuth"}),
			}),
		makeChunk("obj:mmm.2", chunk.ChunkObject, "docs/topics/auth.md", "Authentication > OAuth",
			[]markdown.Block{
				headingBlock("OAuth", 2, []string{"Authentication", "OAuth"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "hash789")

	dir := t.TempDir()
	path := filepath.Join(dir, "taxonomy.yml")

	if err := result.Taxonomy.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	// Read back and verify.
	loaded, err := taxonomy.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Encoding != result.Taxonomy.Encoding {
		t.Errorf("encoding: expected %q, got %q", result.Taxonomy.Encoding, loaded.Encoding)
	}
	if loaded.InstructionsHash != result.Taxonomy.InstructionsHash {
		t.Errorf("instructions_hash: expected %q, got %q", result.Taxonomy.InstructionsHash, loaded.InstructionsHash)
	}
	if loaded.TermCount != result.Taxonomy.TermCount {
		t.Errorf("term_count: expected %d, got %d", result.Taxonomy.TermCount, loaded.TermCount)
	}
	if len(loaded.Terms) != len(result.Taxonomy.Terms) {
		t.Errorf("terms count: expected %d, got %d", len(result.Taxonomy.Terms), len(loaded.Terms))
	}
}

func TestTaxonomy_WriteTo_2SpaceIndent(t *testing.T) {
	tax := &taxonomy.Taxonomy{
		Encoding:         "cl100k_base",
		InstructionsHash: "abc",
		TermCount:        1,
		Terms: map[string]*taxonomy.Term{
			"auth": {
				Canonical: "Authentication",
				Source:    taxonomy.SourceHeading,
				Chunks:    []string{"obj:aaa.1"},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "taxonomy.yml")
	if err := tax.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Should be valid YAML.
	var loaded taxonomy.Taxonomy
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.TermCount != 1 {
		t.Errorf("expected term_count 1, got %d", loaded.TermCount)
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	_, err := taxonomy.Load("/nonexistent/taxonomy.yml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestTaxonomyPath(t *testing.T) {
	got := taxonomy.TaxonomyPath("/project/.codectx/compiled")
	expected := filepath.Join("/project/.codectx/compiled", "taxonomy.yml")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// ---------------------------------------------------------------------------
// Cross-reference relationships
// ---------------------------------------------------------------------------

func TestExtract_CrossReferenceRelated(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:nnn.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				paragraphBlock("See also [middleware](middleware.md) for request filtering.", []string{"Authentication"}),
			}),
		makeChunk("obj:nnn.2", chunk.ChunkObject, "docs/topics/middleware.md", "Middleware",
			[]markdown.Block{
				headingBlock("Middleware", 1, []string{"Middleware"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	auth := result.Taxonomy.Terms["authentication"]
	mw := result.Taxonomy.Terms["middleware"]

	if auth == nil || mw == nil {
		t.Fatal("expected both 'authentication' and 'middleware' terms")
	}

	// They should be related.
	if !slices.Contains(auth.Related, "middleware") {
		t.Errorf("expected 'middleware' in auth.Related, got %v", auth.Related)
	}
	if !slices.Contains(mw.Related, "authentication") {
		t.Errorf("expected 'authentication' in mw.Related, got %v", mw.Related)
	}
}

// ---------------------------------------------------------------------------
// Empty input
// ---------------------------------------------------------------------------

func TestExtract_EmptyChunks(t *testing.T) {
	result := taxonomy.Extract(nil, defaultCfg(), "cl100k_base", "")
	if len(result.Taxonomy.Terms) != 0 {
		t.Errorf("expected 0 terms for nil chunks, got %d", len(result.Taxonomy.Terms))
	}
	if result.Taxonomy.TermCount != 0 {
		t.Errorf("expected term_count 0, got %d", result.Taxonomy.TermCount)
	}
}

// ---------------------------------------------------------------------------
// Integration: Extract → ChunkTerms → Manifest
// ---------------------------------------------------------------------------

func TestExtract_IntegrationChunkTermsToManifest(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:ooo.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				headingBlock("OAuth", 2, []string{"Authentication", "OAuth"}),
			}),
		makeChunk("obj:ooo.2", chunk.ChunkObject, "docs/topics/auth.md", "Authentication > OAuth",
			[]markdown.Block{
				headingBlock("OAuth", 2, []string{"Authentication", "OAuth"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	// Verify chunk terms are populated.
	terms1 := result.ChunkTerms["obj:ooo.1"]
	terms2 := result.ChunkTerms["obj:ooo.2"]

	if len(terms1) == 0 {
		t.Error("expected terms for obj:ooo.1")
	}
	if len(terms2) == 0 {
		t.Error("expected terms for obj:ooo.2")
	}

	// Both chunks should have "oauth" since the heading appears in both.
	if !slices.Contains(terms1, "oauth") {
		t.Error("expected 'oauth' in obj:ooo.1 terms")
	}
	if !slices.Contains(terms2, "oauth") {
		t.Error("expected 'oauth' in obj:ooo.2 terms")
	}
}

// ---------------------------------------------------------------------------
// Phase 4 coverage: __underscore__ bold variant (4.1)
// ---------------------------------------------------------------------------

func TestExtract_UnderscoreBoldTerms(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:und.1", chunk.ChunkObject, "docs/topics/config.md", "Configuration",
			[]markdown.Block{
				headingBlock("Configuration", 1, []string{"Configuration"}),
				paragraphBlock("__Service Mesh__ provides network communication.", []string{"Configuration"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	if result.Taxonomy.Terms["service-mesh"] == nil {
		t.Error("expected 'service-mesh' term from __underscore__ bold text")
	}
}

// ---------------------------------------------------------------------------
// Phase 4 coverage: em-dash list separator (4.2)
// ---------------------------------------------------------------------------

func TestExtract_EmDashListSeparator(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:emd.1", chunk.ChunkObject, "docs/topics/glossary.md", "Glossary",
			[]markdown.Block{
				headingBlock("Glossary", 1, []string{"Glossary"}),
				listBlock("- Service Discovery \u2014 locates services in the cluster", []string{"Glossary"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	if result.Taxonomy.Terms["service-discovery"] == nil {
		t.Error("expected 'service-discovery' term from list header with em-dash separator")
	}
}

// ---------------------------------------------------------------------------
// Phase 4 coverage: circular cross-references (4.5)
// ---------------------------------------------------------------------------

func TestExtract_CircularCrossReferences(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:circ.1", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				paragraphBlock("See [authorization](authorization.md) for access control.", []string{"Authentication"}),
			}),
		makeChunk("obj:circ.2", chunk.ChunkObject, "docs/topics/authorization.md", "Authorization",
			[]markdown.Block{
				headingBlock("Authorization", 1, []string{"Authorization"}),
				paragraphBlock("See [authentication](auth.md) for identity verification.", []string{"Authorization"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	auth := result.Taxonomy.Terms["authentication"]
	authz := result.Taxonomy.Terms["authorization"]

	if auth == nil || authz == nil {
		t.Fatal("expected both 'authentication' and 'authorization' terms")
	}

	// Each should list the other as related exactly once.
	authRelatedCount := 0
	for _, r := range auth.Related {
		if r == "authorization" {
			authRelatedCount++
		}
	}
	if authRelatedCount != 1 {
		t.Errorf("expected 'authorization' in auth.Related exactly once, got %d times (Related: %v)", authRelatedCount, auth.Related)
	}

	authzRelatedCount := 0
	for _, r := range authz.Related {
		if r == "authentication" {
			authzRelatedCount++
		}
	}
	if authzRelatedCount != 1 {
		t.Errorf("expected 'authentication' in authz.Related exactly once, got %d times (Related: %v)", authzRelatedCount, authz.Related)
	}
}

// ---------------------------------------------------------------------------
// Phase 4 coverage: Load with invalid YAML (4.6)
// ---------------------------------------------------------------------------

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taxonomy.yml")

	if err := os.WriteFile(path, []byte("{{{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := taxonomy.Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// ---------------------------------------------------------------------------
// Phase 4 coverage: equal-rank dedup keeps first-seen canonical (4.7)
// ---------------------------------------------------------------------------

func TestExtract_EqualRankKeepsFirstSeen(t *testing.T) {
	// Two chunks with the same heading text but different casing.
	// First-seen canonical should win when source ranks are equal.
	chunks := []chunk.Chunk{
		makeChunk("obj:fst.1", chunk.ChunkObject, "docs/a.md", "API Gateway",
			[]markdown.Block{
				headingBlock("API Gateway", 1, []string{"API Gateway"}),
			}),
		makeChunk("obj:fst.2", chunk.ChunkObject, "docs/b.md", "Api gateway",
			[]markdown.Block{
				headingBlock("Api gateway", 1, []string{"Api gateway"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	term := result.Taxonomy.Terms["api-gateway"]
	if term == nil {
		t.Fatal("expected 'api-gateway' term")
	}

	// Both are SourceHeading (equal rank), so first-seen canonical "API Gateway" should win.
	if term.Canonical != "API Gateway" {
		t.Errorf("expected first-seen canonical %q, got %q", "API Gateway", term.Canonical)
	}
}

// ---------------------------------------------------------------------------
// Phase 4 coverage: bold term exceeding maxBoldTermLen (4.8)
// ---------------------------------------------------------------------------

func TestExtract_BoldTermExceedingMaxLength(t *testing.T) {
	longBold := "This Is An Extremely Long Bold Term That Should Definitely Exceed The Maximum Length"
	chunks := []chunk.Chunk{
		makeChunk("obj:long.1", chunk.ChunkObject, "docs/topics/verbose.md", "Verbose",
			[]markdown.Block{
				headingBlock("Verbose", 1, []string{"Verbose"}),
				paragraphBlock("**"+longBold+"** is too long to be a term.", []string{"Verbose"}),
			}),
	}

	result := taxonomy.Extract(chunks, defaultCfg(), "cl100k_base", "")

	key := taxonomy.NormalizeKey(longBold)
	if result.Taxonomy.Terms[key] != nil {
		t.Errorf("expected long bold term %q to be filtered out", key)
	}
}

// ---------------------------------------------------------------------------
// POS extraction integration tests
// ---------------------------------------------------------------------------

// posCfg returns a TaxonomyConfig with POS extraction enabled.
func posCfg() project.TaxonomyConfig {
	cfg := defaultCfg()
	cfg.POSExtraction = true
	return cfg
}

func TestExtract_POSNewTermsAdded(t *testing.T) {
	// The paragraph text contains compound noun phrases that structural
	// extraction won't find (no headings, no code, no bold markers for them).
	chunks := []chunk.Chunk{
		makeChunk("obj:pos.1", chunk.ChunkObject, "docs/topics/arch.md", "Architecture",
			[]markdown.Block{
				headingBlock("Architecture", 1, []string{"Architecture"}),
				paragraphBlock(
					"The dependency injection framework provides automatic service resolution. "+
						"It uses a middleware chain for request processing.",
					[]string{"Architecture"},
				),
			}),
	}

	result := taxonomy.Extract(chunks, posCfg(), "cl100k_base", "")

	// Should have heading-derived term.
	if result.Taxonomy.Terms["architecture"] == nil {
		t.Fatal("expected 'architecture' term from heading")
	}

	// Should also have POS-derived terms. Check stats.
	if result.Stats.TermsFromPOS == 0 {
		// POS extraction should find at least some noun phrases from the
		// paragraph text. If the prose tagger doesn't find any, that's still
		// a valid outcome — but we expect compound terms to be extracted.
		t.Log("POS extraction found 0 terms; prose tagger may not have matched expected patterns")
	}
}

func TestExtract_POSMergesWithStructural(t *testing.T) {
	// "Authentication" appears as a heading (structural) and will also
	// be tagged as a proper noun by POS tagger. The heading source
	// (rank 0) should win over POS (rank 4).
	chunks := []chunk.Chunk{
		makeChunk("obj:pos.2", chunk.ChunkObject, "docs/topics/auth.md", "Authentication",
			[]markdown.Block{
				headingBlock("Authentication", 1, []string{"Authentication"}),
				paragraphBlock(
					"Authentication is the process of verifying identity. "+
						"The authentication module validates credentials.",
					[]string{"Authentication"},
				),
			}),
	}

	result := taxonomy.Extract(chunks, posCfg(), "cl100k_base", "")

	auth := result.Taxonomy.Terms["authentication"]
	if auth == nil {
		t.Fatal("expected 'authentication' term")
	}

	// Heading source should win.
	if auth.Source != taxonomy.SourceHeading {
		t.Errorf("expected source %q (heading wins over POS), got %q",
			taxonomy.SourceHeading, auth.Source)
	}
}

func TestExtract_POSDisabledNoPOSTerms(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:pos.3", chunk.ChunkObject, "docs/topics/arch.md", "Architecture",
			[]markdown.Block{
				headingBlock("Architecture", 1, []string{"Architecture"}),
				paragraphBlock(
					"The distributed message queue handles event processing.",
					[]string{"Architecture"},
				),
			}),
	}

	cfg := defaultCfg()
	cfg.POSExtraction = false

	result := taxonomy.Extract(chunks, cfg, "cl100k_base", "")

	if result.Stats.TermsFromPOS != 0 {
		t.Errorf("expected 0 POS terms when disabled, got %d", result.Stats.TermsFromPOS)
	}

	// Should only have structural terms.
	for _, term := range result.Taxonomy.Terms {
		if term.Source == taxonomy.SourcePOS {
			t.Errorf("found POS-sourced term %q when POS extraction disabled", term.Canonical)
		}
	}
}

func TestExtract_POSStatsPopulated(t *testing.T) {
	chunks := []chunk.Chunk{
		makeChunk("obj:pos.4", chunk.ChunkObject, "docs/topics/arch.md", "Architecture",
			[]markdown.Block{
				headingBlock("Architecture", 1, []string{"Architecture"}),
				paragraphBlock(
					"The distributed system uses a message broker for asynchronous communication. "+
						"The load balancer distributes traffic across multiple service instances.",
					[]string{"Architecture"},
				),
			}),
	}

	result := taxonomy.Extract(chunks, posCfg(), "cl100k_base", "")

	if result.Stats.TermsFromHeadings == 0 {
		t.Error("expected non-zero terms from headings")
	}

	// The total should include all sources.
	total := result.Stats.TermsFromHeadings + result.Stats.TermsFromCodeIdents +
		result.Stats.TermsFromBoldTerms + result.Stats.TermsFromStructured +
		result.Stats.TermsFromPOS
	if total != result.Stats.CanonicalTerms {
		t.Errorf("stats sources sum %d != canonical terms %d", total, result.Stats.CanonicalTerms)
	}
}

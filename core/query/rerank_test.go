package query

import (
	"testing"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
)

func defaultGraphConfig() project.GraphRerankConfig {
	return project.DefaultQueryConfig().GraphRerank
}

func testManifest() *manifest.Manifest {
	prev := "obj:abc.1"
	next := "obj:abc.3"
	specID := "spec:abc.1"
	parentID := "obj:abc.2"

	return &manifest.Manifest{
		Objects: map[string]*manifest.ManifestEntry{
			"obj:abc.1": {
				Type:   "object",
				Source: "topics/auth/README.md",
				Adjacent: &manifest.Adjacency{
					Next: &next,
				},
			},
			"obj:abc.2": {
				Type:      "object",
				Source:    "topics/auth/README.md",
				SpecChunk: &specID,
				Adjacent: &manifest.Adjacency{
					Previous: &prev,
					Next:     &next,
				},
			},
			"obj:abc.3": {
				Type:   "object",
				Source: "topics/auth/README.md",
				Adjacent: &manifest.Adjacency{
					Previous: &parentID,
				},
			},
			"obj:def.1": {
				Type:   "object",
				Source: "topics/jwt/README.md",
			},
		},
		Specs: map[string]*manifest.ManifestEntry{
			"spec:abc.1": {
				Type:         "spec",
				Source:       "topics/auth/README.spec.md",
				ParentObject: &parentID,
			},
		},
		System: map[string]*manifest.ManifestEntry{},
	}
}

func TestGraphRerank_AdjacentBoost(t *testing.T) {
	cfg := defaultGraphConfig()
	mfst := testManifest()

	// obj:abc.2 has Previous=obj:abc.1 and Next=obj:abc.3.
	// If obj:abc.1 and obj:abc.3 are both in the window, obj:abc.2 gets adjacent boost.
	results := []RRFResult{
		{ID: "obj:abc.2", RRFScore: 1.0, Sources: map[string]int{"objects": 1}},
		{ID: "obj:abc.1", RRFScore: 0.9, Sources: map[string]int{"objects": 2}},
		{ID: "obj:def.1", RRFScore: 0.8, Sources: map[string]int{"objects": 3}},
	}

	reranked := GraphRerank(results, mfst, nil, cfg, 3)

	for _, r := range reranked {
		if r.ID == "obj:def.1" {
			// Should have no boost — original score 0.8.
			if r.RRFScore != 0.8 {
				t.Errorf("obj:def.1 score = %f, want 0.8 (no boost)", r.RRFScore)
			}
		}
		if r.ID == "obj:abc.2" {
			// Adjacent to obj:abc.1 (Previous, in window) → +15%.
			// Note: obj:abc.3 is NOT in results so Next doesn't trigger.
			expected := 1.0 * (1.0 + cfg.AdjacentBoost)
			if r.RRFScore != expected {
				t.Errorf("obj:abc.2 score = %f, want %f (adjacent boost)", r.RRFScore, expected)
			}
		}
	}
}

func TestGraphRerank_SpecBoost(t *testing.T) {
	cfg := defaultGraphConfig()
	mfst := testManifest()

	// obj:abc.2 has SpecChunk → spec:abc.1. If both are in the window,
	// they should boost each other.
	results := []RRFResult{
		{ID: "obj:abc.2", RRFScore: 1.0, Sources: map[string]int{"objects": 1}},
		{ID: "spec:abc.1", RRFScore: 0.9, Sources: map[string]int{"specs": 1}},
		{ID: "obj:def.1", RRFScore: 0.5, Sources: map[string]int{"objects": 3}},
	}

	reranked := GraphRerank(results, mfst, nil, cfg, 3)

	for _, r := range reranked {
		if r.ID == "obj:abc.2" {
			// Has spec:abc.1 in window → spec boost.
			// Also adjacent to obj:abc.1 which is NOT in window.
			if r.RRFScore <= 1.0 {
				t.Errorf("obj:abc.2 should be boosted, got %f", r.RRFScore)
			}
		}
		if r.ID == "spec:abc.1" {
			// Has ParentObject → obj:abc.2 in window → spec boost.
			if r.RRFScore <= 0.9 {
				t.Errorf("spec:abc.1 should be boosted, got %f", r.RRFScore)
			}
		}
	}
}

func TestGraphRerank_Disabled(t *testing.T) {
	cfg := defaultGraphConfig()
	cfg.Enabled = project.BoolPtr(false)
	mfst := testManifest()

	results := []RRFResult{
		{ID: "obj:abc.1", RRFScore: 1.0, Sources: map[string]int{"objects": 1}},
	}

	reranked := GraphRerank(results, mfst, nil, cfg, 1)

	if reranked[0].RRFScore != 1.0 {
		t.Errorf("disabled graph rerank should not change scores, got %f", reranked[0].RRFScore)
	}
}

func TestGraphRerank_NilManifest(t *testing.T) {
	cfg := defaultGraphConfig()
	results := []RRFResult{
		{ID: "obj:abc.1", RRFScore: 1.0, Sources: map[string]int{"objects": 1}},
	}

	reranked := GraphRerank(results, nil, nil, cfg, 1)
	if reranked[0].RRFScore != 1.0 {
		t.Errorf("nil manifest should not change scores, got %f", reranked[0].RRFScore)
	}
}

func TestGraphRerank_EmptyResults(t *testing.T) {
	cfg := defaultGraphConfig()
	reranked := GraphRerank(nil, testManifest(), nil, cfg, 10)
	if len(reranked) != 0 {
		t.Errorf("expected 0 results, got %d", len(reranked))
	}
}

func TestGraphRerank_WindowSize(t *testing.T) {
	cfg := defaultGraphConfig()
	mfst := testManifest()

	// With topN=1, window = ceil(1 * 2.15) = 3.
	// But we only have 2 results, so window = 2.
	results := []RRFResult{
		{ID: "obj:abc.1", RRFScore: 1.0, Sources: map[string]int{"objects": 1}},
		{ID: "obj:abc.2", RRFScore: 0.5, Sources: map[string]int{"objects": 2}},
	}

	reranked := GraphRerank(results, mfst, nil, cfg, 1)

	// Both should be in the window, so adjacency boost should apply.
	if len(reranked) != 2 {
		t.Fatalf("expected 2 results, got %d", len(reranked))
	}
}

func TestGraphRerank_CrossRefBoost_WithMetadata(t *testing.T) {
	cfg := defaultGraphConfig()
	mfst := testManifest()

	// Create metadata where auth doc references jwt doc.
	meta := &manifest.Metadata{
		Documents: map[string]*manifest.DocumentEntry{
			"topics/auth/README.md": {
				ReferencesTo: []manifest.Reference{
					{Path: "topics/jwt/README.md", Reason: "link"},
				},
			},
			"topics/jwt/README.md": {
				ReferencedBy: []manifest.Reference{
					{Path: "topics/auth/README.md", Reason: "link"},
				},
			},
		},
	}

	// obj:abc.1 is from auth doc, obj:def.1 is from jwt doc.
	// Both score → auth doc cross-refs jwt doc → both get cross-ref boost.
	results := []RRFResult{
		{ID: "obj:abc.1", RRFScore: 1.0, Sources: map[string]int{"objects": 1}},
		{ID: "obj:def.1", RRFScore: 0.5, Sources: map[string]int{"objects": 2}},
	}

	reranked := GraphRerank(results, mfst, meta, cfg, 2)

	// obj:abc.1 should get cross-ref boost since jwt doc also has scored chunks.
	for _, r := range reranked {
		if r.ID == "obj:abc.1" {
			expected := 1.0 * (1.0 + cfg.CrossRefBoost)
			if r.RRFScore != expected {
				t.Errorf("obj:abc.1 score = %f, want %f (cross-ref boost)", r.RRFScore, expected)
			}
		}
	}
}

func TestGraphRerank_CrossRefBoost_NilMetadata(t *testing.T) {
	cfg := defaultGraphConfig()
	mfst := testManifest()

	results := []RRFResult{
		{ID: "obj:abc.1", RRFScore: 1.0, Sources: map[string]int{"objects": 1}},
	}

	// Nil metadata should not panic — just skip cross-ref boost.
	reranked := GraphRerank(results, mfst, nil, cfg, 1)
	if len(reranked) != 1 {
		t.Fatalf("expected 1 result, got %d", len(reranked))
	}
}

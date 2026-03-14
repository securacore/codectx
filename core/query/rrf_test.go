package query

import (
	"testing"

	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/project"
)

func defaultRRFConfig() project.RRFConfig {
	return project.DefaultQueryConfig().RRF
}

func TestWeightedRRF_SingleList(t *testing.T) {
	cfg := defaultRRFConfig()
	lists := map[index.IndexType][]index.ScoredResult{
		index.IndexObjects: {
			{ChunkID: "obj:a.1", Score: 5.0},
			{ChunkID: "obj:b.1", Score: 3.0},
		},
	}

	results := WeightedRRF(lists, cfg)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "obj:a.1" {
		t.Errorf("first result = %q, want obj:a.1", results[0].ID)
	}
	if results[1].ID != "obj:b.1" {
		t.Errorf("second result = %q, want obj:b.1", results[1].ID)
	}
	if results[0].RRFScore <= results[1].RRFScore {
		t.Error("first should score higher than second")
	}
}

func TestWeightedRRF_MultiListFusion(t *testing.T) {
	cfg := defaultRRFConfig()
	// Same chunk appearing in two lists should accumulate score.
	lists := map[index.IndexType][]index.ScoredResult{
		index.IndexObjects: {
			{ChunkID: "obj:a.1", Score: 5.0},
			{ChunkID: "obj:b.1", Score: 3.0},
		},
		index.IndexSpecs: {
			{ChunkID: "spec:c.1", Score: 4.0},
			{ChunkID: "obj:a.1", Score: 2.0}, // appears in both
		},
	}

	results := WeightedRRF(lists, cfg)

	// obj:a.1 should rank first because it appears in both lists.
	if results[0].ID != "obj:a.1" {
		t.Errorf("fused first = %q, want obj:a.1", results[0].ID)
	}

	// It should have sources from both indexes.
	if len(results[0].Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(results[0].Sources))
	}
	if results[0].Sources["objects"] != 1 {
		t.Errorf("objects rank = %d, want 1", results[0].Sources["objects"])
	}
	if results[0].Sources["specs"] != 2 {
		t.Errorf("specs rank = %d, want 2", results[0].Sources["specs"])
	}
}

func TestWeightedRRF_IndexWeights(t *testing.T) {
	// Objects weight 1.0, specs weight 0.1 — objects should dominate.
	cfg := project.RRFConfig{
		K: 60,
		IndexWeights: map[string]float64{
			"objects": 1.0,
			"specs":   0.1,
		},
	}

	lists := map[index.IndexType][]index.ScoredResult{
		index.IndexObjects: {
			{ChunkID: "obj:a.1", Score: 5.0},
		},
		index.IndexSpecs: {
			{ChunkID: "spec:b.1", Score: 5.0},
		},
	}

	results := WeightedRRF(lists, cfg)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Objects chunk should score much higher due to weight.
	if results[0].ID != "obj:a.1" {
		t.Errorf("weighted first = %q, want obj:a.1", results[0].ID)
	}

	// Score ratio should be approximately 10:1.
	ratio := results[0].RRFScore / results[1].RRFScore
	if ratio < 5.0 {
		t.Errorf("score ratio = %f, expected > 5.0 (roughly 10:1)", ratio)
	}
}

func TestWeightedRRF_EmptyLists(t *testing.T) {
	cfg := defaultRRFConfig()
	results := WeightedRRF(nil, cfg)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil lists, got %d", len(results))
	}
}

func TestWeightedRRF_StableSorting(t *testing.T) {
	cfg := defaultRRFConfig()
	// Two chunks at same rank in different indexes should have same RRF score.
	// Tie-breaking should be deterministic by ID.
	lists := map[index.IndexType][]index.ScoredResult{
		index.IndexObjects: {
			{ChunkID: "obj:b.1", Score: 5.0},
		},
		index.IndexSpecs: {
			{ChunkID: "obj:a.1", Score: 5.0},
		},
	}

	results := WeightedRRF(lists, cfg)
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// With equal RRF scores (both rank 1 in their respective lists),
	// tie-break by ID: "obj:a.1" < "obj:b.1".
	// But weights differ: objects=1.0, specs=0.7, so obj:b.1 scores higher.
	if results[0].ID != "obj:b.1" {
		t.Errorf("first = %q, want obj:b.1 (higher index weight)", results[0].ID)
	}
}

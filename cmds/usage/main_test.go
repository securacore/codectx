package usage

import (
	"strings"
	"testing"
	"time"

	coreusage "github.com/securacore/codectx/core/usage"
)

func TestFormatSection_EmptyMetrics(t *testing.T) {
	m := coreusage.Metrics{
		TokensByCaller: make(map[string]int),
		TokensByModel:  make(map[string]int),
	}

	got := formatSection("Test Section", m)

	if !strings.Contains(got, "Test Section") {
		t.Error("output should contain section title")
	}
	if !strings.Contains(got, "Total tokens generated") {
		t.Error("output should contain total tokens label")
	}
	if !strings.Contains(got, "Query invocations") {
		t.Error("output should contain query invocations label")
	}
	if !strings.Contains(got, "Generate invocations") {
		t.Error("output should contain generate invocations label")
	}
	// Should NOT contain cache hit rate with zero invocations.
	if strings.Contains(got, "Cache hit rate") {
		t.Error("should not show cache hit rate with zero generate invocations")
	}
	// Should NOT contain breakdowns with empty maps.
	if strings.Contains(got, "By caller") {
		t.Error("should not show caller breakdown with empty map")
	}
}

func TestFormatSection_WithData(t *testing.T) {
	now := time.Now().UnixNano()
	m := coreusage.Metrics{
		TotalTokens:         15000,
		QueryInvocations:    42,
		GenerateInvocations: 10,
		CacheHits:           3,
		TokensByCaller: map[string]int{
			"claude-code": 10000,
			"cursor":      5000,
		},
		TokensByModel: map[string]int{
			"claude-sonnet": 15000,
		},
		FirstSeen:   now - 86400*1e9, // 1 day ago
		LastUpdated: now,
	}

	got := formatSection("Usage Stats", m)

	// Check section header.
	if !strings.Contains(got, "Usage Stats") {
		t.Error("output should contain section title")
	}

	// Check stats.
	if !strings.Contains(got, "15,000") {
		t.Error("output should contain formatted total tokens")
	}
	if !strings.Contains(got, "42") {
		t.Error("output should contain query invocations count")
	}

	// Check cache hit rate.
	if !strings.Contains(got, "Cache hit rate") {
		t.Error("output should contain cache hit rate")
	}
	if !strings.Contains(got, "30.0%") {
		t.Error("output should contain 30.0% cache hit rate")
	}

	// Check breakdowns.
	if !strings.Contains(got, "By caller") {
		t.Error("output should contain caller breakdown")
	}
	if !strings.Contains(got, "claude-code") {
		t.Error("output should contain claude-code caller")
	}
	if !strings.Contains(got, "cursor") {
		t.Error("output should contain cursor caller")
	}
	if !strings.Contains(got, "By model") {
		t.Error("output should contain model breakdown")
	}
	if !strings.Contains(got, "claude-sonnet") {
		t.Error("output should contain claude-sonnet model")
	}

	// Check timestamps.
	if !strings.Contains(got, "Tracking since") {
		t.Error("output should contain tracking since timestamp")
	}
	if !strings.Contains(got, "Last updated") {
		t.Error("output should contain last updated timestamp")
	}
}

func TestFormatSection_LastCompile(t *testing.T) {
	now := time.Now().UnixNano()
	m := coreusage.Metrics{
		TokensByCaller: make(map[string]int),
		TokensByModel:  make(map[string]int),
		FirstSeen:      now,
		LastUpdated:    now,
		LastCompile:    now,
	}

	got := formatSection("Test", m)
	if !strings.Contains(got, "Last compile sync") {
		t.Error("output should contain last compile sync when set")
	}
}

func TestFormatBreakdown_Sorted(t *testing.T) {
	var b strings.Builder
	breakdown := map[string]int{
		"small":  100,
		"large":  900,
		"medium": 500,
	}

	formatBreakdown(&b, breakdown, 1500)
	got := b.String()

	// Entries should be sorted by tokens descending.
	largeIdx := strings.Index(got, "large")
	mediumIdx := strings.Index(got, "medium")
	smallIdx := strings.Index(got, "small")

	if largeIdx == -1 || mediumIdx == -1 || smallIdx == -1 {
		t.Fatalf("missing entries in output: %q", got)
	}

	if largeIdx > mediumIdx || mediumIdx > smallIdx {
		t.Errorf("entries not sorted by tokens descending: large=%d, medium=%d, small=%d",
			largeIdx, mediumIdx, smallIdx)
	}
}

func TestFormatBreakdown_Percentages(t *testing.T) {
	var b strings.Builder
	breakdown := map[string]int{
		"only": 1000,
	}

	formatBreakdown(&b, breakdown, 1000)
	got := b.String()

	if !strings.Contains(got, "100.0%") {
		t.Errorf("expected 100.0%% in output, got %q", got)
	}
}

func TestFormatBreakdown_ZeroTotal(t *testing.T) {
	var b strings.Builder
	breakdown := map[string]int{
		"test": 0,
	}

	formatBreakdown(&b, breakdown, 0)
	got := b.String()

	// Should not panic and should show 0.0%.
	if !strings.Contains(got, "0.0%") {
		t.Errorf("expected 0.0%% for zero total, got %q", got)
	}
}

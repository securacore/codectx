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
	// Empty state — should show concise message, not full zero listing.
	if !strings.Contains(got, "No activity recorded") {
		t.Error("empty metrics should show 'No activity recorded'")
	}
	// Should NOT show detail lines in empty state.
	if strings.Contains(got, "Total tokens generated") {
		t.Error("empty state should not show detail lines")
	}
	if strings.Contains(got, "Query invocations") {
		t.Error("empty state should not show query invocations line")
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
		TotalTokens:      100,
		QueryInvocations: 1,
		TokensByCaller:   make(map[string]int),
		TokensByModel:    make(map[string]int),
		FirstSeen:        now,
		LastUpdated:      now,
		LastCompile:      now,
	}

	got := formatSection("Test", m)
	if !strings.Contains(got, "Last compile sync") {
		t.Error("output should contain last compile sync when set")
	}
}

func TestFormatSection_ZeroTimestamps(t *testing.T) {
	m := coreusage.Metrics{
		TotalTokens:      500,
		QueryInvocations: 5,
		TokensByCaller:   make(map[string]int),
		TokensByModel:    make(map[string]int),
		FirstSeen:        0,
		LastUpdated:      0,
	}

	got := formatSection("Test", m)

	// Should show stats (has activity) but skip zero timestamps.
	if !strings.Contains(got, "500") {
		t.Error("should show token count")
	}
	if strings.Contains(got, "Tracking since") {
		t.Error("should not show Tracking since when FirstSeen is 0")
	}
	if strings.Contains(got, "Last updated") {
		t.Error("should not show Last updated when LastUpdated is 0")
	}
	if strings.Contains(got, "1969") {
		t.Error("should not show Unix epoch date")
	}
}

func TestHasActivity(t *testing.T) {
	tests := []struct {
		name string
		m    coreusage.Metrics
		want bool
	}{
		{"all zero", coreusage.Metrics{}, false},
		{"has tokens", coreusage.Metrics{TotalTokens: 1}, true},
		{"has queries", coreusage.Metrics{QueryInvocations: 1}, true},
		{"has generates", coreusage.Metrics{GenerateInvocations: 1}, true},
		{"cache hits only", coreusage.Metrics{CacheHits: 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasActivity(tt.m); got != tt.want {
				t.Errorf("hasActivity() = %v, want %v", got, tt.want)
			}
		})
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

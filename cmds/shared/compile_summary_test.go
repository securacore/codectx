package shared

import (
	"regexp"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/compile"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func TestRenderCompactCompileSummary_Basic(t *testing.T) {
	t.Parallel()

	result := &compile.Result{
		TotalFiles:    100,
		TotalChunks:   500,
		TotalTokens:   50000,
		TaxonomyTerms: 200,
		TotalSeconds:  3.5,
	}

	output := stripANSI(RenderCompactCompileSummary(result))

	if !strings.Contains(output, "Compilation complete") {
		t.Error("expected 'Compilation complete' header")
	}
	if !strings.Contains(output, "100 files") {
		t.Error("expected file count")
	}
	if !strings.Contains(output, "Compiled") {
		t.Error("expected 'Compiled' label")
	}
	if !strings.Contains(output, "Taxonomy") {
		t.Error("expected 'Taxonomy' label")
	}
	if !strings.Contains(output, "Time") {
		t.Error("expected 'Time' label")
	}
}

func TestRenderCompactCompileSummary_NoTaxonomy(t *testing.T) {
	t.Parallel()

	result := &compile.Result{
		TotalFiles:    10,
		TotalChunks:   50,
		TotalTokens:   5000,
		TaxonomyTerms: 0,
		TotalSeconds:  1.2,
	}

	output := stripANSI(RenderCompactCompileSummary(result))

	if strings.Contains(output, "Taxonomy") {
		t.Error("expected no 'Taxonomy' line when terms = 0")
	}
	if !strings.Contains(output, "Compiled") {
		t.Error("expected 'Compiled' label")
	}
}

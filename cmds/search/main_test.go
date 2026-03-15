package search

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/registry"
)

func TestRenderSearchResults_Empty(t *testing.T) {
	t.Parallel()

	output := renderSearchResults("react", nil, 0)
	if !strings.Contains(output, `"react"`) {
		t.Error("expected query string in output")
	}
	if strings.Contains(output, "Add with") {
		t.Error("should not show install hint for empty results")
	}
}

func TestRenderSearchResults_SingleResult(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{
			Name:          "react-patterns",
			Author:        "community",
			FullName:      "community/codectx-react-patterns",
			Description:   "React component patterns",
			Stars:         342,
			LatestVersion: "2.4.0",
			HasRelease:    true,
		},
	}

	output := renderSearchResults("react", results, 0)

	// Package ref in accent.
	if !strings.Contains(output, "react-patterns@community") {
		t.Error("expected package ref in output")
	}
	// Version with v prefix.
	if !strings.Contains(output, "v2.4.0") {
		t.Error("expected version in output")
	}
	// Star count.
	if !strings.Contains(output, "342") {
		t.Error("expected star count in output")
	}
	// Description.
	if !strings.Contains(output, "React component patterns") {
		t.Error("expected description in output")
	}
	// Repo as KeyValue.
	if !strings.Contains(output, "Repo") {
		t.Error("expected Repo label in output")
	}
	if !strings.Contains(output, "community/codectx-react-patterns") {
		t.Error("expected full repo name in output")
	}
	// Install hint with codectx add.
	if !strings.Contains(output, "Add with") {
		t.Error("expected install hint")
	}
	if !strings.Contains(output, "codectx add react-patterns@community:latest") {
		t.Error("expected add command in output")
	}
	// Summary line.
	if !strings.Contains(output, "Found 1 packages") {
		t.Error("expected summary count in output")
	}
}

func TestRenderSearchResults_MultipleResults(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{
			Name:          "react-patterns",
			Author:        "community",
			FullName:      "community/codectx-react-patterns",
			Stars:         342,
			LatestVersion: "2.4.0",
			HasRelease:    true,
		},
		{
			Name:          "react-testing",
			Author:        "community",
			FullName:      "community/codectx-react-testing",
			Stars:         89,
			LatestVersion: "1.0.0",
			HasRelease:    true,
		},
	}

	output := renderSearchResults("react", results, 0)

	if !strings.Contains(output, "1.") {
		t.Error("expected numbered first result")
	}
	if !strings.Contains(output, "2.") {
		t.Error("expected numbered second result")
	}
}

func TestRenderSearchResults_SummaryCountAllInstallable(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", FullName: "x/codectx-a", LatestVersion: "1.0.0", HasRelease: true},
		{Name: "b", Author: "x", FullName: "x/codectx-b", LatestVersion: "2.0.0", HasRelease: true},
	}

	output := renderSearchResults("test", results, 0)

	// When all are installable, no "(N installable)" qualifier.
	if !strings.Contains(output, "Found 2 packages") {
		t.Error("expected summary count")
	}
	if strings.Contains(output, "installable") {
		t.Error("should not show installable qualifier when all are installable")
	}
}

func TestRenderSearchResults_SummaryCountPartialInstallable(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", FullName: "x/codectx-a", LatestVersion: "1.0.0", HasRelease: true},
		{Name: "b", Author: "x", FullName: "x/codectx-b", LatestVersion: "2.0.0", HasRelease: false},
		{Name: "c", Author: "x", FullName: "x/codectx-c"},
	}

	output := renderSearchResults("test", results, 0)

	if !strings.Contains(output, "Found 3 packages (1 installable)") {
		t.Error("expected summary with installable count")
	}
}

func TestFormatResult_Installable(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:          "test-pkg",
		Author:        "org",
		FullName:      "org/codectx-test-pkg",
		Description:   "A test package",
		Stars:         42,
		LatestVersion: "1.0.0",
		HasRelease:    true,
	}

	output := formatResult(1, r)

	// Package ref.
	if !strings.Contains(output, "test-pkg@org") {
		t.Error("expected package ref")
	}
	// Bold version.
	if !strings.Contains(output, "v1.0.0") {
		t.Error("expected version")
	}
	// Repo KeyValue.
	if !strings.Contains(output, "Repo") {
		t.Error("expected Repo label")
	}
	if !strings.Contains(output, "org/codectx-test-pkg") {
		t.Error("expected full repo name")
	}
	// Stars.
	if !strings.Contains(output, "42") {
		t.Error("expected star count")
	}
	// Description.
	if !strings.Contains(output, "A test package") {
		t.Error("expected description")
	}
	// No warnings.
	if strings.Contains(output, "no release archive") {
		t.Error("should not show warning for installable package")
	}
	if strings.Contains(output, "no version tags") {
		t.Error("should not show no-tags warning for package with version")
	}
}

func TestFormatResult_NoReleaseArchive(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:          "test-pkg",
		Author:        "org",
		FullName:      "org/codectx-test-pkg",
		LatestVersion: "1.0.0",
		HasRelease:    false,
	}

	output := formatResult(1, r)

	if !strings.Contains(output, "no release archive") {
		t.Error("expected 'no release archive' warning")
	}
	// Should still show version.
	if !strings.Contains(output, "v1.0.0") {
		t.Error("expected version even with no release")
	}
}

func TestFormatResult_NoVersionTags(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:     "test-pkg",
		Author:   "org",
		FullName: "org/codectx-test-pkg",
	}

	output := formatResult(1, r)

	if !strings.Contains(output, "no version tags") {
		t.Error("expected 'no version tags' warning")
	}
	// Should NOT show "v" prefix with no version.
	if strings.Contains(output, " v ") || strings.Contains(output, " v\n") {
		t.Error("should not show bare 'v' prefix without a version")
	}
}

func TestFormatResult_NoStars(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:          "test-pkg",
		Author:        "org",
		FullName:      "org/codectx-test-pkg",
		LatestVersion: "1.0.0",
		HasRelease:    true,
		Stars:         0,
	}

	output := formatResult(1, r)

	// Should not show star indicator when stars == 0.
	if strings.Contains(output, "(* 0)") {
		t.Error("should not show star count when zero")
	}
}

func TestFormatResult_NoDescription(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:          "test-pkg",
		Author:        "org",
		FullName:      "org/codectx-test-pkg",
		LatestVersion: "1.0.0",
		HasRelease:    true,
	}

	output := formatResult(1, r)

	// Repo line should still be present.
	if !strings.Contains(output, "org/codectx-test-pkg") {
		t.Error("expected full name in output")
	}
}

func TestSortResults_InstallableFirst(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "no-tags", Author: "a"},
		{Name: "no-release", Author: "b", LatestVersion: "1.0.0", HasRelease: false},
		{Name: "installable", Author: "c", LatestVersion: "2.0.0", HasRelease: true},
	}

	sortResults(results)

	if results[0].Name != "installable" {
		t.Errorf("expected installable first, got %s", results[0].Name)
	}
	if results[1].Name != "no-release" {
		t.Errorf("expected no-release second, got %s", results[1].Name)
	}
	if results[2].Name != "no-tags" {
		t.Errorf("expected no-tags last, got %s", results[2].Name)
	}
}

func TestSortResults_PreservesStarOrder(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", LatestVersion: "1.0.0", HasRelease: true, Stars: 100},
		{Name: "b", Author: "x", LatestVersion: "2.0.0", HasRelease: true, Stars: 50},
		{Name: "c", Author: "x", LatestVersion: "3.0.0", HasRelease: true, Stars: 10},
	}

	sortResults(results)

	// All same priority — stable sort preserves original star order.
	if results[0].Name != "a" || results[1].Name != "b" || results[2].Name != "c" {
		t.Error("stable sort should preserve original star order within same priority")
	}
}

func TestFirstInstallable_ReturnsInstallable(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "no-release", Author: "a", LatestVersion: "1.0.0", HasRelease: false},
		{Name: "installable", Author: "b", LatestVersion: "2.0.0", HasRelease: true},
	}

	got := firstInstallable(results)
	if got.Name != "installable" {
		t.Errorf("expected installable, got %s", got.Name)
	}
}

func TestFirstInstallable_FallsBackToFirst(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "no-release", Author: "a", LatestVersion: "1.0.0", HasRelease: false},
		{Name: "no-tags", Author: "b"},
	}

	got := firstInstallable(results)
	if got.Name != "no-release" {
		t.Errorf("expected first result as fallback, got %s", got.Name)
	}
}

func TestRenderSearchResults_InstallHintUsesInstallableResult(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "no-release", Author: "a", FullName: "a/codectx-no-release", LatestVersion: "1.0.0", HasRelease: false},
		{Name: "installable", Author: "b", FullName: "b/codectx-installable", LatestVersion: "2.0.0", HasRelease: true},
	}

	output := renderSearchResults("test", results, 0)
	if !strings.Contains(output, "codectx add installable@b:latest") {
		t.Error("expected install hint to use the installable result")
	}
}

func TestCountInstallable(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", HasRelease: true},
		{Name: "b", HasRelease: false},
		{Name: "c", HasRelease: true},
		{Name: "d"},
	}

	got := countInstallable(results)
	if got != 2 {
		t.Errorf("expected 2 installable, got %d", got)
	}
}

func TestFormatSummaryLine_AllInstallable(t *testing.T) {
	t.Parallel()

	line := formatSummaryLine(3, 3)
	if !strings.Contains(line, "Found 3 packages") {
		t.Error("expected package count")
	}
	if strings.Contains(line, "installable") {
		t.Error("should not show installable when all are installable")
	}
}

func TestFormatSummaryLine_PartialInstallable(t *testing.T) {
	t.Parallel()

	line := formatSummaryLine(5, 2)
	if !strings.Contains(line, "Found 5 packages (2 installable)") {
		t.Errorf("expected summary with installable count, got: %s", line)
	}
}

func TestFilterInstallable(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", HasRelease: true},
		{Name: "b", Author: "x", HasRelease: false},
		{Name: "c", Author: "x", HasRelease: true},
	}

	filtered, hidden := shared.FilterInstallable(results)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 installable, got %d", len(filtered))
	}
	if hidden != 1 {
		t.Errorf("expected 1 hidden, got %d", hidden)
	}
}

func TestFilterInstallable_AllHidden(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", HasRelease: false},
		{Name: "b", Author: "x", HasRelease: false},
	}

	filtered, hidden := shared.FilterInstallable(results)
	if len(filtered) != 0 {
		t.Fatalf("expected 0 installable, got %d", len(filtered))
	}
	if hidden != 2 {
		t.Errorf("expected 2 hidden, got %d", hidden)
	}
}

func TestRenderSearchResults_HiddenCount(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", FullName: "x/codectx-a", LatestVersion: "1.0.0", HasRelease: true},
	}

	output := renderSearchResults("test", results, 3)
	if !strings.Contains(output, "3 packages hidden") {
		t.Error("expected hidden count note in output")
	}
	if !strings.Contains(output, "--show-uninstallable") {
		t.Error("expected --show-uninstallable flag hint")
	}
}

func TestRenderSearchResults_NoHiddenCount(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", FullName: "x/codectx-a", LatestVersion: "1.0.0", HasRelease: true},
	}

	output := renderSearchResults("test", results, 0)
	if strings.Contains(output, "hidden") {
		t.Error("should not show hidden note when count is 0")
	}
}

func TestRenderSearchResults_HiddenCountSingular(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{Name: "a", Author: "x", FullName: "x/codectx-a", LatestVersion: "1.0.0", HasRelease: true},
	}

	output := renderSearchResults("test", results, 1)
	if !strings.Contains(output, "1 package hidden") {
		t.Error("expected singular 'package' for count of 1")
	}
}

func TestResultPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   registry.SearchResult
		expected int
	}{
		{"installable", registry.SearchResult{LatestVersion: "1.0.0", HasRelease: true}, 0},
		{"no release", registry.SearchResult{LatestVersion: "1.0.0", HasRelease: false}, 1},
		{"no version", registry.SearchResult{}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resultPriority(tt.result)
			if got != tt.expected {
				t.Errorf("resultPriority() = %d, want %d", got, tt.expected)
			}
		})
	}
}

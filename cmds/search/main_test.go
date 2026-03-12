package search

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/registry"
)

func TestRenderSearchResults_Empty(t *testing.T) {
	t.Parallel()

	output := renderSearchResults("react", nil)
	if !strings.Contains(output, `"react"`) {
		t.Error("expected query string in output")
	}
	// No install hint for empty results.
	if strings.Contains(output, "Install with") {
		t.Error("should not show install hint for empty results")
	}
}

func TestRenderSearchResults_SingleResult(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{
			Name:          "react-patterns",
			Org:           "community",
			FullName:      "community/codectx-react-patterns",
			Description:   "React component patterns",
			Stars:         342,
			LatestVersion: "2.4.0",
		},
	}

	output := renderSearchResults("react", results)

	if !strings.Contains(output, "react-patterns@community") {
		t.Error("expected package name in output")
	}
	if !strings.Contains(output, "v2.4.0") {
		t.Error("expected version in output")
	}
	if !strings.Contains(output, "342") {
		t.Error("expected star count in output")
	}
	if !strings.Contains(output, "React component patterns") {
		t.Error("expected description in output")
	}
	if !strings.Contains(output, "Install with") {
		t.Error("expected install hint")
	}
	if !strings.Contains(output, "codectx install react-patterns@community:latest") {
		t.Error("expected install command in output")
	}
}

func TestRenderSearchResults_MultipleResults(t *testing.T) {
	t.Parallel()

	results := []registry.SearchResult{
		{
			Name:          "react-patterns",
			Org:           "community",
			FullName:      "community/codectx-react-patterns",
			Stars:         342,
			LatestVersion: "2.4.0",
		},
		{
			Name:          "react-testing",
			Org:           "community",
			FullName:      "community/codectx-react-testing",
			Stars:         89,
			LatestVersion: "1.0.0",
		},
	}

	output := renderSearchResults("react", results)

	if !strings.Contains(output, "1.") {
		t.Error("expected numbered first result")
	}
	if !strings.Contains(output, "2.") {
		t.Error("expected numbered second result")
	}
}

func TestFormatResult_NoVersion(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:     "test-pkg",
		Org:      "org",
		FullName: "org/codectx-test-pkg",
	}

	output := formatResult(1, r)
	if !strings.Contains(output, "no tags") {
		t.Error("expected 'no tags' for empty version")
	}
}

func TestFormatResult_NoStars(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:          "test-pkg",
		Org:           "org",
		FullName:      "org/codectx-test-pkg",
		LatestVersion: "1.0.0",
		Stars:         0,
	}

	output := formatResult(1, r)
	// Should not contain the star indicator when stars == 0.
	if strings.Contains(output, "* 0") {
		t.Error("should not show star count when zero")
	}
}

func TestFormatResult_NoDescription(t *testing.T) {
	t.Parallel()

	r := registry.SearchResult{
		Name:          "test-pkg",
		Org:           "org",
		FullName:      "org/codectx-test-pkg",
		LatestVersion: "1.0.0",
	}

	output := formatResult(1, r)

	// Full name line should be present.
	if !strings.Contains(output, "org/codectx-test-pkg") {
		t.Error("expected full name in output")
	}
}

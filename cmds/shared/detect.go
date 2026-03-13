package shared

import (
	"github.com/securacore/codectx/core/detect"
	"github.com/securacore/codectx/core/project"
)

// DetectProviderCapabilities checks the detection results for Claude CLI
// binary and Anthropic API key availability.
func DetectProviderCapabilities(detection detect.Result) (hasCLI, hasAPI bool) {
	for _, t := range detection.Tools {
		if t.Binary == "claude" {
			hasCLI = true
			break
		}
	}
	for _, p := range detection.Providers {
		if p.Name == "Anthropic" {
			hasAPI = true
			break
		}
	}
	return hasCLI, hasAPI
}

// AutoSelectProvider returns the appropriate provider string based on
// detected capabilities. When both CLI and API are available, CLI is
// preferred (non-interactive default). Returns empty string if neither
// is available.
func AutoSelectProvider(hasCLI, hasAPI bool) string {
	switch {
	case hasCLI:
		return project.ProviderCLI
	case hasAPI:
		return project.ProviderAPI
	default:
		return ""
	}
}

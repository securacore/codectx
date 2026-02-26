// Package ai provides AI tool detection and integration for codectx.
// It discovers locally installed AI tools (Claude Code, opencode, Ollama)
// and manages the provider abstraction for communicating with them.
package ai

// Provider describes an AI tool that codectx can detect and integrate with.
type Provider struct {
	// ID is the configuration identifier (e.g., "claude", "opencode", "ollama").
	ID string

	// Name is the human-readable display name (e.g., "Claude Code").
	Name string

	// Binary is the executable name expected on PATH.
	Binary string
}

// DetectionResult holds the outcome of checking a single provider's availability.
type DetectionResult struct {
	Provider Provider

	// Path is the resolved binary path. Empty when the binary is not found.
	Path string

	// Found reports whether the binary exists on PATH.
	Found bool
}

// Providers is the default registry of supported AI providers.
var Providers = []Provider{
	{ID: "claude", Name: "Claude Code", Binary: "claude"},
	{ID: "opencode", Name: "opencode", Binary: "opencode"},
	{ID: "ollama", Name: "Ollama", Binary: "ollama"},
}

// ProviderByID returns the provider with the given ID and true,
// or a zero Provider and false if no match is found.
func ProviderByID(id string) (Provider, bool) {
	for _, p := range Providers {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

package llm

import (
	"context"
	"os/exec"

	"github.com/securacore/codectx/core/project"
)

// Sender sends structured requests to an LLM and returns typed responses.
// Implemented by both the API provider (apiSender) and the CLI provider
// (cliSender).
type Sender interface {
	// SendAliases sends a batch alias generation request and returns
	// structured aliases. The system parameter is the taxonomy-generation
	// instruction content; prompt is the formatted batch of terms.
	SendAliases(ctx context.Context, system, prompt string) (*AliasResponse, error)

	// SendBridges sends a batch bridge generation request and returns
	// structured bridge summaries. The system parameter is the bridge-summaries
	// instruction content; prompt is the formatted batch of chunk pairs.
	SendBridges(ctx context.Context, system, prompt string) (*BridgeResponse, error)
}

// LookPathFunc is the function used to locate binaries. Defaults to exec.LookPath.
// Override in tests to control CLI detection.
var LookPathFunc = exec.LookPath

// NewSender creates the appropriate Sender based on provider configuration.
//
// For provider "api": requires a non-empty apiKey. Uses the Anthropic SDK.
// For provider "cli": requires the claude binary on PATH. Uses headless CLI.
// For empty provider: auto-detects — tries API first (if key present), then
// CLI (if binary found).
//
// Returns (nil, nil) if no provider is available. This is not an error;
// it signals that LLM augmentation should be skipped gracefully.
func NewSender(provider, apiKey, model, claudeBinary string) (Sender, error) {
	switch provider {
	case project.ProviderAPI:
		if apiKey == "" {
			return nil, nil
		}
		return newAPISender(apiKey, model)

	case project.ProviderCLI:
		path, err := LookPathFunc(claudeBinary)
		if err != nil {
			return nil, nil //nolint:nilerr // binary not found = skip
		}
		return newCLISender(path, model), nil

	default:
		// Auto-detect: try API first, then CLI.
		if apiKey != "" {
			return newAPISender(apiKey, model)
		}
		path, err := LookPathFunc(claudeBinary)
		if err != nil {
			return nil, nil //nolint:nilerr // nothing available
		}
		return newCLISender(path, model), nil
	}
}

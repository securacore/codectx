package shared

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/ai"
)

// ValidateAIProvider checks that the given provider ID is known and that
// its binary is available on PATH. Returns nil on success. This is the
// single validation point for all commands that accept an AI provider ID
// (codectx set, codectx ai setup, future AI commands).
func ValidateAIProvider(id string) error {
	provider, ok := ai.ProviderByID(id)
	if !ok {
		known := make([]string, len(ai.Providers))
		for i, p := range ai.Providers {
			known[i] = p.ID
		}
		return fmt.Errorf("unknown AI provider %q — known providers: %s", id, strings.Join(known, ", "))
	}

	result := ai.DetectProvider(provider)
	if !result.Found {
		return fmt.Errorf("AI provider %q: binary %q not found on PATH", id, provider.Binary)
	}

	return nil
}

// ValidateAIClass checks that the given model class ID is in the known
// registry. Returns nil on success. Unlike provider validation, class
// validation does not check for binaries — classes are documentation
// targets, not executable tools.
func ValidateAIClass(id string) error {
	if _, ok := ai.ClassByID(id); ok {
		return nil
	}

	known := make([]string, len(ai.Classes))
	for i, c := range ai.Classes {
		known[i] = c.ID
	}
	return fmt.Errorf("unknown model class %q — known classes: %s", id, strings.Join(known, ", "))
}

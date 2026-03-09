package ai

import "os/exec"

// Detect checks all registered providers for availability and returns
// the results. Each result indicates whether the provider's binary
// was found on PATH and, if so, its resolved path.
func Detect() []DetectionResult {
	results := make([]DetectionResult, 0, len(Providers))
	for _, p := range Providers {
		results = append(results, DetectProvider(p))
	}
	return results
}

// DetectProvider checks a single provider for availability by looking
// up its binary on PATH using exec.LookPath.
func DetectProvider(p Provider) DetectionResult {
	path, err := exec.LookPath(p.Binary)
	return DetectionResult{
		Provider: p,
		Path:     path,
		Found:    err == nil,
	}
}

// Found filters a slice of DetectionResults to only those that were found.
func Found(results []DetectionResult) []DetectionResult {
	var found []DetectionResult
	for _, r := range results {
		if r.Found {
			found = append(found, r)
		}
	}
	return found
}

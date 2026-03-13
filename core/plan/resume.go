package plan

import (
	"fmt"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/query"
)

// ResumeResult holds the result of a plan resume operation.
type ResumeResult struct {
	// Plan is the loaded plan.
	Plan *Plan

	// Check is the dependency hash check result.
	Check *CheckResult

	// GenerateResults contains the results from replaying chunks.
	// Only populated when all hashes match (instant replay).
	GenerateResults []*query.GenerateResult

	// Output is the formatted output string ready for display.
	Output string
}

// Resume performs the plan resume operation:
//  1. Load plan.yaml, identify current_step
//  2. Check each dependency's current content hash against stored hash
//  3. If all hashes match: replay current step's chunks via generate
//  4. If any hashes changed: report drift with stored queries
//
// The compiledDir and encoding parameters are needed for chunk replay via
// RunGenerate. If replay is not needed (drift detected), these can be empty.
func Resume(planPath, compiledDir, encoding string) (*ResumeResult, error) {
	// Load the plan.
	p, err := Load(planPath)
	if err != nil {
		return nil, fmt.Errorf("loading plan: %w", err)
	}

	// Validate current step exists.
	current := p.CurrentStepEntry()
	if current == nil {
		return &ResumeResult{
			Plan:   p,
			Output: FormatStatus(p, nil),
		}, nil
	}

	// Load compiled hashes for dependency checking.
	hashesPath := manifest.HashesPath(compiledDir)
	hashes, err := manifest.LoadHashes(hashesPath)
	if err != nil {
		return nil, fmt.Errorf("loading compiled hashes: %w", err)
	}

	// Check dependencies.
	check := CheckDependencies(p.Dependencies, hashes)

	result := &ResumeResult{
		Plan:  p,
		Check: check,
	}

	if check.AllMatch {
		// All hashes match — replay chunks for instant context reconstruction.
		generateOutputs, genResults, replayErr := replayChunks(current, compiledDir, encoding)
		if replayErr != nil {
			return nil, fmt.Errorf("replaying chunks: %w", replayErr)
		}
		result.GenerateResults = genResults
		result.Output = FormatResumeMatch(p, generateOutputs)
	} else {
		// Hashes changed — report drift with stored queries.
		result.Output = FormatResumeDrift(p, check)
	}

	return result, nil
}

// replayChunks replays the current step's stored chunk lists via RunGenerate.
// Each entry in step.Chunks is a comma-delimited string of chunk IDs
// representing one generate call.
func replayChunks(step *Step, compiledDir, encoding string) ([]string, []*query.GenerateResult, error) {
	if len(step.Chunks) == 0 {
		return nil, nil, nil
	}

	var outputs []string
	var results []*query.GenerateResult

	for _, chunkStr := range step.Chunks {
		chunkIDs := query.ParseChunkIDs(chunkStr)
		if len(chunkIDs) == 0 {
			continue
		}

		genResult, err := query.RunGenerate(compiledDir, encoding, chunkIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("generating chunks %q: %w", chunkStr, err)
		}

		results = append(results, genResult)
		outputs = append(outputs, query.FormatGenerateSummary(genResult, "", "", false))
	}

	return outputs, results, nil
}

package compile

import (
	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/taxonomy"
)

// chunksForSources returns the chunk IDs from the manifest that belong to
// the given source file paths. Used to identify chunks that need to be
// removed during incremental compilation (deleted or modified sources).
func chunksForSources(mfst *manifest.Manifest, sourcePaths []string) []string {
	pathSet := make(map[string]bool, len(sourcePaths))
	for _, p := range sourcePaths {
		pathSet[p] = true
	}

	var ids []string

	collectFrom := func(entries map[string]*manifest.ManifestEntry) {
		for id, entry := range entries {
			if pathSet[entry.Source] {
				ids = append(ids, id)
			}
		}
	}

	collectFrom(mfst.Objects)
	collectFrom(mfst.Specs)
	collectFrom(mfst.System)

	return ids
}

// chunksFromManifest extracts LoadedChunk metadata from the manifest for
// the given source file paths. Used to reconstruct Chunk structs from disk
// in incremental mode.
func chunksFromManifest(mfst *manifest.Manifest, sourcePaths []string) []chunk.LoadedChunk {
	pathSet := make(map[string]bool, len(sourcePaths))
	for _, p := range sourcePaths {
		pathSet[p] = true
	}

	var loaded []chunk.LoadedChunk

	extractFrom := func(entries map[string]*manifest.ManifestEntry) {
		for id, entry := range entries {
			if !pathSet[entry.Source] {
				continue
			}
			loaded = append(loaded, chunk.LoadedChunk{
				ID:          id,
				Type:        chunk.ChunkType(entry.Type),
				Source:      entry.Source,
				Heading:     entry.Heading,
				Sequence:    entry.Sequence,
				TotalInFile: entry.TotalInFile,
				Tokens:      entry.Tokens,
			})
		}
	}

	extractFrom(mfst.Objects)
	extractFrom(mfst.Specs)
	extractFrom(mfst.System)

	return loaded
}

// preserveUnchangedAliases copies aliases from the previous taxonomy to
// the current taxonomy for terms whose source chunks are all unchanged.
// This avoids re-extracting aliases for stable terms during incremental builds.
//
// A term's aliases are preserved if ALL chunks that contributed to the term
// are in the unchangedChunkIDs set.
func preserveUnchangedAliases(
	current *taxonomy.Taxonomy,
	previous *taxonomy.Taxonomy,
	chunkTerms map[string][]string,
	unchangedChunkIDs map[string]bool,
) {
	if current == nil || previous == nil {
		return
	}

	// Build reverse map: term key -> contributing chunk IDs.
	termChunks := make(map[string][]string)
	for chunkID, terms := range chunkTerms {
		for _, termKey := range terms {
			termChunks[termKey] = append(termChunks[termKey], chunkID)
		}
	}

	for key, term := range current.Terms {
		// Skip if term already has aliases (from current extraction).
		if len(term.Aliases) > 0 {
			continue
		}

		// Check if the previous taxonomy had aliases for this term.
		prevTerm, ok := previous.Terms[key]
		if !ok || len(prevTerm.Aliases) == 0 {
			continue
		}

		// Check if all contributing chunks are unchanged.
		chunks := termChunks[key]
		if len(chunks) == 0 {
			// Term exists but no chunk mapping — preserve anyway since
			// it may come from a structural position that's stable.
			term.Aliases = prevTerm.Aliases
			continue
		}

		allUnchanged := true
		for _, chunkID := range chunks {
			if !unchangedChunkIDs[chunkID] {
				allUnchanged = false
				break
			}
		}

		if allUnchanged {
			term.Aliases = prevTerm.Aliases
		}
	}
}

// preserveUnchangedBridges copies bridge summaries from the previous manifest
// to the current manifest for chunk pairs where both the "from" and "to"
// chunks are unchanged.
func preserveUnchangedBridges(
	current *manifest.Manifest,
	previous *manifest.Manifest,
	unchangedChunkIDs map[string]bool,
) {
	copyBridges := func(currEntries, prevEntries map[string]*manifest.ManifestEntry) {
		for id, entry := range currEntries {
			// Skip if already has a bridge (from current compilation pass).
			if entry.BridgeToNext != nil {
				continue
			}

			// Skip if not unchanged.
			if !unchangedChunkIDs[id] {
				continue
			}

			// Check if the next chunk is also unchanged.
			if entry.Adjacent == nil || entry.Adjacent.Next == nil {
				continue
			}
			nextID := *entry.Adjacent.Next
			if !unchangedChunkIDs[nextID] {
				continue
			}

			// Copy bridge from previous manifest.
			prevEntry, ok := prevEntries[id]
			if !ok || prevEntry.BridgeToNext == nil {
				continue
			}

			entry.BridgeToNext = prevEntry.BridgeToNext
		}
	}

	copyBridges(current.Objects, previous.Objects)
	copyBridges(current.System, previous.System)
}

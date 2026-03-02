package compile

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/internal/util"
)

// ConflictEntry records a single deduplication or conflict event.
type ConflictEntry struct {
	Section    string // "foundation", "topics", "prompts", "plans"
	ID         string
	WinnerPkg  string // "local" or "name@author"
	SkippedPkg string
	Reason     string // "duplicate" or "conflict"
}

// seenEntry tracks the origin and file hash for a previously merged entry.
type seenEntry struct {
	pkg  string // "local" or "name@author"
	hash string // SHA256 hex of file content, empty if file missing
}

// DeduplicationReport summarizes all dedup and conflict events from a compile.
type DeduplicationReport struct {
	Duplicates []ConflictEntry
	Conflicts  []ConflictEntry
}

// HasConflicts returns true if any entries had different content for the same ID.
func (r *DeduplicationReport) HasConflicts() bool {
	return len(r.Conflicts) > 0
}

// Total returns the total number of dedup + conflict events.
func (r *DeduplicationReport) Total() int {
	return len(r.Duplicates) + len(r.Conflicts)
}

// mergeManifestDedup merges src entries into dst with deduplication.
// It checks each entry's ID against the seen map. If an ID was already merged:
//   - Same file content (SHA256): skip silently (dedup), record as duplicate.
//   - Different file content: skip (precedence wins), record as conflict.
//
// srcRoot is the filesystem root for the source package (to read file content).
// dstRoot is the filesystem root for already-merged entries.
// srcPkg is the display name of the source package (e.g., "react@org").
// seen maps "section:id" to seenEntry and is mutated in place.
func mergeManifestDedup(
	dst *manifest.Manifest,
	src *manifest.Manifest,
	dstRoot, srcRoot, srcPkg string,
	seen map[string]seenEntry,
) []ConflictEntry {
	var events []ConflictEntry

	for _, e := range src.Foundation {
		key := "foundation:" + e.ID
		if ev, skip := checkDedup(key, e.Path, "foundation", srcRoot, dstRoot, srcPkg, seen); skip {
			events = append(events, ev)
			continue
		}
		dst.Foundation = append(dst.Foundation, e)
		seen[key] = seenEntry{pkg: srcPkg, hash: fileHash(filepath.Join(srcRoot, e.Path))}
	}

	for _, e := range src.Application {
		key := "application:" + e.ID
		if ev, skip := checkDedup(key, e.Path, "application", srcRoot, dstRoot, srcPkg, seen); skip {
			events = append(events, ev)
			continue
		}
		dst.Application = append(dst.Application, e)
		seen[key] = seenEntry{pkg: srcPkg, hash: fileHash(filepath.Join(srcRoot, e.Path))}
	}

	for _, e := range src.Topics {
		key := "topics:" + e.ID
		if ev, skip := checkDedup(key, e.Path, "topics", srcRoot, dstRoot, srcPkg, seen); skip {
			events = append(events, ev)
			continue
		}
		dst.Topics = append(dst.Topics, e)
		seen[key] = seenEntry{pkg: srcPkg, hash: fileHash(filepath.Join(srcRoot, e.Path))}
	}

	for _, e := range src.Prompts {
		key := "prompts:" + e.ID
		if ev, skip := checkDedup(key, e.Path, "prompts", srcRoot, dstRoot, srcPkg, seen); skip {
			events = append(events, ev)
			continue
		}
		dst.Prompts = append(dst.Prompts, e)
		seen[key] = seenEntry{pkg: srcPkg, hash: fileHash(filepath.Join(srcRoot, e.Path))}
	}

	for _, e := range src.Plans {
		key := "plans:" + e.ID
		if ev, skip := checkDedup(key, e.Path, "plans", srcRoot, dstRoot, srcPkg, seen); skip {
			events = append(events, ev)
			continue
		}
		dst.Plans = append(dst.Plans, e)
		seen[key] = seenEntry{pkg: srcPkg, hash: fileHash(filepath.Join(srcRoot, e.Path))}
	}

	return events
}

// checkDedup checks whether an entry should be skipped due to deduplication.
// Returns (event, true) if the entry should be skipped, (zero, false) if it should be included.
func checkDedup(
	key, path, section, srcRoot, dstRoot, srcPkg string,
	seen map[string]seenEntry,
) (ConflictEntry, bool) {
	existing, found := seen[key]
	if !found {
		return ConflictEntry{}, false
	}

	srcHash := fileHash(filepath.Join(srcRoot, path))

	if srcHash == existing.hash && srcHash != "" {
		// Same content: silent dedup.
		return ConflictEntry{
			Section:    section,
			ID:         util.KeyID(key),
			WinnerPkg:  existing.pkg,
			SkippedPkg: srcPkg,
			Reason:     "duplicate",
		}, true
	}

	// Different content: precedence wins.
	return ConflictEntry{
		Section:    section,
		ID:         util.KeyID(key),
		WinnerPkg:  existing.pkg,
		SkippedPkg: srcPkg,
		Reason:     "conflict",
	}, true
}

// fileHash returns the hex SHA256 of a file, or empty string if the file
// cannot be read (e.g., does not exist).
func fileHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// collectActiveIDs returns a map of "section:id" for all entries in a manifest.
// Used for early collision detection in the add command.
func CollectActiveIDs(m *manifest.Manifest) map[string]bool {
	ids := make(map[string]bool)
	for _, e := range m.Foundation {
		ids["foundation:"+e.ID] = true
	}
	for _, e := range m.Application {
		ids["application:"+e.ID] = true
	}
	for _, e := range m.Topics {
		ids["topics:"+e.ID] = true
	}
	for _, e := range m.Prompts {
		ids["prompts:"+e.ID] = true
	}
	for _, e := range m.Plans {
		ids["plans:"+e.ID] = true
	}
	return ids
}

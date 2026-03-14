package compile

import (
	"github.com/securacore/codectx/core/manifest"
)

// FileChange classifies a source file's change status relative to the
// previous compilation.
type FileChange int

const (
	// FileNew indicates the file was not present in the previous hashes.
	FileNew FileChange = iota

	// FileModified indicates the file's content hash differs from the previous compilation.
	FileModified

	// FileUnchanged indicates the file's content hash is identical to the previous compilation.
	FileUnchanged
)

// ChangeSet holds the results of comparing current source files against
// previous compilation hashes.
type ChangeSet struct {
	// Status maps each relative source path to its change classification.
	Status map[string]FileChange

	// Deleted lists source paths that existed in the previous hashes
	// but were not found among the current source files.
	Deleted []string

	// Counts for quick access.
	NewCount       int
	ModifiedCount  int
	UnchangedCount int
}

// HasChanges reports whether any files are new, modified, or deleted.
func (cs *ChangeSet) HasChanges() bool {
	return cs.NewCount > 0 || cs.ModifiedCount > 0 || len(cs.Deleted) > 0
}

// ClassifyFiles compares current source files and their hashes against
// the previous compilation's hashes. Each file is classified as new,
// modified, or unchanged.
//
// The currentHashes parameter maps relative source paths to their current
// "sha256:<hex>" content hashes. The previousHashes parameter is the Hashes
// loaded from the previous compilation (may be nil for first compile).
func ClassifyFiles(currentHashes map[string]string, previousHashes *manifest.Hashes) *ChangeSet {
	cs := &ChangeSet{
		Status: make(map[string]FileChange, len(currentHashes)),
	}

	// If no previous hashes, everything is new.
	if previousHashes == nil || len(previousHashes.Files) == 0 {
		for path := range currentHashes {
			cs.Status[path] = FileNew
			cs.NewCount++
		}
		return cs
	}

	// Classify each current file.
	for path, currentHash := range currentHashes {
		prevHash, existed := previousHashes.Files[path]
		switch {
		case !existed:
			cs.Status[path] = FileNew
			cs.NewCount++
		case currentHash != prevHash:
			cs.Status[path] = FileModified
			cs.ModifiedCount++
		default:
			cs.Status[path] = FileUnchanged
			cs.UnchangedCount++
		}
	}

	// Find deleted files (in previous but not in current).
	for path := range previousHashes.Files {
		if _, exists := currentHashes[path]; !exists {
			cs.Deleted = append(cs.Deleted, path)
		}
	}

	return cs
}

// InstructionChanges tracks which system instruction directories changed
// between compilations.
type InstructionChanges struct {
	ContextAssembly bool
}

// AnyChanged reports whether any system instruction directory changed.
func (ic *InstructionChanges) AnyChanged() bool {
	return ic.ContextAssembly
}

// DetectInstructionChanges compares current system directory hashes against
// the previous compilation's system hashes. Returns which instruction
// directories have changed.
//
// A directory is considered changed if:
//   - It exists now but didn't exist before (or vice versa)
//   - Its hash differs from the previous value
func DetectInstructionChanges(currentSystem, previousSystem map[string]string) *InstructionChanges {
	ic := &InstructionChanges{}

	check := func(name string) bool {
		curr, currOK := currentSystem[name]
		prev, prevOK := previousSystem[name]

		// Both missing — no change.
		if !currOK && !prevOK {
			return false
		}
		// One missing, the other present — changed.
		if currOK != prevOK {
			return true
		}
		// Both present — compare hashes.
		return curr != prev
	}

	ic.ContextAssembly = check("context-assembly")

	return ic
}

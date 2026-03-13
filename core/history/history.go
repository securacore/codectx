// Package history implements the structured history and generate cache system
// for codectx.
//
// History is stored under docs/.codectx/history/ with three components:
//   - queries/[nanoTs].[queryHash12].json — one per codectx query invocation
//   - chunks/[nanoTs].[chunkSetHash12].json — one per codectx generate invocation
//   - docs/[nanoTs].[contentHash12].md — one per codectx generate invocation
//
// All timestamps use nanosecond precision. Filenames use timestamp-first
// ordering so lexicographic sort equals chronological order. The history
// directory is lazily created on first use by EnsureDir, which also ensures
// the project .gitignore includes the history entry. History is capped at
// 100 MB; when exceeded, CheckAndPrune retains only the most recent files.
//
// The generate cache uses chunk_set_hash + compile_hash as a composite key.
// Cache lookups glob the chunks/ directory, verify full hashes and document
// existence, and return the cached document path on hit.
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
)

const (
	// QueriesDir is the subdirectory for query history entries.
	QueriesDir = "queries"

	// ChunksDir is the subdirectory for generate/chunks history entries.
	ChunksDir = "chunks"

	// DocsDir is the subdirectory for generated document snapshots.
	DocsDir = "docs"

	// MaxSize is the maximum total size of the history directory (100 MB).
	MaxSize = 100 * 1024 * 1024

	// PruneKeep is the number of files to retain per subdirectory after pruning.
	PruneKeep = 5

	// ShortHashLen is the number of hex characters from SHA-256 used in
	// filenames and display identifiers.
	ShortHashLen = 12

	// HistorySubdir is the subdirectory name under .codectx/.
	HistorySubdir = "history"

	// hashPrefix is prepended to all hash values (e.g. "sha256:abc123...").
	hashPrefix = "sha256:"
)

// QueryEntry represents a single query invocation record.
type QueryEntry struct {
	Ts          int64  `json:"ts"`
	QueryHash   string `json:"query_hash"`
	Raw         string `json:"raw"`
	Expanded    string `json:"expanded"`
	ResultCount int    `json:"result_count"`
	CompileHash string `json:"compile_hash"`
	Caller      string `json:"caller"`
	SessionID   string `json:"session_id"`
	Model       string `json:"model"`
}

// ChunksEntry represents a single generate invocation record.
type ChunksEntry struct {
	Ts           int64    `json:"ts"`
	ChunkSetHash string   `json:"chunk_set_hash"`
	Chunks       []string `json:"chunks"`
	TokenCount   int      `json:"token_count"`
	ContentHash  string   `json:"content_hash"`
	CompileHash  string   `json:"compile_hash"`
	DocFile      string   `json:"doc_file"`
	CacheHit     bool     `json:"cache_hit"`
	Caller       string   `json:"caller"`
	SessionID    string   `json:"session_id"`
	Model        string   `json:"model"`
}

// ShortHash truncates a full hash to ShortHashLen characters.
// If the hash has the "sha256:" prefix, that prefix is stripped first.
// Returns the input unchanged if it is already shorter.
func ShortHash(hash string) string {
	h := strings.TrimPrefix(hash, hashPrefix)
	if len(h) > ShortHashLen {
		return h[:ShortHashLen]
	}
	return h
}

// HistoryDir returns the absolute path to the history directory for a project.
func HistoryDir(projectDir string, cfg *project.Config) string {
	return filepath.Join(project.RootDir(projectDir, cfg), project.CodectxDir, HistorySubdir)
}

// EnsureDir creates the history directory structure (queries/, chunks/, docs/)
// and ensures the project .gitignore includes the history entry. Called by
// command code before any write operation. Idempotent.
func EnsureDir(histDir, projectDir, root string) error {
	for _, sub := range []string{QueriesDir, ChunksDir, DocsDir} {
		if err := os.MkdirAll(filepath.Join(histDir, sub), project.DirPerm); err != nil {
			return fmt.Errorf("creating history/%s directory: %w", sub, err)
		}
	}

	if err := project.EnsureGitignore(projectDir, root); err != nil {
		return fmt.Errorf("ensuring gitignore: %w", err)
	}

	return nil
}

// --- Hash computation ---

// QueryHash computes the SHA-256 hash of a raw query string.
// Returns the hash in "sha256:<hex>" format.
func QueryHash(rawQuery string) string {
	sum := sha256.Sum256([]byte(rawQuery))
	return hashPrefix + hex.EncodeToString(sum[:])
}

// ChunkSetHash computes the SHA-256 hash of a sorted, comma-joined set of
// chunk IDs. Sorting before hashing ensures chunk ID order does not affect
// the cache key.
func ChunkSetHash(chunkIDs []string) string {
	sorted := make([]string, len(chunkIDs))
	copy(sorted, chunkIDs)
	sort.Strings(sorted)
	raw := strings.Join(sorted, ",")
	sum := sha256.Sum256([]byte(raw))
	return hashPrefix + hex.EncodeToString(sum[:])
}

// ContentHash computes the SHA-256 hash of document content.
// Returns the hash in "sha256:<hex>" format.
func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hashPrefix + hex.EncodeToString(sum[:])
}

// CompileHash computes the SHA-256 hash of the hashes.yml file content
// from the compiled directory. This serves as the cache invalidation signal.
func CompileHash(compiledDir string) (string, error) {
	hashesPath := manifest.HashesPath(compiledDir)
	data, err := os.ReadFile(hashesPath)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hashPrefix + hex.EncodeToString(sum[:]), nil
}

// --- Write operations ---

// WriteQueryEntry writes a query history entry as a JSON file in queries/.
// Filename format: [nanoTs].[queryHash12].json
func WriteQueryEntry(histDir string, entry QueryEntry) error {
	shortHash := ShortHash(entry.QueryHash)
	filename := fmt.Sprintf("%d.%s.json", entry.Ts, shortHash)
	path := filepath.Join(histDir, QueriesDir, filename)
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling query entry: %w", err)
	}
	return os.WriteFile(path, data, project.FilePerm)
}

// WriteChunksEntry writes a chunks history entry as a JSON file in chunks/.
// Filename format: [nanoTs].[chunkSetHash12].json
func WriteChunksEntry(histDir string, entry ChunksEntry) error {
	shortHash := ShortHash(entry.ChunkSetHash)
	filename := fmt.Sprintf("%d.%s.json", entry.Ts, shortHash)
	path := filepath.Join(histDir, ChunksDir, filename)
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling chunks entry: %w", err)
	}
	return os.WriteFile(path, data, project.FilePerm)
}

// SaveDocument writes the generated document to history/docs/[nanoTs].[contentHash12].md.
// Returns the filename (not full path) for use in ChunksEntry.DocFile.
func SaveDocument(histDir string, content []byte, contentHash string, ts int64) (filename string, err error) {
	shortHash := ShortHash(contentHash)
	filename = fmt.Sprintf("%d.%s.md", ts, shortHash)
	path := filepath.Join(histDir, DocsDir, filename)

	if err = os.WriteFile(path, content, project.FilePerm); err != nil {
		return "", fmt.Errorf("writing history document: %w", err)
	}

	return filename, nil
}

// AnnotateDocument appends an HTML warning comment to an existing history document.
// Used when post-save operations (like index writes) fail, so the document
// records what went wrong.
func AnnotateDocument(histDir, docFile, warning string) error {
	path := filepath.Join(histDir, DocsDir, docFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, project.FilePerm)
	if err != nil {
		return fmt.Errorf("opening document for annotation: %w", err)
	}

	annotation := fmt.Sprintf("\n<!-- codectx:warning %s -->\n", warning)
	if _, err := f.WriteString(annotation); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing annotation: %w", err)
	}
	return f.Close()
}

// --- Read operations ---

// ReadQueryHistory reads query entries from history/queries/, sorted by
// timestamp descending (newest first). If n > 0, at most n entries are returned.
// If n <= 0, all entries are returned.
func ReadQueryHistory(histDir string, n int) ([]QueryEntry, error) {
	pattern := filepath.Join(histDir, QueriesDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing query history: %w", err)
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// Lexicographic descending = newest first (timestamp-prefixed filenames).
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	if n > 0 && len(matches) > n {
		matches = matches[:n]
	}

	entries := make([]QueryEntry, 0, len(matches))
	for _, path := range matches {
		entry, readErr := readQueryEntry(path)
		if readErr != nil {
			continue // skip malformed files
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ReadChunksHistory reads chunks entries from history/chunks/, sorted by
// timestamp descending (newest first). If n > 0, at most n entries are returned.
// If n <= 0, all entries are returned.
func ReadChunksHistory(histDir string, n int) ([]ChunksEntry, error) {
	pattern := filepath.Join(histDir, ChunksDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing chunks history: %w", err)
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// Lexicographic descending = newest first (timestamp-prefixed filenames).
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	if n > 0 && len(matches) > n {
		matches = matches[:n]
	}

	entries := make([]ChunksEntry, 0, len(matches))
	for _, path := range matches {
		entry, readErr := readChunksEntry(path)
		if readErr != nil {
			continue // skip malformed files
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ShowDocument finds and returns the content of a history document by hash prefix.
// The hashPrefix is matched against the hash portion of filenames in docs/.
// Since filenames are [nanoTs].[hash12].md, the match pattern is *.[hashPrefix]*.md.
// If multiple matches exist, the newest (lexicographically last) is returned.
func ShowDocument(histDir, hashPrefix string) (string, error) {
	docsPath := filepath.Join(histDir, DocsDir)
	pattern := filepath.Join(docsPath, "*."+hashPrefix+"*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("globbing history docs: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no document found matching hash prefix %q", hashPrefix)
	}

	// Lexicographic sort: newest is last (timestamp-prefixed filenames).
	sort.Strings(matches)
	content, err := os.ReadFile(matches[len(matches)-1])
	if err != nil {
		return "", fmt.Errorf("reading document: %w", err)
	}
	return string(content), nil
}

// Clear removes all history data (all files in queries/, chunks/, and docs/).
func Clear(histDir string) error {
	for _, sub := range []string{QueriesDir, ChunksDir, DocsDir} {
		dir := filepath.Join(histDir, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading %s directory: %w", sub, err)
		}
		for _, e := range entries {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return fmt.Errorf("removing %s/%s: %w", sub, e.Name(), err)
			}
		}
	}
	return nil
}

// --- Generate cache ---

// GenerateCacheLookup searches the history for a cached generate result
// matching the given chunk IDs and current compilation state. Returns the
// path to the cached document and true on a cache hit, or empty string and
// false on a miss.
func GenerateCacheLookup(histDir string, chunkIDs []string, compiledDir string) (docPath string, hit bool) {
	chunkSetHash := ChunkSetHash(chunkIDs)
	compileHash, err := CompileHash(compiledDir)
	if err != nil {
		return "", false // compile hash unavailable — treat as miss
	}

	shortHash := ShortHash(chunkSetHash)
	pattern := filepath.Join(histDir, ChunksDir, "*."+shortHash+".json")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", false
	}

	// Lexicographic descending = newest first (timestamp-prefixed filenames).
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	for _, match := range matches {
		entry, readErr := readChunksEntry(match)
		if readErr != nil {
			continue
		}

		// Verify full hash matches (not just the 12-char prefix).
		if entry.ChunkSetHash != chunkSetHash {
			continue
		}

		// Verify compilation state matches.
		if entry.CompileHash != compileHash {
			continue
		}

		// Verify the docs/ file still exists (may have been pruned).
		docPath = filepath.Join(histDir, DocsDir, entry.DocFile)
		if _, statErr := os.Stat(docPath); os.IsNotExist(statErr) {
			continue
		}

		return docPath, true
	}

	return "", false
}

// --- High-level operations ---

// LogGenerate performs the full generate-history workflow: ensures the history
// directory exists, saves the document, writes a chunks entry, and prunes
// if needed. Returns the saved document filename. If the document is saved
// but the chunks entry write fails, the document is annotated with the error.
func LogGenerate(histDir, projectDir, root string, document []byte, chunkIDs []string, tokens int, contentHash, compileHash string, cacheHit bool, caller CallerContext) (docFile string, err error) {
	if err = EnsureDir(histDir, projectDir, root); err != nil {
		return "", fmt.Errorf("initializing history directory: %w", err)
	}

	ts := time.Now().UnixNano()

	docFile, err = SaveDocument(histDir, document, contentHash, ts)
	if err != nil {
		return "", fmt.Errorf("saving history document: %w", err)
	}

	entry := ChunksEntry{
		Ts:           ts,
		ChunkSetHash: ChunkSetHash(chunkIDs),
		Chunks:       chunkIDs,
		TokenCount:   tokens,
		ContentHash:  contentHash,
		CompileHash:  compileHash,
		DocFile:      docFile,
		CacheHit:     cacheHit,
		Caller:       caller.Caller,
		SessionID:    caller.SessionID,
		Model:        caller.Model,
	}

	if appendErr := WriteChunksEntry(histDir, entry); appendErr != nil {
		// Document saved but entry write failed — annotate the document.
		_ = AnnotateDocument(histDir, docFile, "chunks entry write failed: "+appendErr.Error())
		return docFile, fmt.Errorf("writing chunks entry: %w", appendErr)
	}

	if pruneErr := CheckAndPrune(histDir); pruneErr != nil {
		return docFile, fmt.Errorf("pruning history: %w", pruneErr)
	}

	return docFile, nil
}

// LogQuery performs the full query-history workflow: ensures the history
// directory exists, writes a query entry, and prunes if needed.
func LogQuery(histDir, projectDir, root, rawQuery, expandedQuery string, totalResults int, compileHash string, caller CallerContext) error {
	if err := EnsureDir(histDir, projectDir, root); err != nil {
		return fmt.Errorf("initializing history directory: %w", err)
	}

	entry := QueryEntry{
		Ts:          time.Now().UnixNano(),
		QueryHash:   QueryHash(rawQuery),
		Raw:         rawQuery,
		Expanded:    expandedQuery,
		ResultCount: totalResults,
		CompileHash: compileHash,
		Caller:      caller.Caller,
		SessionID:   caller.SessionID,
		Model:       caller.Model,
	}

	if err := WriteQueryEntry(histDir, entry); err != nil {
		return fmt.Errorf("writing query entry: %w", err)
	}

	if err := CheckAndPrune(histDir); err != nil {
		return fmt.Errorf("pruning history: %w", err)
	}

	return nil
}

// --- Pruning ---

// CheckAndPrune enforces the MaxSize limit on the history directory.
// If the total size exceeds MaxSize, it prunes each subdirectory to the
// most recent PruneKeep files.
func CheckAndPrune(histDir string) error {
	size, err := dirSize(histDir)
	if err != nil {
		return fmt.Errorf("computing history size: %w", err)
	}

	if size <= MaxSize {
		return nil
	}

	for _, sub := range []string{QueriesDir, ChunksDir, DocsDir} {
		if err := PruneDirectory(filepath.Join(histDir, sub), PruneKeep); err != nil {
			return fmt.Errorf("pruning %s: %w", sub, err)
		}
	}

	return nil
}

// PruneDirectory deletes all but the keepN most recent files in a directory.
// Timestamp-first filenames mean lexicographic sort equals chronological order.
func PruneDirectory(dir string, keepN int) error {
	entries, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return err
	}
	if len(entries) <= keepN {
		return nil
	}

	// Lexicographic sort: oldest first (timestamp-prefixed filenames).
	sort.Strings(entries)

	// Delete oldest entries (beginning of sorted list).
	for _, path := range entries[:len(entries)-keepN] {
		_ = os.Remove(path) // best-effort — individual errors ignored
	}

	return nil
}

// --- Internal helpers ---

// readQueryEntry reads and parses a single query entry JSON file.
func readQueryEntry(path string) (QueryEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return QueryEntry{}, err
	}
	var entry QueryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return QueryEntry{}, err
	}
	return entry, nil
}

// readChunksEntry reads and parses a single chunks entry JSON file.
func readChunksEntry(path string) (ChunksEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ChunksEntry{}, err
	}
	var entry ChunksEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return ChunksEntry{}, err
	}
	return entry, nil
}

// dirSize computes the total size of all files in a directory tree.
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

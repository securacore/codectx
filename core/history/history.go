// Package history implements the query and generate history system for codectx.
//
// History is stored under docs/.codectx/history/ with three components:
//   - query.history — tab-separated log of codectx query invocations
//   - chunks.history — tab-separated log of codectx generate invocations
//   - docs/<shortHash>.<nanoTs>.md — full generated document snapshots
//
// All timestamps use nanosecond precision. The history directory is lazily
// created on first use by EnsureDir, which also ensures the project .gitignore
// includes the history entry. History is capped at 100 MB; when exceeded,
// CheckAndPrune truncates index files and retains only the most recent documents.
package history

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/securacore/codectx/core/project"
)

const (
	// QueryHistoryFile is the filename for query history entries.
	QueryHistoryFile = "query.history"

	// ChunksHistoryFile is the filename for generate/chunks history entries.
	ChunksHistoryFile = "chunks.history"

	// DocsDir is the subdirectory for generated document snapshots.
	DocsDir = "docs"

	// MaxSize is the maximum total size of the history directory (100 MB).
	MaxSize = 100 * 1024 * 1024

	// PruneKeep is the number of entries/documents to retain after pruning.
	PruneKeep = 5

	// ShortHashLen is the number of hex characters from SHA-256 used in doc
	// filenames and display identifiers.
	ShortHashLen = 12

	// HistorySubdir is the subdirectory name under .codectx/.
	HistorySubdir = "history"

	// fieldSep is the tab character used as TSV field separator.
	fieldSep = "\t"
)

// ShortHash truncates a full hex hash to ShortHashLen characters.
// Returns the input unchanged if it is already shorter.
func ShortHash(hash string) string {
	if len(hash) > ShortHashLen {
		return hash[:ShortHashLen]
	}
	return hash
}

// QueryEntry represents a single parsed line from query.history.
type QueryEntry struct {
	// Timestamp is the nanosecond Unix timestamp of the query.
	Timestamp int64

	// RawQuery is the original query string the user provided.
	RawQuery string

	// ExpandedQuery is the query after alias/synonym expansion.
	ExpandedQuery string

	// ResultCount is the total number of results returned.
	ResultCount int
}

// ChunksEntry represents a single parsed line from chunks.history.
type ChunksEntry struct {
	// Timestamp is the nanosecond Unix timestamp of the generate invocation.
	Timestamp int64

	// ChunkIDs is the list of chunk IDs included in the generated document.
	ChunkIDs []string

	// TokenCount is the total token count of the generated document.
	TokenCount int

	// ContentHash is the full hex SHA-256 hash of the generated document.
	ContentHash string
}

// HistoryDir returns the absolute path to the history directory for a project.
func HistoryDir(projectDir string, cfg *project.Config) string {
	return filepath.Join(project.RootDir(projectDir, cfg), project.CodectxDir, HistorySubdir)
}

// EnsureDir creates the history directory structure (history/ and history/docs/)
// and ensures the project .gitignore includes the history entry. Called by
// command code before any write operation. Idempotent.
func EnsureDir(histDir, projectDir, root string) error {
	docsPath := filepath.Join(histDir, DocsDir)
	if err := os.MkdirAll(docsPath, project.DirPerm); err != nil {
		return fmt.Errorf("creating history directory: %w", err)
	}

	if err := project.EnsureGitignore(projectDir, root); err != nil {
		return fmt.Errorf("ensuring gitignore: %w", err)
	}

	return nil
}

// AppendQuery appends a query entry to query.history as a TSV line.
// Format: <nanoTs>\t<rawQuery>\t<expandedQuery>\t<resultCount>
func AppendQuery(histDir, rawQuery, expandedQuery string, resultCount int) error {
	line := fmt.Sprintf("%d%s%s%s%s%s%d\n",
		time.Now().UnixNano(), fieldSep,
		escapeTSV(rawQuery), fieldSep,
		escapeTSV(expandedQuery), fieldSep,
		resultCount,
	)
	return appendToFile(filepath.Join(histDir, QueryHistoryFile), line)
}

// AppendChunks appends a chunks entry to chunks.history as a TSV line.
// Format: <nanoTs>\t<chunkID1,chunkID2,...>\t<tokenCount>\t<contentHash>
func AppendChunks(histDir string, chunkIDs []string, tokenCount int, contentHash string) error {
	line := fmt.Sprintf("%d%s%s%s%d%s%s\n",
		time.Now().UnixNano(), fieldSep,
		strings.Join(chunkIDs, ","), fieldSep,
		tokenCount, fieldSep,
		contentHash,
	)
	return appendToFile(filepath.Join(histDir, ChunksHistoryFile), line)
}

// SaveDocument writes the generated document to history/docs/<shortHash>.<nanoTs>.md.
// Returns the short hash (first 12 hex chars of SHA-256), the content hash (full hex),
// and the full file path.
func SaveDocument(histDir, content string) (shortHash, contentHash, filePath string, err error) {
	hash := sha256.Sum256([]byte(content))
	contentHash = fmt.Sprintf("%x", hash)
	shortHash = contentHash[:ShortHashLen]

	ts := time.Now().UnixNano()
	filename := fmt.Sprintf("%s.%d.md", shortHash, ts)
	filePath = filepath.Join(histDir, DocsDir, filename)

	if err = os.WriteFile(filePath, []byte(content), project.FilePerm); err != nil {
		return "", "", "", fmt.Errorf("writing history document: %w", err)
	}

	return shortHash, contentHash, filePath, nil
}

// AnnotateDocument appends an HTML warning comment to an existing history document.
// Used when post-save operations (like index writes) fail, so the document
// records what went wrong.
func AnnotateDocument(filePath, warning string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, project.FilePerm)
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

// CheckAndPrune enforces the MaxSize limit on the history directory.
// If the total size exceeds MaxSize, it truncates both .history files to the
// last PruneKeep lines and deletes all but the PruneKeep most recent documents.
func CheckAndPrune(histDir string) error {
	size, err := dirSize(histDir)
	if err != nil {
		return fmt.Errorf("computing history size: %w", err)
	}

	if size <= MaxSize {
		return nil
	}

	// Truncate index files.
	for _, name := range []string{QueryHistoryFile, ChunksHistoryFile} {
		path := filepath.Join(histDir, name)
		if err := truncateToLastN(path, PruneKeep); err != nil {
			return fmt.Errorf("truncating %s: %w", name, err)
		}
	}

	// Prune documents to most recent PruneKeep.
	if err := pruneDocuments(filepath.Join(histDir, DocsDir), PruneKeep); err != nil {
		return fmt.Errorf("pruning documents: %w", err)
	}

	return nil
}

// ReadQueryHistory reads the last n entries from query.history.
// Returns entries in chronological order (oldest first).
// If n <= 0, all entries are returned.
func ReadQueryHistory(histDir string, n int) ([]QueryEntry, error) {
	lines, err := readLastN(filepath.Join(histDir, QueryHistoryFile), n)
	if err != nil {
		return nil, err
	}

	entries := make([]QueryEntry, 0, len(lines))
	for _, line := range lines {
		entry, err := parseQueryLine(line)
		if err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ReadChunksHistory reads the last n entries from chunks.history.
// Returns entries in chronological order (oldest first).
// If n <= 0, all entries are returned.
func ReadChunksHistory(histDir string, n int) ([]ChunksEntry, error) {
	lines, err := readLastN(filepath.Join(histDir, ChunksHistoryFile), n)
	if err != nil {
		return nil, err
	}

	entries := make([]ChunksEntry, 0, len(lines))
	for _, line := range lines {
		entry, err := parseChunksLine(line)
		if err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ShowDocument finds and returns the content of a history document by hash prefix.
// The hashPrefix must match the beginning of the document filename's hash component.
// Returns an error if no match is found or multiple documents match.
func ShowDocument(histDir, hashPrefix string) (string, error) {
	docsPath := filepath.Join(histDir, DocsDir)
	entries, err := os.ReadDir(docsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no history documents found")
		}
		return "", fmt.Errorf("reading history docs: %w", err)
	}

	var matches []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// Filename format: <hash>.<nanoTs>.md
		if strings.HasPrefix(e.Name(), hashPrefix) {
			matches = append(matches, filepath.Join(docsPath, e.Name()))
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no document found matching hash prefix %q", hashPrefix)
	case 1:
		content, err := os.ReadFile(matches[0])
		if err != nil {
			return "", fmt.Errorf("reading document: %w", err)
		}
		return string(content), nil
	default:
		// Multiple matches — pick the most recent (highest timestamp).
		sort.Strings(matches)
		content, err := os.ReadFile(matches[len(matches)-1])
		if err != nil {
			return "", fmt.Errorf("reading document: %w", err)
		}
		return string(content), nil
	}
}

// Clear removes all history data (both .history files and all docs).
func Clear(histDir string) error {
	for _, name := range []string{QueryHistoryFile, ChunksHistoryFile} {
		path := filepath.Join(histDir, name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", name, err)
		}
	}

	docsPath := filepath.Join(histDir, DocsDir)
	entries, err := os.ReadDir(docsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading docs directory: %w", err)
	}
	for _, e := range entries {
		if err := os.Remove(filepath.Join(docsPath, e.Name())); err != nil {
			return fmt.Errorf("removing %s: %w", e.Name(), err)
		}
	}

	return nil
}

// --- high-level operations ---
// These combine EnsureDir + operation + CheckAndPrune into single calls.
// Errors are returned, not swallowed — callers decide how to handle them.

// LogGenerate performs the full generate-history workflow: ensures the history
// directory exists, saves the document, appends to chunks.history, and prunes
// if needed. Returns the saved document path. If the document is saved but the
// index append fails, the document is annotated with the error.
func LogGenerate(histDir, projectDir, root, document string, chunkIDs []string, tokens int, contentHash string) (docPath string, err error) {
	if err = EnsureDir(histDir, projectDir, root); err != nil {
		return "", fmt.Errorf("initializing history directory: %w", err)
	}

	_, _, docPath, err = SaveDocument(histDir, document)
	if err != nil {
		return "", fmt.Errorf("saving history document: %w", err)
	}

	if appendErr := AppendChunks(histDir, chunkIDs, tokens, contentHash); appendErr != nil {
		// Document saved but index write failed — annotate the document.
		_ = AnnotateDocument(docPath, "chunks.history write failed: "+appendErr.Error())
		return docPath, fmt.Errorf("logging to chunks.history: %w", appendErr)
	}

	if pruneErr := CheckAndPrune(histDir); pruneErr != nil {
		return docPath, fmt.Errorf("pruning history: %w", pruneErr)
	}

	return docPath, nil
}

// LogQuery performs the full query-history workflow: ensures the history
// directory exists, appends to query.history, and prunes if needed.
func LogQuery(histDir, projectDir, root, rawQuery, expandedQuery string, totalResults int) error {
	if err := EnsureDir(histDir, projectDir, root); err != nil {
		return fmt.Errorf("initializing history directory: %w", err)
	}

	if err := AppendQuery(histDir, rawQuery, expandedQuery, totalResults); err != nil {
		return fmt.Errorf("logging query: %w", err)
	}

	if err := CheckAndPrune(histDir); err != nil {
		return fmt.Errorf("pruning history: %w", err)
	}

	return nil
}

// --- internal helpers ---

// appendToFile appends data to a file, creating it if it doesn't exist.
func appendToFile(path, data string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, project.FilePerm)
	if err != nil {
		return fmt.Errorf("opening %s: %w", filepath.Base(path), err)
	}

	if _, err := f.WriteString(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing to %s: %w", filepath.Base(path), err)
	}
	return f.Close()
}

// escapeTSV replaces tabs and newlines in a string to make it safe for TSV.
func escapeTSV(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// dirSize computes the total size of all files in a directory tree.
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// truncateToLastN keeps only the last n lines of a file.
// If the file doesn't exist or has <= n lines, it's a no-op.
func truncateToLastN(path string, n int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) <= n {
		return nil
	}

	kept := lines[len(lines)-n:]
	return os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), project.FilePerm)
}

// pruneDocuments deletes all but the keep most recent .md files in a directory.
// "Most recent" is determined by the nanosecond timestamp in the filename.
func pruneDocuments(docsPath string, keep int) error {
	entries, err := os.ReadDir(docsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Collect .md files.
	var mdFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e)
		}
	}

	if len(mdFiles) <= keep {
		return nil
	}

	// Sort by name descending. Since filenames contain nanosecond timestamps,
	// lexicographic sort on the timestamp portion gives chronological order.
	// We extract timestamps for proper numeric sorting.
	type fileWithTS struct {
		name string
		ts   int64
	}
	files := make([]fileWithTS, 0, len(mdFiles))
	for _, e := range mdFiles {
		ts := extractTimestamp(e.Name())
		files = append(files, fileWithTS{name: e.Name(), ts: ts})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ts > files[j].ts // newest first
	})

	// Delete everything after the first 'keep' entries.
	for _, f := range files[keep:] {
		if err := os.Remove(filepath.Join(docsPath, f.name)); err != nil {
			return err
		}
	}

	return nil
}

// extractTimestamp extracts the nanosecond timestamp from a filename like
// "a1b2c3d4e5f6.1741532400123456789.md". Returns 0 if parsing fails.
func extractTimestamp(name string) int64 {
	// Strip .md suffix.
	name = strings.TrimSuffix(name, ".md")
	// Find the last dot — everything after it is the timestamp.
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return 0
	}
	ts, err := strconv.ParseInt(name[idx+1:], 10, 64)
	if err != nil {
		return 0
	}
	return ts
}

// readLastN reads the last n non-empty lines from a file.
// If n <= 0, all lines are returned. Returns nil (not error) if file doesn't exist.
func readLastN(path string, n int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}

	raw := strings.TrimRight(string(data), "\n")
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(raw, "\n")

	// Filter empty lines.
	nonEmpty := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}

	if n > 0 && len(nonEmpty) > n {
		nonEmpty = nonEmpty[len(nonEmpty)-n:]
	}
	return nonEmpty, nil
}

// parseQueryLine parses a single TSV line from query.history.
func parseQueryLine(line string) (QueryEntry, error) {
	fields := strings.SplitN(line, fieldSep, 4)
	if len(fields) != 4 {
		return QueryEntry{}, fmt.Errorf("expected 4 fields, got %d", len(fields))
	}

	ts, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return QueryEntry{}, fmt.Errorf("parsing timestamp: %w", err)
	}

	count, err := strconv.Atoi(fields[3])
	if err != nil {
		return QueryEntry{}, fmt.Errorf("parsing result count: %w", err)
	}

	return QueryEntry{
		Timestamp:     ts,
		RawQuery:      fields[1],
		ExpandedQuery: fields[2],
		ResultCount:   count,
	}, nil
}

// parseChunksLine parses a single TSV line from chunks.history.
func parseChunksLine(line string) (ChunksEntry, error) {
	fields := strings.SplitN(line, fieldSep, 4)
	if len(fields) != 4 {
		return ChunksEntry{}, fmt.Errorf("expected 4 fields, got %d", len(fields))
	}

	ts, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return ChunksEntry{}, fmt.Errorf("parsing timestamp: %w", err)
	}

	var chunkIDs []string
	if fields[1] != "" {
		chunkIDs = strings.Split(fields[1], ",")
	}

	tokens, err := strconv.Atoi(fields[2])
	if err != nil {
		return ChunksEntry{}, fmt.Errorf("parsing token count: %w", err)
	}

	return ChunksEntry{
		Timestamp:   ts,
		ChunkIDs:    chunkIDs,
		TokenCount:  tokens,
		ContentHash: fields[3],
	}, nil
}

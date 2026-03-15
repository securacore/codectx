// This file provides shared generate-cache-hit logic used by both the
// generate and prompt commands. The ServeCacheHit function performs the
// common sequence of reading the cached document, recovering token count
// from history, writing a cache-hit chunks entry, and updating usage
// metrics. Callers handle formatting and output themselves.

package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/usage"
)

// CacheHitParams holds the inputs needed to serve a generate cache hit.
type CacheHitParams struct {
	DocPath     string                // path to the cached document file
	ChunkIDs    []string              // chunk IDs that produced this cache entry
	HistDir     string                // project history directory
	CompiledDir string                // compiled output directory (for compile hash)
	UsageFile   string                // path to the local usage metrics file
	Caller      history.CallerContext // caller context (tool, session, model)
}

// CacheHitResult holds the values recovered from a cache hit, allowing
// callers to build their own summary/header output.
type CacheHitResult struct {
	Content    []byte // raw document content
	TokenCount int    // token count recovered from history (0 if not found)
	Hash       string // content hash of the cached document
	DocFile    string // base filename of the cached document
}

// ServeCacheHit handles the common generate cache-hit sequence shared by
// the generate and prompt commands:
//
//  1. Read the cached document from disk.
//  2. Compute content and compile hashes.
//  3. Scan recent history chunks entries to recover the token count.
//  4. Write a new chunks entry with CacheHit=true (best-effort).
//  5. Update usage metrics (best-effort).
//
// The caller is responsible for formatting summaries and calling
// OutputDocument, since generate and prompt use different formatters.
func ServeCacheHit(p CacheHitParams) (*CacheHitResult, error) {
	content, err := os.ReadFile(p.DocPath)
	if err != nil {
		return nil, fmt.Errorf("reading cached document: %w", err)
	}

	contentHash := history.ContentHash(content)
	compileHash, _ := history.CompileHash(p.CompiledDir)

	// Recover token count from the most recent matching chunks entry.
	tokenCount := 0
	if entries, readErr := history.ReadChunksHistory(p.HistDir, 0); readErr == nil {
		for _, e := range entries {
			if e.ContentHash == contentHash {
				tokenCount = e.TokenCount
				break
			}
		}
	}

	// Write chunks entry recording the cache hit (best-effort).
	docFile := filepath.Base(p.DocPath)
	entry := history.ChunksEntry{
		Ts:           time.Now().UnixNano(),
		ChunkSetHash: history.ChunkSetHash(p.ChunkIDs),
		Chunks:       p.ChunkIDs,
		TokenCount:   tokenCount,
		ContentHash:  contentHash,
		CompileHash:  compileHash,
		DocFile:      docFile,
		CacheHit:     true,
		Caller:       p.Caller.Caller,
		SessionID:    p.Caller.SessionID,
		Model:        p.Caller.Model,
	}
	if writeErr := history.WriteChunksEntry(p.HistDir, entry); writeErr != nil {
		WarnBestEffort("Writing cache-hit entry", writeErr)
	}

	// Update usage (best-effort).
	if usageErr := usage.UpdateGenerate(p.UsageFile, tokenCount, true, p.Caller); usageErr != nil {
		WarnBestEffort("Updating usage metrics", usageErr)
	}

	return &CacheHitResult{
		Content:    content,
		TokenCount: tokenCount,
		Hash:       contentHash,
		DocFile:    docFile,
	}, nil
}

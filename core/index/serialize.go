package index

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/project"
)

// indexFileName is the name of the serialized index file within each
// index type subdirectory.
const indexFileName = "index.gob"

// Save writes all three BM25 indexes to the compiled directory.
// Creates files at:
//   - compiledDir/bm25/objects/index.gob
//   - compiledDir/bm25/specs/index.gob
//   - compiledDir/bm25/system/index.gob
//
// Parent directories are created as needed.
func (idx *Index) Save(compiledDir string) error {
	for it, bm25 := range idx.Indexes {
		dir := filepath.Join(compiledDir, project.BM25Dir, string(it))
		if err := os.MkdirAll(dir, project.DirPerm); err != nil {
			return fmt.Errorf("creating index directory %s: %w", dir, err)
		}

		path := filepath.Join(dir, indexFileName)
		if err := saveIndex(path, bm25); err != nil {
			return fmt.Errorf("saving %s index: %w", it, err)
		}
	}
	return nil
}

// Load reads all three BM25 indexes from the compiled directory.
// Returns an error if any index file is missing or corrupted.
//
// The loaded indexes are ready for querying — no Build() call is needed
// since the IDF cache and average document length are part of the
// serialized state.
func Load(compiledDir string) (*Index, error) {
	idx := &Index{
		Indexes: make(map[IndexType]*BM25, 3),
	}

	for _, it := range AllIndexTypes() {
		path := filepath.Join(compiledDir, project.BM25Dir, string(it), indexFileName)
		bm25, err := loadIndex(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s index: %w", it, err)
		}
		idx.Indexes[it] = bm25
	}

	return idx, nil
}

// saveIndex writes a single BM25 index to a gob-encoded file.
func saveIndex(path string, bm25 *BM25) (retErr error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("closing file %s: %w", path, closeErr)
		}
	}()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(bm25); err != nil {
		return fmt.Errorf("encoding index: %w", err)
	}

	return nil
}

// loadIndex reads a single BM25 index from a gob-encoded file.
func loadIndex(path string) (result *BM25, retErr error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("closing file %s: %w", path, closeErr)
		}
	}()

	var bm25 BM25
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&bm25); err != nil {
		return nil, fmt.Errorf("decoding index from %s: %w", path, err)
	}

	return &bm25, nil
}

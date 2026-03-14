package index

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/project"
	"golang.org/x/sync/errgroup"
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

// Load reads all three BM25 indexes from the compiled directory in parallel.
// Returns an error if any index file is missing or corrupted.
//
// The loaded indexes are ready for querying — no Build() call is needed
// since the IDF cache and average document length are part of the
// serialized state.
func Load(compiledDir string) (*Index, error) {
	types := allIndexTypes()
	loaded := make([]*BM25, len(types))

	var g errgroup.Group
	for i, it := range types {
		path := filepath.Join(compiledDir, project.BM25Dir, string(it), indexFileName)
		g.Go(func() error {
			bm25, err := loadIndex(path)
			if err != nil {
				return fmt.Errorf("loading %s index: %w", it, err)
			}
			loaded[i] = bm25
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	idx := &Index{
		Indexes: make(map[IndexType]*BM25, len(types)),
	}
	for i, it := range types {
		idx.Indexes[it] = loaded[i]
	}

	return idx, nil
}

// SaveFieldIndex writes all three BM25F indexes to the compiled directory.
// Creates files at:
//   - compiledDir/bm25f/objects/index.gob
//   - compiledDir/bm25f/specs/index.gob
//   - compiledDir/bm25f/system/index.gob
func (idx *FieldIndex) SaveFieldIndex(compiledDir string) error {
	for it, bm25f := range idx.Indexes {
		dir := filepath.Join(compiledDir, project.BM25FDir, string(it))
		if err := os.MkdirAll(dir, project.DirPerm); err != nil {
			return fmt.Errorf("creating field index directory %s: %w", dir, err)
		}

		path := filepath.Join(dir, indexFileName)
		if err := saveFieldIndex(path, bm25f); err != nil {
			return fmt.Errorf("saving %s field index: %w", it, err)
		}
	}
	return nil
}

// LoadFieldIndex reads all three BM25F indexes from the compiled directory
// in parallel. Returns an error if any index file is missing or corrupted.
func LoadFieldIndex(compiledDir string) (*FieldIndex, error) {
	types := allIndexTypes()
	loaded := make([]*BM25F, len(types))

	var g errgroup.Group
	for i, it := range types {
		path := filepath.Join(compiledDir, project.BM25FDir, string(it), indexFileName)
		g.Go(func() error {
			bm25f, err := loadFieldIndex(path)
			if err != nil {
				return fmt.Errorf("loading %s field index: %w", it, err)
			}
			loaded[i] = bm25f
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	idx := &FieldIndex{
		Indexes: make(map[IndexType]*BM25F, len(types)),
	}
	for i, it := range types {
		idx.Indexes[it] = loaded[i]
	}

	return idx, nil
}

// saveGob writes a gob-encoded value to a file.
func saveGob(path string, v any) (retErr error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("closing file %s: %w", path, closeErr)
		}
	}()

	if err := gob.NewEncoder(f).Encode(v); err != nil {
		return fmt.Errorf("encoding gob to %s: %w", path, err)
	}
	return nil
}

// loadGob reads a gob-encoded value from a file into the given pointer.
func loadGob(path string, v any) (retErr error) {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("closing file %s: %w", path, closeErr)
		}
	}()

	if err := gob.NewDecoder(f).Decode(v); err != nil {
		return fmt.Errorf("decoding gob from %s: %w", path, err)
	}
	return nil
}

// saveIndex writes a single BM25 index to a gob-encoded file.
func saveIndex(path string, bm25 *BM25) error {
	return saveGob(path, bm25)
}

// loadIndex reads a single BM25 index from a gob-encoded file.
func loadIndex(path string) (*BM25, error) {
	var bm25 BM25
	if err := loadGob(path, &bm25); err != nil {
		return nil, err
	}
	return &bm25, nil
}

// saveFieldIndex writes a single BM25F index to a gob-encoded file.
func saveFieldIndex(path string, bm25f *BM25F) error {
	return saveGob(path, bm25f)
}

// loadFieldIndex reads a single BM25F index from a gob-encoded file.
func loadFieldIndex(path string) (*BM25F, error) {
	var bm25f BM25F
	if err := loadGob(path, &bm25f); err != nil {
		return nil, err
	}
	return &bm25f, nil
}

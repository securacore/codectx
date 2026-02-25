package compile

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ObjectStore manages the content-addressed object store.
// Files are stored as {16-char SHA256 prefix}.md in a flat directory.
// Identical content produces the same hash, giving natural deduplication.
type ObjectStore struct {
	dir string // absolute path to the objects/ directory
}

// NewObjectStore creates an ObjectStore rooted at the given directory.
func NewObjectStore(dir string) *ObjectStore {
	return &ObjectStore{dir: dir}
}

// Store writes content to the object store and returns its hash.
// If an object with the same hash already exists, it is not overwritten
// (content-addressed writes are idempotent).
func (s *ObjectStore) Store(content []byte) (string, error) {
	hash := ContentHash(content)
	path := filepath.Join(s.dir, hash+".md")

	// Skip if already exists (idempotent).
	if _, err := os.Stat(path); err == nil {
		return hash, nil
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", fmt.Errorf("create objects directory: %w", err)
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", fmt.Errorf("write object %s: %w", hash, err)
	}

	return hash, nil
}

// StoreAs writes content to the object store under the given hash.
// Unlike Store, the hash is provided externally rather than computed from content.
// This supports the two-pass compilation model where the hash is based on raw
// source content but the stored content has rewritten links.
// If an object with the same hash already exists, it is not overwritten.
func (s *ObjectStore) StoreAs(hash string, content []byte) error {
	path := filepath.Join(s.dir, hash+".md")

	// Skip if already exists (idempotent).
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create objects directory: %w", err)
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write object %s: %w", hash, err)
	}

	return nil
}

// ContentHash computes the 16-char hex SHA256 prefix of the given data.
// This provides 64 bits of collision resistance (~4 billion files).
func ContentHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)[:16]
}

// ObjectPath returns the relative path for a hash (e.g., "objects/a1b2c3d4e5f67890.md").
// This is the value stored in compiled manifest entries.
func ObjectPath(hash string) string {
	return fmt.Sprintf("objects/%s.md", hash)
}

// Prune removes objects not in the active set.
// Returns the number of files removed.
func (s *ObjectStore) Prune(active map[string]bool) (int, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read objects directory: %w", err)
	}

	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		hash := strings.TrimSuffix(name, ".md")
		if !active[hash] {
			if err := os.Remove(filepath.Join(s.dir, name)); err != nil {
				return removed, fmt.Errorf("remove orphan %s: %w", name, err)
			}
			removed++
		}
	}

	return removed, nil
}

// List returns the set of hashes currently in the object store.
func (s *ObjectStore) List() (map[string]bool, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("read objects directory: %w", err)
	}

	hashes := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		hash := strings.TrimSuffix(e.Name(), ".md")
		hashes[hash] = true
	}

	return hashes, nil
}

// Read returns the content of an object by hash.
func (s *ObjectStore) Read(hash string) ([]byte, error) {
	path := filepath.Join(s.dir, hash+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", hash, err)
	}
	return data, nil
}

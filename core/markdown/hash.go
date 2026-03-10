package markdown

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash computes a SHA-256 hex digest of the given content.
// Used for content-based cache invalidation in incremental compilation.
func Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

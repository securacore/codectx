// Package util provides small shared helpers used across core and cmd packages.
package util

// ToSet converts a string slice to a set for O(1) lookups.
func ToSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// SplitKey splits a "section:id" key into its parts.
func SplitKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// KeyID extracts the ID portion from a "section:id" key.
// Returns the full key unchanged if no colon is present.
func KeyID(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[i+1:]
		}
	}
	return key
}

package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadManifest reads and parses the manifest YAML file at the given path.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	// Ensure maps are non-nil for safe iteration.
	if m.Objects == nil {
		m.Objects = make(map[string]*ManifestEntry)
	}
	if m.Specs == nil {
		m.Specs = make(map[string]*ManifestEntry)
	}
	if m.System == nil {
		m.System = make(map[string]*ManifestEntry)
	}

	return &m, nil
}

// LoadHashes reads and parses the hashes YAML file at the given path.
func LoadHashes(path string) (*Hashes, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading hashes: %w", err)
	}

	var h Hashes
	if err := yaml.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parsing hashes: %w", err)
	}

	// Ensure maps are non-nil for safe iteration.
	if h.Files == nil {
		h.Files = make(map[string]string)
	}
	if h.System == nil {
		h.System = make(map[string]string)
	}

	return &h, nil
}

// LoadMetadata reads and parses the metadata YAML file at the given path.
func LoadMetadata(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metadata: %w", err)
	}

	var m Metadata
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	if m.Documents == nil {
		m.Documents = make(map[string]*DocumentEntry)
	}

	return &m, nil
}

package manifest_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/manifest"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// HashFile
// ---------------------------------------------------------------------------

func TestHashFile_ReturnsCorrectHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	content := []byte("# Hello World\n\nSome content here.\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := manifest.HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	sum := sha256.Sum256(content)
	want := "sha256:" + hex.EncodeToString(sum[:])
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestHashFile_HasSHA256Prefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := manifest.HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", hash)
	}
}

func TestHashFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := manifest.HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	sum := sha256.Sum256(nil)
	want := "sha256:" + hex.EncodeToString(sum[:])
	if hash != want {
		t.Errorf("expected %q, got %q", want, hash)
	}
}

func TestHashFile_NonexistentFile(t *testing.T) {
	_, err := manifest.HashFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestHashFile_DifferentContentsDifferentHashes(t *testing.T) {
	dir := t.TempDir()

	path1 := filepath.Join(dir, "a.txt")
	path2 := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(path1, []byte("content A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path2, []byte("content B"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash1, err := manifest.HashFile(path1)
	if err != nil {
		t.Fatalf("HashFile a: %v", err)
	}
	hash2, err := manifest.HashFile(path2)
	if err != nil {
		t.Fatalf("HashFile b: %v", err)
	}

	if hash1 == hash2 {
		t.Error("different contents should produce different hashes")
	}
}

func TestHashFile_SameContentSameHash(t *testing.T) {
	dir := t.TempDir()
	content := []byte("identical content")

	path1 := filepath.Join(dir, "a.txt")
	path2 := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(path1, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path2, content, 0o644); err != nil {
		t.Fatal(err)
	}

	hash1, err := manifest.HashFile(path1)
	if err != nil {
		t.Fatalf("HashFile a: %v", err)
	}
	hash2, err := manifest.HashFile(path2)
	if err != nil {
		t.Fatalf("HashFile b: %v", err)
	}

	if hash1 != hash2 {
		t.Error("identical contents should produce identical hashes")
	}
}

// ---------------------------------------------------------------------------
// HashDir
// ---------------------------------------------------------------------------

func TestHashDir_SingleFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := manifest.HashDir(dir)
	if err != nil {
		t.Fatalf("HashDir: %v", err)
	}

	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", hash)
	}
}

func TestHashDir_MultipleFiles_Deterministic(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"a.md": "content A",
		"b.md": "content B",
		"c.md": "content C",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	hash1, err := manifest.HashDir(dir)
	if err != nil {
		t.Fatalf("first HashDir: %v", err)
	}
	hash2, err := manifest.HashDir(dir)
	if err != nil {
		t.Fatalf("second HashDir: %v", err)
	}

	if hash1 != hash2 {
		t.Error("HashDir should be deterministic for same directory contents")
	}
}

func TestHashDir_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	hash, err := manifest.HashDir(dir)
	if err != nil {
		t.Fatalf("HashDir: %v", err)
	}

	// Empty dir should hash to sha256 of empty string.
	sum := sha256.Sum256(nil)
	want := "sha256:" + hex.EncodeToString(sum[:])
	if hash != want {
		t.Errorf("expected %q for empty dir, got %q", want, hash)
	}
}

func TestHashDir_NonexistentDirectory(t *testing.T) {
	_, err := manifest.HashDir("/nonexistent/path/dir")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestHashDir_DifferentContentsDifferentHashes(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir1, "file.txt"), []byte("version 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "file.txt"), []byte("version 2"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash1, err := manifest.HashDir(dir1)
	if err != nil {
		t.Fatalf("HashDir dir1: %v", err)
	}
	hash2, err := manifest.HashDir(dir2)
	if err != nil {
		t.Fatalf("HashDir dir2: %v", err)
	}

	if hash1 == hash2 {
		t.Error("directories with different contents should have different hashes")
	}
}

func TestHashDir_NestedFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.md"), []byte("root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.md"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := manifest.HashDir(dir)
	if err != nil {
		t.Fatalf("HashDir: %v", err)
	}

	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", hash)
	}

	// Hash should differ from hashing just the root file.
	dirRootOnly := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirRootOnly, "root.md"), []byte("root"), 0o644); err != nil {
		t.Fatal(err)
	}
	hashRootOnly, err := manifest.HashDir(dirRootOnly)
	if err != nil {
		t.Fatalf("HashDir root only: %v", err)
	}

	if hash == hashRootOnly {
		t.Error("directory with nested files should hash differently from directory with only root file")
	}
}

// ---------------------------------------------------------------------------
// HashSystemDirs
// ---------------------------------------------------------------------------

func TestHashSystemDirs_HashesExistingDirs(t *testing.T) {
	topicsDir := t.TempDir()

	// Create the context-assembly system instruction dir.
	ctxDir := filepath.Join(topicsDir, "context-assembly")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "README.md"), []byte("# Context Assembly"), 0o644); err != nil {
		t.Fatal(err)
	}

	hashes, err := manifest.HashSystemDirs(topicsDir)
	if err != nil {
		t.Fatalf("HashSystemDirs: %v", err)
	}

	if len(hashes) != 1 {
		t.Errorf("expected 1 entry, got %d", len(hashes))
	}
	if _, ok := hashes["context-assembly"]; !ok {
		t.Error("expected context-assembly hash")
	}
}

func TestHashSystemDirs_SkipsMissingDirs(t *testing.T) {
	topicsDir := t.TempDir()

	hashes, err := manifest.HashSystemDirs(topicsDir)
	if err != nil {
		t.Fatalf("HashSystemDirs: %v", err)
	}

	if len(hashes) != 0 {
		t.Errorf("expected 0 entries for empty topics dir, got %d", len(hashes))
	}
}

func TestHashSystemDirs_AllDirs(t *testing.T) {
	topicsDir := t.TempDir()

	for _, name := range manifest.SystemInstructionDirs() {
		dir := filepath.Join(topicsDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	hashes, err := manifest.HashSystemDirs(topicsDir)
	if err != nil {
		t.Fatalf("HashSystemDirs: %v", err)
	}

	expected := len(manifest.SystemInstructionDirs())
	if len(hashes) != expected {
		t.Errorf("expected %d entries, got %d", expected, len(hashes))
	}

	for _, name := range manifest.SystemInstructionDirs() {
		h, ok := hashes[name]
		if !ok {
			t.Errorf("missing hash for %s", name)
			continue
		}
		if !strings.HasPrefix(h, "sha256:") {
			t.Errorf("%s hash missing sha256: prefix: %q", name, h)
		}
	}
}

// ---------------------------------------------------------------------------
// BuildHashes
// ---------------------------------------------------------------------------

func TestBuildHashes_SetsCompiledAt(t *testing.T) {
	h := manifest.BuildHashes(nil, nil)
	if h.CompiledAt == "" {
		t.Error("expected compiled_at to be set")
	}
}

func TestBuildHashes_NilMaps(t *testing.T) {
	h := manifest.BuildHashes(nil, nil)
	if h.Files == nil {
		t.Error("expected Files to be initialized (not nil)")
	}
	if h.System == nil {
		t.Error("expected System to be initialized (not nil)")
	}
}

func TestBuildHashes_WithFiles(t *testing.T) {
	files := map[string]string{
		"docs/topics/auth/jwt.md": "sha256:abc123",
		"docs/topics/api/rest.md": "sha256:def456",
	}

	h := manifest.BuildHashes(files, nil)
	if len(h.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(h.Files))
	}
	if h.Files["docs/topics/auth/jwt.md"] != "sha256:abc123" {
		t.Error("file hash mismatch")
	}
}

func TestBuildHashes_WithSystem(t *testing.T) {
	system := map[string]string{
		"taxonomy-generation": "sha256:aaa111",
		"bridge-summaries":    "sha256:bbb222",
	}

	h := manifest.BuildHashes(nil, system)
	if len(h.System) != 2 {
		t.Errorf("expected 2 system entries, got %d", len(h.System))
	}
	if h.System["taxonomy-generation"] != "sha256:aaa111" {
		t.Error("system hash mismatch")
	}
}

func TestBuildHashes_WithBoth(t *testing.T) {
	files := map[string]string{"file.md": "sha256:111"}
	system := map[string]string{"dir": "sha256:222"}

	h := manifest.BuildHashes(files, system)
	if len(h.Files) != 1 || len(h.System) != 1 {
		t.Errorf("expected 1 file and 1 system, got %d and %d", len(h.Files), len(h.System))
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func TestHashes_WriteTo_RoundTrip(t *testing.T) {
	files := map[string]string{
		"docs/topics/auth/jwt.md": "sha256:abc123def456",
		"docs/topics/api/rest.md": "sha256:789012345678",
	}
	system := map[string]string{
		"taxonomy-generation": "sha256:aaa111bbb222",
	}

	h := manifest.BuildHashes(files, system)

	dir := t.TempDir()
	path := filepath.Join(dir, "hashes.yml")

	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading hashes: %v", err)
	}

	content := string(data)

	// Should have the header comment.
	if !strings.HasPrefix(content, "# codectx content hashes") {
		t.Error("expected header comment")
	}

	// Should be valid YAML that can be unmarshaled.
	var loaded manifest.Hashes
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling hashes: %v", err)
	}

	if len(loaded.Files) != 2 {
		t.Errorf("round-trip files: expected 2, got %d", len(loaded.Files))
	}
	if len(loaded.System) != 1 {
		t.Errorf("round-trip system: expected 1, got %d", len(loaded.System))
	}
	if loaded.Files["docs/topics/auth/jwt.md"] != "sha256:abc123def456" {
		t.Error("round-trip file hash mismatch")
	}
	if loaded.System["taxonomy-generation"] != "sha256:aaa111bbb222" {
		t.Error("round-trip system hash mismatch")
	}
	if loaded.CompiledAt == "" {
		t.Error("round-trip compiled_at missing")
	}
}

func TestHashes_WriteTo_2SpaceIndent(t *testing.T) {
	h := manifest.BuildHashes(
		map[string]string{"file.md": "sha256:abc"},
		map[string]string{"dir": "sha256:def"},
	)

	dir := t.TempDir()
	path := filepath.Join(dir, "hashes.yml")
	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "\t") {
		t.Error("hashes should not contain tabs")
	}
}

func TestHashes_WriteTo_EmptyMaps(t *testing.T) {
	h := manifest.BuildHashes(nil, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "hashes.yml")
	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded manifest.Hashes
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	if len(loaded.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(loaded.Files))
	}
	if len(loaded.System) != 0 {
		t.Errorf("expected 0 system entries, got %d", len(loaded.System))
	}
}

// ---------------------------------------------------------------------------
// HashesPath
// ---------------------------------------------------------------------------

func TestHashesPath(t *testing.T) {
	got := manifest.HashesPath("/project/.codectx/compiled")
	expected := filepath.Join("/project/.codectx/compiled", "hashes.yml")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

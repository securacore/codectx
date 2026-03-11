package markdown_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/testutil"
)

func TestWalkFiles_FindsMarkdown(t *testing.T) {
	root := t.TempDir()
	testutil.MustWriteFile(t, filepath.Join(root, "README.md"), "# Root\n")
	testutil.MustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth\n")
	testutil.MustWriteFile(t, filepath.Join(root, "topics", "api.md"), "# API\n")

	files, err := markdown.WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	// Should be sorted alphabetically by relative path.
	if files[0].RelPath != "README.md" {
		t.Errorf("expected first file README.md, got %s", files[0].RelPath)
	}
	if files[1].RelPath != "topics/api.md" {
		t.Errorf("expected second file topics/api.md, got %s", files[1].RelPath)
	}
	if files[2].RelPath != "topics/auth.md" {
		t.Errorf("expected third file topics/auth.md, got %s", files[2].RelPath)
	}
}

func TestWalkFiles_SkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	testutil.MustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth\n")
	testutil.MustWriteFile(t, filepath.Join(root, ".hidden", "secret.md"), "# Secret\n")
	testutil.MustWriteFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main")

	files, err := markdown.WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file (hidden dirs skipped), got %d", len(files))
	}
	if files[0].RelPath != "topics/auth.md" {
		t.Errorf("expected topics/auth.md, got %s", files[0].RelPath)
	}
}

func TestWalkFiles_SkipsNonMarkdown(t *testing.T) {
	root := t.TempDir()
	testutil.MustWriteFile(t, filepath.Join(root, "doc.md"), "# Doc\n")
	testutil.MustWriteFile(t, filepath.Join(root, "image.png"), "PNG data")
	testutil.MustWriteFile(t, filepath.Join(root, "config.yml"), "key: value")

	files, err := markdown.WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 markdown file, got %d", len(files))
	}
}

func TestWalkFiles_EmptyDirectory(t *testing.T) {
	root := t.TempDir()

	files, err := markdown.WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files for empty dir, got %d", len(files))
	}
}

func TestWalkFiles_NonexistentDir(t *testing.T) {
	_, err := markdown.WalkFiles("/nonexistent/path/12345")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestWalkFiles_AbsPathsAreAbsolute(t *testing.T) {
	root := t.TempDir()
	testutil.MustWriteFile(t, filepath.Join(root, "doc.md"), "# Doc\n")

	files, err := markdown.WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if !filepath.IsAbs(files[0].AbsPath) {
		t.Errorf("AbsPath should be absolute, got %s", files[0].AbsPath)
	}

	// Verify the file actually exists at AbsPath.
	if _, err := os.Stat(files[0].AbsPath); err != nil {
		t.Errorf("file should exist at AbsPath %s: %v", files[0].AbsPath, err)
	}
}

func TestWalkFiles_ForwardSlashRelPaths(t *testing.T) {
	root := t.TempDir()
	testutil.MustWriteFile(t, filepath.Join(root, "a", "b", "c.md"), "# C\n")

	files, err := markdown.WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].RelPath != "a/b/c.md" {
		t.Errorf("expected forward-slash relative path, got %s", files[0].RelPath)
	}
}

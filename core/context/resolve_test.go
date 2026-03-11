package context_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/context"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/testutil"
)

// mustWriteFile delegates to the shared test helper.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	testutil.MustWriteFile(t, path, content)
}

// ---------------------------------------------------------------------------
// ParseRef
// ---------------------------------------------------------------------------

func TestParseRef_Local(t *testing.T) {
	refType, pkg, subpath := context.ParseRef("foundation/coding-standards")
	if refType != context.RefLocal {
		t.Errorf("expected RefLocal, got %v", refType)
	}
	if pkg != "" {
		t.Errorf("expected empty package, got %q", pkg)
	}
	if subpath != "foundation/coding-standards" {
		t.Errorf("expected subpath %q, got %q", "foundation/coding-standards", subpath)
	}
}

func TestParseRef_PackageSpecific(t *testing.T) {
	refType, pkg, subpath := context.ParseRef("react-patterns@community/foundation/component-principles")
	if refType != context.RefPackageSpecific {
		t.Errorf("expected RefPackageSpecific, got %v", refType)
	}
	if pkg != "react-patterns" {
		t.Errorf("expected package %q, got %q", "react-patterns", pkg)
	}
	if subpath != "foundation/component-principles" {
		t.Errorf("expected subpath %q, got %q", "foundation/component-principles", subpath)
	}
}

func TestParseRef_PackageBare(t *testing.T) {
	refType, pkg, subpath := context.ParseRef("company-standards@acme")
	if refType != context.RefPackageBare {
		t.Errorf("expected RefPackageBare, got %v", refType)
	}
	if pkg != "company-standards" {
		t.Errorf("expected package %q, got %q", "company-standards", pkg)
	}
	if subpath != "" {
		t.Errorf("expected empty subpath, got %q", subpath)
	}
}

func TestParseRef_SingleSegmentLocal(t *testing.T) {
	refType, pkg, subpath := context.ParseRef("overview.md")
	if refType != context.RefLocal {
		t.Errorf("expected RefLocal, got %v", refType)
	}
	if pkg != "" {
		t.Errorf("expected empty package, got %q", pkg)
	}
	if subpath != "overview.md" {
		t.Errorf("expected subpath %q, got %q", "overview.md", subpath)
	}
}

// ---------------------------------------------------------------------------
// Resolve — local paths
// ---------------------------------------------------------------------------

func TestResolve_LocalDirectory(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "coding-standards", "README.md"), "# Coding Standards\n")
	mustWriteFile(t, filepath.Join(root, "foundation", "coding-standards", "naming.md"), "# Naming\n")

	entries, err := context.Resolve(root, "", []string{"foundation/coding-standards"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Reference != "foundation/coding-standards" {
		t.Errorf("expected reference %q, got %q", "foundation/coding-standards", e.Reference)
	}
	if e.Type != context.RefLocal {
		t.Errorf("expected RefLocal, got %v", e.Type)
	}
	if e.Title != "Coding Standards" {
		t.Errorf("expected title %q, got %q", "Coding Standards", e.Title)
	}
	if len(e.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(e.Files))
	}

	// Files should be sorted.
	if e.Files[0].RelPath != "README.md" {
		t.Errorf("expected first file README.md, got %q", e.Files[0].RelPath)
	}
	if e.Files[1].RelPath != "naming.md" {
		t.Errorf("expected second file naming.md, got %q", e.Files[1].RelPath)
	}
}

func TestResolve_LocalSingleFile(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "overview.md"), "# Overview\n")

	entries, err := context.Resolve(root, "", []string{"foundation/overview.md"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if len(e.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(e.Files))
	}
	if e.Files[0].RelPath != "overview.md" {
		t.Errorf("expected file overview.md, got %q", e.Files[0].RelPath)
	}
}

func TestResolve_LocalSkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "standards", "README.md"), "# Standards\n")
	mustWriteFile(t, filepath.Join(root, "foundation", "standards", ".hidden", "secret.md"), "# Secret\n")

	entries, err := context.Resolve(root, "", []string{"foundation/standards"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(entries[0].Files) != 1 {
		t.Errorf("expected 1 file (hidden dir skipped), got %d", len(entries[0].Files))
	}
}

func TestResolve_LocalSkipsNonMarkdown(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "README.md"), "# Foundation\n")
	mustWriteFile(t, filepath.Join(root, "foundation", "diagram.png"), "PNG data")

	entries, err := context.Resolve(root, "", []string{"foundation"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(entries[0].Files) != 1 {
		t.Errorf("expected 1 markdown file, got %d", len(entries[0].Files))
	}
}

// ---------------------------------------------------------------------------
// Resolve — package references
// ---------------------------------------------------------------------------

func TestResolve_PackageSpecific(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, ".codectx", "packages")

	mustWriteFile(t, filepath.Join(pkgDir, "react-patterns", "foundation", "component-principles", "README.md"),
		"# Component Principles\n")

	entries, err := context.Resolve(root, pkgDir, []string{
		"react-patterns@community/foundation/component-principles",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Type != context.RefPackageSpecific {
		t.Errorf("expected RefPackageSpecific, got %v", e.Type)
	}
	if e.PackageName != "react-patterns" {
		t.Errorf("expected package name %q, got %q", "react-patterns", e.PackageName)
	}
	if e.Title != "Component Principles" {
		t.Errorf("expected title %q, got %q", "Component Principles", e.Title)
	}
	if len(e.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(e.Files))
	}
}

func TestResolve_PackageBare(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, ".codectx", "packages")

	mustWriteFile(t, filepath.Join(pkgDir, "company-standards", "foundation", "coding.md"), "# Coding\n")
	mustWriteFile(t, filepath.Join(pkgDir, "company-standards", "topics", "review.md"), "# Review\n")

	entries, err := context.Resolve(root, pkgDir, []string{"company-standards@acme"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Type != context.RefPackageBare {
		t.Errorf("expected RefPackageBare, got %v", e.Type)
	}
	if e.Title != "Company Standards" {
		t.Errorf("expected title %q, got %q", "Company Standards", e.Title)
	}
	if len(e.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(e.Files))
	}
}

// ---------------------------------------------------------------------------
// Resolve — multiple entries
// ---------------------------------------------------------------------------

func TestResolve_MultipleEntries(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "coding", "README.md"), "# Coding\n")
	mustWriteFile(t, filepath.Join(root, "foundation", "arch", "README.md"), "# Architecture\n")

	entries, err := context.Resolve(root, "", []string{
		"foundation/coding",
		"foundation/arch",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Order matches input order.
	if entries[0].Title != "Coding" {
		t.Errorf("expected first entry title %q, got %q", "Coding", entries[0].Title)
	}
	if entries[1].Title != "Arch" {
		t.Errorf("expected second entry title %q, got %q", "Arch", entries[1].Title)
	}
}

// ---------------------------------------------------------------------------
// Resolve — error cases
// ---------------------------------------------------------------------------

func TestResolve_NonexistentPath(t *testing.T) {
	root := t.TempDir()

	_, err := context.Resolve(root, "", []string{"foundation/nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestResolve_EmptyAlwaysLoaded(t *testing.T) {
	root := t.TempDir()

	entries, err := context.Resolve(root, "", nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestResolve_SkipsEmptyStrings(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "foundation", "README.md"), "# Foundation\n")

	entries, err := context.Resolve(root, "", []string{"", "foundation", ""})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (empty strings skipped), got %d", len(entries))
	}
}

func TestResolve_DirectoryWithNoMarkdown(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "foundation", "images")
	if err := os.MkdirAll(dir, project.DirPerm); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "logo.png"), "PNG data")

	_, err := context.Resolve(root, "", []string{"foundation/images"})
	if err == nil {
		t.Fatal("expected error for directory with no markdown files")
	}
}

// ---------------------------------------------------------------------------
// Title derivation
// ---------------------------------------------------------------------------

func TestResolve_TitleFromNestedPath(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "foundation", "error-handling", "README.md"), "# Error Handling\n")

	entries, err := context.Resolve(root, "", []string{"foundation/error-handling"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if entries[0].Title != "Error Handling" {
		t.Errorf("expected title %q, got %q", "Error Handling", entries[0].Title)
	}
}

// ---------------------------------------------------------------------------
// RefType.String
// ---------------------------------------------------------------------------

func TestRefType_String(t *testing.T) {
	if context.RefLocal.String() != "local" {
		t.Errorf("expected %q, got %q", "local", context.RefLocal.String())
	}
	if context.RefPackageSpecific.String() != "package" {
		t.Errorf("expected %q, got %q", "package", context.RefPackageSpecific.String())
	}
	if context.RefPackageBare.String() != "package" {
		t.Errorf("expected %q, got %q", "package", context.RefPackageBare.String())
	}
}

package embed_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/embed"
)

func TestSystemFiles_ReturnsAllExpectedFiles(t *testing.T) {
	files := embed.SystemFiles()

	if len(files) != 3 {
		t.Fatalf("expected 3 system files, got %d", len(files))
	}

	// Verify all dest paths start with "system/".
	for _, f := range files {
		if !strings.HasPrefix(f.DestPath, "system/") {
			t.Errorf("expected dest path to start with system/, got %q", f.DestPath)
		}
	}

	// Verify all embed paths start with "defaults/".
	for _, f := range files {
		if !strings.HasPrefix(f.EmbedPath, "defaults/") {
			t.Errorf("expected embed path to start with defaults/, got %q", f.EmbedPath)
		}
	}
}

func TestReadFile_ReadsEmbeddedContent(t *testing.T) {
	files := embed.SystemFiles()

	for _, f := range files {
		data, err := embed.ReadFile(f.EmbedPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", f.EmbedPath, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("expected %s to have content", f.EmbedPath)
		}
		// All default files should start with a markdown heading.
		if !strings.HasPrefix(string(data), "# ") {
			t.Errorf("expected %s to start with markdown heading, got %q", f.EmbedPath, string(data[:20]))
		}
	}
}

func TestReadFile_ErrorsOnMissingFile(t *testing.T) {
	_, err := embed.ReadFile("defaults/nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSystemFiles_DestPathsAreUnique(t *testing.T) {
	files := embed.SystemFiles()
	seen := make(map[string]bool)

	for _, f := range files {
		if seen[f.DestPath] {
			t.Errorf("duplicate dest path: %s", f.DestPath)
		}
		seen[f.DestPath] = true
	}
}

func TestSystemFiles_CoversExpectedTopics(t *testing.T) {
	files := embed.SystemFiles()

	expectedPaths := map[string]bool{
		"system/foundation/documentation-protocol/README.md": false,
		"system/foundation/history/README.md":                false,
		"system/topics/context-assembly/README.md":           false,
	}

	for _, f := range files {
		if _, ok := expectedPaths[f.DestPath]; ok {
			expectedPaths[f.DestPath] = true
		}
	}

	for path, found := range expectedPaths {
		if !found {
			t.Errorf("expected system file not found: %s", path)
		}
	}
}

// --- PackageTemplateFiles ---

func TestPackageTemplateFiles_ReturnsExpectedFiles(t *testing.T) {
	files := embed.PackageTemplateFiles()

	if len(files) != 1 {
		t.Fatalf("expected 1 package template file, got %d", len(files))
	}

	// Verify dest path is the release workflow.
	if files[0].DestPath != ".github/workflows/release.yml" {
		t.Errorf("expected dest path %q, got %q", ".github/workflows/release.yml", files[0].DestPath)
	}

	// Verify embed path is within package/.
	if !strings.HasPrefix(files[0].EmbedPath, "package/") {
		t.Errorf("expected embed path to start with package/, got %q", files[0].EmbedPath)
	}
}

func TestReadPackageFile_ReadsContent(t *testing.T) {
	data, err := embed.ReadPackageFile("package/release.yml")
	if err != nil {
		t.Fatalf("failed to read package/release.yml: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected package/release.yml to have content")
	}
}

func TestReadPackageFile_ErrorsOnMissing(t *testing.T) {
	_, err := embed.ReadPackageFile("package/nonexistent.yml")
	if err == nil {
		t.Error("expected error for nonexistent package file")
	}
}

// --- Config templates ---

func TestReadConfigTemplate_ReadsAllTemplates(t *testing.T) {
	templates := []string{
		"codectx.yml",
		"ai.yml",
		"preferences.yml",
		"package-codectx.yml",
		"usage.yml",
		"global-usage.yml",
	}

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			data, err := embed.ReadConfigTemplate(name)
			if err != nil {
				t.Fatalf("failed to read config template %q: %v", name, err)
			}
			if len(data) == 0 {
				t.Errorf("expected config template %q to have content", name)
			}
			if !strings.HasPrefix(string(data), "#") {
				t.Errorf("expected config template %q to start with YAML comment (#), got %q", name, string(data[:20]))
			}
		})
	}
}

func TestReadConfigTemplate_ErrorsOnMissing(t *testing.T) {
	_, err := embed.ReadConfigTemplate("nonexistent.yml")
	if err == nil {
		t.Error("expected error for nonexistent config template")
	}
}

func TestConfigTemplatePath_Format(t *testing.T) {
	got := embed.ConfigTemplatePath("ai.yml")
	want := "defaults/config/ai.yml.tmpl"
	if got != want {
		t.Errorf("ConfigTemplatePath(%q) = %q, want %q", "ai.yml", got, want)
	}
}

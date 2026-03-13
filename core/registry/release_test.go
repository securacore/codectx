package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v68/github"
)

func TestReleaseAssetURL_Found(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/community/codectx-react-patterns/releases/tags/v2.0.0", func(w http.ResponseWriter, _ *http.Request) {
		release := github.RepositoryRelease{
			TagName: github.Ptr("v2.0.0"),
			Assets: []*github.ReleaseAsset{
				{
					Name:               github.Ptr("package.tar.gz"),
					BrowserDownloadURL: github.Ptr("https://github.com/community/codectx-react-patterns/releases/download/v2.0.0/package.tar.gz"),
				},
				{
					Name:               github.Ptr("checksums.txt"),
					BrowserDownloadURL: github.Ptr("https://github.com/community/codectx-react-patterns/releases/download/v2.0.0/checksums.txt"),
				},
			},
		}
		jsonRespond(w, release)
	})

	gh, _ := newTestGitHubClient(t, mux)
	url, err := gh.ReleaseAssetURL(context.Background(), "community", "codectx-react-patterns", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "https://github.com/community/codectx-react-patterns/releases/download/v2.0.0/package.tar.gz"
	if url != expected {
		t.Errorf("url: got %q, want %q", url, expected)
	}
}

func TestReleaseAssetURL_NoMatchingAsset(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/org/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		release := github.RepositoryRelease{
			TagName: github.Ptr("v1.0.0"),
			Assets: []*github.ReleaseAsset{
				{
					Name:               github.Ptr("source.tar.gz"),
					BrowserDownloadURL: github.Ptr("https://example.com/source.tar.gz"),
				},
			},
		}
		jsonRespond(w, release)
	})

	gh, _ := newTestGitHubClient(t, mux)
	_, err := gh.ReleaseAssetURL(context.Background(), "org", "repo", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for missing package.tar.gz asset")
	}
}

func TestReleaseAssetURL_NoRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/org/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		jsonRespond(w, github.ErrorResponse{Message: "Not Found"})
	})

	gh, _ := newTestGitHubClient(t, mux)
	_, err := gh.ReleaseAssetURL(context.Background(), "org", "repo", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for missing release")
	}
}

func TestReleaseAssetURLForDep(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/community/codectx-react-patterns/releases/tags/v2.0.0", func(w http.ResponseWriter, _ *http.Request) {
		release := github.RepositoryRelease{
			TagName: github.Ptr("v2.0.0"),
			Assets: []*github.ReleaseAsset{
				{
					Name:               github.Ptr("package.tar.gz"),
					BrowserDownloadURL: github.Ptr("https://example.com/package.tar.gz"),
				},
			},
		}
		jsonRespond(w, release)
	})

	gh, _ := newTestGitHubClient(t, mux)
	dk := DepKey{Name: "react-patterns", Author: "community"}
	url, err := gh.ReleaseAssetURLForDep(context.Background(), dk, "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/package.tar.gz" {
		t.Errorf("url: got %q, want %q", url, "https://example.com/package.tar.gz")
	}
}

func TestArchiveConfigReader_ReadDeps(t *testing.T) {
	// Create a temp dir to act as cache.
	cacheDir := t.TempDir()

	// Pre-populate the cache with a codectx.yml.
	pkgDir := filepath.Join(cacheDir, "test@org-1.0.0")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgContent := `name: test
author: org
version: 1.0.0
dependencies:
  dep-a@other: ">=1.0.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "codectx.yml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// ArchiveConfigReader with nil GH/HTTP — should use cache, not network.
	reader := &ArchiveConfigReader{CacheDir: cacheDir}
	dk := DepKey{Name: "test", Author: "org"}

	deps, err := reader.ReadDeps(context.Background(), dk, "1.0.0", "github.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
	if deps["dep-a@other"] != ">=1.0.0" {
		t.Errorf("dep-a@other: got %q, want %q", deps["dep-a@other"], ">=1.0.0")
	}
}

func TestArchiveConfigReader_ReadDeps_CacheMiss(t *testing.T) {
	// Create a mock server that serves the release asset endpoint but
	// returns 404 — simulating a missing release.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/org/codectx-test/releases/tags/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	gh, _ := newTestGitHubClient(t, mux)
	reader := &ArchiveConfigReader{
		GH:       gh,
		HTTP:     http.DefaultClient,
		CacheDir: t.TempDir(),
	}

	dk := DepKey{Name: "test", Author: "org"}
	_, err := reader.ReadDeps(context.Background(), dk, "1.0.0", "github.com")
	if err == nil {
		t.Fatal("expected error when release not found and cache empty")
	}
}

func TestLoadPackageConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "codectx.yml")

	content := `name: my-pkg
author: myorg
version: 2.0.0
dependencies:
  dep-a@other: ">=1.0.0"
  dep-b@another: "latest"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadPackageConfigFromFile(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "my-pkg" {
		t.Errorf("name: got %q, want %q", cfg.Name, "my-pkg")
	}
	if len(cfg.Dependencies) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(cfg.Dependencies))
	}
}

func TestLoadPackageConfigFromFile_Missing(t *testing.T) {
	_, err := loadPackageConfigFromFile("/nonexistent/codectx.yml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v68/github"
)

// jsonRespond writes v as a JSON response. Error is intentionally ignored
// since this is only used in test HTTP handlers.
func jsonRespond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v) //nolint:errcheck // test helper
}

// newTestGitHubClient creates a GitHubClient backed by a test HTTP server.
// The caller provides handler routes via the mux.
func newTestGitHubClient(t *testing.T, mux *http.ServeMux) (*GitHubClient, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := github.NewClient(nil)
	serverURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = serverURL

	return &GitHubClient{client: client}, server
}

func TestSearchPackages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search/repositories", func(w http.ResponseWriter, r *http.Request) {
		result := github.RepositoriesSearchResult{
			Total: github.Ptr(2),
			Repositories: []*github.Repository{
				{
					Name:            github.Ptr("codectx-react-patterns"),
					FullName:        github.Ptr("community/codectx-react-patterns"),
					Description:     github.Ptr("React component patterns"),
					StargazersCount: github.Ptr(342),

					Owner: &github.User{
						Login: github.Ptr("community"),
					},
				},
				{
					Name:            github.Ptr("codectx-react-testing"),
					FullName:        github.Ptr("community/codectx-react-testing"),
					Description:     github.Ptr("React testing patterns"),
					StargazersCount: github.Ptr(89),

					Owner: &github.User{
						Login: github.Ptr("community"),
					},
				},
			},
		}
		jsonRespond(w, result)
	})

	gh, _ := newTestGitHubClient(t, mux)
	results, err := gh.SearchPackages(context.Background(), "react", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r1 := results[0]
	if r1.Name != "react-patterns" {
		t.Errorf("name: got %q, want %q", r1.Name, "react-patterns")
	}
	if r1.Author != "community" {
		t.Errorf("author: got %q, want %q", r1.Author, "community")
	}
	if r1.Stars != 342 {
		t.Errorf("stars: got %d, want %d", r1.Stars, 342)
	}
	if r1.Description != "React component patterns" {
		t.Errorf("description: got %q, want %q", r1.Description, "React component patterns")
	}

	r2 := results[1]
	if r2.Name != "react-testing" {
		t.Errorf("name: got %q, want %q", r2.Name, "react-testing")
	}
}

func TestSearchPackages_Empty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search/repositories", func(w http.ResponseWriter, _ *http.Request) {
		result := github.RepositoriesSearchResult{
			Total:        github.Ptr(0),
			Repositories: []*github.Repository{},
		}
		jsonRespond(w, result)
	})

	gh, _ := newTestGitHubClient(t, mux)
	results, err := gh.SearchPackages(context.Background(), "nonexistent", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchPackages_SkipsShortNames(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search/repositories", func(w http.ResponseWriter, _ *http.Request) {
		result := github.RepositoriesSearchResult{
			Total: github.Ptr(1),
			Repositories: []*github.Repository{
				{
					// Name too short to have a package name after stripping prefix.
					Name:     github.Ptr("codectx-"),
					FullName: github.Ptr("user/codectx-"),
					Owner:    &github.User{Login: github.Ptr("user")},
				},
			},
		}
		jsonRespond(w, result)
	})

	gh, _ := newTestGitHubClient(t, mux)
	results, err := gh.SearchPackages(context.Background(), "test", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (short name filtered), got %d", len(results))
	}
}

func TestListRepoTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/community/codectx-react-patterns/tags", func(w http.ResponseWriter, _ *http.Request) {
		tags := []*github.RepositoryTag{
			{Name: github.Ptr("v2.3.1")},
			{Name: github.Ptr("v2.3.0")},
			{Name: github.Ptr("v2.2.0")},
			{Name: github.Ptr("v1.0.0")},
		}
		jsonRespond(w, tags)
	})

	gh, _ := newTestGitHubClient(t, mux)
	tags, err := gh.ListRepoTags(context.Background(), "community", "codectx-react-patterns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tags) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(tags))
	}
	if tags[0] != "v2.3.1" {
		t.Errorf("first tag: got %q, want %q", tags[0], "v2.3.1")
	}
}

func TestListRepoTags_Pagination(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/org/repo/tags", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		page := r.URL.Query().Get("page")

		var tags []*github.RepositoryTag
		if page == "" || page == "1" {
			tags = []*github.RepositoryTag{
				{Name: github.Ptr("v2.0.0")},
				{Name: github.Ptr("v1.1.0")},
			}
			// Signal there's a next page.
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/org/repo/tags?page=2>; rel="next"`, r.Host))
		} else {
			tags = []*github.RepositoryTag{
				{Name: github.Ptr("v1.0.0")},
			}
		}

		jsonRespond(w, tags)
	})

	gh, _ := newTestGitHubClient(t, mux)
	tags, err := gh.ListRepoTags(context.Background(), "org", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tags) != 3 {
		t.Fatalf("expected 3 tags across pages, got %d: %v", len(tags), tags)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (pagination), got %d", callCount)
	}
}

func TestSearchPackages_DefaultLimit(t *testing.T) {
	// When limit <= 0, SearchPackages should default to 10.
	var capturedPerPage int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search/repositories", func(w http.ResponseWriter, r *http.Request) {
		capturedPerPage = 0
		pp := r.URL.Query().Get("per_page")
		if pp != "" {
			_, _ = fmt.Sscanf(pp, "%d", &capturedPerPage)
		}

		result := github.RepositoriesSearchResult{
			Total:        github.Ptr(0),
			Repositories: []*github.Repository{},
		}
		jsonRespond(w, result)
	})

	gh, _ := newTestGitHubClient(t, mux)
	_, err := gh.SearchPackages(context.Background(), "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPerPage != 10 {
		t.Errorf("expected per_page=10 (default), got %d", capturedPerPage)
	}

	_, err = gh.SearchPackages(context.Background(), "test", -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPerPage != 10 {
		t.Errorf("expected per_page=10 for negative limit, got %d", capturedPerPage)
	}
}

func TestNewGitHubClient(t *testing.T) {
	gh := NewGitHubClient("")
	if gh == nil {
		t.Fatal("expected non-nil client")
	}
	if gh.client == nil {
		t.Fatal("expected non-nil underlying github client")
	}
}

func TestNewGitHubClient_WithToken(t *testing.T) {
	// Verify that providing a token results in authenticated requests.
	var authHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search/repositories", func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		result := github.RepositoriesSearchResult{
			Total:        github.Ptr(0),
			Repositories: []*github.Repository{},
		}
		jsonRespond(w, result)
	})

	gh, _ := newTestGitHubClient(t, mux)
	// Override the client with an authenticated one pointing at our test server.
	gh.client = gh.client.WithAuthToken("test-token-xyz")

	_, err := gh.SearchPackages(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authHeader == "" {
		t.Error("expected Authorization header to be set with token")
	}
	if authHeader != "Bearer test-token-xyz" {
		t.Errorf("unexpected auth header: got %q, want %q", authHeader, "Bearer test-token-xyz")
	}
}

func TestGitHubToken_Env(t *testing.T) {
	// Both unset.
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	if got := GitHubToken(); got != "" {
		t.Errorf("expected empty token, got %q", got)
	}

	// GH_TOKEN only.
	t.Setenv("GH_TOKEN", "gh-cli-token")
	if got := GitHubToken(); got != "gh-cli-token" {
		t.Errorf("expected %q from GH_TOKEN, got %q", "gh-cli-token", got)
	}

	// GITHUB_TOKEN takes precedence.
	t.Setenv("GITHUB_TOKEN", "actions-token")
	if got := GitHubToken(); got != "actions-token" {
		t.Errorf("expected %q from GITHUB_TOKEN, got %q", "actions-token", got)
	}
}

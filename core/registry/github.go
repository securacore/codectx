package registry

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v68/github"
)

// GitHubToken returns a GitHub personal access token from the environment.
// It checks GITHUB_TOKEN first, then falls back to GH_TOKEN (used by the
// GitHub CLI). Returns an empty string if neither is set.
func GitHubToken() string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("GH_TOKEN")
}

// GitHubClient wraps the go-github client for package discovery and metadata.
// When created without a token, uses unauthenticated access (60 requests/hour).
// With a token (GITHUB_TOKEN), the rate limit increases to 5,000 requests/hour.
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a GitHub API client. If token is non-empty, the
// client authenticates with it (raising the rate limit to 5,000 req/hr).
// If token is empty, the client is unauthenticated (60 req/hr).
func NewGitHubClient(token string) *GitHubClient {
	client := github.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}
	return &GitHubClient{client: client}
}

// SearchResult represents a single package found via GitHub search.
type SearchResult struct {
	// Name is the package name without the codectx- prefix.
	Name string

	// Org is the GitHub owner/organization.
	Org string

	// FullName is the full "owner/repo" path on GitHub.
	FullName string

	// Description is the repository description.
	Description string

	// Stars is the number of GitHub stars.
	Stars int

	// LatestVersion is the most recent semver tag (resolved separately).
	LatestVersion string
}

// SearchPackages searches GitHub for codectx packages matching the query.
// The query is combined with the codectx- prefix convention to find packages.
//
// Results are sorted by stars (descending) to surface popular packages first.
func (gh *GitHubClient) SearchPackages(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Build the GitHub search query: "codectx- <user query> in:name"
	searchQuery := fmt.Sprintf("codectx- %s in:name", query)

	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: limit,
		},
	}

	result, _, err := gh.client.Search.Repositories(ctx, searchQuery, opts)
	if err != nil {
		return nil, fmt.Errorf("searching GitHub: %w", err)
	}

	var results []SearchResult
	for _, repo := range result.Repositories {
		name := repo.GetName()
		if len(name) <= len(RepoPrefix) {
			continue
		}

		// Strip the codectx- prefix to get the package name.
		pkgName := name[len(RepoPrefix):]

		org := ""
		if repo.GetOwner() != nil {
			org = repo.GetOwner().GetLogin()
		}

		results = append(results, SearchResult{
			Name:        pkgName,
			Org:         org,
			FullName:    repo.GetFullName(),
			Description: repo.GetDescription(),
			Stars:       repo.GetStargazersCount(),
		})
	}

	return results, nil
}

// ListRepoTags fetches all version tags for a GitHub repository.
// Returns tag names (e.g. "v1.0.0", "v2.3.1") with pagination.
func (gh *GitHubClient) ListRepoTags(ctx context.Context, owner, repo string) ([]string, error) {
	var allTags []string
	opts := &github.ListOptions{PerPage: 100}

	for {
		tags, resp, err := gh.client.Repositories.ListTags(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("listing tags for %s/%s: %w", owner, repo, err)
		}

		for _, tag := range tags {
			allTags = append(allTags, tag.GetName())
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allTags, nil
}

package resolve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	repoPrefix = "codectx-"
)

// searchBaseURL is the base URL for the GitHub search API.
// Tests override this to point at an httptest server.
var searchBaseURL = "https://api.github.com/search/repositories"

// SearchResult holds a single search result from the GitHub API.
type SearchResult struct {
	Name        string // package name (codectx- prefix stripped)
	Author      string // GitHub owner login
	Description string
	URL         string // repository HTML URL
	Stars       int
}

// githubSearchResponse maps the GitHub search API response.
type githubSearchResponse struct {
	TotalCount int                `json:"total_count"`
	Items      []githubSearchItem `json:"items"`
}

type githubSearchItem struct {
	FullName    string      `json:"full_name"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	HTMLURL     string      `json:"html_url"`
	Stars       int         `json:"stargazers_count"`
	Owner       githubOwner `json:"owner"`
}

type githubOwner struct {
	Login string `json:"login"`
}

// Search queries GitHub for codectx packages matching the given query.
// If author is non-empty, results are scoped to that user/organization.
func Search(query, author string) ([]SearchResult, error) {
	q := fmt.Sprintf("%s%s in:name", repoPrefix, query)
	if author != "" {
		q += fmt.Sprintf(" user:%s", author)
	}

	reqURL := fmt.Sprintf("%s?q=%s&sort=stars&order=desc&per_page=25",
		searchBaseURL, url.QueryEscape(q))

	return doSearch(reqURL)
}

// doSearch performs the HTTP request and parses the response.
// Extracted for testability.
func doSearch(reqURL string) ([]SearchResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	return parseSearchResponse(resp.Body)
}

// parseSearchResponse reads and parses the GitHub search response body.
func parseSearchResponse(r io.Reader) ([]SearchResult, error) {
	var ghResp githubSearchResponse
	if err := json.NewDecoder(r).Decode(&ghResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var results []SearchResult
	for _, item := range ghResp.Items {
		// Only include repos that follow the codectx- naming convention.
		if !strings.HasPrefix(item.Name, repoPrefix) {
			continue
		}

		name := strings.TrimPrefix(item.Name, repoPrefix)
		results = append(results, SearchResult{
			Name:        name,
			Author:      item.Owner.Login,
			Description: item.Description,
			URL:         item.HTMLURL,
			Stars:       item.Stars,
		})
	}

	return results, nil
}

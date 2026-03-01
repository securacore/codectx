package resolve

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseURL extracts a PackageRef and source URL from a raw GitHub URL.
// It validates the repository follows the codectx- naming convention.
//
// Accepted forms:
//   - https://github.com/org/codectx-react
//   - https://github.com/org/codectx-react.git
//   - https://github.com/org/codectx-react/tree/v1.2.0
//
// Returns the PackageRef (with version if extracted from /tree/ref)
// and the Git clone URL.
func ParseURL(rawURL string) (*PackageRef, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	// Validate GitHub host.
	if u.Host != "github.com" && u.Host != "www.github.com" {
		return nil, "", fmt.Errorf("only GitHub URLs are supported, got host %q", u.Host)
	}

	// Parse path segments: /author/repo[/tree/ref]
	path := strings.Trim(u.Path, "/")
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return nil, "", fmt.Errorf("url must contain owner and repository: %s", rawURL)
	}

	author := segments[0]
	repo := segments[1]

	// Strip .git suffix if present.
	repo = strings.TrimSuffix(repo, ".git")

	// Validate codectx- naming convention.
	if !strings.HasPrefix(repo, repoPrefix) {
		return nil, "", fmt.Errorf("repository %q does not follow the codectx- naming convention", repo)
	}

	name := strings.TrimPrefix(repo, repoPrefix)
	if name == "" {
		return nil, "", fmt.Errorf("empty package name after stripping codectx- prefix from %q", repo)
	}

	ref := &PackageRef{
		Name:   name,
		Author: author,
	}

	// Extract version hint from /tree/<ref> path.
	if len(segments) >= 4 && segments[2] == "tree" {
		versionHint := segments[3]
		ref.Version = strings.TrimPrefix(versionHint, "v")
	}

	// Build the clone URL.
	source := fmt.Sprintf("https://github.com/%s/%s%s.git", author, repoPrefix, name)

	return ref, source, nil
}

// IsURL returns true if the input looks like a URL.
func IsURL(input string) bool {
	return strings.Contains(input, "://")
}

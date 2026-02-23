package resolve

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Resolve connects to a remote Git repository, lists semver tags,
// and finds the best version matching the constraint in ref.Version.
// If ref.Version is empty, the latest semver tag is selected.
func Resolve(ref *PackageRef, source string) (*ResolvedPackage, error) {
	// List remote references (tags) without cloning.
	remote := git.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{source},
	})

	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list remote %s: %w", source, err)
	}

	// Collect semver tags.
	var versions []*semver.Version
	tagMap := make(map[string]string) // version string -> tag name

	for _, r := range refs {
		name := r.Name().Short()
		if !r.Name().IsTag() {
			continue
		}

		// Strip "v" prefix for semver parsing.
		versionStr := strings.TrimPrefix(name, "v")
		v, err := semver.NewVersion(versionStr)
		if err != nil {
			continue // skip non-semver tags
		}

		versions = append(versions, v)
		tagMap[v.String()] = name
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no semver tags found in %s", source)
	}

	matched, err := matchVersion(versions, ref.Version)
	if err != nil {
		return nil, err
	}

	tag := tagMap[matched.String()]

	return &ResolvedPackage{
		Name:    ref.Name,
		Author:  ref.Author,
		Version: matched.String(),
		Source:  source,
		Tag:     tag,
	}, nil
}

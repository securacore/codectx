package resolve

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// matchVersion finds the best matching version from a list of versions.
// If constraint is empty, the latest version is returned.
// If constraint is provided, the highest version satisfying it is returned.
func matchVersion(versions []*semver.Version, constraint string) (*semver.Version, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions provided")
	}

	if constraint == "" {
		// No constraint: pick the latest version.
		best := versions[0]
		for _, v := range versions[1:] {
			if v.GreaterThan(best) {
				best = v
			}
		}
		return best, nil
	}

	// Parse the version constraint.
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return nil, fmt.Errorf("invalid version constraint %q: %w", constraint, err)
	}

	var matched *semver.Version
	for _, v := range versions {
		if c.Check(v) {
			if matched == nil || v.GreaterThan(matched) {
				matched = v
			}
		}
	}

	if matched == nil {
		return nil, fmt.Errorf("no version matching %q", constraint)
	}

	return matched, nil
}

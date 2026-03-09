package resolve

import "fmt"

// Parse parses a package identifier string into a PackageRef.
// Accepted forms:
//   - "name"                -> Name only
//   - "name:version"        -> Name + version constraint
//   - "name@author"         -> Name + author
//   - "name@author:version" -> Fully qualified
func Parse(input string) (*PackageRef, error) {
	if input == "" {
		return nil, fmt.Errorf("empty package identifier")
	}

	ref := &PackageRef{}

	// Split on ":" first to separate version.
	nameAuthor := input
	colonIdx := lastIndex(input, ':')
	if colonIdx >= 0 {
		nameAuthor = input[:colonIdx]
		ref.Version = input[colonIdx+1:]
		if ref.Version == "" {
			return nil, fmt.Errorf("empty version in %q", input)
		}
	}

	// Split name@author on "@".
	atIdx := lastIndex(nameAuthor, '@')
	if atIdx >= 0 {
		ref.Name = nameAuthor[:atIdx]
		ref.Author = nameAuthor[atIdx+1:]
		if ref.Author == "" {
			return nil, fmt.Errorf("empty author in %q", input)
		}
	} else {
		ref.Name = nameAuthor
	}

	if ref.Name == "" {
		return nil, fmt.Errorf("empty package name in %q", input)
	}

	return ref, nil
}

// lastIndex returns the index of the last occurrence of sep in s,
// or -1 if not found.
func lastIndex(s string, sep byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep {
			return i
		}
	}
	return -1
}

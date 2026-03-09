package resolve

import "fmt"

// InferSource constructs a Git repository URL from a package name and author.
// Convention: name@author maps to https://github.com/[author]/codectx-[name].git
func InferSource(name, author string) string {
	return fmt.Sprintf("https://github.com/%s/codectx-%s.git", author, name)
}

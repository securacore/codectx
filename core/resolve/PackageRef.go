package resolve

// PackageRef represents a parsed package identifier.
// It holds the components extracted from shorthand forms like
// "name", "name:version", "name@author", or "name@author:version".
type PackageRef struct {
	Name    string
	Author  string
	Version string
}

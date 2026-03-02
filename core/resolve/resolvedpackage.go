package resolve

// ResolvedPackage holds the result of a successful version resolution.
type ResolvedPackage struct {
	Name    string
	Author  string
	Version string // exact resolved version (e.g., "1.2.0")
	Source  string // Git repository URL
	Tag     string // Git tag name (e.g., "v1.2.0")
}

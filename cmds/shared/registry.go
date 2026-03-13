package shared

import (
	"fmt"

	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
)

// PrintConflicts prints styled warnings for any version conflicts
// found during dependency resolution. Used by both install and update.
func PrintConflicts(conflicts []registry.Conflict) {
	for _, conflict := range conflicts {
		fmt.Print(tui.WarnMsg{
			Title: fmt.Sprintf("Version conflict: %s", conflict.PackageRef),
			Detail: func() []string {
				var lines []string
				for requester, version := range conflict.Versions {
					lines = append(lines, fmt.Sprintf("%s requires %s", requester, version))
				}
				return lines
			}(),
		}.Render())
	}
}

// FilterInstallable removes search results that don't have a release archive.
// Returns the filtered list and the count of hidden results.
func FilterInstallable(results []registry.SearchResult) ([]registry.SearchResult, int) {
	filtered := make([]registry.SearchResult, 0, len(results))
	hidden := 0
	for _, r := range results {
		if r.HasRelease {
			filtered = append(filtered, r)
		} else {
			hidden++
		}
	}
	return filtered, hidden
}

// SaveLockOrError writes the lock file, printing a styled error if it fails.
// Returns nil on success or the write error on failure.
func SaveLockOrError(lockPath string, result *registry.ResolveResult, commitSHAs map[string]string, reg string) error {
	lf := registry.ToLockFile(result, commitSHAs, reg)
	if err := registry.SaveLock(lockPath, lf); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to write lock file",
			Detail: []string{err.Error()},
		}.Render())
		return err
	}
	return nil
}

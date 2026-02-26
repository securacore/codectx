package manifest

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Discover scans standard documentation directories under pkgDir and returns
// a new manifest containing all entries from existing plus any entries found
// on disk that are not already declared. It merges missing entries into every
// section, regardless of whether the section already has some entries.
//
// The four standard directories are:
//   - foundation/*/README.md → FoundationEntry (ID from directory name, Files from sibling .md files)
//   - topics/*/README.md   → TopicEntry (ID from directory name, Files from sibling .md files)
//   - prompts/*/README.md  → PromptEntry (ID from directory name)
//   - plans/*/README.md    → PlanEntry (ID from directory name, PlanState from plan.yml if present)
//
// Description is extracted from the first markdown heading (# ...) in each
// entry's primary file. If no heading is found, the ID is used as a fallback.
//
// Missing directories are silently skipped. Unreadable files are skipped.
// The returned manifest preserves all metadata from existing.
func Discover(pkgDir string, existing *Manifest) *Manifest {
	// Start with a copy of existing entries.
	result := &Manifest{
		Name:        existing.Name,
		Author:      existing.Author,
		Version:     existing.Version,
		Description: existing.Description,
		Foundation:  copyFoundation(existing.Foundation),
		Application: copyApplication(existing.Application),
		Topics:      copyTopics(existing.Topics),
		Prompts:     copyPrompts(existing.Prompts),
		Plans:       copyPlans(existing.Plans),
	}

	// Build sets of existing IDs for each section.
	foundationIDs := idSet(existing.Foundation, func(e FoundationEntry) string { return e.ID })
	applicationIDs := idSet(existing.Application, func(e ApplicationEntry) string { return e.ID })
	topicIDs := idSet(existing.Topics, func(e TopicEntry) string { return e.ID })
	promptIDs := idSet(existing.Prompts, func(e PromptEntry) string { return e.ID })
	planIDs := idSet(existing.Plans, func(e PlanEntry) string { return e.ID })

	// Discover foundation entries: foundation/*.md
	discoverFoundation(pkgDir, foundationIDs, result)

	// Discover application entries: application/*/README.md
	discoverApplication(pkgDir, applicationIDs, result)

	// Discover topic entries: topics/*/README.md
	discoverTopics(pkgDir, topicIDs, result)

	// Discover prompt entries: prompts/*/README.md
	discoverPrompts(pkgDir, promptIDs, result)

	// Discover plan entries: plans/*/README.md
	discoverPlans(pkgDir, planIDs, result)

	return result
}

// discoverFoundation scans foundation/ for subdirectories containing README.md.
func discoverFoundation(pkgDir string, existing map[string]bool, result *Manifest) {
	dir := filepath.Join(pkgDir, "foundation")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // directory missing or unreadable
	}

	// Sort by name for deterministic ordering.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		id := e.Name()
		if existing[id] {
			continue
		}

		readmePath := filepath.Join("foundation", id, "README.md")
		absReadme := filepath.Join(pkgDir, readmePath)
		if _, err := os.Stat(absReadme); err != nil {
			continue // no README.md, skip
		}

		desc := extractDescription(absReadme, id)

		entry := FoundationEntry{
			ID:          id,
			Path:        readmePath,
			Description: desc,
		}

		// Discover sub-files (other .md files in the foundation directory).
		entry.Files = discoverFoundationFiles(pkgDir, id)

		// Check for spec directory.
		specPath := filepath.Join("foundation", id, "spec", "README.md")
		if _, err := os.Stat(filepath.Join(pkgDir, specPath)); err == nil {
			entry.Spec = specPath
		}

		result.Foundation = append(result.Foundation, entry)
	}
}

// discoverFoundationFiles finds additional .md files in a foundation directory
// (excluding README.md) and returns their paths relative to the package root.
func discoverFoundationFiles(pkgDir, foundationID string) []string {
	dir := filepath.Join(pkgDir, "foundation", foundationID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		files = append(files, filepath.Join("foundation", foundationID, e.Name()))
	}
	return files
}

// discoverApplication scans application/ for subdirectories containing README.md.
// Application entries follow the same pattern as topics (spec, files) plus
// support for the load field.
func discoverApplication(pkgDir string, existing map[string]bool, result *Manifest) {
	dir := filepath.Join(pkgDir, "application")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		id := e.Name()
		if existing[id] {
			continue
		}

		readmePath := filepath.Join("application", id, "README.md")
		absReadme := filepath.Join(pkgDir, readmePath)
		if _, err := os.Stat(absReadme); err != nil {
			continue // no README.md, skip
		}

		desc := extractDescription(absReadme, id)

		entry := ApplicationEntry{
			ID:          id,
			Path:        readmePath,
			Description: desc,
		}

		// Discover sub-files (other .md files in the application directory).
		entry.Files = discoverApplicationFiles(pkgDir, id)

		// Check for spec directory.
		specPath := filepath.Join("application", id, "spec", "README.md")
		if _, err := os.Stat(filepath.Join(pkgDir, specPath)); err == nil {
			entry.Spec = specPath
		}

		result.Application = append(result.Application, entry)
	}
}

// discoverApplicationFiles finds additional .md files in an application directory
// (excluding README.md) and returns their paths relative to the package root.
func discoverApplicationFiles(pkgDir, appID string) []string {
	dir := filepath.Join(pkgDir, "application", appID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		files = append(files, filepath.Join("application", appID, e.Name()))
	}
	return files
}

// discoverTopics scans topics/ for subdirectories containing README.md.
func discoverTopics(pkgDir string, existing map[string]bool, result *Manifest) {
	dir := filepath.Join(pkgDir, "topics")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		id := e.Name()
		if existing[id] {
			continue
		}

		readmePath := filepath.Join("topics", id, "README.md")
		absReadme := filepath.Join(pkgDir, readmePath)
		if _, err := os.Stat(absReadme); err != nil {
			continue // no README.md, skip
		}

		desc := extractDescription(absReadme, id)

		entry := TopicEntry{
			ID:          id,
			Path:        readmePath,
			Description: desc,
		}

		// Discover sub-files (other .md files in the topic directory).
		entry.Files = discoverTopicFiles(pkgDir, id)

		// Check for spec directory.
		specPath := filepath.Join("topics", id, "spec", "README.md")
		if _, err := os.Stat(filepath.Join(pkgDir, specPath)); err == nil {
			entry.Spec = specPath
		}

		result.Topics = append(result.Topics, entry)
	}
}

// discoverTopicFiles finds additional .md files in a topic directory
// (excluding README.md) and returns their paths relative to the package root.
func discoverTopicFiles(pkgDir, topicID string) []string {
	dir := filepath.Join(pkgDir, "topics", topicID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		files = append(files, filepath.Join("topics", topicID, e.Name()))
	}
	return files
}

// discoverPrompts scans prompts/ for subdirectories containing README.md.
func discoverPrompts(pkgDir string, existing map[string]bool, result *Manifest) {
	dir := filepath.Join(pkgDir, "prompts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		id := e.Name()
		if existing[id] {
			continue
		}

		readmePath := filepath.Join("prompts", id, "README.md")
		absReadme := filepath.Join(pkgDir, readmePath)
		if _, err := os.Stat(absReadme); err != nil {
			continue
		}

		desc := extractDescription(absReadme, id)

		result.Prompts = append(result.Prompts, PromptEntry{
			ID:          id,
			Path:        readmePath,
			Description: desc,
		})
	}
}

// discoverPlans scans plans/ for subdirectories containing README.md.
func discoverPlans(pkgDir string, existing map[string]bool, result *Manifest) {
	dir := filepath.Join(pkgDir, "plans")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		id := e.Name()
		if existing[id] {
			continue
		}

		readmePath := filepath.Join("plans", id, "README.md")
		absReadme := filepath.Join(pkgDir, readmePath)
		if _, err := os.Stat(absReadme); err != nil {
			continue
		}

		desc := extractDescription(absReadme, id)

		entry := PlanEntry{
			ID:          id,
			Path:        readmePath,
			Description: desc,
		}

		// Check for plan.yml.
		planStatePath := filepath.Join("plans", id, "plan.yml")
		if _, err := os.Stat(filepath.Join(pkgDir, planStatePath)); err == nil {
			entry.PlanState = planStatePath
		}

		result.Plans = append(result.Plans, entry)
	}
}

// extractDescription reads the first markdown heading from a file.
// Returns the heading text (without the # prefix) or the fallback string.
func extractDescription(path, fallback string) string {
	f, err := os.Open(path)
	if err != nil {
		return fallback
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}

	return fallback
}

// idSet builds a set of IDs from a slice using the given accessor.
func idSet[T any](entries []T, id func(T) string) map[string]bool {
	s := make(map[string]bool, len(entries))
	for _, e := range entries {
		s[id(e)] = true
	}
	return s
}

// copyApplication returns a shallow copy of an application entry slice.
func copyApplication(entries []ApplicationEntry) []ApplicationEntry {
	if entries == nil {
		return nil
	}
	out := make([]ApplicationEntry, len(entries))
	copy(out, entries)
	return out
}

// copyFoundation returns a shallow copy of a foundation entry slice.
func copyFoundation(entries []FoundationEntry) []FoundationEntry {
	if entries == nil {
		return nil
	}
	out := make([]FoundationEntry, len(entries))
	copy(out, entries)
	return out
}

// copyTopics returns a shallow copy of a topic entry slice.
func copyTopics(entries []TopicEntry) []TopicEntry {
	if entries == nil {
		return nil
	}
	out := make([]TopicEntry, len(entries))
	copy(out, entries)
	return out
}

// copyPrompts returns a shallow copy of a prompt entry slice.
func copyPrompts(entries []PromptEntry) []PromptEntry {
	if entries == nil {
		return nil
	}
	out := make([]PromptEntry, len(entries))
	copy(out, entries)
	return out
}

// copyPlans returns a shallow copy of a plan entry slice.
func copyPlans(entries []PlanEntry) []PlanEntry {
	if entries == nil {
		return nil
	}
	out := make([]PlanEntry, len(entries))
	copy(out, entries)
	return out
}

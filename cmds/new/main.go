package new

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

// kebabCase matches lowercase kebab-case identifiers: a-z, 0-9, hyphens,
// starting and ending with an alphanumeric character.
var kebabCase = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Command is the parent command for scaffolding new documentation entries.
var Command = &cli.Command{
	Name:  "new",
	Usage: "Scaffold a new documentation entry",
	Commands: []*cli.Command{
		foundationCommand,
		topicCommand,
		promptCommand,
		planCommand,
		applicationCommand,
	},
}

// sectionKind identifies the type of documentation section being scaffolded.
type sectionKind string

const (
	kindFoundation  sectionKind = "foundation"
	kindTopic       sectionKind = "topic"
	kindPrompt      sectionKind = "prompt"
	kindPlan        sectionKind = "plan"
	kindApplication sectionKind = "application"
)

// sectionDir returns the filesystem directory name for a section kind.
// Most kinds map directly to their name, but "topic" maps to "topics",
// "prompt" maps to "prompts", and "plan" maps to "plans".
func sectionDir(kind sectionKind) string {
	switch kind {
	case kindTopic:
		return "topics"
	case kindPrompt:
		return "prompts"
	case kindPlan:
		return "plans"
	default:
		return string(kind)
	}
}

// scaffold validates the name, creates directories and template files, runs
// manifest sync, and prints a summary. It is the shared logic for all
// subcommands.
func scaffold(kind sectionKind, name string) error {
	if !kebabCase.MatchString(name) {
		return fmt.Errorf("invalid name %q: must be lowercase kebab-case (e.g. my-entry)", name)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	docsDir := cfg.DocsDir()
	entryDir := filepath.Join(docsDir, sectionDir(kind), name)

	// Check for duplicates.
	if _, err := os.Stat(entryDir); err == nil {
		return fmt.Errorf("%s %q already exists", kind, name)
	}

	// Create the entry directory.
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write template files based on kind.
	readme := filepath.Join(entryDir, "README.md")
	title := kebabToTitle(name)
	if err := os.WriteFile(readme, []byte(fmt.Sprintf("# %s\n", title)), 0o644); err != nil {
		return fmt.Errorf("write README.md: %w", err)
	}

	var extraFiles []string

	switch kind {
	case kindTopic, kindApplication:
		// Create spec/README.md.
		specDir := filepath.Join(entryDir, "spec")
		if err := os.MkdirAll(specDir, 0o755); err != nil {
			return fmt.Errorf("create spec directory: %w", err)
		}
		specReadme := filepath.Join(specDir, "README.md")
		if err := os.WriteFile(specReadme, []byte(fmt.Sprintf("# %s Spec\n", title)), 0o644); err != nil {
			return fmt.Errorf("write spec/README.md: %w", err)
		}
		extraFiles = append(extraFiles, filepath.Join(sectionDir(kind), name, "spec", "README.md"))

	case kindPlan:
		// Create plan.yml with initial state.
		planYML := filepath.Join(entryDir, "plan.yml")
		planContent := fmt.Sprintf("plan: %s\nstatus: not_started\nsummary: \"\"\n", name)
		if err := os.WriteFile(planYML, []byte(planContent), 0o644); err != nil {
			return fmt.Errorf("write plan.yml: %w", err)
		}
		extraFiles = append(extraFiles, filepath.Join(sectionDir(kind), name, "plan.yml"))
	}

	// Run manifest sync.
	manifestPath := filepath.Join(docsDir, "manifest.yml")
	existing, err := manifest.Load(manifestPath)
	if err != nil {
		existing = &manifest.Manifest{Name: cfg.Name}
	}

	result := manifest.Sync(docsDir, existing)
	if err := manifest.Write(manifestPath, result); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Print summary.
	ui.Done(fmt.Sprintf("Created %s %q", kind, name))
	ui.Blank()
	ui.Header("Files:")
	ui.Item(filepath.Join(sectionDir(kind), name, "README.md"))
	for _, f := range extraFiles {
		ui.Item(f)
	}
	ui.Blank()

	return nil
}

// kebabToTitle converts a kebab-case string to a Title Case string.
// For example, "my-entry" becomes "My Entry".
func kebabToTitle(s string) string {
	words := splitKebab(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = string(w[0]-32) + w[1:]
		}
	}
	result := ""
	for i, w := range words {
		if i > 0 {
			result += " "
		}
		result += w
	}
	return result
}

// splitKebab splits a kebab-case string into words.
func splitKebab(s string) []string {
	var words []string
	current := ""
	for _, c := range s {
		if c == '-' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

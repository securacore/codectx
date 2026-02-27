package sync

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:  "sync",
	Usage: "Synchronize manifest with the filesystem and infer relationships from links",
	Action: func(ctx context.Context, c *cli.Command) error {
		return run()
	},
}

func run() error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	docsDir := cfg.DocsDir()
	manifestPath := filepath.Join(docsDir, "manifest.yml")

	// Load existing manifest, or start fresh if none exists.
	existing, err := manifest.Load(manifestPath)
	if err != nil {
		// If the manifest doesn't exist or is invalid, start with an empty one
		// but preserve name from config.
		existing = &manifest.Manifest{
			Name: cfg.Name,
		}
	}

	// Capture counts before sync for reporting.
	beforeCounts := sectionCounts(existing)

	result := manifest.Sync(docsDir, existing)

	if err := manifest.Write(manifestPath, result); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	afterCounts := sectionCounts(result)
	printSummary(beforeCounts, afterCounts, result)
	ui.Blank()

	return nil
}

// counts holds entry counts per section.
type counts struct {
	foundation  int
	application int
	topics      int
	prompts     int
	plans       int
}

func (c counts) total() int {
	return c.foundation + c.application + c.topics + c.prompts + c.plans
}

func sectionCounts(m *manifest.Manifest) counts {
	return counts{
		foundation:  len(m.Foundation),
		application: len(m.Application),
		topics:      len(m.Topics),
		prompts:     len(m.Prompts),
		plans:       len(m.Plans),
	}
}

// countRelationships returns the total number of depends_on entries across all sections.
func countRelationships(m *manifest.Manifest) int {
	n := 0
	for _, e := range m.Foundation {
		n += len(e.DependsOn)
	}
	for _, e := range m.Application {
		n += len(e.DependsOn)
	}
	for _, e := range m.Topics {
		n += len(e.DependsOn)
	}
	for _, e := range m.Prompts {
		n += len(e.DependsOn)
	}
	for _, e := range m.Plans {
		n += len(e.DependsOn)
	}
	return n
}

func printSummary(before, after counts, result *manifest.Manifest) {
	added := after.total() - before.total()
	rels := countRelationships(result)

	ui.Done(fmt.Sprintf("Synced manifest (%d entries)", after.total()))
	ui.Blank()

	if added > 0 {
		ui.KV("Discovered", added, 16)
	} else if added < 0 {
		ui.KV("Removed", -added, 16)
	}

	if after.foundation > 0 {
		ui.KV("Foundation", after.foundation, 16)
	}
	if after.application > 0 {
		ui.KV("Application", after.application, 16)
	}
	if after.topics > 0 {
		ui.KV("Topics", after.topics, 16)
	}
	if after.prompts > 0 {
		ui.KV("Prompts", after.prompts, 16)
	}
	if after.plans > 0 {
		ui.KV("Plans", after.plans, 16)
	}

	if rels > 0 {
		ui.KV("Relationships", rels, 16)
	}
}

// Package generate implements the `codectx generate` command which assembles
// specific chunks into a single coherent reading document.
//
// The command accepts comma-separated chunk IDs (from codectx query output),
// loads the corresponding chunk content and manifest metadata, groups by type
// (Instructions, System, Reasoning), and writes the assembled document to
// /tmp/codectx/.
//
// Usage:
//
//	codectx generate "obj:a1b2c3.03,obj:a1b2c3.04,spec:f7g8h9.02"
package generate

import (
	"context"
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/project"
	corequery "github.com/securacore/codectx/core/query"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx generate`.
var Command = &cli.Command{
	Name:      "generate",
	Usage:     "Assemble chunks into a reading document",
	ArgsUsage: "<chunk-id>,<chunk-id>,...",
	Description: `Assemble specific chunks into a single coherent reading document.
Accepts obj:, spec:, and sys: prefixed chunk IDs from codectx query output.

The generated document is written to /tmp/codectx/ and a summary is
printed to stdout including token count and related chunks.

Examples:
  codectx generate "obj:a1b2c3.03,spec:f7g8h9.02"
  codectx generate "obj:a1b2c3.03,obj:a1b2c3.04,spec:f7g8h9.02"`,
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing chunk IDs",
			Detail: []string{
				"Usage: codectx generate \"<chunk-id>,<chunk-id>,...\"",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Find chunk IDs with:", Command: "codectx query \"search terms\""},
				{Text: "Generate from results:", Command: "codectx generate \"obj:a1b2c3.03,spec:f7g8h9.02\""},
			},
		}.Render())
		return fmt.Errorf("missing chunk IDs")
	}

	// Parse comma-separated chunk IDs, trimming whitespace.
	raw := cmd.Args().First()
	parts := strings.Split(raw, ",")
	chunkIDs := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id != "" {
			chunkIDs = append(chunkIDs, id)
		}
	}

	if len(chunkIDs) == 0 {
		fmt.Print(tui.ErrorMsg{
			Title: "No valid chunk IDs provided",
			Detail: []string{
				"Expected comma-separated chunk IDs like: obj:a1b2c3.03,spec:f7g8h9.02",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Find chunk IDs with:", Command: "codectx query \"search terms\""},
			},
		}.Render())
		return fmt.Errorf("no valid chunk IDs")
	}

	// --- Step 1: Discover and load the project ---
	projectDir, cfg, err := project.DiscoverAndLoad(".")
	if err != nil {
		fmt.Print(tui.ProjectNotFoundError())
		return fmt.Errorf("project not found: %w", err)
	}

	// --- Step 2: Resolve compiled directory and load encoding ---
	compiledDir := corequery.CompiledDir(projectDir, cfg)

	encoding := project.DefaultEncoding
	aiCfg, aiErr := project.LoadAIConfigForProject(projectDir, cfg)
	if aiErr == nil && aiCfg.Compilation.Encoding != "" {
		encoding = aiCfg.Compilation.Encoding
	}

	// --- Step 3: Run generate ---
	result, err := corequery.RunGenerate(compiledDir, encoding, chunkIDs)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Generate failed",
			Detail: []string{
				err.Error(),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Verify chunk IDs exist:", Command: "codectx query \"search terms\""},
				{Text: "Compile documentation first:", Command: "codectx compile"},
			},
		}.Render())
		return fmt.Errorf("generate failed: %w", err)
	}

	// --- Step 4: Display summary ---
	fmt.Print(corequery.FormatGenerateSummary(result))

	return nil
}

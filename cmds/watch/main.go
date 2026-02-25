package watch

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/securacore/codectx/core/config"
	corewatch "github.com/securacore/codectx/core/watch"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:  "watch",
	Usage: "Watch for changes and recompile automatically",
	Action: func(ctx context.Context, c *cli.Command) error {
		return run(ctx)
	},
}

func run(ctx context.Context) error {
	// Validate config exists before starting watch loop.
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Set up signal handling for clean shutdown.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	w := corewatch.New(configFile)

	// Start watch loop in background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(ctx)
	}()

	ui.Done(fmt.Sprintf("Watching %s for changes...", cfg.DocsDir()))
	ui.Blank()

	// Process results until shutdown.
	for {
		select {
		case result := <-w.Results():
			printResult(result)
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func printResult(r corewatch.Result) {
	ts := r.Timestamp.Format(time.TimeOnly)

	// Print sync activity when entries were discovered or removed.
	if s := r.Sync; s != nil && (s.Discovered > 0 || s.Removed > 0) {
		printSyncResult(s, ts)
	}

	if r.Error != nil {
		ui.Fail(fmt.Sprintf("%s [%s]", r.Error, ts))
		return
	}

	if r.Compiled == nil {
		return
	}

	if r.Compiled.UpToDate {
		return
	}

	parts := []string{
		fmt.Sprintf("%d objects", r.Compiled.ObjectsStored),
	}
	if r.Compiled.ObjectsPruned > 0 {
		parts = append(parts, fmt.Sprintf("%d pruned", r.Compiled.ObjectsPruned))
	}
	if r.Compiled.Packages > 0 {
		parts = append(parts, fmt.Sprintf("%d packages", r.Compiled.Packages))
	}

	msg := fmt.Sprintf("Compiled (%s) [%s]", joinParts(parts), ts)
	ui.Done(msg)

	if r.Compiled.Dedup.HasConflicts() {
		ui.Warn(fmt.Sprintf("%d conflict(s)", len(r.Compiled.Dedup.Conflicts)))
	}
}

func printSyncResult(s *corewatch.SyncResult, ts string) {
	parts := []string{
		fmt.Sprintf("%d entries", s.Entries),
	}
	if s.Discovered > 0 {
		parts = append(parts, fmt.Sprintf("+%d discovered", s.Discovered))
	}
	if s.Removed > 0 {
		parts = append(parts, fmt.Sprintf("-%d removed", s.Removed))
	}
	if s.Relationships > 0 {
		parts = append(parts, fmt.Sprintf("%d relationships", s.Relationships))
	}

	ui.Done(fmt.Sprintf("Synced (%s) [%s]", joinParts(parts), ts))
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

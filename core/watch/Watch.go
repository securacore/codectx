package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
)

const defaultDebounce = 300 * time.Millisecond

// Result reports what happened after a watch-triggered compile.
type Result struct {
	Compiled  *compile.Result // nil if config load failed
	Error     error           // non-nil on compile or config error
	Timestamp time.Time
}

// Option configures a Watcher.
type Option func(*Watcher)

// WithDebounce sets the debounce window for coalescing rapid file changes.
// The default is 300ms.
func WithDebounce(d time.Duration) Option {
	return func(w *Watcher) {
		w.debounce = d
	}
}

// Watcher monitors filesystem changes and triggers recompilation.
type Watcher struct {
	configFile string
	debounce   time.Duration
	results    chan Result
}

// New creates a Watcher that monitors the filesystem for changes and
// triggers compilation. The configFile is the path to codectx.yml.
func New(configFile string, opts ...Option) *Watcher {
	w := &Watcher{
		configFile: configFile,
		debounce:   defaultDebounce,
		results:    make(chan Result, 8),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Results returns the channel that receives compile outcomes.
// Consumers should read this channel to receive status updates.
func (w *Watcher) Results() <-chan Result {
	return w.results
}

// Run starts the watch loop. It performs an initial compile, then
// watches for filesystem changes and recompiles on each debounced event.
// Blocks until ctx is canceled.
func (w *Watcher) Run(ctx context.Context) error {
	// Initial compile.
	w.compileAndSend()

	// Load config to determine watch paths.
	cfg, err := config.Load(w.configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer func() { _ = fsw.Close() }()

	docsDir := cfg.DocsDir()
	outputDir := cfg.OutputDir()

	// Resolve to absolute paths for reliable prefix matching.
	absOutputDir, _ := filepath.Abs(outputDir)

	// Watch codectx.yml.
	if err := fsw.Add(w.configFile); err != nil {
		return fmt.Errorf("watch config file: %w", err)
	}

	// Walk docs dir and add all directories recursively.
	if err := addDirRecursive(fsw, docsDir, absOutputDir); err != nil {
		return fmt.Errorf("watch docs directory: %w", err)
	}

	trigger := make(chan struct{}, 1)
	var timer *time.Timer

	for {
		select {
		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}

			if shouldIgnore(event.Name, absOutputDir) {
				continue
			}

			// Auto-watch new directories.
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = addDirRecursive(fsw, event.Name, absOutputDir)
				}
			}

			// Reset debounce timer.
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(w.debounce, func() {
				select {
				case trigger <- struct{}{}:
				default:
				}
			})

		case <-trigger:
			w.compileAndSend()

			// Check if config changed watch paths.
			newCfg, loadErr := config.Load(w.configFile)
			if loadErr == nil && newCfg.DocsDir() != docsDir {
				// Docs dir changed; rewire watcher.
				_ = fsw.Remove(docsDir)
				docsDir = newCfg.DocsDir()
				outputDir = newCfg.OutputDir()
				absOutputDir, _ = filepath.Abs(outputDir)
				_ = addDirRecursive(fsw, docsDir, absOutputDir)
			}

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			w.results <- Result{
				Error:     fmt.Errorf("watcher error: %w", err),
				Timestamp: time.Now(),
			}

		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return nil
		}
	}
}

// compileAndSend loads config, runs compile, and sends the result.
func (w *Watcher) compileAndSend() {
	cfg, err := config.Load(w.configFile)
	if err != nil {
		w.results <- Result{
			Error:     fmt.Errorf("load config: %w", err),
			Timestamp: time.Now(),
		}
		return
	}

	compiled, err := compile.Compile(cfg)
	w.results <- Result{
		Compiled:  compiled,
		Error:     err,
		Timestamp: time.Now(),
	}
}

// addDirRecursive walks root and adds all directories to the watcher,
// skipping any path under excludeDir.
func addDirRecursive(fsw *fsnotify.Watcher, root, excludeDir string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible directories
		}
		if !d.IsDir() {
			return nil
		}

		absPath, _ := filepath.Abs(path)
		if isUnderDir(absPath, excludeDir) {
			return filepath.SkipDir
		}

		return fsw.Add(path)
	})
}

// shouldIgnore returns true if the event path should be ignored.
func shouldIgnore(path, excludeDir string) bool {
	absPath, _ := filepath.Abs(path)
	return isUnderDir(absPath, excludeDir)
}

// isUnderDir reports whether path is equal to or under dir.
func isUnderDir(path, dir string) bool {
	if dir == "" {
		return false
	}
	if path == dir {
		return true
	}
	return strings.HasPrefix(path, dir+string(filepath.Separator))
}

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
	"github.com/securacore/codectx/core/manifest"
)

const defaultDebounce = 300 * time.Millisecond
const defaultPollInterval = 30 * time.Second

// Result reports what happened after a watch-triggered sync and compile.
type Result struct {
	Compiled  *compile.Result // nil if config load failed
	Sync      *SyncResult     // nil if sync was not attempted
	Error     error           // non-nil on compile or config error
	Timestamp time.Time
}

// SyncResult reports sync activity from a watch cycle.
type SyncResult struct {
	Entries       int // total entries after sync
	Discovered    int // net new entries (added - removed, when positive)
	Removed       int // net entries removed (removed - added, when positive)
	Relationships int // total depends_on relationships
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

// WithPollInterval sets the polling heartbeat interval. The watcher
// periodically runs a compile as a safety net for missed filesystem
// events. The fingerprint mechanism makes this cheap: if nothing
// changed, Compile returns UpToDate immediately. Set to 0 to disable.
// The default is 30s.
func WithPollInterval(d time.Duration) Option {
	return func(w *Watcher) {
		w.pollInterval = d
	}
}

// Watcher monitors filesystem changes and triggers recompilation.
type Watcher struct {
	configFile   string
	debounce     time.Duration
	pollInterval time.Duration
	results      chan Result
}

// New creates a Watcher that monitors the filesystem for changes and
// triggers compilation. The configFile is the path to codectx.yml.
func New(configFile string, opts ...Option) *Watcher {
	w := &Watcher{
		configFile:   configFile,
		debounce:     defaultDebounce,
		pollInterval: defaultPollInterval,
		results:      make(chan Result, 8),
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

	// Polling heartbeat as a safety net for missed filesystem events.
	// The fingerprint mechanism makes this cheap: Compile returns
	// UpToDate immediately if nothing changed.
	var pollCh <-chan time.Time
	if w.pollInterval > 0 {
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()
		pollCh = ticker.C
	}

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

		case <-pollCh:
			w.compileAndSend()

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

// compileAndSend syncs the local manifest, runs compile, and sends the result.
// Sync runs before compile to discover new entries, remove stale entries, and
// infer relationships from links. The sync result is included so consumers
// (like the watch CLI) can report sync activity.
func (w *Watcher) compileAndSend() {
	cfg, err := config.Load(w.configFile)
	if err != nil {
		w.results <- Result{
			Error:     fmt.Errorf("load config: %w", err),
			Timestamp: time.Now(),
		}
		return
	}

	// Sync local manifest before compile.
	syncResult := w.syncLocal(cfg)

	compiled, err := compile.Compile(cfg)
	w.results <- Result{
		Compiled:  compiled,
		Sync:      syncResult,
		Error:     err,
		Timestamp: time.Now(),
	}
}

// syncLocal runs manifest.Sync on the local docs directory, writes back
// the result, and returns sync stats. Returns nil if the manifest cannot
// be loaded (e.g., first run before init).
func (w *Watcher) syncLocal(cfg *config.Config) *SyncResult {
	docsDir := cfg.DocsDir()
	manifestPath := filepath.Join(docsDir, "manifest.yml")

	existing, err := manifest.Load(manifestPath)
	if err != nil {
		return nil
	}

	beforeTotal := countManifestEntries(existing)
	synced := manifest.Sync(docsDir, existing)
	afterTotal := countManifestEntries(synced)
	rels := countRelationships(synced)

	// Write back (conditional — skips if content unchanged).
	_ = manifest.Write(manifestPath, synced)

	sr := &SyncResult{
		Entries:       afterTotal,
		Relationships: rels,
	}

	delta := afterTotal - beforeTotal
	if delta > 0 {
		sr.Discovered = delta
	} else if delta < 0 {
		sr.Removed = -delta
	}

	return sr
}

// countManifestEntries returns the total number of entries across all sections.
func countManifestEntries(m *manifest.Manifest) int {
	return len(m.Foundation) + len(m.Application) + len(m.Topics) + len(m.Prompts) + len(m.Plans)
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

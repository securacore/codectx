// Package usage implements local and global usage metrics tracking for codectx.
//
// Two YAML files track invocation and token usage at different scopes:
//   - usage.yml — gitignored, local machine, updated on every query/generate
//   - global_usage.yml — checked in, project lifetime, updated on codectx compile
//
// All writes are best-effort. Concurrent invocations may race on
// read-modify-write; lost updates are acceptable for volume measurements.
package usage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/embed"
	"gopkg.in/yaml.v3"
)

const (
	// LocalFile is the filename for local machine usage metrics.
	LocalFile = "usage.yml"

	// GlobalFile is the filename for project lifetime usage metrics.
	GlobalFile = "global_usage.yml"

	// localHeader is the YAML comment header for usage.yml.
	localHeader = "# codectx local machine usage — gitignored\n# Rolled into global_usage.yml on codectx compile.\n\n"

	// globalHeader is the YAML comment header for global_usage.yml.
	globalHeader = "# codectx project lifetime usage — checked into version control\n# Updated on codectx compile from local usage.yml.\n\n"
)

// Metrics tracks invocation and token usage.
type Metrics struct {
	TotalTokens         int            `yaml:"total_tokens"`
	QueryInvocations    int            `yaml:"query_invocations"`
	GenerateInvocations int            `yaml:"generate_invocations"`
	CacheHits           int            `yaml:"cache_hits"`
	TokensByCaller      map[string]int `yaml:"tokens_by_caller"`
	TokensByModel       map[string]int `yaml:"tokens_by_model"`
	Project             string         `yaml:"project,omitempty"`
	FirstSeen           int64          `yaml:"first_seen"`
	LastUpdated         int64          `yaml:"last_updated"`
	LastCompile         int64          `yaml:"last_compile,omitempty"`
}

// LocalPath returns the path to usage.yml for a project.
func LocalPath(projectDir string, cfg *project.Config) string {
	return filepath.Join(project.RootDir(projectDir, cfg), project.CodectxDir, LocalFile)
}

// GlobalPath returns the path to global_usage.yml for a project.
func GlobalPath(projectDir string, cfg *project.Config) string {
	return filepath.Join(project.RootDir(projectDir, cfg), project.CodectxDir, GlobalFile)
}

// UpdateQuery increments the query invocation counter in usage.yml.
// Best-effort — errors are returned but callers should warn, not fail.
func UpdateQuery(usageFile string) error {
	m := readOrInit(usageFile)
	m.QueryInvocations++
	m.LastUpdated = time.Now().UnixNano()
	return writeMetrics(usageFile, m, localHeader)
}

// UpdateGenerate adds token counts and increments generate counters in usage.yml.
// Best-effort — errors are returned but callers should warn, not fail.
func UpdateGenerate(usageFile string, tokens int, cacheHit bool, caller history.CallerContext) error {
	m := readOrInit(usageFile)
	m.TotalTokens += tokens
	m.GenerateInvocations++
	if cacheHit {
		m.CacheHits++
	}
	m.TokensByCaller[caller.Caller] += tokens
	m.TokensByModel[caller.Model] += tokens
	m.LastUpdated = time.Now().UnixNano()
	return writeMetrics(usageFile, m, localHeader)
}

// SyncGlobal merges local usage.yml into global_usage.yml, then resets
// the local file to zero. Called by codectx compile after the pipeline
// completes.
//
// If the global write succeeds but the local reset fails, the next compile
// will re-add the same local counts — a slight overcount, acceptable for
// volume measurements.
func SyncGlobal(localFile, globalFile, projectName string) error {
	local := readOrInit(localFile)

	// Nothing to sync if local has no activity.
	if local.TotalTokens == 0 && local.QueryInvocations == 0 && local.GenerateInvocations == 0 {
		return nil
	}

	global := readOrInit(globalFile)

	global.TotalTokens += local.TotalTokens
	global.QueryInvocations += local.QueryInvocations
	global.GenerateInvocations += local.GenerateInvocations
	global.CacheHits += local.CacheHits

	for caller, tokens := range local.TokensByCaller {
		global.TokensByCaller[caller] += tokens
	}
	for model, tokens := range local.TokensByModel {
		global.TokensByModel[model] += tokens
	}

	global.Project = projectName
	global.LastUpdated = time.Now().UnixNano()
	global.LastCompile = time.Now().UnixNano()

	if err := writeMetrics(globalFile, global, globalHeader); err != nil {
		return fmt.Errorf("writing global usage: %w", err)
	}

	// Reset local to zero after successful sync.
	return writeMetrics(localFile, initMetrics(), localHeader)
}

// ReadLocal reads the local usage metrics. Returns zero-initialized metrics
// if the file does not exist or cannot be read.
func ReadLocal(path string) Metrics {
	return readOrInit(path)
}

// ReadGlobal reads the global usage metrics. Returns zero-initialized metrics
// if the file does not exist or cannot be read.
func ReadGlobal(path string) Metrics {
	return readOrInit(path)
}

// initFile creates a usage file with zero values if it does not exist.
// Never overwrites an existing file.
func initFile(path, projectName, header string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // file exists, do not overwrite
	}

	m := initMetrics()
	if projectName != "" {
		m.Project = projectName
	}
	return writeMetrics(path, m, header)
}

// InitLocalFile creates usage.yml with zero values if it does not exist.
// Uses the embedded commented template for self-documenting defaults.
func InitLocalFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // file exists, do not overwrite
	}

	tmpl, err := embed.ReadConfigTemplate("usage.yml")
	if err != nil {
		// Fall back to struct-based write if template is unavailable.
		return initFile(path, "", localHeader)
	}

	return project.WriteConfigFromTemplate(path, tmpl, struct {
		FirstSeen int64
	}{
		FirstSeen: time.Now().UnixNano(),
	})
}

// InitGlobalFile creates global_usage.yml with zero values if it does not exist.
// Uses the embedded commented template for self-documenting defaults.
func InitGlobalFile(path, projectName string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // file exists, do not overwrite
	}

	tmpl, err := embed.ReadConfigTemplate("global-usage.yml")
	if err != nil {
		// Fall back to struct-based write if template is unavailable.
		return initFile(path, projectName, globalHeader)
	}

	return project.WriteConfigFromTemplate(path, tmpl, struct {
		Project   string
		FirstSeen int64
	}{
		Project:   projectName,
		FirstSeen: time.Now().UnixNano(),
	})
}

// --- Internal helpers ---

// initMetrics returns a zero-initialized Metrics with allocated maps.
func initMetrics() Metrics {
	return Metrics{
		TokensByCaller: map[string]int{},
		TokensByModel:  map[string]int{},
		FirstSeen:      time.Now().UnixNano(),
	}
}

// readOrInit reads a usage file and returns its metrics. If the file does
// not exist or cannot be parsed, returns zero-initialized metrics.
func readOrInit(path string) Metrics {
	data, err := os.ReadFile(path)
	if err != nil {
		return initMetrics()
	}

	var m Metrics
	_ = yaml.Unmarshal(data, &m)

	if m.TokensByCaller == nil {
		m.TokensByCaller = map[string]int{}
	}
	if m.TokensByModel == nil {
		m.TokensByModel = map[string]int{}
	}

	return m
}

// writeMetrics serializes metrics to YAML at the given path.
func writeMetrics(path string, m Metrics, header string) error {
	return project.WriteYAMLFile(path, header, m)
}

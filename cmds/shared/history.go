// Package shared provides helpers used across multiple codectx commands.
//
// This file contains history-related helpers for commands that log to the
// query or generate history.

package shared

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
)

// ResolveHistoryDir discovers the project and returns the history directory path.
func ResolveHistoryDir() (histDir, projectDir string, cfg *project.Config, err error) {
	projectDir, cfg, err = DiscoverProject()
	if err != nil {
		return "", "", nil, err
	}
	return history.HistoryDir(projectDir, cfg), projectDir, cfg, nil
}

// WarnHistory prints a best-effort warning for a failed history operation
// to stderr using the standard WarnMsg format.
func WarnHistory(action string, err error) {
	fmt.Fprint(os.Stderr, tui.WarnMsg{
		Title:  fmt.Sprintf("History: %s failed", action),
		Detail: []string{err.Error()},
	}.Render())
}

// WarnBestEffort prints a best-effort warning for any non-critical operation
// to stderr using the standard WarnMsg format. Use this for cache writes,
// usage updates, and other operations that should never block the primary command.
func WarnBestEffort(action string, err error) {
	fmt.Fprint(os.Stderr, tui.WarnMsg{
		Title:  fmt.Sprintf("%s failed", action),
		Detail: []string{err.Error()},
	}.Render())
}

// BuildHistoryPath builds a display-friendly relative path to a history
// doc file. If the path cannot be made relative to the current directory,
// the absolute path is returned. Used by generate and prompt commands.
func BuildHistoryPath(histDir, docFile string) string {
	if docFile == "" {
		return ""
	}
	fullPath := filepath.Join(histDir, history.DocsDir, docFile)
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		if realCwd, evalErr := filepath.EvalSymlinks(cwd); evalErr == nil {
			cwd = realCwd
		}
		if rel, relErr := filepath.Rel(cwd, fullPath); relErr == nil {
			return rel
		}
	}
	return fullPath
}

// ResolveTopN determines the number of query results to return.
// If flagValue is positive, it is used directly. Otherwise, the default
// is loaded from the AI config or falls back to project.DefaultResultsCount.
// Used by both query and prompt commands.
func ResolveTopN(flagValue int, projectDir string, cfg *project.Config) int {
	if flagValue > 0 {
		return flagValue
	}

	if cfg != nil {
		aiCfg, aiErr := project.LoadAIConfigForProject(projectDir, cfg)
		if aiErr == nil && aiCfg.Consumption.ResultsCount > 0 {
			return aiCfg.Consumption.ResultsCount
		}
	}

	return project.DefaultResultsCount
}

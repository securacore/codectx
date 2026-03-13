// Package shared provides helpers used across multiple codectx commands.
//
// This file contains history-related helpers for commands that log to the
// query or generate history.

package shared

import (
	"fmt"
	"os"

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

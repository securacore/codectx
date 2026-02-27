package activate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// viewState tracks which view is active in the TUI.
type viewState int

const (
	viewPackages viewState = iota // package list (main view)
	viewEntries                   // entry drill-in for a single package
)

// packageItem holds the working state for a single package in the TUI.
type packageItem struct {
	name       string
	author     string
	version    string
	activation config.Activation  // working copy (mutated by user)
	original   config.Activation  // snapshot for change detection
	manifest   *manifest.Manifest // loaded on first drill-in (nil until then)
	loadErr    error              // non-nil if manifest failed to load
}

// entryItem represents a single activatable entry in the drill-in view.
type entryItem struct {
	section string
	id      string
	label   string
	active  bool
}

// activateModel is the bubbletea model for the activate TUI.
type activateModel struct {
	// Data.
	packages []packageItem
	entries  []entryItem

	// State.
	view       viewState
	cursor     int
	drillIndex int  // which package we're drilling into
	modified   bool // has anything changed?
	saved      bool // was save performed?
	quitting   bool

	// Config reference.
	cfg     *config.Config
	docsDir string

	// Terminal dimensions.
	width  int
	height int
}

// Styles — aliases to the shared ui palette.
var (
	activeTitleStyle  = ui.BoldStyle
	activeHintStyle   = ui.DimStyle
	activeGreenStyle  = ui.GreenStyle
	activeYellowStyle = ui.YellowStyle
	activeAccentStyle = ui.AccentStyle
)

func newActivateModel(cfg *config.Config) activateModel {
	docsDir := cfg.DocsDir()
	packages := make([]packageItem, len(cfg.Packages))
	for i, pkg := range cfg.Packages {
		packages[i] = packageItem{
			name:       pkg.Name,
			author:     pkg.Author,
			version:    pkg.Version,
			activation: pkg.Active,
			original:   pkg.Active,
		}
	}

	return activateModel{
		packages: packages,
		cfg:      cfg,
		docsDir:  docsDir,
		view:     viewPackages,
	}
}

func (m activateModel) Init() tea.Cmd {
	return nil
}

func (m activateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global quit.
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}

		switch m.view {
		case viewPackages:
			return m.updatePackages(msg)
		case viewEntries:
			return m.updateEntries(msg)
		}
	}

	return m, nil
}

func (m activateModel) updatePackages(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.packages)-1 {
			m.cursor++
		}
	case " ":
		// Toggle: none -> all, all -> none, partial -> none.
		if m.cursor < len(m.packages) {
			pkg := &m.packages[m.cursor]
			if pkg.activation.IsNone() {
				pkg.activation = config.Activation{Mode: "all"}
			} else {
				pkg.activation = config.Activation{Mode: "none"}
			}
			m.modified = m.hasChanges()
		}
	case "enter":
		// Drill into entry selection.
		if m.cursor < len(m.packages) {
			m.drillIndex = m.cursor
			m.loadEntries()
			m.view = viewEntries
			m.cursor = 0
		}
	case "s":
		if m.modified {
			m.saved = true
			m.quitting = true
			return m, tea.Quit
		}
	case "q", "esc":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m activateModel) updateEntries(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case " ":
		// Toggle individual entry.
		if m.cursor < len(m.entries) {
			m.entries[m.cursor].active = !m.entries[m.cursor].active
		}
	case "a":
		// Select all.
		for i := range m.entries {
			m.entries[i].active = true
		}
	case "n":
		// Deselect all.
		for i := range m.entries {
			m.entries[i].active = false
		}
	case "enter", "esc":
		// Go back to package view — apply changes.
		m.applyEntryChanges()
		m.view = viewPackages
		m.cursor = m.drillIndex
		m.modified = m.hasChanges()
	}

	return m, nil
}

// loadEntries loads the manifest for the current drill-in package and
// populates the entries slice.
func (m *activateModel) loadEntries() {
	pkg := &m.packages[m.drillIndex]

	// Load manifest if not already loaded.
	if pkg.manifest == nil && pkg.loadErr == nil {
		pkgDir := filepath.Join(m.docsDir, "packages", fmt.Sprintf("%s@%s", pkg.name, pkg.author))
		pkgManifestPath := filepath.Join(pkgDir, "manifest.yml")
		loaded, err := manifest.Load(pkgManifestPath)
		if err != nil {
			pkg.loadErr = err
			m.entries = nil
			return
		}
		pkg.manifest = manifest.Discover(pkgDir, loaded)
	}

	if pkg.loadErr != nil {
		m.entries = nil
		return
	}

	// Build entry list with current activation state.
	activeIDs := buildActiveIDSet(pkg.activation, pkg.manifest)
	var entries []entryItem

	for _, e := range pkg.manifest.Foundation {
		entries = append(entries, entryItem{
			section: "foundation",
			id:      e.ID,
			label:   fmt.Sprintf("[foundation] %s - %s", e.ID, e.Description),
			active:  activeIDs["foundation:"+e.ID],
		})
	}
	for _, e := range pkg.manifest.Application {
		entries = append(entries, entryItem{
			section: "application",
			id:      e.ID,
			label:   fmt.Sprintf("[application] %s - %s", e.ID, e.Description),
			active:  activeIDs["application:"+e.ID],
		})
	}
	for _, e := range pkg.manifest.Topics {
		entries = append(entries, entryItem{
			section: "topics",
			id:      e.ID,
			label:   fmt.Sprintf("[topics] %s - %s", e.ID, e.Description),
			active:  activeIDs["topics:"+e.ID],
		})
	}
	for _, e := range pkg.manifest.Prompts {
		entries = append(entries, entryItem{
			section: "prompts",
			id:      e.ID,
			label:   fmt.Sprintf("[prompts] %s - %s", e.ID, e.Description),
			active:  activeIDs["prompts:"+e.ID],
		})
	}
	for _, e := range pkg.manifest.Plans {
		entries = append(entries, entryItem{
			section: "plans",
			id:      e.ID,
			label:   fmt.Sprintf("[plans] %s - %s", e.ID, e.Description),
			active:  activeIDs["plans:"+e.ID],
		})
	}

	m.entries = entries
}

// buildActiveIDSet returns the set of active "section:id" keys for a
// package given its activation state and manifest.
func buildActiveIDSet(activation config.Activation, m *manifest.Manifest) map[string]bool {
	if activation.IsAll() {
		ids := make(map[string]bool)
		for _, e := range m.Foundation {
			ids["foundation:"+e.ID] = true
		}
		for _, e := range m.Application {
			ids["application:"+e.ID] = true
		}
		for _, e := range m.Topics {
			ids["topics:"+e.ID] = true
		}
		for _, e := range m.Prompts {
			ids["prompts:"+e.ID] = true
		}
		for _, e := range m.Plans {
			ids["plans:"+e.ID] = true
		}
		return ids
	}
	if activation.IsNone() || activation.Map == nil {
		return nil
	}
	ids := make(map[string]bool)
	for _, id := range activation.Map.Foundation {
		ids["foundation:"+id] = true
	}
	for _, id := range activation.Map.Application {
		ids["application:"+id] = true
	}
	for _, id := range activation.Map.Topics {
		ids["topics:"+id] = true
	}
	for _, id := range activation.Map.Prompts {
		ids["prompts:"+id] = true
	}
	for _, id := range activation.Map.Plans {
		ids["plans:"+id] = true
	}
	return ids
}

// applyEntryChanges converts the current entry toggle states back into an
// Activation on the drilled-in package.
func (m *activateModel) applyEntryChanges() {
	pkg := &m.packages[m.drillIndex]
	if pkg.manifest == nil {
		return
	}

	totalEntries := len(pkg.manifest.Foundation) + len(pkg.manifest.Application) +
		len(pkg.manifest.Topics) + len(pkg.manifest.Prompts) + len(pkg.manifest.Plans)

	activeCount := 0
	for _, e := range m.entries {
		if e.active {
			activeCount++
		}
	}

	if activeCount == 0 {
		pkg.activation = config.Activation{Mode: "none"}
		return
	}
	if activeCount == totalEntries {
		pkg.activation = config.Activation{Mode: "all"}
		return
	}

	// Build granular activation map.
	am := &config.ActivationMap{}
	for _, e := range m.entries {
		if !e.active {
			continue
		}
		switch e.section {
		case "foundation":
			am.Foundation = append(am.Foundation, e.id)
		case "application":
			am.Application = append(am.Application, e.id)
		case "topics":
			am.Topics = append(am.Topics, e.id)
		case "prompts":
			am.Prompts = append(am.Prompts, e.id)
		case "plans":
			am.Plans = append(am.Plans, e.id)
		}
	}
	pkg.activation = config.Activation{Map: am}
}

// hasChanges checks whether any package activation differs from its original.
func (m activateModel) hasChanges() bool {
	for _, pkg := range m.packages {
		if !activationsEqual(pkg.activation, pkg.original) {
			return true
		}
	}
	return false
}

// activationsEqual compares two Activation values for equality.
func activationsEqual(a, b config.Activation) bool {
	// Both string modes.
	if a.Map == nil && b.Map == nil {
		modeA := a.Mode
		modeB := b.Mode
		if modeA == "" {
			modeA = "none"
		}
		if modeB == "" {
			modeB = "none"
		}
		return modeA == modeB
	}
	// One is map, one is not.
	if (a.Map == nil) != (b.Map == nil) {
		return false
	}
	// Both are maps.
	return slicesEqual(a.Map.Foundation, b.Map.Foundation) &&
		slicesEqual(a.Map.Application, b.Map.Application) &&
		slicesEqual(a.Map.Topics, b.Map.Topics) &&
		slicesEqual(a.Map.Prompts, b.Map.Prompts) &&
		slicesEqual(a.Map.Plans, b.Map.Plans)
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- View ---

func (m activateModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.view {
	case viewPackages:
		return m.viewPackages()
	case viewEntries:
		return m.viewEntries()
	}
	return ""
}

func (m activateModel) viewPackages() string {
	var lines []string

	lines = append(lines, "")
	lines = append(lines, "  "+activeTitleStyle.Render("Package Activation"))
	lines = append(lines, "")

	if len(m.packages) == 0 {
		lines = append(lines, "  "+activeHintStyle.Render("No packages installed."))
	} else {
		// Calculate column widths.
		nameW := 0
		versionW := 0
		for _, pkg := range m.packages {
			label := fmt.Sprintf("%s@%s", pkg.name, pkg.author)
			if len(label) > nameW {
				nameW = len(label)
			}
			if len(pkg.version) > versionW {
				versionW = len(pkg.version)
			}
		}

		for i, pkg := range m.packages {
			icon := m.activationIcon(pkg.activation, pkg.manifest)
			label := fmt.Sprintf("%s@%s", pkg.name, pkg.author)
			status := m.activationStatus(pkg)

			row := fmt.Sprintf("  %s %-*s  %-*s  %s", icon, nameW, label, versionW, pkg.version, status)

			if i == m.cursor {
				lines = append(lines, activeAccentStyle.Render(row))
			} else {
				lines = append(lines, row)
			}
		}
	}

	// Pad to push help bar to bottom.
	topCount := len(lines)
	bottomCount := 2
	padding := m.height - topCount - bottomCount
	if padding < 1 {
		padding = 1
	}
	for i := 0; i < padding; i++ {
		lines = append(lines, "")
	}

	// Help bar.
	help := m.packageHelpText()
	lines = append(lines, "  "+activeHintStyle.Render(help))

	return strings.Join(lines, "\n")
}

func (m activateModel) viewEntries() string {
	var lines []string
	pkg := m.packages[m.drillIndex]

	lines = append(lines, "")
	lines = append(lines, "  "+activeTitleStyle.Render(fmt.Sprintf("%s@%s  %s", pkg.name, pkg.author, pkg.version)))
	lines = append(lines, "")

	if pkg.loadErr != nil {
		lines = append(lines, "  "+ui.RedStyle.Render(ui.SymbolFail+" "+pkg.loadErr.Error()))
	} else if len(m.entries) == 0 {
		lines = append(lines, "  "+activeHintStyle.Render("No entries in this package."))
	} else {
		// Determine visible window for scrolling.
		maxVisible := m.height - 8
		if maxVisible < 3 {
			maxVisible = 3
		}
		start, end := 0, len(m.entries)
		if end > maxVisible {
			if m.cursor < maxVisible {
				end = maxVisible
			} else {
				end = m.cursor + 1
				start = end - maxVisible
			}
		}

		if start > 0 {
			lines = append(lines, "  "+activeHintStyle.Render(fmt.Sprintf("  ... %d above", start)))
		}

		for i := start; i < end; i++ {
			e := m.entries[i]
			var check string
			if e.active {
				check = activeGreenStyle.Render(ui.SymbolDone)
			} else {
				check = activeHintStyle.Render(ui.SymbolInactive)
			}

			if i == m.cursor {
				row := fmt.Sprintf("  %s %s", check, activeAccentStyle.Render(e.label))
				lines = append(lines, activeAccentStyle.Render("\u25b8")+row)
			} else {
				row := fmt.Sprintf("   %s %s", check, e.label)
				if e.active {
					lines = append(lines, activeGreenStyle.Render(row))
				} else {
					lines = append(lines, row)
				}
			}
		}

		if end < len(m.entries) {
			lines = append(lines, "  "+activeHintStyle.Render(fmt.Sprintf("  ... %d more", len(m.entries)-end)))
		}
	}

	// Pad to push help bar to bottom.
	topCount := len(lines)
	bottomCount := 2
	padding := m.height - topCount - bottomCount
	if padding < 1 {
		padding = 1
	}
	for i := 0; i < padding; i++ {
		lines = append(lines, "")
	}

	help := m.entryHelpText()
	lines = append(lines, "  "+activeHintStyle.Render(help))

	return strings.Join(lines, "\n")
}

// activationIcon returns the visual indicator for a package's activation state.
func (m activateModel) activationIcon(a config.Activation, mf *manifest.Manifest) string {
	if a.IsAll() {
		return activeGreenStyle.Render(ui.SymbolActive)
	}
	if a.IsGranular() {
		return activeYellowStyle.Render(ui.SymbolPartial)
	}
	return activeHintStyle.Render(ui.SymbolInactive)
}

// activationStatus returns a status label for a package.
func (m activateModel) activationStatus(pkg packageItem) string {
	if pkg.activation.IsAll() {
		return activeGreenStyle.Render("all")
	}
	if pkg.activation.IsNone() {
		return activeHintStyle.Render("none")
	}
	count := activationEntryCount(pkg.activation)
	if pkg.manifest != nil {
		total := len(pkg.manifest.Foundation) + len(pkg.manifest.Application) +
			len(pkg.manifest.Topics) + len(pkg.manifest.Prompts) + len(pkg.manifest.Plans)
		return activeYellowStyle.Render(fmt.Sprintf("%d of %d entries", count, total))
	}
	return activeYellowStyle.Render(fmt.Sprintf("%d entries", count))
}

func (m activateModel) packageHelpText() string {
	parts := []string{"\u2191/\u2193 navigate", "space toggle", "enter edit entries"}
	if m.modified {
		parts = append(parts, "s save")
	}
	parts = append(parts, "q quit")
	return strings.Join(parts, "  ")
}

func (m activateModel) entryHelpText() string {
	return "\u2191/\u2193 navigate  space toggle  a all  n none  esc back"
}

// --- Launch ---

// runInteractive launches the full-screen activation TUI.
func runInteractive() error {
	if !ui.IsTTY() {
		return fmt.Errorf("interactive activation requires a terminal; use: codectx activate <package@author>")
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Packages) == 0 {
		ui.Done("No packages installed. Use: codectx add <package>")
		return nil
	}

	m := newActivateModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return err
	}

	final := result.(activateModel)
	if !final.saved {
		return nil
	}

	// Apply changes to config.
	for i, pkg := range final.packages {
		cfg.Packages[i].Active = pkg.activation
	}

	if err := config.Write(configFile, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	ui.Done("Activation updated")
	for _, pkg := range final.packages {
		if !activationsEqual(pkg.activation, pkg.original) {
			ui.KV(fmt.Sprintf("%s@%s", pkg.name, pkg.author), activationLabel(pkg.activation), 30)
		}
	}

	if err := shared.MaybeAutoCompile(cfg); err != nil {
		return err
	}

	ui.Blank()
	return nil
}

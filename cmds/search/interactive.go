package search

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/securacore/codectx/core/resolve"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// searchState represents the current state of the interactive search.
type searchState int

const (
	stateInput     searchState = iota // text input focused, waiting for query
	stateSearching                    // spinner active, API request in flight
	stateResults                      // results displayed, cursor navigation active
)

// searchDoneMsg carries the results of a completed search operation.
type searchDoneMsg struct {
	results []resolve.SearchResult
	err     error
}

// searchModel is the bubbletea model for interactive package search.
type searchModel struct {
	input    textinput.Model
	spinner  spinner.Model
	state    searchState
	results  []resolve.SearchResult
	cursor   int
	err      error
	width    int
	height   int
	author   string // pre-set author filter (from --author flag)
	searched bool   // true after at least one search completes
	selected *resolve.SearchResult
	quitting bool
}

// Styles for the interactive view — built from the shared ui palette.
var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	hintStyle     = lipgloss.NewStyle().Foreground(ui.Dim)
	errorStyle    = lipgloss.NewStyle().Foreground(ui.Red)
	selectorStyle = lipgloss.NewStyle().Foreground(ui.Accent)
	selectedStyle = lipgloss.NewStyle().Foreground(ui.Accent)
	headerStyle   = lipgloss.NewStyle().Foreground(ui.Dim)
)

func newSearchModel(author string) searchModel {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.Accent)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(ui.Accent)

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(ui.Dim)

	return searchModel{
		input:   ti,
		spinner: sp,
		state:   stateInput,
		author:  author,
	}
}

func (m searchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Give the text input most of the width.
		m.input.Width = max(m.width-10, 20)
		return m, nil

	case searchDoneMsg:
		m.searched = true
		m.results = msg.results
		m.err = msg.err
		m.cursor = 0
		if m.err != nil || len(m.results) == 0 {
			// Re-focus input so the user can try again.
			m.state = stateInput
			m.input.Focus()
			return m, textinput.Blink
		}
		m.state = stateResults
		return m, nil

	case spinner.TickMsg:
		if m.state == stateSearching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Global: quit.
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
			m.quitting = true
			return m, tea.Quit
		}

		switch m.state {
		case stateInput:
			if msg.Type == tea.KeyEnter {
				query := strings.TrimSpace(m.input.Value())
				if query == "" {
					return m, nil
				}
				m.state = stateSearching
				m.err = nil
				m.input.Blur()
				return m, tea.Batch(m.spinner.Tick, searchCmd(query, m.author))
			}
			// Delegate all other keys to the text input.
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd

		case stateResults:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.results)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.results) > 0 {
					r := m.results[m.cursor]
					m.selected = &r
					m.quitting = true
					return m, tea.Quit
				}
			case "/":
				m.state = stateInput
				m.input.Focus()
				m.input.SetValue("")
				m.results = nil
				m.err = nil
				m.searched = false
				return m, textinput.Blink
			}
		}
	}

	return m, nil
}

func (m searchModel) View() string {
	if m.quitting {
		return ""
	}

	var lines []string

	// Title.
	lines = append(lines, "")
	lines = append(lines, "  "+titleStyle.Render("codectx search"))
	lines = append(lines, "")

	// Input line.
	inputLine := "  " + m.input.View()
	if m.state == stateInput && strings.TrimSpace(m.input.Value()) != "" {
		inputLine += "  " + hintStyle.Render("enter to search")
	}
	lines = append(lines, inputLine)
	lines = append(lines, "")

	// Content area.
	switch m.state {
	case stateInput:
		if m.err != nil {
			lines = append(lines, "  "+errorStyle.Render(ui.SymbolFail+" "+m.err.Error()))
		} else if m.searched {
			lines = append(lines, "  "+hintStyle.Render("No packages found. Try a different query."))
		} else {
			lines = append(lines, "  "+hintStyle.Render("Type a package name to search GitHub."))
		}
	case stateSearching:
		lines = append(lines, "  "+m.spinner.View()+" "+hintStyle.Render("Searching..."))
	case stateResults:
		lines = append(lines, m.renderTableLines()...)
		lines = append(lines, "")
		lines = append(lines,
			"  "+hintStyle.Render(fmt.Sprintf("%d package(s) found", len(m.results))))
	}

	// Pad to push help bar toward the bottom.
	topCount := len(lines)
	bottomCount := 2 // blank line + help
	padding := m.height - topCount - bottomCount
	if padding < 1 {
		padding = 1
	}
	for i := 0; i < padding; i++ {
		lines = append(lines, "")
	}

	// Help bar.
	lines = append(lines, "  "+hintStyle.Render(m.helpText()))

	return strings.Join(lines, "\n")
}

// helpText returns context-sensitive keybinding hints.
func (m searchModel) helpText() string {
	switch m.state {
	case stateInput:
		if strings.TrimSpace(m.input.Value()) != "" {
			return "enter search  esc quit"
		}
		return "esc quit"
	case stateSearching:
		return "esc cancel"
	case stateResults:
		if len(m.results) > 0 {
			return "\u2191/\u2193 navigate  enter select  / new search  esc quit"
		}
		return "/ new search  esc quit"
	}
	return ""
}

// renderTableLines builds the results table as a slice of lines.
func (m searchModel) renderTableLines() []string {
	if len(m.results) == 0 {
		return nil
	}

	const gap = 2

	// Calculate column widths.
	pkgW := len("PACKAGE")
	starsW := len("STARS")
	for _, r := range m.results {
		pkg := fmt.Sprintf("%s@%s", r.Name, r.Author)
		if len(pkg) > pkgW {
			pkgW = len(pkg)
		}
		s := strconv.Itoa(r.Stars)
		if len(s) > starsW {
			starsW = len(s)
		}
	}

	// Max description width.
	maxDesc := m.width - 4 - pkgW - gap - starsW - gap
	if maxDesc < 10 {
		maxDesc = 10
	}

	var lines []string

	// Header row.
	hdr := fmt.Sprintf("%-*s%-*s%s",
		pkgW+gap, "PACKAGE",
		starsW+gap, "STARS",
		"DESCRIPTION",
	)
	lines = append(lines, "  "+headerStyle.Render(hdr))

	// Determine visible window for scrolling.
	maxVisible := m.height - 12 // overhead: title(3) + input(2) + header(1) + count(2) + help(2) + pad(2)
	if maxVisible < 3 {
		maxVisible = 3
	}
	start, end := 0, len(m.results)
	if end > maxVisible {
		// Keep cursor in view.
		if m.cursor < maxVisible {
			end = maxVisible
		} else {
			end = m.cursor + 1
			start = end - maxVisible
		}
	}

	// Data rows.
	for i := start; i < end; i++ {
		r := m.results[i]
		pkg := fmt.Sprintf("%s@%s", r.Name, r.Author)
		stars := strconv.Itoa(r.Stars)
		desc := r.Description
		if len(desc) > maxDesc {
			desc = desc[:maxDesc-3] + "..."
		}

		row := fmt.Sprintf("%-*s%-*s%s",
			pkgW+gap, pkg,
			starsW+gap, stars,
			desc,
		)

		if i == m.cursor {
			lines = append(lines, selectorStyle.Render("\u25b8 ")+selectedStyle.Render(row))
		} else {
			lines = append(lines, "  "+row)
		}
	}

	// Scroll indicator.
	if end < len(m.results) {
		lines = append(lines,
			"  "+hintStyle.Render(fmt.Sprintf("  ... %d more", len(m.results)-end)))
	}
	if start > 0 {
		// Prepend a "more above" hint after the header.
		hint := "  " + hintStyle.Render(fmt.Sprintf("  ... %d above", start))
		lines = append(lines[:1], append([]string{hint}, lines[1:]...)...)
	}

	return lines
}

// searchCmd returns a tea.Cmd that performs the search asynchronously.
func searchCmd(query, author string) tea.Cmd {
	return func() tea.Msg {
		results, err := resolve.Search(query, author)
		return searchDoneMsg{results: results, err: err}
	}
}

// runInteractive launches the full-screen interactive search TUI.
func runInteractive(author string) error {
	if !ui.IsTTY() {
		return fmt.Errorf("interactive search requires a terminal; use: codectx search <query>")
	}

	m := newSearchModel(author)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return err
	}

	final := result.(searchModel)
	if final.selected != nil {
		ui.Blank()
		ui.Done(fmt.Sprintf("%s@%s", final.selected.Name, final.selected.Author))
		if final.selected.Description != "" {
			ui.KV("Description", final.selected.Description, 14)
		}
		ui.KV("Stars", final.selected.Stars, 14)
		ui.Blank()
		fmt.Printf("  Run: codectx add %s@%s\n\n", final.selected.Name, final.selected.Author)
	}

	return nil
}

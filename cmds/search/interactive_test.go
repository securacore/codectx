package search

import (
	"testing"

	"github.com/securacore/codectx/core/resolve"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearchModel_defaults(t *testing.T) {
	m := newSearchModel("")
	assert.Equal(t, stateInput, m.state)
	assert.Empty(t, m.author)
	assert.False(t, m.searched)
	assert.Nil(t, m.selected)
	assert.False(t, m.quitting)
}

func TestNewSearchModel_withAuthor(t *testing.T) {
	m := newSearchModel("facebook")
	assert.Equal(t, "facebook", m.author)
}

func TestSearchModel_windowSize(t *testing.T) {
	m := newSearchModel("")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(searchModel)
	assert.Equal(t, 120, result.width)
	assert.Equal(t, 40, result.height)
}

func TestSearchModel_quitCtrlC(t *testing.T) {
	m := newSearchModel("")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(searchModel)
	assert.True(t, result.quitting)
	assert.NotNil(t, cmd)
}

func TestSearchModel_quitEsc(t *testing.T) {
	m := newSearchModel("")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(searchModel)
	assert.True(t, result.quitting)
	assert.NotNil(t, cmd)
}

func TestSearchModel_enterEmptyInput(t *testing.T) {
	m := newSearchModel("")
	// Input is empty by default; pressing Enter should not start a search.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(searchModel)
	assert.Equal(t, stateInput, result.state)
}

func TestSearchModel_enterWithInput(t *testing.T) {
	m := newSearchModel("")
	m.input.SetValue("react")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(searchModel)
	assert.Equal(t, stateSearching, result.state)
	assert.NotNil(t, cmd) // should have batch of spinner.Tick + searchCmd
}

func TestSearchModel_searchDoneWithResults(t *testing.T) {
	m := newSearchModel("")
	m.state = stateSearching
	m.width = 100
	m.height = 30

	results := []resolve.SearchResult{
		{Name: "react", Author: "org", Description: "React docs", Stars: 42},
		{Name: "go", Author: "org", Description: "Go docs", Stars: 10},
	}

	updated, _ := m.Update(searchDoneMsg{results: results})
	result := updated.(searchModel)
	assert.Equal(t, stateResults, result.state)
	assert.Len(t, result.results, 2)
	assert.Equal(t, 0, result.cursor)
	assert.True(t, result.searched)
}

func TestSearchModel_searchDoneEmpty(t *testing.T) {
	m := newSearchModel("")
	m.state = stateSearching

	updated, _ := m.Update(searchDoneMsg{results: nil})
	result := updated.(searchModel)
	// Empty results should re-focus input.
	assert.Equal(t, stateInput, result.state)
	assert.True(t, result.searched)
}

func TestSearchModel_searchDoneError(t *testing.T) {
	m := newSearchModel("")
	m.state = stateSearching

	updated, _ := m.Update(searchDoneMsg{err: assert.AnError})
	result := updated.(searchModel)
	// Error should re-focus input.
	assert.Equal(t, stateInput, result.state)
	assert.Error(t, result.err)
	assert.True(t, result.searched)
}

func TestSearchModel_navigateResults(t *testing.T) {
	m := newSearchModel("")
	m.state = stateResults
	m.results = []resolve.SearchResult{
		{Name: "a", Author: "x"},
		{Name: "b", Author: "x"},
		{Name: "c", Author: "x"},
	}
	m.cursor = 0

	// Move down.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := updated.(searchModel)
	assert.Equal(t, 1, result.cursor)

	// Move down again.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result = updated.(searchModel)
	assert.Equal(t, 2, result.cursor)

	// Already at bottom, should stay.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result = updated.(searchModel)
	assert.Equal(t, 2, result.cursor)

	// Move up.
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result = updated.(searchModel)
	assert.Equal(t, 1, result.cursor)
}

func TestSearchModel_selectResult(t *testing.T) {
	m := newSearchModel("")
	m.state = stateResults
	m.results = []resolve.SearchResult{
		{Name: "react", Author: "org", Stars: 42},
	}
	m.cursor = 0

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(searchModel)
	assert.True(t, result.quitting)
	require.NotNil(t, result.selected)
	assert.Equal(t, "react", result.selected.Name)
	assert.Equal(t, "org", result.selected.Author)
	assert.NotNil(t, cmd)
}

func TestSearchModel_slashNewSearch(t *testing.T) {
	m := newSearchModel("")
	m.state = stateResults
	m.results = []resolve.SearchResult{{Name: "react", Author: "org"}}
	m.searched = true
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	result := updated.(searchModel)
	assert.Equal(t, stateInput, result.state)
	assert.Empty(t, result.input.Value())
	assert.Nil(t, result.results)
	assert.False(t, result.searched)
}

func TestSearchModel_viewInput(t *testing.T) {
	m := newSearchModel("")
	m.width = 80
	m.height = 24
	view := m.View()
	assert.Contains(t, view, "codectx search")
	assert.Contains(t, view, "Type a package name")
	assert.Contains(t, view, "esc quit")
}

func TestSearchModel_viewResults(t *testing.T) {
	m := newSearchModel("")
	m.state = stateResults
	m.width = 100
	m.height = 30
	m.searched = true
	m.results = []resolve.SearchResult{
		{Name: "react", Author: "org", Description: "React docs", Stars: 42},
	}

	view := m.View()
	assert.Contains(t, view, "PACKAGE")
	assert.Contains(t, view, "STARS")
	assert.Contains(t, view, "DESCRIPTION")
	assert.Contains(t, view, "react@org")
	assert.Contains(t, view, "42")
	assert.Contains(t, view, "1 package(s) found")
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "select")
}

func TestSearchModel_viewSearching(t *testing.T) {
	m := newSearchModel("")
	m.state = stateSearching
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "Searching...")
	assert.Contains(t, view, "cancel")
}

func TestSearchModel_viewError(t *testing.T) {
	m := newSearchModel("")
	m.state = stateInput
	m.err = assert.AnError
	m.searched = true
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, assert.AnError.Error())
}

func TestSearchModel_viewQuitting(t *testing.T) {
	m := newSearchModel("")
	m.quitting = true
	view := m.View()
	assert.Empty(t, view)
}

func TestSearchModel_helpTextInput(t *testing.T) {
	m := newSearchModel("")
	m.state = stateInput
	assert.Equal(t, "esc quit", m.helpText())

	m.input.SetValue("react")
	assert.Contains(t, m.helpText(), "enter search")
}

func TestSearchModel_helpTextResults(t *testing.T) {
	m := newSearchModel("")
	m.state = stateResults
	m.results = []resolve.SearchResult{{Name: "a", Author: "x"}}
	help := m.helpText()
	assert.Contains(t, help, "navigate")
	assert.Contains(t, help, "select")
	assert.Contains(t, help, "new search")
}

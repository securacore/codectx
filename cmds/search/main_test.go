package search

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/securacore/codectx/core/resolve"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "search", Command.Name)
	assert.NotEmpty(t, Command.Usage)
	assert.Equal(t, "<query>", Command.ArgsUsage)
}

func TestCommand_authorFlag(t *testing.T) {
	require := assert.New(t)
	require.Len(Command.Flags, 1)
	flag := Command.Flags[0]
	assert.Equal(t, "author", flag.Names()[0])
}

// captureStdout runs fn and returns whatever it writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

// --- runWith ---

func TestRunWith_noResults(t *testing.T) {
	search := func(query, author string) ([]resolve.SearchResult, error) {
		assert.Equal(t, "react", query)
		assert.Equal(t, "", author)
		return nil, nil
	}

	out := captureStdout(t, func() {
		err := runWith("react", "", search)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "No packages found")
}

func TestRunWith_withResults(t *testing.T) {
	search := func(query, author string) ([]resolve.SearchResult, error) {
		return []resolve.SearchResult{
			{Name: "react", Author: "facebook", Description: "React conventions", Stars: 42},
			{Name: "go", Author: "securacore", Description: "Go conventions", Stars: 10},
		}, nil
	}

	out := captureStdout(t, func() {
		err := runWith("react", "", search)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "react@facebook")
	assert.Contains(t, out, "42")
	assert.Contains(t, out, "go@securacore")
	assert.Contains(t, out, "10")
	assert.Contains(t, out, "2 package(s) found")
}

func TestRunWith_searchError(t *testing.T) {
	search := func(query, author string) ([]resolve.SearchResult, error) {
		return nil, fmt.Errorf("rate limited")
	}

	err := runWith("react", "", search)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search:")
	assert.Contains(t, err.Error(), "rate limited")
}

func TestRunWith_passesAuthor(t *testing.T) {
	var capturedAuthor string
	search := func(query, author string) ([]resolve.SearchResult, error) {
		capturedAuthor = author
		return nil, nil
	}

	out := captureStdout(t, func() {
		err := runWith("react", "facebook", search)
		require.NoError(t, err)
	})
	assert.Equal(t, "facebook", capturedAuthor)
	assert.Contains(t, out, "No packages found")
}

func TestRunWith_truncatesLongDescription(t *testing.T) {
	longDesc := "This is a very long description that exceeds sixty characters and should be truncated"
	search := func(query, author string) ([]resolve.SearchResult, error) {
		return []resolve.SearchResult{
			{Name: "pkg", Author: "org", Description: longDesc, Stars: 1},
		}, nil
	}

	out := captureStdout(t, func() {
		err := runWith("pkg", "", search)
		require.NoError(t, err)
	})
	// Description should be truncated to 57 chars + "..."
	assert.Contains(t, out, "...")
	assert.NotContains(t, out, longDesc)
	assert.Contains(t, out, "1 package(s) found")
}

func TestRunWith_singleResult(t *testing.T) {
	search := func(query, author string) ([]resolve.SearchResult, error) {
		return []resolve.SearchResult{
			{Name: "tailwind", Author: "org", Description: "Tailwind CSS", Stars: 100},
		}, nil
	}

	out := captureStdout(t, func() {
		err := runWith("tailwind", "", search)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "tailwind@org")
	assert.Contains(t, out, "100")
	assert.Contains(t, out, "1 package(s) found")
	assert.Contains(t, out, "codectx add <package>")
}

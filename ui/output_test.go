package ui

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout runs fn and returns whatever it writes to os.Stdout.
// ANSI escape codes are included in the output; use assert.Contains
// for assertions since styled text wraps content in escape sequences.
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

// --- Done ---

func TestDone(t *testing.T) {
	out := captureStdout(t, func() { Done("Compiled to .codectx") })
	assert.Contains(t, out, SymbolDone)
	assert.Contains(t, out, "Compiled to .codectx")
}

// --- Warn ---

func TestWarn(t *testing.T) {
	out := captureStdout(t, func() { Warn("2 conflicts") })
	assert.Contains(t, out, SymbolWarn)
	assert.Contains(t, out, "2 conflicts")
}

// --- Fail ---

func TestFail(t *testing.T) {
	out := captureStdout(t, func() { Fail("resolve failed") })
	assert.Contains(t, out, SymbolFail)
	assert.Contains(t, out, "resolve failed")
}

// --- Step ---

func TestStep(t *testing.T) {
	out := captureStdout(t, func() { Step("Resolving...") })
	assert.Contains(t, out, "Resolving...")
}

// --- Header ---

func TestHeader(t *testing.T) {
	out := captureStdout(t, func() { Header("Created:") })
	assert.Contains(t, out, "Created:")
}

// --- Item ---

func TestItem(t *testing.T) {
	out := captureStdout(t, func() { Item("codectx.yml") })
	assert.Contains(t, out, SymbolBullet)
	assert.Contains(t, out, "codectx.yml")
}

// --- ItemDetail ---

func TestItemDetail(t *testing.T) {
	out := captureStdout(t, func() { ItemDetail("CLAUDE.md", "backed up") })
	assert.Contains(t, out, SymbolBullet)
	assert.Contains(t, out, "CLAUDE.md")
	assert.Contains(t, out, "backed up")
}

// --- KV ---

func TestKV(t *testing.T) {
	out := captureStdout(t, func() { KV("Files copied", 42, 16) })
	assert.Contains(t, out, "Files copied")
	assert.Contains(t, out, "42")
}

// --- Canceled ---

func TestCanceled(t *testing.T) {
	out := captureStdout(t, func() { Canceled() })
	assert.Contains(t, out, "Canceled.")
}

// --- Table ---

func TestTable_basic(t *testing.T) {
	headers := []string{"PACKAGE", "STARS", "DESCRIPTION"}
	rows := [][]string{
		{"react@org", "42", "React conventions"},
		{"go@google", "10", "Go patterns"},
	}

	out := captureStdout(t, func() { Table(headers, rows, 2) })
	assert.Contains(t, out, "PACKAGE")
	assert.Contains(t, out, "react@org")
	assert.Contains(t, out, "go@google")
	assert.Contains(t, out, "42")
	assert.Contains(t, out, "React conventions")
}

func TestTable_empty(t *testing.T) {
	out := captureStdout(t, func() { Table([]string{}, [][]string{}, 2) })
	assert.Empty(t, out)
}

func TestTable_noRows(t *testing.T) {
	headers := []string{"NAME", "VALUE"}
	out := captureStdout(t, func() { Table(headers, [][]string{}, 2) })
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "VALUE")
}

func TestTable_raggedRows(t *testing.T) {
	headers := []string{"NAME", "VALUE", "EXTRA"}
	rows := [][]string{
		{"full", "row", "data"},
		{"short"},        // fewer cells than headers
		{"two", "cells"}, // also fewer
	}

	out := captureStdout(t, func() { Table(headers, rows, 2) })
	// All header labels should appear.
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "VALUE")
	assert.Contains(t, out, "EXTRA")
	// All provided cell values should appear.
	assert.Contains(t, out, "full")
	assert.Contains(t, out, "short")
	assert.Contains(t, out, "two")
	assert.Contains(t, out, "cells")
}

func TestTable_singleColumn(t *testing.T) {
	headers := []string{"NAME"}
	rows := [][]string{
		{"alpha"},
		{"beta"},
	}

	out := captureStdout(t, func() { Table(headers, rows, 2) })
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "beta")
}

// --- Blank ---

func TestBlank(t *testing.T) {
	out := captureStdout(t, func() { Blank() })
	assert.Equal(t, "\n", out)
}

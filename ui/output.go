package ui

import (
	"fmt"
	"strings"
)

// Done prints a success message with a green checkmark.
//
//	✓ Compiled to .codectx
func Done(msg string) {
	fmt.Printf("  %s %s\n", greenStyle.Render(SymbolDone), msg)
}

// Warn prints a warning message with a yellow indicator.
//
//	! 2 conflict(s) detected
func Warn(msg string) {
	fmt.Printf("  %s %s\n", yellowStyle.Render(SymbolWarn), msg)
}

// Fail prints an error message with a red indicator.
//
//	✗ resolve failed
func Fail(msg string) {
	fmt.Printf("  %s %s\n", redStyle.Render(SymbolFail), msg)
}

// Step prints a dim in-progress message (non-TTY fallback for spinner).
//
//	○ Resolving react@org...
func Step(msg string) {
	fmt.Printf("  %s %s\n", dimStyle.Render(SymbolSpinner), dimStyle.Render(msg))
}

// Header prints a bold section label.
//
//	Created:
func Header(msg string) {
	fmt.Printf("  %s\n", boldStyle.Render(msg))
}

// Item prints an indented list item with a dim bullet.
//
//   - codectx.yml
func Item(msg string) {
	fmt.Printf("  %s %s\n", dimStyle.Render(SymbolBullet), msg)
}

// ItemDetail prints a list item with a dim parenthetical detail.
//
//   - .windsurfrules (backed up)
func ItemDetail(msg, detail string) {
	fmt.Printf("  %s %s %s\n", dimStyle.Render(SymbolBullet), msg, dimStyle.Render("("+detail+")"))
}

// KV prints a key-value pair with the key dimmed and right-padded.
// The width parameter sets the total key column width.
//
//	Files copied   42
func KV(key string, value any, width int) {
	format := fmt.Sprintf("  %%-%ds %%v\n", width)
	fmt.Printf(format, dimStyle.Render(key), value)
}

// Cancelled prints a cancellation message.
func Cancelled() {
	fmt.Printf("  %s\n", dimStyle.Render("Cancelled."))
}

// Table prints a column-aligned table with dim headers.
// The gap parameter controls spacing between columns.
func Table(headers []string, rows [][]string, gap int) {
	if len(headers) == 0 {
		return
	}

	// Compute column widths from headers and data.
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Build format string.
	parts := make([]string, len(headers))
	for i, w := range widths {
		if i == len(widths)-1 {
			parts[i] = "%s" // last column: no padding
		} else {
			parts[i] = fmt.Sprintf("%%-%ds", w+gap)
		}
	}
	format := "  " + strings.Join(parts, "") + "\n"

	// Print dim header.
	headerArgs := make([]any, len(headers))
	for i, h := range headers {
		if i == len(headers)-1 {
			headerArgs[i] = dimStyle.Render(h)
		} else {
			padded := fmt.Sprintf(fmt.Sprintf("%%-%ds", widths[i]+gap), h)
			headerArgs[i] = dimStyle.Render(padded)
		}
	}
	fmt.Printf("  %s\n", strings.TrimRight(fmt.Sprintf(strings.Repeat("%s", len(headers)), headerArgs...), " "))

	// Print rows.
	for _, row := range rows {
		args := make([]any, len(headers))
		for i := range headers {
			if i < len(row) {
				args[i] = row[i]
			} else {
				args[i] = ""
			}
		}
		fmt.Printf(format, args...)
	}
}

// Blank prints an empty line for visual spacing.
func Blank() {
	fmt.Println()
}

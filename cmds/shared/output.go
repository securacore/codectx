package shared

import (
	"fmt"
	"os"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
)

// OutputDocumentParams configures document output routing.
type OutputDocumentParams struct {
	Content  []byte
	FilePath string // empty = stdout
	Header   string // printed before content (prompt only, empty for generate)
	Footer   string // printed after content (summary)
}

// OutputDocument writes document content to file or stdout, routing
// metadata (header/footer) to the appropriate destination.
//
// When FilePath is set, the document is written to the file and both
// Header and Footer are printed to stdout for the user. When FilePath
// is empty (pipe-friendly mode), Header and Footer go to stderr so only
// the document content reaches stdout.
func OutputDocument(p OutputDocumentParams) error {
	if p.FilePath != "" {
		if err := os.WriteFile(p.FilePath, p.Content, project.FilePerm); err != nil {
			fmt.Print(tui.ErrorMsg{
				Title:  "Failed to write file",
				Detail: []string{err.Error()},
			}.Render())
			return fmt.Errorf("writing file: %w", err)
		}
		fmt.Print(p.Header)
		fmt.Print(p.Footer)
	} else {
		fmt.Fprint(os.Stderr, p.Header)
		fmt.Print(string(p.Content))
		fmt.Fprint(os.Stderr, p.Footer)
	}
	return nil
}

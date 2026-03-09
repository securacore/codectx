package shared

import (
	"fmt"
	"os"
	"strings"
)

// EnsureGitignoreEntry adds the given entry to the .gitignore file at the
// specified path if it is not already present. Creates the file if it does
// not exist. The entry should be the exact line to match/add (e.g., ".codectx/").
func EnsureGitignoreEntry(gitignorePath, entry string) error {
	if data, err := os.ReadFile(gitignorePath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == entry {
				return nil
			}
		}
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open .gitignore: %w", err)
		}
		defer func() { _ = f.Close() }()
		if len(data) > 0 && data[len(data)-1] != '\n' {
			if _, err := f.WriteString("\n"); err != nil {
				return fmt.Errorf("write newline to .gitignore: %w", err)
			}
		}
		if _, err := f.WriteString(entry + "\n"); err != nil {
			return fmt.Errorf("append to .gitignore: %w", err)
		}
		return nil
	}

	return os.WriteFile(gitignorePath, []byte(entry+"\n"), 0o644)
}

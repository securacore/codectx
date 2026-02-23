package link

// Tool defines an AI tool entry point file.
type Tool struct {
	Name   string // display name (e.g., "Claude Code")
	File   string // entry point filename (e.g., "CLAUDE.md")
	SubDir string // optional subdirectory (e.g., ".github" for Copilot)
}

// Tools is the built-in list of supported AI tools.
var Tools = []Tool{
	{Name: "Claude Code", File: "CLAUDE.md"},
	{Name: "Agents", File: "AGENTS.md"},
	{Name: "Cursor", File: ".cursorrules"},
	{Name: "Windsurf", File: ".windsurfrules"},
	{Name: "GitHub Copilot", File: "copilot-instructions.md", SubDir: ".github"},
}

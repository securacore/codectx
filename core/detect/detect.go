// Package detect scans the system for installed AI CLI tools and configured
// API keys. The detection results are used during codectx init to set sensible
// defaults in ai.yml based on what's actually available on the user's system.
package detect

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/securacore/codectx/core/project"
)

// Tool represents a detected AI CLI tool with its binary path and version.
type Tool struct {
	// Name is the human-readable name of the tool.
	Name string

	// Binary is the executable name (e.g., "claude", "opencode").
	Binary string

	// Path is the absolute path to the binary, as resolved by LookPath.
	Path string

	// Version is the version string returned by the tool, if available.
	Version string
}

// Provider represents a detected API provider based on environment variables.
type Provider struct {
	// Name is the provider name (e.g., "Anthropic", "OpenAI").
	Name string

	// EnvVar is the environment variable that was found set.
	EnvVar string

	// DefaultModel is the recommended default compilation model for this provider.
	DefaultModel string
}

// Result holds the complete detection output.
type Result struct {
	// Tools lists all detected AI CLI tools, in priority order.
	Tools []Tool

	// Providers lists all detected API providers based on env vars.
	Providers []Provider

	// RecommendedModel is the best default model based on what was detected.
	// Falls back to "claude-sonnet-4-20250514" if nothing is detected.
	RecommendedModel string

	// RecommendedEncoding is the tokenizer encoding for the recommended model.
	RecommendedEncoding string
}

// toolSpec defines a tool to scan for.
type toolSpec struct {
	name       string
	binary     string
	versionCmd []string // command to get version (e.g., ["claude", "--version"])
}

// providerSpec defines an API provider to check via environment variables.
type providerSpec struct {
	name         string
	envVar       string
	defaultModel string
}

// knownTools is the ordered list of AI CLI tools to scan for.
// Order reflects detection priority — first found with a matching
// provider gets to set the recommended model.
var knownTools = []toolSpec{
	{name: "Claude Code", binary: binaryClaude, versionCmd: []string{binaryClaude, "--version"}},
	{name: "Codex", binary: binaryCodex, versionCmd: []string{binaryCodex, "--version"}},
	{name: "Amp", binary: "amp", versionCmd: []string{"amp", "--version"}},
	{name: "Aider", binary: "aider", versionCmd: []string{"aider", "--version"}},
	{name: "OpenCode", binary: "opencode", versionCmd: []string{"opencode", "--version"}},
	{name: "Ollama", binary: "ollama", versionCmd: []string{"ollama", "--version"}},
	{name: "Goose", binary: "goose", versionCmd: []string{"goose", "--version"}},
	{name: "Cursor", binary: "cursor", versionCmd: []string{"cursor", "--version"}},
}

// knownProviders is the ordered list of API providers to check.
// Order reflects recommendation priority.
var knownProviders = []providerSpec{
	{name: providerAnthropic, envVar: "ANTHROPIC_API_KEY", defaultModel: project.DefaultModel},
	{name: providerOpenAI, envVar: "OPENAI_API_KEY", defaultModel: modelGPT4o},
	{name: "Google", envVar: "GEMINI_API_KEY", defaultModel: "gemini-2.0-flash"},
	{name: "Google", envVar: "GOOGLE_API_KEY", defaultModel: "gemini-2.0-flash"},
	{name: "Groq", envVar: "GROQ_API_KEY", defaultModel: "llama-3.3-70b-versatile"},
	{name: "DeepSeek", envVar: "DEEPSEEK_API_KEY", defaultModel: "deepseek-chat"},
	{name: "xAI", envVar: "XAI_API_KEY", defaultModel: "grok-2"},
	{name: "OpenRouter", envVar: "OPENROUTER_API_KEY", defaultModel: "anthropic/claude-sonnet-4"},
	{name: "Mistral", envVar: "MISTRAL_API_KEY", defaultModel: "mistral-large-latest"},
}

const (
	// modelGPT4o is the OpenAI GPT-4o model identifier.
	modelGPT4o = "gpt-4o"

	// providerAnthropic is the Anthropic provider name.
	providerAnthropic = "Anthropic"

	// providerOpenAI is the OpenAI provider name.
	providerOpenAI = "OpenAI"

	// binaryClaude is the Claude Code CLI binary name.
	binaryClaude = "claude"

	// binaryCodex is the Codex CLI binary name.
	binaryCodex = "codex"
)

// LookPathFunc is the function used to locate binaries. Defaults to exec.LookPath.
// Override in tests to control detection results.
var LookPathFunc = exec.LookPath

// GetenvFunc is the function used to read environment variables. Defaults to os.Getenv.
// Override in tests to control detection results.
var GetenvFunc = os.Getenv

// RunCommandFunc is the function used to run version commands. Defaults to runCommand.
// Override in tests to control version output.
var RunCommandFunc = runCommand

// Scan performs a full system scan for AI tools and API providers.
// It returns the detection result with a recommended model based on
// what was found.
func Scan() Result {
	result := Result{
		RecommendedModel:    project.DefaultModel,
		RecommendedEncoding: project.DefaultEncoding,
	}

	// Scan for installed tools.
	for _, spec := range knownTools {
		path, err := LookPathFunc(spec.binary)
		if err != nil {
			continue
		}

		tool := Tool{
			Name:   spec.name,
			Binary: spec.binary,
			Path:   path,
		}

		// Try to get version.
		if version, err := RunCommandFunc(spec.versionCmd[0], spec.versionCmd[1:]...); err == nil {
			tool.Version = cleanVersion(version)
		}

		result.Tools = append(result.Tools, tool)
	}

	// Scan for API provider environment variables.
	seen := make(map[string]bool)
	for _, spec := range knownProviders {
		if GetenvFunc(spec.envVar) != "" && !seen[spec.name] {
			result.Providers = append(result.Providers, Provider{
				Name:         spec.name,
				EnvVar:       spec.envVar,
				DefaultModel: spec.defaultModel,
			})
			seen[spec.name] = true
		}
	}

	// Set recommended model based on detection priority.
	result.RecommendedModel, result.RecommendedEncoding = recommendModel(result)

	return result
}

// recommendModel selects the best default model based on detected tools and providers.
func recommendModel(r Result) (string, string) {
	// Priority 1: If Anthropic API key is set, use Claude.
	for _, p := range r.Providers {
		if p.Name == providerAnthropic {
			return project.DefaultModel, EncodingForModel(project.DefaultModel)
		}
	}

	// Priority 2: If Claude Code is installed (implies Anthropic access).
	for _, t := range r.Tools {
		if t.Binary == binaryClaude {
			return project.DefaultModel, EncodingForModel(project.DefaultModel)
		}
	}

	// Priority 3: If OpenAI API key is set, use GPT-4o.
	for _, p := range r.Providers {
		if p.Name == providerOpenAI {
			return modelGPT4o, EncodingForModel(modelGPT4o)
		}
	}

	// Priority 4: If Codex is installed (implies OpenAI access).
	for _, t := range r.Tools {
		if t.Binary == binaryCodex {
			return modelGPT4o, EncodingForModel(modelGPT4o)
		}
	}

	// Priority 5: First available provider.
	if len(r.Providers) > 0 {
		return r.Providers[0].DefaultModel, EncodingForModel(r.Providers[0].DefaultModel)
	}

	// Fallback.
	return project.DefaultModel, project.DefaultEncoding
}

// runCommand executes a command and returns its stdout output.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// cleanVersion extracts a clean version string from command output.
// Handles formats like:
//   - "2.1.63 (Claude Code)"  → "2.1.63"
//   - "ollama version is 0.17.5" → "0.17.5"
//   - "v1.2.3\n" → "v1.2.3"
func cleanVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Take only the first line.
	if idx := strings.IndexByte(raw, '\n'); idx >= 0 {
		raw = raw[:idx]
	}

	// If it contains "version is", take what follows.
	if idx := strings.Index(raw, "version is "); idx >= 0 {
		return strings.TrimSpace(raw[idx+len("version is "):])
	}

	// If the first field looks like a version (starts with digit or 'v'), use it.
	fields := strings.Fields(raw)
	if len(fields) > 0 {
		first := fields[0]
		if len(first) > 0 && (first[0] >= '0' && first[0] <= '9' || first[0] == 'v') {
			return first
		}
	}

	// Fall back to the full first line.
	return raw
}

// EncodingForModel returns the appropriate tokenizer encoding for a model.
// OpenAI models (gpt-4o, o1, o3-mini) use o200k_base; all others default to cl100k_base.
func EncodingForModel(model string) string {
	switch model {
	case modelGPT4o, "o1", "o3-mini":
		return "o200k_base"
	default:
		return project.DefaultEncoding
	}
}

// HasTools returns true if any AI CLI tools were detected.
func (r Result) HasTools() bool {
	return len(r.Tools) > 0
}

// HasProviders returns true if any API providers were detected.
func (r Result) HasProviders() bool {
	return len(r.Providers) > 0
}

// HasAnything returns true if any tools or providers were detected.
func (r Result) HasAnything() bool {
	return r.HasTools() || r.HasProviders()
}

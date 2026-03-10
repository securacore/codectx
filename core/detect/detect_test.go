package detect_test

import (
	"fmt"
	"testing"

	"github.com/securacore/codectx/core/detect"
)

// withMocks replaces the detection functions with test mocks and restores them after.
func withMocks(
	lookPath func(string) (string, error),
	getenv func(string) string,
	runCmd func(string, ...string) (string, error),
	fn func(),
) {
	origLook := detect.LookPathFunc
	origEnv := detect.GetenvFunc
	origRun := detect.RunCommandFunc

	detect.LookPathFunc = lookPath
	detect.GetenvFunc = getenv
	detect.RunCommandFunc = runCmd

	defer func() {
		detect.LookPathFunc = origLook
		detect.GetenvFunc = origEnv
		detect.RunCommandFunc = origRun
	}()

	fn()
}

func TestScan_NoToolsNoProviders(t *testing.T) {
	withMocks(
		func(string) (string, error) { return "", fmt.Errorf("not found") },
		func(string) string { return "" },
		func(string, ...string) (string, error) { return "", fmt.Errorf("not found") },
		func() {
			result := detect.Scan()

			if result.HasTools() {
				t.Error("expected no tools detected")
			}
			if result.HasProviders() {
				t.Error("expected no providers detected")
			}
			if result.HasAnything() {
				t.Error("expected HasAnything to be false")
			}
			if result.RecommendedModel != "claude-sonnet-4-20250514" {
				t.Errorf("expected default model, got %q", result.RecommendedModel)
			}
			if result.RecommendedEncoding != "cl100k_base" {
				t.Errorf("expected default encoding, got %q", result.RecommendedEncoding)
			}
		},
	)
}

func TestScan_DetectsClaudeCode(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "claude" {
				return "/usr/local/bin/claude", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, args ...string) (string, error) {
			if name == "claude" {
				return "2.1.63 (Claude Code)\n", nil
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()

			if !result.HasTools() {
				t.Fatal("expected tools to be detected")
			}
			if len(result.Tools) != 1 {
				t.Fatalf("expected 1 tool, got %d", len(result.Tools))
			}

			tool := result.Tools[0]
			if tool.Name != "Claude Code" {
				t.Errorf("expected tool name 'Claude Code', got %q", tool.Name)
			}
			if tool.Binary != "claude" {
				t.Errorf("expected binary 'claude', got %q", tool.Binary)
			}
			if tool.Path != "/usr/local/bin/claude" {
				t.Errorf("expected path '/usr/local/bin/claude', got %q", tool.Path)
			}
			if tool.Version != "2.1.63" {
				t.Errorf("expected version '2.1.63', got %q", tool.Version)
			}

			// Claude Code implies Anthropic — model should be Claude.
			if result.RecommendedModel != "claude-sonnet-4-20250514" {
				t.Errorf("expected claude model, got %q", result.RecommendedModel)
			}
		},
	)
}

func TestScan_DetectsOpenAIProvider(t *testing.T) {
	withMocks(
		func(string) (string, error) { return "", fmt.Errorf("not found") },
		func(key string) string {
			if key == "OPENAI_API_KEY" {
				return "sk-test-key"
			}
			return ""
		},
		func(string, ...string) (string, error) { return "", fmt.Errorf("not found") },
		func() {
			result := detect.Scan()

			if !result.HasProviders() {
				t.Fatal("expected providers to be detected")
			}
			if len(result.Providers) != 1 {
				t.Fatalf("expected 1 provider, got %d", len(result.Providers))
			}

			provider := result.Providers[0]
			if provider.Name != "OpenAI" {
				t.Errorf("expected provider 'OpenAI', got %q", provider.Name)
			}

			if result.RecommendedModel != "gpt-4o" {
				t.Errorf("expected gpt-4o model, got %q", result.RecommendedModel)
			}
			if result.RecommendedEncoding != "o200k_base" {
				t.Errorf("expected o200k_base encoding, got %q", result.RecommendedEncoding)
			}
		},
	)
}

func TestScan_AnthropicPriorityOverOpenAI(t *testing.T) {
	withMocks(
		func(string) (string, error) { return "", fmt.Errorf("not found") },
		func(key string) string {
			switch key {
			case "ANTHROPIC_API_KEY":
				return "sk-ant-test"
			case "OPENAI_API_KEY":
				return "sk-openai-test"
			}
			return ""
		},
		func(string, ...string) (string, error) { return "", fmt.Errorf("not found") },
		func() {
			result := detect.Scan()

			if len(result.Providers) != 2 {
				t.Fatalf("expected 2 providers, got %d", len(result.Providers))
			}

			// Anthropic should take priority.
			if result.RecommendedModel != "claude-sonnet-4-20250514" {
				t.Errorf("expected claude model (Anthropic priority), got %q", result.RecommendedModel)
			}
		},
	)
}

func TestScan_MultipleTools(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			switch binary {
			case "claude":
				return "/usr/local/bin/claude", nil
			case "ollama":
				return "/usr/local/bin/ollama", nil
			case "aider":
				return "/usr/local/bin/aider", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, args ...string) (string, error) {
			switch name {
			case "claude":
				return "2.1.63 (Claude Code)", nil
			case "ollama":
				return "ollama version is 0.17.5", nil
			case "aider":
				return "aider v0.82.0\n", nil
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()

			if len(result.Tools) != 3 {
				t.Fatalf("expected 3 tools, got %d", len(result.Tools))
			}

			// Check ordering matches knownTools priority.
			if result.Tools[0].Binary != "claude" {
				t.Errorf("expected claude first, got %q", result.Tools[0].Binary)
			}
			if result.Tools[1].Binary != "aider" {
				t.Errorf("expected aider second, got %q", result.Tools[1].Binary)
			}
			if result.Tools[2].Binary != "ollama" {
				t.Errorf("expected ollama third, got %q", result.Tools[2].Binary)
			}

			// Check version cleaning.
			if result.Tools[0].Version != "2.1.63" {
				t.Errorf("expected claude version '2.1.63', got %q", result.Tools[0].Version)
			}
			if result.Tools[2].Version != "0.17.5" {
				t.Errorf("expected ollama version '0.17.5', got %q", result.Tools[2].Version)
			}
		},
	)
}

func TestScan_VersionCommandFails(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "claude" {
				return "/usr/local/bin/claude", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(string, ...string) (string, error) { return "", fmt.Errorf("command failed") },
		func() {
			result := detect.Scan()

			if len(result.Tools) != 1 {
				t.Fatalf("expected 1 tool, got %d", len(result.Tools))
			}

			// Tool should still be detected, just without version.
			if result.Tools[0].Version != "" {
				t.Errorf("expected empty version when command fails, got %q", result.Tools[0].Version)
			}
		},
	)
}

func TestScan_DeduplicatesGoogleProvider(t *testing.T) {
	withMocks(
		func(string) (string, error) { return "", fmt.Errorf("not found") },
		func(key string) string {
			// Both Google env vars are set.
			switch key {
			case "GEMINI_API_KEY":
				return "gemini-key"
			case "GOOGLE_API_KEY":
				return "google-key"
			}
			return ""
		},
		func(string, ...string) (string, error) { return "", fmt.Errorf("not found") },
		func() {
			result := detect.Scan()

			// Should only have one Google provider, not two.
			googleCount := 0
			for _, p := range result.Providers {
				if p.Name == "Google" {
					googleCount++
				}
			}
			if googleCount != 1 {
				t.Errorf("expected 1 Google provider (deduplicated), got %d", googleCount)
			}
		},
	)
}

func TestScan_RecommendModel_Priority4_CodexImpliesOpenAI(t *testing.T) {
	// Priority 4: Codex is installed (implies OpenAI access) but no Anthropic key
	// and no Claude tool. Should recommend gpt-4o.
	withMocks(
		func(binary string) (string, error) {
			if binary == "codex" {
				return "/usr/local/bin/codex", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" }, // No API keys set
		func(name string, args ...string) (string, error) {
			if name == "codex" {
				return "codex v1.0.0\n", nil
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()

			if !result.HasTools() {
				t.Fatal("expected codex tool to be detected")
			}
			if result.Tools[0].Binary != "codex" {
				t.Errorf("expected codex tool, got %q", result.Tools[0].Binary)
			}

			// Priority 4: Codex installed → recommend gpt-4o.
			if result.RecommendedModel != "gpt-4o" {
				t.Errorf("expected gpt-4o model (Codex implies OpenAI), got %q", result.RecommendedModel)
			}
			if result.RecommendedEncoding != "o200k_base" {
				t.Errorf("expected o200k_base encoding, got %q", result.RecommendedEncoding)
			}
		},
	)
}

func TestScan_RecommendModel_Priority5_FirstNonMajorProvider(t *testing.T) {
	// Priority 5: Only a non-major provider (Groq) is available — no Anthropic,
	// no OpenAI, no Claude, no Codex. Should use the first provider's default model.
	withMocks(
		func(string) (string, error) { return "", fmt.Errorf("not found") }, // No tools
		func(key string) string {
			if key == "GROQ_API_KEY" {
				return "gsk-test-key"
			}
			return ""
		},
		func(string, ...string) (string, error) { return "", fmt.Errorf("not found") },
		func() {
			result := detect.Scan()

			if !result.HasProviders() {
				t.Fatal("expected Groq provider to be detected")
			}
			if result.Providers[0].Name != "Groq" {
				t.Errorf("expected Groq provider, got %q", result.Providers[0].Name)
			}

			// Priority 5: First available provider → Groq's default model.
			if result.RecommendedModel != "llama-3.3-70b-versatile" {
				t.Errorf("expected llama-3.3-70b-versatile model (Groq first provider), got %q", result.RecommendedModel)
			}
			if result.RecommendedEncoding != "cl100k_base" {
				t.Errorf("expected cl100k_base encoding, got %q", result.RecommendedEncoding)
			}
		},
	)
}

// ---------------------------------------------------------------------------
// cleanVersion coverage (tested via Scan with controlled version output)
// ---------------------------------------------------------------------------

func TestScan_CleanVersion_EmptyOutput(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "claude" {
				return "/usr/bin/claude", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, args ...string) (string, error) {
			if name == "claude" {
				return "", nil // Empty version output
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()
			if len(result.Tools) != 1 {
				t.Fatalf("expected 1 tool, got %d", len(result.Tools))
			}
			if result.Tools[0].Version != "" {
				t.Errorf("expected empty version for empty output, got %q", result.Tools[0].Version)
			}
		},
	)
}

func TestScan_CleanVersion_MultilineOutput(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "claude" {
				return "/usr/bin/claude", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, args ...string) (string, error) {
			if name == "claude" {
				return "2.3.0\nExtra info line\n", nil
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()
			// Should take only the first line and extract the version.
			if result.Tools[0].Version != "2.3.0" {
				t.Errorf("expected version '2.3.0' from multiline, got %q", result.Tools[0].Version)
			}
		},
	)
}

func TestScan_CleanVersion_VersionIsFormat(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "ollama" {
				return "/usr/bin/ollama", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, args ...string) (string, error) {
			if name == "ollama" {
				return "ollama version is 0.17.5\n", nil
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()
			if result.Tools[0].Version != "0.17.5" {
				t.Errorf("expected '0.17.5' from 'version is' format, got %q", result.Tools[0].Version)
			}
		},
	)
}

func TestScan_CleanVersion_VPrefix(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "aider" {
				return "/usr/bin/aider", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, args ...string) (string, error) {
			if name == "aider" {
				return "v0.82.0\n", nil
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()
			if result.Tools[0].Version != "v0.82.0" {
				t.Errorf("expected 'v0.82.0' from v-prefixed version, got %q", result.Tools[0].Version)
			}
		},
	)
}

func TestScan_CleanVersion_NonVersionFirstField(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "goose" {
				return "/usr/bin/goose", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, args ...string) (string, error) {
			if name == "goose" {
				return "goose-cli build 2025\n", nil // First field doesn't start with digit or v
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()
			// Should fall back to the full first line.
			if result.Tools[0].Version != "goose-cli build 2025" {
				t.Errorf("expected full first line as version fallback, got %q", result.Tools[0].Version)
			}
		},
	)
}

// ---------------------------------------------------------------------------
// EncodingForModel
// ---------------------------------------------------------------------------

func TestEncodingForModel_GPT4o(t *testing.T) {
	if enc := detect.EncodingForModel(detect.ModelGPT4o); enc != "o200k_base" {
		t.Errorf("expected o200k_base for gpt-4o, got %q", enc)
	}
}

func TestEncodingForModel_O1(t *testing.T) {
	if enc := detect.EncodingForModel("o1"); enc != "o200k_base" {
		t.Errorf("expected o200k_base for o1, got %q", enc)
	}
}

func TestEncodingForModel_O3Mini(t *testing.T) {
	if enc := detect.EncodingForModel("o3-mini"); enc != "o200k_base" {
		t.Errorf("expected o200k_base for o3-mini, got %q", enc)
	}
}

func TestEncodingForModel_Claude(t *testing.T) {
	if enc := detect.EncodingForModel(detect.DefaultModel); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for Claude, got %q", enc)
	}
}

func TestEncodingForModel_Gemini(t *testing.T) {
	if enc := detect.EncodingForModel("gemini-2.0-flash"); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for Gemini, got %q", enc)
	}
}

func TestEncodingForModel_Unknown(t *testing.T) {
	if enc := detect.EncodingForModel("some-custom-model"); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for unknown model, got %q", enc)
	}
}

func TestEncodingForModel_Empty(t *testing.T) {
	if enc := detect.EncodingForModel(""); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for empty string, got %q", enc)
	}
}

// ---------------------------------------------------------------------------
// Result helpers
// ---------------------------------------------------------------------------

func TestResult_HasTools(t *testing.T) {
	empty := detect.Result{}
	if empty.HasTools() {
		t.Error("empty result should not have tools")
	}

	withTool := detect.Result{Tools: []detect.Tool{{Name: "test"}}}
	if !withTool.HasTools() {
		t.Error("result with tool should have tools")
	}
}

func TestResult_HasProviders(t *testing.T) {
	empty := detect.Result{}
	if empty.HasProviders() {
		t.Error("empty result should not have providers")
	}

	withProvider := detect.Result{Providers: []detect.Provider{{Name: "test"}}}
	if !withProvider.HasProviders() {
		t.Error("result with provider should have providers")
	}
}

func TestResult_HasAnything(t *testing.T) {
	empty := detect.Result{}
	if empty.HasAnything() {
		t.Error("empty result should not have anything")
	}

	withTool := detect.Result{Tools: []detect.Tool{{Name: "test"}}}
	if !withTool.HasAnything() {
		t.Error("result with tool should have something")
	}

	withProvider := detect.Result{Providers: []detect.Provider{{Name: "test"}}}
	if !withProvider.HasAnything() {
		t.Error("result with provider should have something")
	}
}

func TestDefaultModel_IsSet(t *testing.T) {
	if detect.DefaultModel == "" {
		t.Error("DefaultModel should not be empty")
	}
	if detect.DefaultModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected default model %q, got %q", "claude-sonnet-4-20250514", detect.DefaultModel)
	}
}

func TestDefaultEncoding_IsSet(t *testing.T) {
	if detect.DefaultEncoding == "" {
		t.Error("DefaultEncoding should not be empty")
	}
	if detect.DefaultEncoding != "cl100k_base" {
		t.Errorf("expected default encoding %q, got %q", "cl100k_base", detect.DefaultEncoding)
	}
}

func TestScan_CleanVersion_WhitespaceOnly(t *testing.T) {
	withMocks(
		func(binary string) (string, error) {
			if binary == "goose" {
				return "/usr/bin/goose", nil
			}
			return "", fmt.Errorf("not found")
		},
		func(string) string { return "" },
		func(name string, _ ...string) (string, error) {
			if name == "goose" {
				return "   \n  \n", nil // Whitespace-only output
			}
			return "", fmt.Errorf("not found")
		},
		func() {
			result := detect.Scan()
			if len(result.Tools) == 0 {
				t.Fatal("expected at least one tool")
			}
			// cleanVersion should return empty string for whitespace-only input.
			if result.Tools[0].Version != "" {
				t.Errorf("expected empty version for whitespace-only output, got %q", result.Tools[0].Version)
			}
		},
	)
}

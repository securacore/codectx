package shared

import (
	"testing"

	"github.com/securacore/codectx/core/detect"
	"github.com/securacore/codectx/core/project"
)

func TestDetectProviderCapabilities(t *testing.T) {
	tests := []struct {
		name    string
		result  detect.Result
		wantCLI bool
		wantAPI bool
	}{
		{
			name:    "empty result",
			result:  detect.Result{},
			wantCLI: false,
			wantAPI: false,
		},
		{
			name: "CLI only",
			result: detect.Result{
				Tools: []detect.Tool{{Binary: "claude"}},
			},
			wantCLI: true,
			wantAPI: false,
		},
		{
			name: "API only",
			result: detect.Result{
				Providers: []detect.Provider{{Name: "Anthropic"}},
			},
			wantCLI: false,
			wantAPI: true,
		},
		{
			name: "both CLI and API",
			result: detect.Result{
				Tools:     []detect.Tool{{Binary: "claude"}},
				Providers: []detect.Provider{{Name: "Anthropic"}},
			},
			wantCLI: true,
			wantAPI: true,
		},
		{
			name: "non-claude tool",
			result: detect.Result{
				Tools: []detect.Tool{{Binary: "cursor"}},
			},
			wantCLI: false,
			wantAPI: false,
		},
		{
			name: "non-anthropic provider",
			result: detect.Result{
				Providers: []detect.Provider{{Name: "OpenAI"}},
			},
			wantCLI: false,
			wantAPI: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCLI, gotAPI := DetectProviderCapabilities(tt.result)
			if gotCLI != tt.wantCLI {
				t.Errorf("hasCLI = %v, want %v", gotCLI, tt.wantCLI)
			}
			if gotAPI != tt.wantAPI {
				t.Errorf("hasAPI = %v, want %v", gotAPI, tt.wantAPI)
			}
		})
	}
}

func TestAutoSelectProvider(t *testing.T) {
	tests := []struct {
		name   string
		hasCLI bool
		hasAPI bool
		want   string
	}{
		{"both available, CLI preferred", true, true, project.ProviderCLI},
		{"only CLI", true, false, project.ProviderCLI},
		{"only API", false, true, project.ProviderAPI},
		{"neither available", false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoSelectProvider(tt.hasCLI, tt.hasAPI)
			if got != tt.want {
				t.Errorf("AutoSelectProvider(%v, %v) = %q, want %q", tt.hasCLI, tt.hasAPI, got, tt.want)
			}
		})
	}
}

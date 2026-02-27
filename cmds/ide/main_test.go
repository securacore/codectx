package ide

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/core/ide/launcher"
	"github.com/securacore/codectx/core/preferences"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command structure ---

func TestCommand_structure(t *testing.T) {
	assert.Equal(t, "ide", Command.Name)
	assert.Equal(t, "Launch an AI documentation authoring session", Command.Usage)
}

func TestCommand_flags(t *testing.T) {
	flags := Command.Flags
	require.Len(t, flags, 2)

	names := make(map[string]bool)
	for _, f := range flags {
		for _, n := range f.Names() {
			names[n] = true
		}
	}
	assert.True(t, names["resume"], "expected --resume flag")
	assert.True(t, names["list"], "expected --list flag")
}

func TestCommand_actionIsWired(t *testing.T) {
	assert.NotNil(t, Command.Action)
}

// --- listSessions ---

func TestListSessions(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, outputDir string)
	}{
		{
			name:  "no sessions",
			setup: func(t *testing.T, outputDir string) {},
		},
		{
			name: "single session with category",
			setup: func(t *testing.T, outputDir string) {
				s := coreide.NewSession("claude")
				s.Title = "Test document"
				s.Category = "topic"
				require.NoError(t, coreide.Save(outputDir, s))
			},
		},
		{
			name: "multiple sessions in different phases",
			setup: func(t *testing.T, outputDir string) {
				s1 := coreide.NewSession("claude")
				s1.Title = "Discover session"
				s1.Phase = coreide.PhaseDiscover
				require.NoError(t, coreide.Save(outputDir, s1))

				s2 := coreide.NewSession("opencode")
				s2.Title = "Draft session"
				s2.Phase = coreide.PhaseDraft
				s2.Category = "foundation"
				require.NoError(t, coreide.Save(outputDir, s2))
			},
		},
		{
			name: "session without category",
			setup: func(t *testing.T, outputDir string) {
				s := coreide.NewSession("claude")
				s.Title = "No category"
				s.Category = ""
				require.NoError(t, coreide.Save(outputDir, s))
			},
		},
		{
			name:  "sessions directory does not exist",
			setup: func(t *testing.T, outputDir string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			outputDir := filepath.Join(dir, ".codectx")
			require.NoError(t, os.MkdirAll(outputDir, 0o755))
			tt.setup(t, outputDir)

			err := listSessions(outputDir)
			assert.NoError(t, err)
		})
	}
}

// --- pickOrCreateSession ---

func TestPickOrCreateSession(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, outputDir string) *coreide.Session
		wantTitle  string
		wantBin    string
		wantPhase  coreide.Phase
		wantResume bool // true = should resume the setup session
	}{
		{
			name: "creates new when no active sessions",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				return nil
			},
			wantTitle:  "New document",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDiscover,
			wantResume: false,
		},
		{
			name: "auto-resumes single active session",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				s := coreide.NewSession("claude")
				s.Title = "Existing session"
				s.Phase = coreide.PhaseDraft
				require.NoError(t, coreide.Save(outputDir, s))
				return s
			},
			wantTitle:  "Existing session",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDraft,
			wantResume: true,
		},
		{
			name: "creates new when only completed sessions exist",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				s := coreide.NewSession("claude")
				s.Title = "Done session"
				s.Phase = coreide.PhaseComplete
				require.NoError(t, coreide.Save(outputDir, s))
				return s
			},
			wantTitle:  "New document",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDiscover,
			wantResume: false,
		},
		{
			name: "persists new session to disk",
			setup: func(t *testing.T, outputDir string) *coreide.Session {
				return nil
			},
			wantTitle:  "New document",
			wantBin:    "claude",
			wantPhase:  coreide.PhaseDiscover,
			wantResume: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			outputDir := filepath.Join(dir, ".codectx")
			require.NoError(t, os.MkdirAll(outputDir, 0o755))

			setupSession := tt.setup(t, outputDir)

			l := launcher.NewClaude("/usr/bin/claude")
			session, err := pickOrCreateSession(outputDir, l)
			require.NoError(t, err)
			require.NotNil(t, session)

			assert.Equal(t, tt.wantTitle, session.Title)
			assert.Equal(t, tt.wantBin, session.Bin)
			assert.Equal(t, tt.wantPhase, session.Phase)

			if tt.wantResume {
				assert.Equal(t, setupSession.ID, session.ID)
			} else if setupSession != nil {
				assert.NotEqual(t, setupSession.ID, session.ID)
			}

			// Every created session should be loadable from disk.
			loaded, err := coreide.Load(outputDir, session.ID)
			require.NoError(t, err)
			assert.Equal(t, session.ID, loaded.ID)
		})
	}
}

// --- assemblePrompt ---

// setupProject creates a temp directory with a codectx.yml and optional
// preferences for testing assemblePrompt.
func setupProject(t *testing.T, prefs *preferences.Preferences) *config.Config {
	t.Helper()
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	docsDir := cfg.DocsDir()
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	outputDir := cfg.OutputDir()
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	if prefs != nil {
		require.NoError(t, preferences.Write(outputDir, prefs))
	}

	return cfg
}

func TestAssemblePrompt(t *testing.T) {
	compTrue := true
	compFalse := false

	tests := []struct {
		name        string
		prefs       *preferences.Preferences
		contains    []string
		notContains []string
	}{
		{
			name:     "includes embedded directive",
			prefs:    nil,
			contains: []string{"documentation", "codectx"},
		},
		{
			name: "includes compression enabled",
			prefs: &preferences.Preferences{
				Compression: &compTrue,
				AI:          &preferences.AIConfig{Class: "claude-sonnet-class"},
			},
			contains: []string{
				"Compression is **enabled**",
				"claude-sonnet-class",
				"Project Preferences",
			},
		},
		{
			name: "includes compression disabled",
			prefs: &preferences.Preferences{
				Compression: &compFalse,
			},
			contains: []string{"Compression is **disabled**"},
		},
		{
			name:        "omits preferences section when empty",
			prefs:       &preferences.Preferences{},
			notContains: []string{"Project Preferences"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := setupProject(t, tt.prefs)

			prompt, err := assemblePrompt(cfg)
			require.NoError(t, err)
			assert.NotEmpty(t, prompt)

			for _, s := range tt.contains {
				assert.Contains(t, prompt, s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, prompt, s)
			}
		})
	}
}

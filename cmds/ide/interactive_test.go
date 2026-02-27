package ide

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/securacore/codectx/core/config"
	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/core/llm"
	"github.com/securacore/codectx/core/manifest"
)

// ansiRe strips ANSI escape sequences for test assertions.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// --- Helpers ---

// stubProvider implements llm.Provider for testing without a real AI backend.
type stubProvider struct{}

func (s *stubProvider) Stream(_ context.Context, _ *llm.Request) (<-chan llm.Event, error) {
	ch := make(chan llm.Event)
	close(ch)
	return ch, nil
}

func (s *stubProvider) ID() string { return "test" }

// testModel creates an ideModel with reasonable defaults for testing.
func testModel() ideModel {
	session := coreide.NewSession("test-provider")
	m := newModel(session, &stubProvider{}, "test system prompt", "/tmp/test-output", "docs", "/tmp/test-root")
	m.width = 120
	m.height = 40
	m.resize()
	return m
}

// --- Model initialization ---

func TestNewModel(t *testing.T) {
	m := testModel()

	assert.Equal(t, viewChat, m.view)
	assert.False(t, m.quitting)
	assert.False(t, m.saved)
	assert.False(t, m.streaming)
	assert.Empty(t, m.messages)
	assert.Empty(t, m.docBlocks)
	assert.Empty(t, m.streamingContent)
	assert.Equal(t, "test system prompt", m.prompt)
	assert.Equal(t, "/tmp/test-output", m.outputDir)
	assert.Equal(t, "/tmp/test-root", m.rootDir)
}

func TestInit(t *testing.T) {
	m := testModel()
	cmd := m.Init()
	// Init returns textarea.Blink command.
	assert.NotNil(t, cmd)
}

// --- Window sizing ---

func TestWindowSizeMsg(t *testing.T) {
	m := testModel()

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	result := updated.(ideModel)

	assert.Nil(t, cmd)
	assert.Equal(t, 200, result.width)
	assert.Equal(t, 60, result.height)
	assert.Equal(t, 200, result.chatViewport.Width)
}

func TestResizeMinChatHeight(t *testing.T) {
	m := testModel()

	// Very small terminal.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 6})
	result := updated.(ideModel)

	// Chat height should be clamped to minimum of 3.
	assert.Equal(t, 3, result.chatViewport.Height)
}

// --- Global quit ---

func TestCtrlCQuits(t *testing.T) {
	m := testModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(ideModel)

	assert.True(t, result.quitting)
	assert.NotNil(t, cmd) // tea.Quit
}

func TestEscQuits(t *testing.T) {
	m := testModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(ideModel)

	assert.True(t, result.quitting)
	assert.NotNil(t, cmd) // tea.Quit
}

// --- Stream messages ---

func TestStreamDelta(t *testing.T) {
	m := testModel()
	m.streaming = true

	updated, _ := m.Update(streamDeltaMsg{content: "Hello "})
	result := updated.(ideModel)
	assert.Equal(t, "Hello ", result.streamingContent)

	updated, _ = result.Update(streamDeltaMsg{content: "world"})
	result = updated.(ideModel)
	assert.Equal(t, "Hello world", result.streamingContent)
}

func TestStreamToolUse(t *testing.T) {
	m := testModel()
	m.streaming = true

	updated, _ := m.Update(streamToolMsg{tool: "Read"})
	result := updated.(ideModel)

	assert.Len(t, result.messages, 1)
	assert.Equal(t, "tool", result.messages[0].role)
	assert.Equal(t, "Read", result.messages[0].content)
}

func TestStreamDone(t *testing.T) {
	m := testModel()
	m.streaming = true
	m.streamingContent = "partial content"

	updated, _ := m.Update(streamDoneMsg{
		sessionID: "sess-123",
		content:   "final response",
	})
	result := updated.(ideModel)

	assert.False(t, result.streaming)
	assert.Empty(t, result.streamingContent)
	assert.Len(t, result.messages, 1)
	assert.Equal(t, "assistant", result.messages[0].role)
	assert.Equal(t, "final response", result.messages[0].content)
	assert.Equal(t, "sess-123", result.session.ProviderSession)
}

func TestStreamDoneWithDocumentBlocks(t *testing.T) {
	m := testModel()
	m.streaming = true

	content := `Here is the document:
<document path="docs/test.md">
# Test
Content here
</document>`

	updated, _ := m.Update(streamDoneMsg{content: content})
	result := updated.(ideModel)

	assert.Len(t, result.docBlocks, 1)
	assert.Equal(t, "docs/test.md", result.docBlocks[0].Path)
}

func TestStreamDoneUsesStreamedContentWhenResultEmpty(t *testing.T) {
	m := testModel()
	m.streaming = true
	m.streamingContent = "streamed content"

	updated, _ := m.Update(streamDoneMsg{content: ""})
	result := updated.(ideModel)

	assert.Equal(t, "streamed content", result.messages[0].content)
}

func TestStreamError(t *testing.T) {
	m := testModel()
	m.streaming = true
	m.streamingContent = "partial"

	updated, _ := m.Update(streamErrMsg{err: assert.AnError})
	result := updated.(ideModel)

	assert.False(t, result.streaming)
	assert.Empty(t, result.streamingContent)
	assert.Len(t, result.messages, 1)
	assert.Equal(t, "error", result.messages[0].role)
}

// --- Chat input ---

func TestInputIgnoredWhileStreaming(t *testing.T) {
	m := testModel()
	m.streaming = true

	// Try to type and send — should be ignored.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(ideModel)

	assert.Nil(t, cmd)
	assert.Empty(t, result.messages)
}

func TestEmptyInputNotSent(t *testing.T) {
	m := testModel()

	// Enter on empty input does not send a message.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(ideModel)

	assert.Empty(t, result.messages)
	assert.False(t, result.streaming)
}

// --- Preview toggle ---

func TestCtrlPWithoutBlocksStaysInChat(t *testing.T) {
	m := testModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	result := updated.(ideModel)

	// No doc blocks, so preview should not open.
	assert.Equal(t, viewChat, result.view)
}

func TestCtrlPWithBlocksOpensPreview(t *testing.T) {
	m := testModel()
	m.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/test.md", Content: "# Test"},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	result := updated.(ideModel)

	assert.Equal(t, viewPreview, result.view)
}

func TestPreviewEscReturnsToChat(t *testing.T) {
	m := testModel()
	m.view = viewPreview
	m.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/test.md", Content: "# Test"},
	}
	m.preview = newPreview(m.docBlocks, m.width, m.height)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(ideModel)

	assert.Equal(t, viewChat, result.view)
}

func TestPreviewCtrlPReturnsToChat(t *testing.T) {
	m := testModel()
	m.view = viewPreview
	m.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/test.md", Content: "# Test"},
	}
	m.preview = newPreview(m.docBlocks, m.width, m.height)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	result := updated.(ideModel)

	assert.Equal(t, viewChat, result.view)
}

func TestPreviewTabCyclesFiles(t *testing.T) {
	blocks := []coreide.DocumentBlock{
		{Path: "docs/a.md", Content: "A"},
		{Path: "docs/b.md", Content: "B"},
	}
	m := testModel()
	m.view = viewPreview
	m.docBlocks = blocks
	m.preview = newPreview(blocks, m.width, m.height)

	assert.Equal(t, 0, m.preview.current)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	result := updated.(ideModel)
	assert.Equal(t, 1, result.preview.current)

	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyTab})
	result = updated.(ideModel)
	assert.Equal(t, 0, result.preview.current) // Wraps around
}

func TestPreviewRequestChangesReturnsToChat(t *testing.T) {
	m := testModel()
	m.view = viewPreview
	m.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/test.md", Content: "# Test"},
	}
	m.preview = newPreview(m.docBlocks, m.width, m.height)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := updated.(ideModel)

	assert.Equal(t, viewChat, result.view)
	assert.Equal(t, "Please change: ", result.input.Value())
}

// --- Write completion ---

func TestWriteCompleteMsg(t *testing.T) {
	m := testModel()

	updated, cmd := m.Update(writeCompleteMsg{files: []string{"docs/test.md"}, compiled: false})
	result := updated.(ideModel)

	assert.True(t, result.saved)
	assert.True(t, result.quitting)
	assert.Equal(t, coreide.PhaseComplete, result.session.Phase)
	assert.NotNil(t, cmd) // tea.Quit
}

// --- View rendering ---

func TestViewQuittingReturnsEmpty(t *testing.T) {
	m := testModel()
	m.quitting = true

	assert.Empty(t, m.View())
}

func TestViewChatContainsHeader(t *testing.T) {
	m := testModel()

	view := m.View()
	assert.Contains(t, view, "codectx ide")
}

func TestViewPreviewShowsPreviewHeader(t *testing.T) {
	m := testModel()
	m.view = viewPreview
	m.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/test.md", Content: "# Test content"},
	}
	m.preview = newPreview(m.docBlocks, m.width, m.height)

	view := m.View()
	assert.Contains(t, view, "PREVIEW")
	assert.Contains(t, view, "docs/test.md")
}

func TestViewChatShowsHelpWithPreviewHint(t *testing.T) {
	m := testModel()
	m.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/test.md", Content: "# Test"},
	}

	view := m.View()
	assert.Contains(t, view, "ctrl+p preview")
	assert.Contains(t, view, "ctrl+w write")
}

func TestViewChatShowsHelpWithoutPreviewHint(t *testing.T) {
	m := testModel()

	view := m.View()
	assert.Contains(t, view, "ctrl+p preview")
	assert.NotContains(t, view, "ctrl+w write")
}

// --- Helper functions ---

func TestVisualLen(t *testing.T) {
	assert.Equal(t, 5, visualLen("hello"))
	assert.Equal(t, 0, visualLen(""))
	// ANSI escape sequences should be stripped.
	assert.Equal(t, 5, visualLen("\x1b[31mhello\x1b[0m"))
}

func TestPadLine(t *testing.T) {
	result := padLine("left", "right", 40)
	assert.Contains(t, result, "left")
	assert.Contains(t, result, "right")
	assert.Len(t, result, 40)
}

func TestPadLineEmptyRight(t *testing.T) {
	result := padLine("left", "", 40)
	assert.Equal(t, "left", result)
}

func TestPadLineNarrowWidth(t *testing.T) {
	// When width is too small, gap should be at least 1.
	result := padLine("left", "right", 5)
	assert.Contains(t, result, "left right")
}

// --- Preview model ---

func TestNewPreviewEmpty(t *testing.T) {
	p := newPreview(nil, 80, 40)
	assert.Equal(t, 0, p.current)
	assert.Empty(t, p.blocks)
}

func TestNewPreviewSetsFirstBlock(t *testing.T) {
	blocks := []coreide.DocumentBlock{
		{Path: "a.md", Content: "AAA"},
		{Path: "b.md", Content: "BBB"},
	}
	p := newPreview(blocks, 80, 40)
	assert.Equal(t, 0, p.current)
	assert.Len(t, p.blocks, 2)
}

func TestNextFileSingleBlock(t *testing.T) {
	p := newPreview([]coreide.DocumentBlock{{Path: "a.md", Content: "A"}}, 80, 40)
	p.nextFile()
	assert.Equal(t, 0, p.current) // No change with single block
}

func TestNextFileWraps(t *testing.T) {
	blocks := []coreide.DocumentBlock{
		{Path: "a.md", Content: "A"},
		{Path: "b.md", Content: "B"},
		{Path: "c.md", Content: "C"},
	}
	p := newPreview(blocks, 80, 40)

	p.nextFile()
	assert.Equal(t, 1, p.current)
	p.nextFile()
	assert.Equal(t, 2, p.current)
	p.nextFile()
	assert.Equal(t, 0, p.current) // Wraps
}

func TestPreviewResize(t *testing.T) {
	p := newPreview(nil, 80, 40)
	p.resize(100, 50)
	assert.Equal(t, 100, p.width)
	assert.Equal(t, 50, p.height)
	assert.Equal(t, 100, p.viewport.Width)
	assert.Equal(t, 46, p.viewport.Height) // 50 - 4
}

// --- Chat rendering ---

func TestRenderChatEmpty(t *testing.T) {
	result := renderChat(nil, "", 80)
	assert.Empty(t, result)
}

func TestRenderChatWithMessages(t *testing.T) {
	msgs := []chatMessage{
		{role: "user", content: "Hello"},
		{role: "assistant", content: "Hi there"},
	}
	result := renderChat(msgs, "", 80)
	plain := stripANSI(result)
	assert.Contains(t, plain, "Hello")
	assert.Contains(t, plain, "Hi there")
}

func TestRenderChatWithStreaming(t *testing.T) {
	result := renderChat(nil, "partial response", 80)
	assert.Contains(t, result, "partial response")
	assert.Contains(t, result, "...")
}

func TestRenderMessageRoles(t *testing.T) {
	tests := []struct {
		role    string
		content string
	}{
		{"user", "user message"},
		{"assistant", "assistant message"},
		{"tool", "Read"},
		{"error", "something went wrong"},
	}

	for _, tt := range tests {
		var b strings.Builder
		renderMessage(&b, chatMessage{role: tt.role, content: tt.content}, 80)
		plain := stripANSI(b.String())
		assert.Contains(t, plain, tt.content)
	}
}

// --- Input handling ---

func TestInputUpdateEnterSendsText(t *testing.T) {
	ta := newTextArea()
	ta.SetValue("hello")

	text, sent := inputUpdate(&ta, tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, sent)
	assert.Equal(t, "hello", text)
	assert.Empty(t, ta.Value()) // Reset after send
}

func TestInputUpdateEnterEmptyNotSent(t *testing.T) {
	ta := newTextArea()

	text, sent := inputUpdate(&ta, tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, sent)
	assert.Empty(t, text)
}

func TestInputUpdateNonEnterNotSent(t *testing.T) {
	ta := newTextArea()
	ta.SetValue("hello")

	text, sent := inputUpdate(&ta, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.False(t, sent)
	assert.Empty(t, text)
}

// --- sendMessage ---

func TestSendMessage_setsState(t *testing.T) {
	m := testModel()

	result, cmd := m.sendMessage("Hello AI")
	updated := result.(ideModel)

	// Should add user message, set streaming, and return a command.
	assert.True(t, updated.streaming)
	assert.Empty(t, updated.streamingContent)
	assert.Len(t, updated.messages, 1)
	assert.Equal(t, "user", updated.messages[0].role)
	assert.Equal(t, "Hello AI", updated.messages[0].content)
	assert.NotNil(t, cmd)
}

func TestSendMessage_cmdProducesStreamDone(t *testing.T) {
	m := testModel()

	_, cmd := m.sendMessage("Hello")

	// Execute the command closure — stubProvider's Stream returns a closed channel,
	// so the loop exits immediately with an empty result.
	msg := cmd()
	done, ok := msg.(streamDoneMsg)
	assert.True(t, ok, "expected streamDoneMsg, got %T", msg)
	assert.Empty(t, done.content) // stub returns empty
}

type errorProvider struct{}

func (e *errorProvider) Stream(_ context.Context, _ *llm.Request) (<-chan llm.Event, error) {
	return nil, assert.AnError
}

func (e *errorProvider) ID() string { return "error-test" }

func TestSendMessage_cmdProducesStreamErr(t *testing.T) {
	session := coreide.NewSession("test")
	m := newModel(session, &errorProvider{}, "prompt", "/tmp/out", "docs", "/tmp/root")
	m.width = 120
	m.height = 40
	m.resize()

	_, cmd := m.sendMessage("Hello")
	msg := cmd()
	_, ok := msg.(streamErrMsg)
	assert.True(t, ok, "expected streamErrMsg, got %T", msg)
}

type deltaProvider struct{}

func (d *deltaProvider) Stream(_ context.Context, _ *llm.Request) (<-chan llm.Event, error) {
	ch := make(chan llm.Event, 4)
	ch <- llm.DeltaEvent("Hello ")
	ch <- llm.DeltaEvent("world")
	ch <- llm.ResultEvent("Hello world", "sess-42", &llm.Usage{InputTokens: 10, OutputTokens: 20})
	close(ch)
	return ch, nil
}

func (d *deltaProvider) ID() string { return "delta-test" }

func TestSendMessage_cmdCollectsStreamEvents(t *testing.T) {
	session := coreide.NewSession("test")
	m := newModel(session, &deltaProvider{}, "prompt", "/tmp/out", "docs", "/tmp/root")
	m.width = 120
	m.height = 40
	m.resize()

	_, cmd := m.sendMessage("Hello")
	msg := cmd()
	done, ok := msg.(streamDoneMsg)
	require.True(t, ok, "expected streamDoneMsg, got %T", msg)
	assert.Equal(t, "Hello world", done.content)
	assert.Equal(t, "sess-42", done.sessionID)
	assert.NotNil(t, done.usage)
	assert.Equal(t, 10, done.usage.InputTokens)
}

type errorEventProvider struct{}

func (e *errorEventProvider) Stream(_ context.Context, _ *llm.Request) (<-chan llm.Event, error) {
	ch := make(chan llm.Event, 2)
	ch <- llm.DeltaEvent("partial ")
	ch <- llm.ErrorEvent("rate limit exceeded")
	close(ch)
	return ch, nil
}

func (e *errorEventProvider) ID() string { return "error-event-test" }

func TestSendMessage_cmdHandlesErrorEvent(t *testing.T) {
	session := coreide.NewSession("test")
	m := newModel(session, &errorEventProvider{}, "prompt", "/tmp/out", "docs", "/tmp/root")
	m.width = 120
	m.height = 40
	m.resize()

	_, cmd := m.sendMessage("Hello")
	msg := cmd()
	errMsg, ok := msg.(streamErrMsg)
	require.True(t, ok, "expected streamErrMsg, got %T", msg)
	assert.Contains(t, errMsg.err.Error(), "rate limit exceeded")
}

// --- writeDocuments ---

func TestWriteDocuments_cmd(t *testing.T) {
	rootDir := t.TempDir()
	docsDir := filepath.Join(rootDir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics"), 0o755))

	// Write a minimal manifest so WriteAndSync can sync.
	m := &manifest.Manifest{Name: "test"}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	session := coreide.NewSession("test")
	model := newModel(session, &stubProvider{}, "prompt", rootDir, docsDir, rootDir)
	model.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/topics/my-topic/README.md", Content: "# My Topic\nContent"},
	}

	cmd := model.writeDocuments()
	assert.NotNil(t, cmd)

	msg := cmd()
	wc, ok := msg.(writeCompleteMsg)
	require.True(t, ok, "expected writeCompleteMsg, got %T", msg)
	assert.Len(t, wc.files, 1)
	assert.Equal(t, "docs/topics/my-topic/README.md", wc.files[0])

	// Verify file was actually written.
	data, err := os.ReadFile(filepath.Join(rootDir, "docs/topics/my-topic/README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "# My Topic")
}

func TestWriteDocuments_cmdError(t *testing.T) {
	session := coreide.NewSession("test")
	// rootDir is /dev/null which will fail to create directories.
	model := newModel(session, &stubProvider{}, "prompt", "/dev/null/impossible", "docs", "/dev/null/impossible")
	model.docBlocks = []coreide.DocumentBlock{
		{Path: "docs/test.md", Content: "test"},
	}

	cmd := model.writeDocuments()
	msg := cmd()
	_, ok := msg.(streamErrMsg)
	assert.True(t, ok, "expected streamErrMsg on write failure, got %T", msg)
}

// --- listSessions ---

func TestListSessions_empty(t *testing.T) {
	outputDir := t.TempDir()
	// No sessions directory at all.
	err := listSessions(outputDir)
	assert.NoError(t, err)
}

func TestListSessions_withSessions(t *testing.T) {
	outputDir := t.TempDir()

	// Create a session.
	s := coreide.NewSession("test-provider")
	s.Title = "Test Document"
	s.Category = "foundation"
	require.NoError(t, coreide.Save(outputDir, s))

	err := listSessions(outputDir)
	assert.NoError(t, err)
}

func TestListSessions_multipleSessions(t *testing.T) {
	outputDir := t.TempDir()

	s1 := coreide.NewSession("test-provider")
	s1.Title = "First"
	require.NoError(t, coreide.Save(outputDir, s1))

	s2 := coreide.NewSession("test-provider")
	s2.Title = "Second"
	s2.Category = "topics"
	require.NoError(t, coreide.Save(outputDir, s2))

	err := listSessions(outputDir)
	assert.NoError(t, err)
}

// --- assemblePrompt ---

func TestAssemblePrompt_withManifest(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	// Write a manifest with entries.
	m := &manifest.Manifest{
		Name:    "test",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Core philosophy"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	cfg := &config.Config{
		Name: "test",
		Config: &config.BuildConfig{
			DocsDir:   docsDir,
			OutputDir: outputDir,
		},
	}

	prompt, err := assemblePrompt(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "philosophy") // manifest summary should include entry
}

func TestAssemblePrompt_noManifest(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	cfg := &config.Config{
		Name: "test",
		Config: &config.BuildConfig{
			DocsDir:   docsDir,
			OutputDir: outputDir,
		},
	}

	// Should gracefully fall back to empty manifest.
	prompt, err := assemblePrompt(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, prompt) // still returns directive even without entries
}

// --- Unknown messages ---

func TestUnknownMessageNoop(t *testing.T) {
	type unknownMsg struct{}
	m := testModel()

	updated, cmd := m.Update(unknownMsg{})
	result := updated.(ideModel)

	assert.Nil(t, cmd)
	assert.Equal(t, viewChat, result.view)
	assert.False(t, result.quitting)
}

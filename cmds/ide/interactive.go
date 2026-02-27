package ide

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/core/llm"
	"github.com/securacore/codectx/ui"
)

// viewState tracks which view is currently active.
type viewState int

const (
	viewChat viewState = iota
	viewPreview
)

// ideModel is the root bubbletea model for the IDE TUI.
type ideModel struct {
	// Core state.
	session  *coreide.Session
	provider llm.Provider
	prompt   string // Assembled system prompt

	// Configuration.
	outputDir string
	docsDir   string // Documentation source directory (e.g., "docs")
	rootDir   string // Project root for file writing

	// View state.
	view     viewState
	quitting bool
	saved    bool // Whether files were written

	// Chat components.
	messages         []chatMessage
	streamingContent string
	streaming        bool
	chatViewport     viewport.Model
	input            textarea.Model

	// Preview.
	preview   previewModel
	docBlocks []coreide.DocumentBlock

	// Dimensions.
	width  int
	height int

	// Context for cancellation.
	ctx    context.Context
	cancel context.CancelFunc
}

// Bubbletea message types.
type streamDeltaMsg struct{ content string }
type streamToolMsg struct{ tool string }
type streamDoneMsg struct {
	sessionID string
	content   string
	usage     *llm.Usage
}
type streamErrMsg struct{ err error }
type writeCompleteMsg struct {
	files    []string
	compiled bool
}

func newModel(session *coreide.Session, provider llm.Provider, prompt, outputDir, docsDir, rootDir string) ideModel {
	ctx, cancel := context.WithCancel(context.Background())
	vp := viewport.New(80, 20)
	ta := newTextArea()

	m := ideModel{
		session:      session,
		provider:     provider,
		prompt:       prompt,
		outputDir:    outputDir,
		docsDir:      docsDir,
		rootDir:      rootDir,
		view:         viewChat,
		chatViewport: vp,
		input:        ta,
		ctx:          ctx,
		cancel:       cancel,
	}

	return m
}

func (m ideModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m ideModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			m.cancel()
			return m, tea.Quit
		}

		switch m.view {
		case viewChat:
			return m.updateChat(msg)
		case viewPreview:
			return m.updatePreview(msg)
		}

	case streamDeltaMsg:
		m.streamingContent += msg.content
		m.chatViewport.SetContent(renderChat(m.messages, m.streamingContent, m.width))
		m.chatViewport.GotoBottom()
		return m, nil

	case streamToolMsg:
		m.messages = append(m.messages, chatMessage{role: "tool", content: msg.tool})
		m.chatViewport.SetContent(renderChat(m.messages, m.streamingContent, m.width))
		m.chatViewport.GotoBottom()
		return m, nil

	case streamDoneMsg:
		m.streaming = false
		m.input.Focus()

		// Finalize the assistant message.
		content := m.streamingContent
		if msg.content != "" {
			content = msg.content
		}
		m.streamingContent = ""
		m.messages = append(m.messages, chatMessage{role: "assistant", content: content})

		// Update session.
		if msg.sessionID != "" {
			m.session.ProviderSession = msg.sessionID
		}
		_ = coreide.Save(m.outputDir, m.session)

		// Check for document blocks.
		blocks := coreide.ParseDocumentBlocks(content)
		if len(blocks) > 0 {
			m.docBlocks = blocks
		}

		m.chatViewport.SetContent(renderChat(m.messages, "", m.width))
		m.chatViewport.GotoBottom()
		return m, nil

	case streamErrMsg:
		m.streaming = false
		m.streamingContent = ""
		m.input.Focus()
		m.messages = append(m.messages, chatMessage{role: "error", content: msg.err.Error()})
		m.chatViewport.SetContent(renderChat(m.messages, "", m.width))
		m.chatViewport.GotoBottom()
		return m, nil

	case writeCompleteMsg:
		m.saved = true
		m.session.Phase = coreide.PhaseComplete
		_ = coreide.Save(m.outputDir, m.session)
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m ideModel) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle preview toggle.
	if msg.Type == tea.KeyCtrlP && len(m.docBlocks) > 0 {
		m.preview = newPreview(m.docBlocks, m.width, m.height)
		m.view = viewPreview
		return m, nil
	}

	// Handle quit.
	if msg.Type == tea.KeyEsc {
		m.quitting = true
		m.cancel()
		return m, tea.Quit
	}

	// If streaming, ignore input except quit keys.
	if m.streaming {
		return m, nil
	}

	// Check if user pressed Enter to send.
	text, sent := inputUpdate(&m.input, msg)
	if sent {
		return m.sendMessage(text)
	}

	// Forward other keys to textarea.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ideModel) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlP:
		m.view = viewChat
		return m, nil

	case tea.KeyEnter:
		// Approve: trigger write.
		return m, m.writeDocuments()

	case tea.KeyTab:
		m.preview.nextFile()
		return m, nil
	}

	switch msg.String() {
	case "r":
		// Request changes: go back to chat.
		m.view = viewChat
		m.input.SetValue("Please change: ")
		m.input.Focus()
		return m, nil
	}

	// Forward scroll keys to preview viewport.
	var cmd tea.Cmd
	m.preview.viewport, cmd = m.preview.viewport.Update(msg)
	return m, cmd
}

func (m ideModel) sendMessage(text string) (tea.Model, tea.Cmd) {
	m.messages = append(m.messages, chatMessage{role: "user", content: text})
	m.streaming = true
	m.streamingContent = ""
	m.input.Blur()

	m.chatViewport.SetContent(renderChat(m.messages, "", m.width))
	m.chatViewport.GotoBottom()

	// Build the request.
	req := &llm.Request{
		Prompt:       text,
		SystemPrompt: m.prompt,
		SessionID:    m.session.ProviderSession,
		Tools:        []string{"Read", "Glob", "Grep"},
		WorkDir:      m.rootDir,
	}

	// Stream in a goroutine.
	return m, func() tea.Msg {
		ch, err := m.provider.Stream(m.ctx, req)
		if err != nil {
			return streamErrMsg{err: err}
		}

		var fullContent strings.Builder
		var sessionID string
		var usage *llm.Usage

		for evt := range ch {
			switch evt.Type {
			case llm.EventDelta:
				fullContent.WriteString(evt.Content)
				// We can't send tea.Msg from here directly during iteration,
				// so we accumulate. For real streaming we'd use p.Send().
				// This simplified version collects the full response.
			case llm.EventToolUse:
				// In the simplified flow, tool use is part of the stream.
			case llm.EventResult:
				sessionID = evt.SessionID
				usage = evt.Usage
				if evt.Content != "" {
					fullContent.Reset()
					fullContent.WriteString(evt.Content)
				}
			case llm.EventError:
				return streamErrMsg{err: fmt.Errorf("%s", evt.Content)}
			}
		}

		return streamDoneMsg{
			sessionID: sessionID,
			content:   fullContent.String(),
			usage:     usage,
		}
	}
}

func (m ideModel) writeDocuments() tea.Cmd {
	return func() tea.Msg {
		written, err := coreide.WriteAndSync(m.rootDir, m.docsDir, m.docBlocks)
		if err != nil {
			return streamErrMsg{err: err}
		}
		return writeCompleteMsg{files: written}
	}
}

func (m ideModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.view {
	case viewPreview:
		return m.preview.renderPreview()
	default:
		return m.viewChat()
	}
}

func (m ideModel) viewChat() string {
	var lines []string

	// Header (2 lines).
	header := renderHeader(
		m.provider.ID(),
		m.session.Category,
		m.session.ID,
		m.session.Target,
		m.session.Phase,
		m.width,
	)
	lines = append(lines, header)

	// Separator.
	lines = append(lines, ui.DimStyle.Render(strings.Repeat("─", m.width)))

	// Chat viewport.
	lines = append(lines, m.chatViewport.View())

	// Separator.
	lines = append(lines, ui.DimStyle.Render(strings.Repeat("─", m.width)))

	// Input area.
	lines = append(lines, m.input.View())

	// Help bar.
	help := "  " + ui.DimStyle.Render("enter send . ctrl+p preview . esc quit")
	if len(m.docBlocks) > 0 {
		help = "  " + ui.DimStyle.Render("enter send . ctrl+p preview . ctrl+w write . esc quit")
	}
	lines = append(lines, help)

	return strings.Join(lines, "\n")
}

func (m *ideModel) resize() {
	headerHeight := 3 // Header + separator
	inputHeight := 5  // Input + separator + help bar
	chatHeight := m.height - headerHeight - inputHeight

	if chatHeight < 3 {
		chatHeight = 3
	}

	m.chatViewport.Width = m.width
	m.chatViewport.Height = chatHeight
	m.input.SetWidth(m.width)

	if m.view == viewPreview {
		m.preview.resize(m.width, m.height)
	}
}

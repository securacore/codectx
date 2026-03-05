# IDE — Implementation Plan

`codectx ide` is an AI documentation authoring pipeline. It provides a conversational TUI where an AI co-author classifies, scopes, drafts, and produces standalone, compilation-optimized documentation for use in codectx packages. The AI has read-only access to existing documentation so it can avoid duplication, maintain cross-reference integrity, and produce content that compiles and deduplicates cleanly.

This is not a general-purpose AI chat tool. Every design decision serves one goal: produce excellent documentation that compiles into content-addressed packages with minimal conflicts and maximum reuse.

---

## Goals

1. Build an LLM provider layer (`core/llm/`) that wraps the Claude CLI binary using its `--output-format stream-json` protocol, with fallbacks to the Anthropic API SDK and Ollama HTTP.
2. Build a session management system (`core/ide/`) that persists conversation state transparently, supports resuming, and auto-cleans completed sessions.
3. Build an embedded documentation directive that instructs the AI to classify, scope, draft, and review documentation against codectx conventions.
4. Build a full-screen bubbletea TUI (`cmds/ide/`) with chat viewport, streaming responses, phase tracking, document preview, and file writing with manifest sync.
5. Enhance `codectx new` to auto-compile and offer `codectx link` after project creation.
6. Ensure the AI has read-only access (`Read`, `Glob`, `Grep`) to existing documentation for context awareness and cross-reference alignment.

## Non-Goals

- General-purpose AI chat or code editing. The AI authors documentation only.
- Replacing `codectx new` for simple scaffolding. `codectx ide` is for AI-assisted authoring; `codectx new` remains for non-interactive scaffolding.
- Supporting every LLM provider at launch. Claude CLI is the primary path; API and Ollama fallbacks are secondary.
- Non-TTY operation. The TUI requires an interactive terminal. Non-TTY users use `codectx new` and their own AI tools.
- Writing files without user approval. All document output goes through a preview step before writing.

---

## Design Decisions

Every decision made during planning is recorded here with rationale.

### D1: Claude CLI as Primary Provider

The Claude Agent SDK (formerly Claude Code SDK) exists in Python and TypeScript but not Go. Both SDKs work by wrapping the `claude` CLI binary and parsing its JSON output. We do the same: spawn `claude -p --output-format stream-json` and parse NDJSON events. This leverages the user's existing Claude Code authentication (OAuth or API key) without requiring separate API key management.

The `claude` CLI supports `--session-id` for native conversation persistence, `--system-prompt` for directive injection, and `--tools` for controlling tool access. These map directly to our requirements.

### D2: Read-Only Tool Access

The AI receives `--tools "Read,Glob,Grep"` when invoked via the Claude CLI. This gives it read-only access to existing documentation so it can:

- Read the manifest to understand the documentation landscape
- Examine existing docs to avoid content duplication (which triggers dedup conflicts during compilation)
- Verify cross-references resolve to real documents
- Align terminology and style with existing material

The AI cannot write, edit, or execute commands. All file output is controlled by the TUI's preview-then-write flow.

### D3: Session Management is Provider-Dependent

For the Claude CLI provider: we store only session metadata (category, phase, target path) in `.codectx/sessions/`. Conversation state is managed by Claude's native `--session-id` mechanism. This avoids duplicating potentially large conversation histories.

For API and Ollama providers: we store the full message history in the session file because these providers are stateless. The message history is replayed on resume.

### D4: Embedded Documentation Directive

The system prompt that drives the AI is embedded in the Go binary via `//go:embed`, following the same pattern as `core/defaults/` for foundation documents. This ensures the directive is always available without depending on compiled output existing. The directive encodes:

- The documentation category taxonomy (foundation, topic, prompt, application)
- The conversation phase protocol (discover, classify, scope, draft, review, finalize)
- Key rules from the documentation, markdown, and ai-authoring foundation documents
- The output protocol for document preview (`<document>` tags)
- Compilation awareness (cross-reference format, dedup implications, standalone requirements)

### D5: Phase-Driven Conversation

The conversation follows six phases. The AI drives progression; the user does not need to know the phases exist. The TUI header displays the current phase as a status indicator.

| Phase | AI Behavior |
|-------|-------------|
| Discover | Ask focused questions to understand the subject, audience, and purpose. One question at a time. |
| Classify | Recommend a documentation category with reasoning. Suggest kebab-case ID and target path. Confirm before proceeding. |
| Scope | Define boundaries using existing documentation as reference. State what the document covers and what it defers. Identify `depends_on` relationships. |
| Draft | Author section by section following codectx conventions. Present each section for feedback. Use Read/Glob to check existing docs for consistency. |
| Review | Validate the complete document against the quality checklist: standalone, one-topic, timeless, AI-first, proper cross-references. |
| Finalize | Present the complete document in `<document>` tag format for TUI preview. |

### D6: Preview-Then-Write Output

The AI never writes files directly. When it produces a document, it wraps the content in `<document path="...">` tags. The TUI parses these tags and presents a rendered preview. The user approves, requests changes, or rejects. On approval, the TUI writes the files, syncs the manifest, and optionally triggers compilation.

### D7: Dynamic Context Injection

The system prompt is assembled at session start by combining:

1. The embedded directive (static, always included)
2. A manifest summary generated from `docs/manifest.yml` (dynamic, shows all existing entry IDs, categories, descriptions, and relationships)
3. Compilation preferences (compression enabled, model class target)

This context is injected via `--system-prompt` for the Claude CLI provider, or as a system message for API/Ollama providers. The manifest summary keeps the AI aware of the documentation landscape without requiring it to read the manifest file itself.

### D8: Init Flow Enhancement

After project initialization, we add two steps: auto-compile (using the existing inline spinner pattern) and an optional link prompt. Since default foundation documents always exist after init, there is always content to compile. This gives the user a complete setup in one command.

### D9: Session Auto-Naming

Sessions start with a UUID identifier. Once the AI classifies the document (phase: classify), the session ID is updated to a human-readable kebab-case name derived from the document title (e.g., `go-error-handling`). If a session with that name already exists, a numeric suffix is appended.

### D10: codectx-Specific LLM Wrapper

The Go Claude CLI wrapper in `core/llm/` is tightly coupled to codectx's needs. It is not designed as an extractable standalone module. This keeps the abstraction minimal and avoids over-engineering interfaces we do not need.

---

## File Map

### New files to create

```
core/llm/
├── provider.go            # Provider interface, Request, Event types
├── message.go             # Message type (role + content)
├── event.go               # Event types (delta, result, tool_use, error)
├── claude.go              # Claude CLI wrapper (stream-json protocol)
├── anthropic.go           # Anthropic API SDK fallback
├── ollama.go              # Ollama HTTP streaming (/api/chat)
├── resolve.go             # Provider detection and selection logic
├── provider_test.go       # Provider interface tests
├── claude_test.go         # Claude CLI wrapper tests (with mock binary)
├── resolve_test.go        # Provider resolution tests

core/ide/
├── session.go             # Session struct, persistence, lifecycle management
├── directive.go           # Embedded system prompt via //go:embed
├── phase.go               # Phase enum, phase state machine
├── context.go             # Dynamic context assembly (manifest summary, prefs)
├── writer.go              # Document tag parser, file writer, manifest sync
├── session_test.go        # Session persistence and lifecycle tests
├── directive_test.go      # Directive assembly tests
├── phase_test.go          # Phase transition tests
├── context_test.go        # Context assembly tests
├── writer_test.go         # Document parser and writer tests
├── content/
│   └── directive.md       # The embedded documentation directive

cmds/ide/
├── main.go                # Command registration, session picker, provider init
├── interactive.go         # Root bubbletea model (state machine, message routing)
├── chat.go                # Chat viewport component (message history, markdown)
├── input.go               # Input area component (textarea, send semantics)
├── header.go              # Header bar component (category, phase, provider)
├── preview.go             # Document preview overlay (approve/reject flow)
├── interactive_test.go    # TUI model tests (Update/View cycle)
```

### Modified files

```
main.go                    # Register cmds/ide command
go.mod                     # Add glamour, anthropic-sdk-go, google/uuid
go.sum                     # Updated by go mod tidy
cmds/init/main.go          # Add auto-compile and link prompt steps
core/ai/claude.go          # Remove Phase 2 stub comment (superseded by core/llm)
core/ai/opencode.go        # Remove Phase 2 stub comment (superseded by core/llm)
ui/spinner.go              # Fix MiniDot spacing (space between icon and text)
```

### Reference files (read, not modified)

```
docs/foundation/documentation/README.md    # Documentation conventions (embedded in directive)
docs/foundation/ai-authoring/README.md     # AI authoring conventions (embedded in directive)
docs/foundation/markdown/README.md         # Markdown conventions (embedded in directive)
docs/foundation/specs/README.md            # Spec template (embedded in directive)
docs/product/ai-integration.md             # AI integration design
docs/product/compilation.md                # Compilation pipeline reference
```

---

## Type Definitions

### LLM Provider Layer (`core/llm/`)

```go
package llm

// Provider streams AI responses for documentation authoring.
type Provider interface {
    // Stream sends a request and returns a channel of streaming events.
    // The channel is closed when the response is complete or an error occurs.
    Stream(ctx context.Context, req *Request) (<-chan Event, error)

    // ID returns the provider identifier (e.g., "claude", "anthropic", "ollama").
    ID() string
}

// Request represents a single exchange with the AI.
type Request struct {
    Prompt       string   // User message for this turn
    SystemPrompt string   // Full system prompt (directive + context)
    SessionID    string   // Provider session ID (empty = new session)
    Model        string   // Model override (empty = provider default)
    Tools        []string // Allowed tools (e.g., ["Read", "Glob", "Grep"])
    WorkDir      string   // Working directory for tool access
}

// Event represents a single streaming event from the provider.
type Event struct {
    Type      EventType // Delta, ToolUse, Result, Error
    Content   string    // Text delta, tool name, result text, or error message
    SessionID string    // Populated on Result events (Claude CLI)
    Usage     *Usage    // Populated on Result events
}

type EventType int

const (
    EventDelta   EventType = iota // Incremental text content
    EventToolUse                  // AI is using a tool (Read, Glob, Grep)
    EventResult                   // Final result with session ID and usage
    EventError                    // Error from the provider
)

// Usage tracks token consumption for a single exchange.
type Usage struct {
    InputTokens  int
    OutputTokens int
}

// Message represents a single message in conversation history.
type Message struct {
    Role    Role   // System, User, Assistant
    Content string
}

type Role int

const (
    RoleSystem    Role = iota
    RoleUser
    RoleAssistant
)
```

### Session Management (`core/ide/`)

```go
package ide

// Session represents a documentation authoring session.
type Session struct {
    ID              string    `yaml:"id"`
    Provider        string    `yaml:"provider"`
    ProviderSession string    `yaml:"provider_session,omitempty"` // Claude CLI session UUID
    Title           string    `yaml:"title"`
    Category        string    `yaml:"category,omitempty"`  // foundation/topic/prompt/application
    Target          string    `yaml:"target,omitempty"`    // e.g., docs/topics/go-error-handling/
    Phase           Phase     `yaml:"phase"`
    Created         time.Time `yaml:"created"`
    Updated         time.Time `yaml:"updated"`
    Messages        []Message `yaml:"messages,omitempty"`  // Full history for stateless providers
    Document        string    `yaml:"document,omitempty"`  // Latest document draft content
}

type Phase int

const (
    PhaseDiscover  Phase = iota
    PhaseClassify
    PhaseScope
    PhaseDraft
    PhaseReview
    PhaseFinalize
    PhaseComplete
)

// DocumentBlock represents a parsed <document> tag from AI output.
type DocumentBlock struct {
    Path    string // Relative path (e.g., docs/topics/example/README.md)
    Content string // Document content
}
```

---

## Claude CLI Protocol

The primary provider communicates with the `claude` binary via its `--output-format stream-json` protocol.

### Invocation

```bash
claude -p \
  --output-format stream-json \
  --system-prompt "<assembled directive + context>" \
  --session-id "<uuid>" \
  --tools "Read,Glob,Grep" \
  --model "<model>" \
  "<user message>"
```

Flags:

| Flag | Value | Purpose |
|------|-------|---------|
| `-p` | (presence) | Non-interactive print mode |
| `--output-format` | `stream-json` | NDJSON streaming events on stdout |
| `--system-prompt` | Full directive string | Documentation authoring directive with dynamic context |
| `--session-id` | UUID string | Conversation persistence (omit for new sessions) |
| `--tools` | `"Read,Glob,Grep"` | Read-only tool access to project files |
| `--model` | Model ID or alias | Optional model override from preferences |

### Event Types

Each line of stdout is a JSON object. Key event types:

| Type | Fields | Meaning |
|------|--------|---------|
| `assistant` | `message.content[].text` | Text content block from the assistant |
| `content_block_delta` | `delta.text` | Incremental text delta (streaming) |
| `result` | `result`, `session_id`, `usage` | Final result with session ID and token counts |

The wrapper reads lines from stdout, parses each as JSON, maps to `Event` types, and sends on the event channel. When the process exits, the channel is closed.

### Error Handling

| Condition | Detection | Response |
|-----------|-----------|----------|
| Auth expired | `result.is_error` + "authentication_error" | Surface "Claude auth expired. Run: claude /login" |
| Binary not found | `exec.LookPath` fails | Fall through to next provider |
| Process crash | Non-zero exit + no result event | Return error event with stderr content |
| Rate limited | `result.is_error` + "rate_limit" | Surface rate limit message, suggest waiting |

---

## Session Lifecycle

### Storage

Sessions are stored as individual YAML files in `.codectx/sessions/`. This directory is inside `.codectx/` which is already gitignored. One file per session.

```
.codectx/sessions/
├── go-error-handling.yml
├── react-hooks-guide.yml
└── 8f2a3b1c.yml            # Not yet classified (UUID prefix)
```

### State Transitions

```
[new] --> Discover --> Classify --> Scope --> Draft --> Review --> Finalize --> [complete]
                                                  ^                  |
                                                  |                  |
                                                  +--- (revisions) --+
```

A session can move backward from Review to Draft if the user requests changes. Finalize can return to Draft through the preview rejection flow. All other transitions are forward-only.

### Auto-Save Triggers

- After every AI response: session metadata updated (phase, title, category, target)
- After classification: session file renamed from UUID to human-readable ID
- After preview approval: phase set to Complete, document content stored
- On user exit (Ctrl+Q, Esc): session preserved at current phase

### Cleanup

On startup, `codectx ide` prunes sessions in `PhaseComplete` that are older than 30 days. Active sessions are never pruned. The user can also resume completed sessions to create follow-up documents.

### Session Picker

When `codectx ide` is invoked with active sessions, a `huh` select form is shown before entering the TUI:

```
? Continue a session or start new?
  > Go Error Handling  [topic]  phase:draft   2 hours ago
    React Hooks Guide  [topic]  phase:scope   yesterday
    Start new conversation
```

Direct resume: `codectx ide --resume <id>`. List all: `codectx ide --list`.

---

## Documentation Directive

The directive is embedded in `core/ide/content/directive.md` via `//go:embed` and assembled with dynamic context at session start.

### Directive Structure

The directive has five sections:

1. **Identity and purpose**: States the AI's role as a documentation authoring assistant for codectx packages.
2. **Phase protocol**: Defines the six-phase conversation flow with explicit behaviors per phase.
3. **Documentation standards**: Key rules from the documentation, markdown, and ai-authoring foundation documents. Condensed for token efficiency.
4. **Compilation awareness**: Rules about cross-references, deduplication, standalone requirements, and content-addressing implications.
5. **Output protocol**: The `<document>` tag format for presenting drafts to the TUI preview system.

### Dynamic Context Assembly (`core/ide/context.go`)

At session start, `AssembleContext()` builds the full system prompt by combining:

1. The embedded directive (static)
2. A manifest summary generated from `docs/manifest.yml`:

```
## Existing Documentation

Foundation:
- philosophy (load:always): Guiding principles for decision-making
- markdown (load:documentation): Markdown formatting conventions
- documentation (load:documentation): Documentation management and strategy
  depends_on: philosophy, markdown

Topics:
- go: Go CLI conventions and patterns
  depends_on: philosophy, specs
- react: React component conventions and patterns
  depends_on: typescript, philosophy, specs

Prompts:
- save: Session state persistence
- docs-audit: Audit documentation against review standards
```

3. Compilation preferences from `.codectx/preferences.yml` (compression setting, model class)

This assembled string is passed as `--system-prompt` to the Claude CLI, or as a system message to API/Ollama providers.

### Output Protocol

The directive instructs the AI to wrap finalized document content in `<document>` tags:

```
<document path="docs/topics/go-error-handling/README.md">
# Go Error Handling

Document content here...
</document>
```

If the category requires a spec:

```
<document path="docs/topics/go-error-handling/spec/README.md">
# Go Error Handling Spec

Spec content here...
</document>
```

The TUI's preview component parses these tags and presents the content for approval.

---

## TUI Architecture (`cmds/ide/`)

### Layout

```
+--------------------------------------------------------------+
| codectx ide  provider:claude     topic . go-error-handling    | <- header.go
|              phase:draft            docs/topics/go-err.../    |
+--------------------------------------------------------------+
|                                                               |
|  * Based on your description, this should be a **topic**      | <- chat.go
|    document. Topics define technology-specific conventions.    |    (viewport)
|                                                               |
|    Suggested structure:                                       |
|    - Path: docs/topics/go-error-handling/                     |
|    - ID: go-error-handling                                    |
|    - Depends on: go (existing Go conventions)                 |
|                                                               |
|  > Yes, that looks right.                                     | <- user message
|                                                               |
|  * Good. Moving to scoping. Let me read the existing go       |
|    topic to understand what's already covered...              |
|    [Reading docs/topics/go/README.md...]                      | <- tool use
|                                                               |
+--------------------------------------------------------------+
| > _                                                           | <- input.go
+--------------------------------------------------------------+
| enter send . ctrl+p preview . ctrl+w write . esc quit         | <- help bar
+--------------------------------------------------------------+
```

### Component Responsibilities

| Component | File | Responsibility |
|-----------|------|----------------|
| Root model | `interactive.go` | State machine, message routing between components, alt-screen program, stream goroutine management |
| Header | `header.go` | Provider name, document category, phase indicator, document ID, target path. Updates reactively from session state. |
| Chat viewport | `chat.go` | Scrollable message history. Renders AI messages with `glamour` (terminal markdown). User messages rendered with accent style. Tool use shown as dim status lines. Streaming deltas appended in real-time. |
| Input area | `input.go` | `bubbles/textarea` component. Enter to send, Shift+Enter for newline. Disabled during AI response streaming. |
| Preview | `preview.go` | Full-screen overlay triggered by ctrl+p or when AI outputs `<document>` tags. Shows rendered markdown. Tab cycles between files. Enter approves. `r` requests changes. Esc returns to chat. |

### Bubbletea Message Types

```go
// streamDeltaMsg carries an incremental text chunk from the AI.
type streamDeltaMsg struct{ content string }

// streamToolMsg indicates the AI is using a tool (Read, Glob, Grep).
type streamToolMsg struct{ tool string }

// streamDoneMsg signals the AI response is complete.
type streamDoneMsg struct {
    sessionID string
    usage     *llm.Usage
}

// streamErrMsg signals an error from the provider.
type streamErrMsg struct{ err error }

// documentMsg carries parsed <document> blocks from the AI response.
type documentMsg struct{ blocks []ide.DocumentBlock }

// writeCompleteMsg signals that files were written and manifest synced.
type writeCompleteMsg struct {
    files    []string
    compiled bool
}
```

### Streaming Integration

1. User presses Enter. Input text is captured and the textarea is disabled.
2. A goroutine calls `provider.Stream(ctx, req)` and reads from the event channel.
3. Each event is sent to the bubbletea program via `p.Send()`:
   - `EventDelta` -> `streamDeltaMsg` (chat viewport appends text)
   - `EventToolUse` -> `streamToolMsg` (chat viewport shows tool status)
   - `EventResult` -> `streamDoneMsg` (session saved, textarea re-enabled)
   - `EventError` -> `streamErrMsg` (error displayed, textarea re-enabled)
4. After the response is complete, the full response text is scanned for `<document>` tags. If found, a `documentMsg` is sent to trigger the preview overlay.

### Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `Enter` | Input area | Send message (or newline if Shift held) |
| `Ctrl+P` | Any | Toggle preview overlay (if document blocks exist) |
| `Ctrl+W` | Preview | Write files (approve) |
| `Esc` | Preview | Return to chat |
| `Esc` | Chat | Quit (save session) |
| `Ctrl+C` | Any | Quit immediately (save session) |
| `Tab` | Preview | Cycle between document files |
| `r` | Preview | Return to chat with "please change..." prompt |
| `Up/Down` | Chat | Scroll viewport |
| `PgUp/PgDn` | Chat | Page scroll viewport |

### Preview-Then-Write Flow

1. AI outputs content with `<document>` tags during the Finalize phase.
2. TUI parses `<document path="...">content</document>` blocks.
3. Preview overlay opens showing rendered markdown.
4. User reviews and either:
   - **Approves** (Enter/Ctrl+W): triggers write flow
   - **Requests changes** (`r`): returns to chat, prefills input with change request
   - **Dismisses** (Esc): returns to chat, no action
5. Write flow (`core/ide/writer.go`):
   a. Creates target directory (e.g., `docs/topics/go-error-handling/`)
   b. Writes all document files (README.md, spec/README.md if present)
   c. Loads `docs/manifest.yml` and runs `manifest.Sync()` to discover the new entry
   d. Writes the updated manifest
   e. If `auto_compile` preference is true, runs `compile.Compile(cfg)`
   f. Sets session phase to Complete
6. TUI exits, prints summary:

```
Created topic: go-error-handling
  docs/topics/go-error-handling/README.md       (2.3 KB)
  docs/topics/go-error-handling/spec/README.md  (890 B)
  Manifest synced. Dependencies: go
  Compiled: 1 new object
```

---

## Init Flow Enhancement

After the current init flow (steps 1-13 in `cmds/init/main.go`), add two steps.

### Step 14: Auto-Compile

```
Compiling documentation...
Compiled: 5 entries, 5 objects (12.4 KB)
```

Uses the existing inline spinner pattern from `cmds/compile/main.go`. Since default foundation documents always exist after init, there is always content to compile. The compile result is printed in the same format as `codectx compile`.

### Step 15: Link Prompt

```
? Set up AI tool integration?
  This creates entry point files so your AI tools automatically
  load project documentation on every session.
  > Yes (recommended)
    No, I'll run 'codectx link' later
```

If the user selects Yes, the existing `core/link/Link()` function is called with the standard tool multi-select prompt from `cmds/link/`. If No, a hint is printed: "Run 'codectx link' when you're ready."

Both steps are skipped in non-TTY mode.

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/charmbracelet/glamour` | latest | Terminal markdown rendering for chat viewport and preview |
| `github.com/anthropics/anthropic-sdk-go` | latest | Anthropic API fallback provider |
| `github.com/google/uuid` | latest | Session ID generation |
| `github.com/charmbracelet/bubbles` | (already in project) | viewport, textarea, spinner components |
| `github.com/charmbracelet/bubbletea` | (already in project) | TUI framework |
| `github.com/charmbracelet/lipgloss` | (already in project) | Terminal styling |
| `github.com/charmbracelet/huh` | (already in project) | Session picker form |
| `github.com/urfave/cli/v3` | (already in project) | CLI command registration |
| `gopkg.in/yaml.v3` | (already in project) | Session YAML persistence |

Only `glamour`, `anthropic-sdk-go`, and `google/uuid` are new dependencies.

---

## Implementation Phases

### Phase 1: LLM Provider Layer

**Goal:** Build the provider interface and Claude CLI wrapper with streaming support.

**Files to create:**

| File | Purpose |
|------|---------|
| `core/llm/provider.go` | Provider interface, Request, Event, EventType, Usage, Message, Role |
| `core/llm/message.go` | Message constructors and helpers |
| `core/llm/event.go` | Event constructors, EventType string representation |
| `core/llm/claude.go` | Claude CLI wrapper: spawn process, parse stream-json NDJSON, emit events |
| `core/llm/resolve.go` | Provider detection: claude binary -> ANTHROPIC_API_KEY -> ollama -> error |
| `core/llm/claude_test.go` | Tests with mock binary or recorded output |
| `core/llm/resolve_test.go` | Provider resolution tests |

**Implementation notes:**

- The Claude CLI wrapper starts a `claude` subprocess per exchange using `os/exec.Command`. It reads stdout line by line using `bufio.Scanner`, parses each line as JSON, maps to Event types, and sends on a channel.
- The wrapper captures stderr for error reporting. If the process exits non-zero with no result event, it returns the stderr content as an error.
- Session ID management: for the first exchange, omit `--session-id`. The result event contains `session_id` which we capture and store. Subsequent exchanges include `--session-id <captured-id>`.
- The `resolve.go` provider detection reuses the existing `core/ai.Detect()` logic to check for the `claude` binary.

**Phase 1 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestClaude_parsesStreamJSON` | A recorded NDJSON stream is parsed into correct Event sequence |
| `TestClaude_capturesSessionID` | Session ID is extracted from result event |
| `TestClaude_handlesAuthError` | Auth error in result is surfaced as EventError with clear message |
| `TestClaude_handlesProcessCrash` | Non-zero exit returns error event with stderr |
| `TestClaude_contextCancellation` | Cancelled context kills the subprocess |
| `TestResolve_claudePreferred` | When claude binary exists, Claude provider is returned |
| `TestResolve_fallbackToAPI` | When no claude binary but ANTHROPIC_API_KEY is set, API provider returned |
| `TestResolve_fallbackToOllama` | When no claude or API key but ollama running, Ollama provider returned |
| `TestResolve_noProvider` | When nothing available, returns descriptive error |
| `TestEventType_String` | String representation of each event type |

**Verification:**

```bash
go test -v ./core/llm/
```

### Phase 2: Session Management

**Goal:** Implement transparent session persistence with auto-save, auto-naming, and cleanup.

**Files to create:**

| File | Purpose |
|------|---------|
| `core/ide/session.go` | Session struct, Load, Save, List, Rename, Cleanup functions |
| `core/ide/phase.go` | Phase enum, String(), ParsePhase(), transition validation |
| `core/ide/session_test.go` | Session lifecycle tests |
| `core/ide/phase_test.go` | Phase transition and serialization tests |

**Implementation notes:**

- `SessionDir()` returns `.codectx/sessions/` using the config's OutputDir.
- `Save(s *Session)` writes to `<SessionDir>/<id>.yml` using yaml.Marshal. Updates the `Updated` timestamp.
- `Load(id string)` reads the YAML file and returns a Session.
- `List()` reads all YAML files in the sessions directory, returns sorted by Updated descending.
- `Active()` filters List() to exclude PhaseComplete sessions.
- `Rename(oldID, newID string)` renames the session file and updates the ID field. Handles collision by appending `-2`, `-3`, etc.
- `Cleanup(maxAge time.Duration)` removes completed sessions older than maxAge.
- Phase transitions are validated: only forward transitions are allowed except Review->Draft (revisions) and Finalize->Draft (preview rejection).

**Phase 2 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestSession_saveAndLoad` | Round-trip save then load preserves all fields |
| `TestSession_list` | Multiple sessions listed in updated-descending order |
| `TestSession_active` | Complete sessions excluded from active list |
| `TestSession_rename` | Session file renamed, ID field updated |
| `TestSession_renameCollision` | Collision appends numeric suffix |
| `TestSession_cleanup` | Old completed sessions removed, active sessions preserved |
| `TestSession_cleanupPreservesRecent` | Completed sessions within maxAge are kept |
| `TestPhase_string` | Each phase has correct string representation |
| `TestPhase_parse` | String-to-phase parsing round-trips |
| `TestPhase_marshalYAML` | Phase serializes as string in YAML |
| `TestPhase_transitionValid` | Forward transitions and Review->Draft allowed |
| `TestPhase_transitionInvalid` | Backward transitions rejected (except allowed ones) |

**Verification:**

```bash
go test -v ./core/ide/
```

### Phase 3: Directive and Context Assembly

**Goal:** Build the embedded documentation directive and dynamic context assembly system.

**Files to create:**

| File | Purpose |
|------|---------|
| `core/ide/directive.go` | Embed directive.md, AssemblePrompt() combining directive + context |
| `core/ide/context.go` | BuildManifestSummary(), BuildPreferencesContext() |
| `core/ide/content/directive.md` | The documentation authoring directive |
| `core/ide/writer.go` | ParseDocumentBlocks(), WriteDocuments(), manifest sync |
| `core/ide/directive_test.go` | Directive assembly tests |
| `core/ide/context_test.go` | Manifest summary generation tests |
| `core/ide/writer_test.go` | Document block parsing and file writing tests |

**Implementation notes:**

- `directive.go` uses `//go:embed content/directive.md` to embed the directive.
- `AssemblePrompt(manifestPath, prefsPath string)` loads the manifest, builds the summary, loads preferences, and concatenates: directive + manifest summary + preferences context.
- `BuildManifestSummary(m *manifest.Manifest)` formats each section's entries as a readable list with IDs, descriptions, load values, and dependencies.
- `ParseDocumentBlocks(text string)` uses a simple state machine to extract `<document path="...">...</document>` blocks from AI response text.
- `WriteDocuments(blocks []DocumentBlock, docsDir string)` creates directories, writes files, and returns the list of written paths.
- After writing, the caller runs `manifest.Sync()` to discover the new entries.

**Phase 3 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestAssemblePrompt_includesDirective` | Output contains the embedded directive content |
| `TestAssemblePrompt_includesManifest` | Output contains manifest summary section |
| `TestAssemblePrompt_includesPreferences` | Output contains preferences context |
| `TestBuildManifestSummary_allSections` | Foundation, topics, prompts all represented |
| `TestBuildManifestSummary_showsDependencies` | depends_on relationships shown |
| `TestBuildManifestSummary_showsLoadValues` | Foundation load:always/documentation shown |
| `TestBuildManifestSummary_emptyManifest` | Empty manifest produces "No existing documentation" |
| `TestParseDocumentBlocks_single` | Single `<document>` block parsed correctly |
| `TestParseDocumentBlocks_multiple` | Multiple blocks parsed with correct paths |
| `TestParseDocumentBlocks_none` | Text without blocks returns empty slice |
| `TestParseDocumentBlocks_preservesContent` | Content including markdown is preserved exactly |
| `TestWriteDocuments_createsDirectory` | Target directory created if absent |
| `TestWriteDocuments_writesFiles` | Files written with correct content |
| `TestWriteDocuments_existingDirectory` | Writing to existing directory does not error |

**Verification:**

```bash
go test -v ./core/ide/
```

### Phase 4: Chat TUI

**Goal:** Build the full-screen bubbletea TUI with chat viewport, streaming responses, input area, and header bar.

**Files to create:**

| File | Purpose |
|------|---------|
| `cmds/ide/main.go` | Command registration, session picker, provider init, TTY guard |
| `cmds/ide/interactive.go` | Root bubbletea model, Init/Update/View, state machine |
| `cmds/ide/chat.go` | Chat viewport: message history, glamour rendering, streaming append |
| `cmds/ide/input.go` | Input area: textarea wrapper, send/newline semantics, enable/disable |
| `cmds/ide/header.go` | Header bar: provider, category, phase, target path |
| `cmds/ide/interactive_test.go` | Model tests: Update cycle, message handling, state transitions |

**Implementation notes:**

- The root model uses `tea.WithAltScreen()` for full-screen mode, consistent with activate and search.
- The chat viewport uses `bubbles/viewport` with a custom content renderer that formats messages with `glamour` for AI messages and `ui.AccentStyle` for user messages.
- The input area uses `bubbles/textarea` with `SetHeight(3)` for multi-line support.
- Streaming: when the user sends a message, the model transitions to a "receiving" state where the input is disabled and the viewport shows a streaming cursor. Each `streamDeltaMsg` appends to the current AI message.
- The header bar is rendered as two lines at the top, using the same `ui.DimStyle` and `ui.BoldStyle` patterns as existing TUIs.
- The help bar at the bottom shows context-sensitive key bindings (different in chat vs. preview mode).
- Window resize handling updates viewport dimensions, consistent with activate and search patterns.

**Phase 4 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestModel_initialState` | Model starts in chat state with empty viewport |
| `TestModel_sendMessage` | Enter triggers stream, disables input, adds user message |
| `TestModel_streamDelta` | Delta messages append to current AI response |
| `TestModel_streamDone` | Done message re-enables input, saves session |
| `TestModel_streamError` | Error message displayed, input re-enabled |
| `TestModel_quit` | Esc sets quitting, returns tea.Quit |
| `TestModel_ctrlC` | Ctrl+C sets quitting, returns tea.Quit |
| `TestModel_windowResize` | Viewport and input dimensions updated |
| `TestHeader_render` | Header shows provider, category, phase, path |
| `TestHeader_phaseUpdate` | Phase indicator updates when session phase changes |
| `TestChat_userMessage` | User message rendered with accent style |
| `TestChat_aiMessage` | AI message rendered with glamour markdown |
| `TestChat_toolUse` | Tool use shown as dim status line |
| `TestChat_scrolling` | Viewport scrolls to bottom on new content |
| `TestInput_sendOnEnter` | Enter with content triggers send |
| `TestInput_emptyBlocked` | Enter with empty content does nothing |
| `TestInput_disabledDuringStream` | Input rejects keystrokes while streaming |

**Verification:**

```bash
go test -v ./cmds/ide/
```

### Phase 5: Preview and Write Flow

**Goal:** Add the document preview overlay and file writing with manifest sync.

**Files to modify/create:**

| File | Action | Purpose |
|------|--------|---------|
| `cmds/ide/preview.go` | Create | Preview overlay component |
| `cmds/ide/interactive.go` | Modify | Add preview state, document message handling, write flow |
| `cmds/ide/interactive_test.go` | Modify | Add preview and write tests |

**Implementation notes:**

- The preview overlay is a full-screen view within the existing alt-screen. It replaces the chat/input area with a rendered document viewport and a context-sensitive help bar.
- When the AI response contains `<document>` blocks, the model automatically parses them and triggers the preview overlay.
- The preview viewport uses `bubbles/viewport` with `glamour`-rendered content.
- Tab cycles between multiple document files (README.md, spec/README.md).
- Enter or Ctrl+W triggers the write flow, which runs in a goroutine and sends a `writeCompleteMsg` on completion.
- After write completion, the TUI exits and prints the summary using `ui.Done()`, `ui.Item()`, and `ui.KV()`.

**Phase 5 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestPreview_triggeredByDocumentBlocks` | Document blocks in AI response open preview |
| `TestPreview_rendersMarkdown` | Preview shows glamour-rendered content |
| `TestPreview_tabCyclesFiles` | Tab switches between README.md and spec/README.md |
| `TestPreview_approveTriggersWrite` | Enter sends write command |
| `TestPreview_requestChanges` | `r` returns to chat with prefilled input |
| `TestPreview_escReturnsToChat` | Esc closes preview without action |
| `TestWrite_createsFiles` | Files written to correct paths |
| `TestWrite_syncsManifest` | manifest.Sync() called after writing |
| `TestWrite_autoCompiles` | Compilation triggered when preference is true |
| `TestWrite_skipsCompile` | Compilation skipped when preference is false |
| `TestWrite_completesSession` | Session phase set to Complete |

**Verification:**

```bash
go test -v ./cmds/ide/ ./core/ide/
```

### Phase 6: Init Enhancement and Command Registration

**Goal:** Wire everything together. Enhance init, register the ide command, add provider fallbacks.

**Files to modify/create:**

| File | Action | Purpose |
|------|--------|---------|
| `main.go` | Modify | Register `cmds/ide.Command` |
| `cmds/init/main.go` | Modify | Add auto-compile and link prompt after init |
| `core/llm/anthropic.go` | Create | Anthropic API fallback provider |
| `core/llm/ollama.go` | Create | Ollama HTTP streaming provider |
| `core/ai/claude.go` | Modify | Remove Phase 2 stub comment |
| `core/ai/opencode.go` | Modify | Remove Phase 2 stub comment |

**Implementation notes for init enhancement:**

- After the current summary output (step 13), call `compile.Compile(cfg)` with the inline spinner pattern.
- Then present a `huh.NewConfirm()` asking about AI tool integration.
- If confirmed, call the existing link flow from `core/link/Link()` with tool selection.
- Both steps are gated on `ui.IsTTY()`.

**Implementation notes for provider fallbacks:**

- `anthropic.go` uses `anthropic-sdk-go` with `client.Messages.NewStreaming()`. Manages conversation history as a `[]Message` slice replayed on each call.
- `ollama.go` uses `POST /api/chat` with `"stream": true`. Manages conversation history the same way. Extends the existing `core/ai/ollama.go` HTTP infrastructure.

**Phase 6 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestCommand_registered` | `ide` command appears in app.Commands |
| `TestCommand_requiresTTY` | Non-TTY returns descriptive error |
| `TestCommand_resumeFlag` | `--resume` flag passes ID to session loader |
| `TestCommand_listFlag` | `--list` flag prints session list and exits |
| `TestInit_compilesAfterSetup` | Init runs compile after creating files |
| `TestInit_promptsForLink` | Init shows link prompt in TTY mode |
| `TestInit_skipsLinkInNonTTY` | Link prompt skipped in non-TTY |
| `TestAnthropic_streamsMessages` | API provider streams delta events |
| `TestAnthropic_resendsHistory` | Full message history sent on each call |
| `TestOllama_streamsChat` | Ollama provider streams chat responses |

**Verification:**

```bash
go test -v ./core/llm/ ./cmds/ide/ ./cmds/init/
just build
```

---

## Success Criteria

### Must-Have (All Phases)

1. `codectx ide` launches a full-screen TUI with chat viewport, input area, and header bar.
2. User can converse with the AI through the Claude CLI provider with streaming responses.
3. The AI classifies documentation into the correct category (foundation, topic, prompt, application).
4. The AI produces document content wrapped in `<document>` tags.
5. The preview overlay renders the document and allows approve/reject/request-changes.
6. Approved documents are written to the correct directory with manifest sync.
7. Sessions are persisted and resumable.
8. `codectx new` compiles and offers link prompt after project creation.
9. All tests pass: `go test ./core/llm/ ./core/ide/ ./cmds/ide/`.
10. Build succeeds: `just build`.

### Should-Have (Phase 4+)

11. AI reads existing documentation via Read/Glob/Grep tools during conversation.
12. The AI identifies `depends_on` relationships with existing entries.
13. Auto-compile triggers after document approval when `auto_compile` is true.
14. Session auto-naming from document title after classification.
15. Glamour-rendered markdown in both chat viewport and preview.

### Nice-to-Have (Phase 6)

16. Anthropic API fallback works for users without Claude CLI.
17. Ollama fallback works for local model users.
18. Token usage displayed in header bar.
19. `codectx ide --list` shows all sessions with status.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Claude CLI stream-json format changes between versions | Low | High | Pin to known event types. Gracefully ignore unknown events. Test against recorded output. |
| Claude CLI auth expires mid-session | Medium | Medium | Detect auth error in result event. Surface clear re-login instructions. Session is preserved for resume. |
| System prompt exceeds Claude's context window | Low | Medium | Manifest summary is a condensed format. Monitor token usage. Truncate summary for very large projects. |
| Glamour rendering breaks terminal layout | Medium | Low | Glamour respects terminal width. Set `glamour.WithWordWrap(width)`. Test with narrow terminals. |
| `<document>` tag parsing fails on complex AI output | Medium | Medium | Simple state machine parser. The tag format is explicit and unambiguous. Include negative test cases. |
| Bubbletea viewport flickers during streaming | Low | Low | Batch delta updates. Only re-render on meaningful content changes, not every single character. |
| Session files accumulate without cleanup | Low | Low | 30-day auto-cleanup on startup. Session files are small (metadata only for Claude provider). |

---

## Future Work (Not In Scope)

These are documented for context but are explicitly excluded from this plan.

1. **Multi-document sessions** — Creating multiple related documents in a single session (e.g., a topic + its spec + related prompt). Would require session-level tracking of multiple document outputs.
2. **Bidirectional streaming** — Using `--input-format stream-json` for a long-running claude process instead of per-exchange subprocess spawning. Lower latency but more complex process management.
3. **Document editing mode** — Reopening an existing document for AI-assisted revision. Currently the tool only creates new documents.
4. **Prompt authoring specialization** — A dedicated prompt-authoring mode with `<execution>` and `<rules>` tag scaffolding. The current directive covers prompts generically.
5. **MCP server integration** — Exposing codectx documentation as an MCP resource for other AI tools to consume.
6. **Collaborative sessions** — Multiple users contributing to the same document session. Requires conflict resolution.

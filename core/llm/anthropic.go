package llm

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	anthropicDefaultModel = "claude-sonnet-4-20250514"
	anthropicMaxTokens    = 8192
)

// Anthropic implements Provider using the Anthropic Messages API.
// It manages conversation history as a []MessageParam slice, replayed
// on each call since the API is stateless.
type Anthropic struct {
	client  anthropic.Client
	model   string
	history []anthropic.MessageParam
}

// NewAnthropic creates a provider backed by the Anthropic API.
// The API key is read from the ANTHROPIC_API_KEY environment variable
// by the SDK automatically.
func NewAnthropic() *Anthropic {
	return &Anthropic{
		client: anthropic.NewClient(),
		model:  anthropicDefaultModel,
	}
}

func (a *Anthropic) ID() string { return "anthropic" }

// Stream sends a message to the Anthropic API and returns streaming events.
// Conversation history is replayed on each call for multi-turn continuity.
func (a *Anthropic) Stream(ctx context.Context, req *Request) (<-chan Event, error) {
	// Append user message to history.
	a.history = append(a.history, anthropic.NewUserMessage(
		anthropic.NewTextBlock(req.Prompt),
	))

	// Build system prompt.
	var system []anthropic.TextBlockParam
	if req.SystemPrompt != "" {
		system = []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		}
	}

	// Determine model.
	model := a.model
	if req.Model != "" {
		model = req.Model
	}

	// Start streaming.
	stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: anthropicMaxTokens,
		System:    system,
		Messages:  a.history,
	})

	ch := make(chan Event, 32)

	go func() {
		defer close(ch)

		message := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				ch <- ErrorEvent("accumulate error: " + err.Error())
				return
			}

			switch variant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch delta := variant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					ch <- DeltaEvent(delta.Text)
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- ErrorEvent(err.Error())
			return
		}

		// Extract the full response text for history.
		var fullText string
		for _, block := range message.Content {
			if text, ok := block.AsAny().(anthropic.TextBlock); ok {
				fullText += text.Text
			}
		}

		// Append assistant response to history.
		a.history = append(a.history, message.ToParam())

		// Emit result.
		ch <- ResultEvent(fullText, "", &Usage{
			InputTokens:  int(message.Usage.InputTokens),
			OutputTokens: int(message.Usage.OutputTokens),
		})
	}()

	return ch, nil
}

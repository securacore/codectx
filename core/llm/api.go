package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// maxTokens is the maximum output tokens for LLM responses.
const maxTokens = 4096

// maxRetries is the maximum number of retries for transient API errors.
const maxRetries = 3

// apiSender implements Sender using the Anthropic Messages API via the
// official Go SDK. Structured output uses the tool-use pattern: a tool
// is defined with the desired JSON schema and the model is forced to
// invoke it, guaranteeing schema-conformant output.
type apiSender struct {
	client *anthropic.Client
	model  string
}

// newAPISender creates an API sender from an API key and model name.
func newAPISender(apiKey, model string, opts ...option.RequestOption) (*apiSender, error) {
	allOpts := append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)
	client := anthropic.NewClient(allOpts...)
	return &apiSender{
		client: &client,
		model:  model,
	}, nil
}

// SendAliases sends a batch alias generation request using the tool-use pattern.
func (s *apiSender) SendAliases(ctx context.Context, system, prompt string) (*AliasResponse, error) {
	schema, err := schemaFromJSON(aliasJSONSchema)
	if err != nil {
		return nil, fmt.Errorf("building alias schema: %w", err)
	}

	raw, err := s.sendWithTool(ctx, system, prompt, "generate_aliases", "Generate aliases for taxonomy terms", schema)
	if err != nil {
		return nil, err
	}

	var resp AliasResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing alias response: %w", err)
	}
	return &resp, nil
}

// SendBridges sends a batch bridge generation request using the tool-use pattern.
func (s *apiSender) SendBridges(ctx context.Context, system, prompt string) (*BridgeResponse, error) {
	schema, err := schemaFromJSON(bridgeJSONSchema)
	if err != nil {
		return nil, fmt.Errorf("building bridge schema: %w", err)
	}

	raw, err := s.sendWithTool(ctx, system, prompt, "generate_bridges", "Generate bridge summaries for chunk boundaries", schema)
	if err != nil {
		return nil, err
	}

	var resp BridgeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing bridge response: %w", err)
	}
	return &resp, nil
}

// schemaFromJSON converts a JSON Schema string into the Anthropic SDK's
// ToolInputSchemaParam. This ensures both the CLI provider (which uses
// the JSON string directly) and the API provider (which needs the param
// struct) share the same canonical schema definition from schema.go.
func schemaFromJSON(jsonSchema string) (anthropic.ToolInputSchemaParam, error) {
	var parsed struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal([]byte(jsonSchema), &parsed); err != nil {
		return anthropic.ToolInputSchemaParam{}, fmt.Errorf("parsing JSON schema: %w", err)
	}
	return anthropic.ToolInputSchemaParam{
		Properties: parsed.Properties,
	}, nil
}

// sendWithTool sends a message that forces the model to use a specific tool,
// returning the tool's input as raw JSON. This is the standard Anthropic
// pattern for structured output.
func (s *apiSender) sendWithTool(ctx context.Context, system, prompt, toolName, toolDesc string, schema anthropic.ToolInputSchemaParam) (json.RawMessage, error) {
	tool := anthropic.ToolParam{
		Name:        toolName,
		Description: anthropic.String(toolDesc),
		InputSchema: schema,
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(s.model),
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: system},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
		Tools: []anthropic.ToolUnionParam{
			{OfTool: &tool},
		},
		ToolChoice: anthropic.ToolChoiceParamOfTool(toolName),
	}

	var lastErr error
	for attempt := range maxRetries {
		msg, err := s.client.Messages.New(ctx, params)
		if err != nil {
			lastErr = err
			// Exponential backoff: 1s, 2s, 4s.
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}

		// Extract the tool use block.
		for _, block := range msg.Content {
			if tu, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				raw, marshalErr := json.Marshal(tu.Input)
				if marshalErr != nil {
					return nil, fmt.Errorf("marshaling tool input: %w", marshalErr)
				}
				return raw, nil
			}
		}

		return nil, fmt.Errorf("no tool use block in response")
	}

	return nil, fmt.Errorf("API request failed after %d retries: %w", maxRetries, lastErr)
}

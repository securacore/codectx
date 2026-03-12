package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// TestSchemaFromJSON_AliasSchema verifies the alias JSON schema parses correctly.
func TestSchemaFromJSON_AliasSchema(t *testing.T) {
	schema, err := schemaFromJSON(aliasJSONSchema)
	if err != nil {
		t.Fatalf("schemaFromJSON(aliasJSONSchema): %v", err)
	}

	if schema.Properties == nil {
		t.Fatal("expected non-nil Properties")
	}

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatalf("expected Properties to be map[string]any, got %T", schema.Properties)
	}

	terms, ok := props["terms"]
	if !ok {
		t.Fatal("expected 'terms' property in schema")
	}

	termsMap, ok := terms.(map[string]any)
	if !ok {
		t.Fatalf("expected terms to be map[string]any, got %T", terms)
	}
	if termsMap["type"] != "array" {
		t.Errorf("expected terms type 'array', got %v", termsMap["type"])
	}
}

// TestSchemaFromJSON_BridgeSchema verifies the bridge JSON schema parses correctly.
func TestSchemaFromJSON_BridgeSchema(t *testing.T) {
	schema, err := schemaFromJSON(bridgeJSONSchema)
	if err != nil {
		t.Fatalf("schemaFromJSON(bridgeJSONSchema): %v", err)
	}

	if schema.Properties == nil {
		t.Fatal("expected non-nil Properties")
	}

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatalf("expected Properties to be map[string]any, got %T", schema.Properties)
	}

	bridges, ok := props["bridges"]
	if !ok {
		t.Fatal("expected 'bridges' property in schema")
	}

	bridgesMap, ok := bridges.(map[string]any)
	if !ok {
		t.Fatalf("expected bridges to be map[string]any, got %T", bridges)
	}
	if bridgesMap["type"] != "array" {
		t.Errorf("expected bridges type 'array', got %v", bridgesMap["type"])
	}
}

// TestSchemaFromJSON_InvalidJSON verifies error handling for invalid JSON.
func TestSchemaFromJSON_InvalidJSON(t *testing.T) {
	_, err := schemaFromJSON("not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestSchemaFromJSON_NoProperties verifies handling of schema with no properties field.
func TestSchemaFromJSON_NoProperties(t *testing.T) {
	schema, err := schemaFromJSON(`{"type": "object"}`)
	if err != nil {
		t.Fatalf("schemaFromJSON: %v", err)
	}

	// When "properties" is absent from JSON, unmarshaling produces nil map.
	props, ok := schema.Properties.(map[string]any)
	if ok && len(props) > 0 {
		t.Errorf("expected empty or nil Properties, got %v", schema.Properties)
	}
}

// ---------------------------------------------------------------------------
// API sender with mock HTTP server
// ---------------------------------------------------------------------------

// toolUseResponse builds a minimal Anthropic Messages API response JSON that
// contains a tool_use content block with the given input payload.
func toolUseResponse(toolID, toolName string, input any) string {
	inputJSON, _ := json.Marshal(input)
	return fmt.Sprintf(`{
		"id": "msg_test",
		"type": "message",
		"role": "assistant",
		"model": "test-model",
		"content": [
			{
				"type": "tool_use",
				"id": %q,
				"name": %q,
				"input": %s
			}
		],
		"stop_reason": "tool_use",
		"stop_sequence": null,
		"usage": {"input_tokens": 10, "output_tokens": 20}
	}`, toolID, toolName, string(inputJSON))
}

func newTestAPISender(t *testing.T, handler http.HandlerFunc) *apiSender {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	sender, err := newAPISender("test-key", "test-model",
		option.WithBaseURL(server.URL),
		option.WithMaxRetries(0),
	)
	if err != nil {
		t.Fatalf("newAPISender: %v", err)
	}
	return sender
}

func TestAPISender_SendAliases_Success(t *testing.T) {
	aliasInput := map[string]any{
		"terms": []map[string]any{
			{"key": "auth", "aliases": []string{"authentication", "login"}},
		},
	}

	sender := newTestAPISender(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, toolUseResponse("tool_1", "generate_aliases", aliasInput))
	})

	resp, err := sender.SendAliases(context.Background(), "system", "prompt")
	if err != nil {
		t.Fatalf("SendAliases: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Terms) != 1 {
		t.Fatalf("expected 1 term, got %d", len(resp.Terms))
	}
	if resp.Terms[0].Key != "auth" {
		t.Errorf("expected key 'auth', got %q", resp.Terms[0].Key)
	}
	if len(resp.Terms[0].Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(resp.Terms[0].Aliases))
	}
}

func TestAPISender_SendBridges_Success(t *testing.T) {
	bridgeInput := map[string]any{
		"bridges": []map[string]any{
			{"chunk_id": "obj:a1.01", "summary": "Auth flow established"},
		},
	}

	sender := newTestAPISender(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, toolUseResponse("tool_2", "generate_bridges", bridgeInput))
	})

	resp, err := sender.SendBridges(context.Background(), "system", "prompt")
	if err != nil {
		t.Fatalf("SendBridges: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Bridges) != 1 {
		t.Fatalf("expected 1 bridge, got %d", len(resp.Bridges))
	}
	if resp.Bridges[0].ChunkID != "obj:a1.01" {
		t.Errorf("expected chunk_id 'obj:a1.01', got %q", resp.Bridges[0].ChunkID)
	}
}

func TestAPISender_SendAliases_APIError(t *testing.T) {
	sender := newTestAPISender(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"server error"}}`)
	})

	_, err := sender.SendAliases(context.Background(), "system", "prompt")
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestAPISender_SendWithTool_NoToolUseBlock(t *testing.T) {
	// Return a response with only a text block, no tool_use.
	sender := newTestAPISender(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"model": "test-model",
			"content": [
				{"type": "text", "text": "I cannot do that"}
			],
			"stop_reason": "end_turn",
			"stop_sequence": null,
			"usage": {"input_tokens": 10, "output_tokens": 20}
		}`)
	})

	_, err := sender.SendAliases(context.Background(), "system", "prompt")
	if err == nil {
		t.Fatal("expected error when no tool_use block in response")
	}
	if !strings.Contains(err.Error(), "no tool use block") {
		t.Errorf("expected 'no tool use block' error, got: %v", err)
	}
}

func TestAPISender_SendWithTool_ContextCanceled(t *testing.T) {
	// Server always returns error, context is canceled before retries complete.
	sender := newTestAPISender(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"overloaded"}}`)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := sender.SendAliases(ctx, "system", "prompt")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

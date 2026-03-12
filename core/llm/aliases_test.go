package llm

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockSender is a test double for the Sender interface.
type mockSender struct {
	aliasResponses  []*AliasResponse
	aliasErrors     []error
	bridgeResponses []*BridgeResponse
	bridgeErrors    []error
	aliasCalls      int
	bridgeCalls     int
}

func (m *mockSender) SendAliases(_ context.Context, _, _ string) (*AliasResponse, error) {
	idx := m.aliasCalls
	m.aliasCalls++
	if idx < len(m.aliasErrors) && m.aliasErrors[idx] != nil {
		return nil, m.aliasErrors[idx]
	}
	if idx < len(m.aliasResponses) {
		return m.aliasResponses[idx], nil
	}
	return &AliasResponse{}, nil
}

func (m *mockSender) SendBridges(_ context.Context, _, _ string) (*BridgeResponse, error) {
	idx := m.bridgeCalls
	m.bridgeCalls++
	if idx < len(m.bridgeErrors) && m.bridgeErrors[idx] != nil {
		return nil, m.bridgeErrors[idx]
	}
	if idx < len(m.bridgeResponses) {
		return m.bridgeResponses[idx], nil
	}
	return &BridgeResponse{}, nil
}

func TestBuildAliasBatchPrompt(t *testing.T) {
	terms := []*aliasRequest{
		{
			Key:       "authentication",
			Canonical: "Authentication",
			Source:    "heading",
			Narrower:  []string{"jwt", "oauth"},
			Related:   []string{"authorization"},
		},
		{
			Key:       "jwt",
			Canonical: "JWT",
			Source:    "code_identifier",
			Broader:   "authentication",
		},
	}

	prompt := buildAliasBatchPrompt(terms, 10)

	// Verify key structural elements.
	if !strings.Contains(prompt, "Maximum 10 aliases") {
		t.Error("expected max alias count in prompt")
	}
	if !strings.Contains(prompt, "Key: authentication") {
		t.Error("expected 'authentication' term key")
	}
	if !strings.Contains(prompt, "Key: jwt") {
		t.Error("expected 'jwt' term key")
	}
	if !strings.Contains(prompt, "Broader: authentication") {
		t.Error("expected broader relationship for jwt")
	}
	if !strings.Contains(prompt, "Broader: (none)") {
		t.Error("expected '(none)' for top-level term")
	}
	if !strings.Contains(prompt, "Narrower: jwt, oauth") {
		t.Error("expected narrower terms for authentication")
	}
}

func TestGenerateAliases_MockSender(t *testing.T) {
	sender := &mockSender{
		aliasResponses: []*AliasResponse{
			{
				Terms: []AliasTermResponse{
					{Key: "authentication", Aliases: []string{"auth", "login", "sign-in"}},
					{Key: "jwt", Aliases: []string{"JSON Web Token", "bearer token"}},
				},
			},
		},
	}

	terms := []*aliasRequest{
		{Key: "authentication", Canonical: "Authentication", Source: "heading"},
		{Key: "jwt", Canonical: "JWT", Source: "code_identifier", Broader: "authentication"},
	}

	result := generateAliases(context.Background(), aliasGenConfig{
		sender: sender, terms: terms, instructions: "instructions", maxAliases: 10, batchSize: 50,
	})

	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}
	if result.TotalAliases != 5 {
		t.Errorf("expected 5 total aliases, got %d", result.TotalAliases)
	}
	if len(result.Aliases["authentication"]) != 3 {
		t.Errorf("expected 3 aliases for 'authentication', got %d", len(result.Aliases["authentication"]))
	}
	if len(result.Aliases["jwt"]) != 2 {
		t.Errorf("expected 2 aliases for 'jwt', got %d", len(result.Aliases["jwt"]))
	}
}

func TestGenerateAliases_EmptyTerms(t *testing.T) {
	sender := &mockSender{}
	result := generateAliases(context.Background(), aliasGenConfig{
		sender: sender, instructions: "instructions", maxAliases: 10, batchSize: 50,
	})

	if result.TotalAliases != 0 {
		t.Errorf("expected 0 aliases, got %d", result.TotalAliases)
	}
	if sender.aliasCalls != 0 {
		t.Errorf("expected 0 sender calls, got %d", sender.aliasCalls)
	}
}

func TestGenerateAliases_BatchSplit(t *testing.T) {
	// 7 terms with batch size 3 should produce 3 batches (3+3+1).
	sender := &mockSender{
		aliasResponses: []*AliasResponse{
			{Terms: []AliasTermResponse{{Key: "a", Aliases: []string{"a1"}}, {Key: "b", Aliases: []string{"b1"}}, {Key: "c", Aliases: []string{"c1"}}}},
			{Terms: []AliasTermResponse{{Key: "d", Aliases: []string{"d1"}}, {Key: "e", Aliases: []string{"e1"}}, {Key: "f", Aliases: []string{"f1"}}}},
			{Terms: []AliasTermResponse{{Key: "g", Aliases: []string{"g1"}}}},
		},
	}

	terms := make([]*aliasRequest, 7)
	for i := range terms {
		key := string(rune('a' + i))
		terms[i] = &aliasRequest{Key: key, Canonical: strings.ToUpper(key), Source: "heading"}
	}

	result := generateAliases(context.Background(), aliasGenConfig{
		sender: sender, terms: terms, instructions: "instructions", maxAliases: 10, batchSize: 3,
	})

	if sender.aliasCalls != 3 {
		t.Errorf("expected 3 batches, got %d", sender.aliasCalls)
	}
	if result.TotalAliases != 7 {
		t.Errorf("expected 7 total aliases, got %d", result.TotalAliases)
	}
}

func TestGenerateAliases_BatchError(t *testing.T) {
	// Second batch fails; first and third succeed.
	sender := &mockSender{
		aliasResponses: []*AliasResponse{
			{Terms: []AliasTermResponse{{Key: "a", Aliases: []string{"a1"}}}},
			nil, // This won't be used because of the error.
			{Terms: []AliasTermResponse{{Key: "c", Aliases: []string{"c1"}}}},
		},
		aliasErrors: []error{
			nil,
			fmt.Errorf("API rate limited"),
			nil,
		},
	}

	terms := []*aliasRequest{
		{Key: "a", Canonical: "A", Source: "heading"},
		{Key: "b", Canonical: "B", Source: "heading"},
		{Key: "c", Canonical: "C", Source: "heading"},
	}

	result := generateAliases(context.Background(), aliasGenConfig{
		sender: sender, terms: terms, instructions: "instructions", maxAliases: 10, batchSize: 1,
	})

	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
	if result.TotalAliases != 2 {
		t.Errorf("expected 2 total aliases (a + c), got %d", result.TotalAliases)
	}
}

func TestGenerateAliases_UnknownKeyFiltered(t *testing.T) {
	sender := &mockSender{
		aliasResponses: []*AliasResponse{
			{
				Terms: []AliasTermResponse{
					{Key: "auth", Aliases: []string{"login"}},
					{Key: "unknown-key", Aliases: []string{"should-be-ignored"}},
				},
			},
		},
	}

	terms := []*aliasRequest{
		{Key: "auth", Canonical: "Auth", Source: "heading"},
	}

	result := generateAliases(context.Background(), aliasGenConfig{
		sender: sender, terms: terms, instructions: "instructions", maxAliases: 10, batchSize: 50,
	})

	if _, ok := result.Aliases["unknown-key"]; ok {
		t.Error("expected unknown key to be filtered out")
	}
	if result.TotalAliases != 1 {
		t.Errorf("expected 1 alias, got %d", result.TotalAliases)
	}
}

func TestGroupByBranch(t *testing.T) {
	terms := []*aliasRequest{
		{Key: "jwt", Broader: "authentication"},
		{Key: "authentication"},
		{Key: "oauth", Broader: "authentication"},
		{Key: "error-handling"},
		{Key: "middleware", Broader: "error-handling"},
	}

	grouped := groupByBranch(terms)

	// Terms with broader="authentication" should be adjacent.
	var authGroup []string
	for _, g := range grouped {
		if g.Broader == "authentication" {
			authGroup = append(authGroup, g.Key)
		}
	}
	if len(authGroup) != 2 || authGroup[0] != "jwt" || authGroup[1] != "oauth" {
		t.Errorf("expected [jwt, oauth] in auth group, got %v", authGroup)
	}

	// Top-level terms (no broader) should be last.
	last := grouped[len(grouped)-1]
	secondLast := grouped[len(grouped)-2]
	topLevel := map[string]bool{last.Key: true, secondLast.Key: true}
	if !topLevel["authentication"] || !topLevel["error-handling"] {
		t.Errorf("expected top-level terms at end, got %q and %q", secondLast.Key, last.Key)
	}
}

func TestGenerateAliases_DefaultBatchAndMaxAlias(t *testing.T) {
	// Pass 0 for both batchSize and maxAliasCount to trigger defaults.
	sender := &mockSender{
		aliasResponses: []*AliasResponse{
			{Terms: []AliasTermResponse{{Key: "auth", Aliases: []string{"login"}}}},
		},
	}

	terms := []*aliasRequest{
		{Key: "auth", Canonical: "Auth", Source: "heading"},
	}

	result := generateAliases(context.Background(), aliasGenConfig{
		sender: sender, terms: terms, instructions: "instructions",
	})

	if result.TotalAliases != 1 {
		t.Errorf("expected 1 alias, got %d", result.TotalAliases)
	}
}

func TestApplyMaxAliases(t *testing.T) {
	resp := &AliasResponse{
		Terms: []AliasTermResponse{
			{Key: "auth", Aliases: []string{"a", "b", "c", "d", "e"}},
			{Key: "jwt", Aliases: []string{"x", "y"}},
		},
	}

	applyMaxAliases(resp, 3)

	if len(resp.Terms[0].Aliases) != 3 {
		t.Errorf("expected 3 aliases for auth, got %d", len(resp.Terms[0].Aliases))
	}
	if len(resp.Terms[1].Aliases) != 2 {
		t.Errorf("expected 2 aliases for jwt (unchanged), got %d", len(resp.Terms[1].Aliases))
	}
}

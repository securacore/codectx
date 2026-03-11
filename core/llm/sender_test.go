package llm_test

import (
	"errors"
	"testing"

	"github.com/securacore/codectx/core/llm"
	"github.com/securacore/codectx/core/project"
)

func TestNewSender_APIWithKey(t *testing.T) {
	sender, err := llm.NewSender(project.ProviderAPI, "test-key", "claude-sonnet-4-20250514", "claude")
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}
	if sender == nil {
		t.Fatal("expected non-nil sender for API with key")
	}
}

func TestNewSender_APIWithoutKey(t *testing.T) {
	sender, err := llm.NewSender(project.ProviderAPI, "", "claude-sonnet-4-20250514", "claude")
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}
	if sender != nil {
		t.Fatal("expected nil sender for API without key")
	}
}

func TestNewSender_CLIWithBinary(t *testing.T) {
	// Mock LookPath to find the binary.
	orig := llm.LookPathFunc
	llm.LookPathFunc = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/local/bin/claude", nil
		}
		return "", errors.New("not found")
	}
	defer func() { llm.LookPathFunc = orig }()

	sender, err := llm.NewSender(project.ProviderCLI, "", "claude-sonnet-4-20250514", "claude")
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}
	if sender == nil {
		t.Fatal("expected non-nil sender for CLI with binary")
	}
}

func TestNewSender_CLIWithoutBinary(t *testing.T) {
	orig := llm.LookPathFunc
	llm.LookPathFunc = func(_ string) (string, error) {
		return "", errors.New("not found")
	}
	defer func() { llm.LookPathFunc = orig }()

	sender, err := llm.NewSender(project.ProviderCLI, "", "claude-sonnet-4-20250514", "claude")
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}
	if sender != nil {
		t.Fatal("expected nil sender for CLI without binary")
	}
}

func TestNewSender_AutoDetect_APIFirst(t *testing.T) {
	// Both API key and CLI are available; API should be chosen.
	orig := llm.LookPathFunc
	llm.LookPathFunc = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/local/bin/claude", nil
		}
		return "", errors.New("not found")
	}
	defer func() { llm.LookPathFunc = orig }()

	sender, err := llm.NewSender("", "test-key", "claude-sonnet-4-20250514", "claude")
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}
	if sender == nil {
		t.Fatal("expected non-nil sender for auto-detect with API key")
	}
}

func TestNewSender_AutoDetect_CLIFallback(t *testing.T) {
	// No API key, but CLI is available.
	orig := llm.LookPathFunc
	llm.LookPathFunc = func(file string) (string, error) {
		if file == "claude" {
			return "/usr/local/bin/claude", nil
		}
		return "", errors.New("not found")
	}
	defer func() { llm.LookPathFunc = orig }()

	sender, err := llm.NewSender("", "", "claude-sonnet-4-20250514", "claude")
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}
	if sender == nil {
		t.Fatal("expected non-nil sender for auto-detect with CLI")
	}
}

func TestNewSender_AutoDetect_NothingAvailable(t *testing.T) {
	orig := llm.LookPathFunc
	llm.LookPathFunc = func(_ string) (string, error) {
		return "", errors.New("not found")
	}
	defer func() { llm.LookPathFunc = orig }()

	sender, err := llm.NewSender("", "", "claude-sonnet-4-20250514", "claude")
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}
	if sender != nil {
		t.Fatal("expected nil sender when nothing available")
	}
}

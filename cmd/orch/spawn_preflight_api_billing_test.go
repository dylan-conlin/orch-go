package main

import (
	"os"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/config"
)

func TestCheckAPIKeyBilling_NoAPIKeys(t *testing.T) {
	// Ensure no API keys are set
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")

	err := checkAPIKeyBilling(nil)
	if err != nil {
		t.Errorf("Expected no error when no API keys are set, got: %v", err)
	}
}

func TestCheckAPIKeyBilling_AnthropicKeyBlocked(t *testing.T) {
	// Save original values
	origAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	origAllowBilling := spawnAllowAPIBilling
	defer func() {
		if origAnthropicKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", origAnthropicKey)
		}
		spawnAllowAPIBilling = origAllowBilling
	}()

	// Set API key and ensure flag is false
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	spawnAllowAPIBilling = false

	err := checkAPIKeyBilling(nil)
	if err == nil {
		t.Error("Expected error when ANTHROPIC_API_KEY is set without --allow-api-billing")
	}

	if err != nil && !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("Error message should mention ANTHROPIC_API_KEY, got: %v", err)
	}
}

func TestCheckAPIKeyBilling_OpenAIKeyBlocked(t *testing.T) {
	// Save original values
	origOpenAIKey := os.Getenv("OPENAI_API_KEY")
	origAllowBilling := spawnAllowAPIBilling
	defer func() {
		if origOpenAIKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", origOpenAIKey)
		}
		spawnAllowAPIBilling = origAllowBilling
	}()

	// Set API key and ensure flag is false
	os.Setenv("OPENAI_API_KEY", "sk-proj-test-key")
	spawnAllowAPIBilling = false

	err := checkAPIKeyBilling(nil)
	if err == nil {
		t.Error("Expected error when OPENAI_API_KEY is set without --allow-api-billing")
	}

	if err != nil && !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("Error message should mention OPENAI_API_KEY, got: %v", err)
	}
}

func TestCheckAPIKeyBilling_BothKeysBlocked(t *testing.T) {
	// Save original values
	origAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	origOpenAIKey := os.Getenv("OPENAI_API_KEY")
	origAllowBilling := spawnAllowAPIBilling
	defer func() {
		if origAnthropicKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", origAnthropicKey)
		}
		if origOpenAIKey == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", origOpenAIKey)
		}
		spawnAllowAPIBilling = origAllowBilling
	}()

	// Set both API keys and ensure flag is false
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	os.Setenv("OPENAI_API_KEY", "sk-proj-test-key")
	spawnAllowAPIBilling = false

	err := checkAPIKeyBilling(nil)
	if err == nil {
		t.Error("Expected error when both API keys are set without --allow-api-billing")
	}

	if err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "ANTHROPIC_API_KEY") || !strings.Contains(errMsg, "OPENAI_API_KEY") {
			t.Errorf("Error message should mention both API keys, got: %v", err)
		}
	}
}

func TestCheckAPIKeyBilling_AllowedViaFlag(t *testing.T) {
	// Save original values
	origAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	origAllowBilling := spawnAllowAPIBilling
	defer func() {
		if origAnthropicKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", origAnthropicKey)
		}
		spawnAllowAPIBilling = origAllowBilling
	}()

	// Set API key and set flag to true
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	spawnAllowAPIBilling = true

	err := checkAPIKeyBilling(nil)
	if err != nil {
		t.Errorf("Expected no error when --allow-api-billing is set, got: %v", err)
	}
}

func TestCheckAPIKeyBilling_AllowedViaConfig(t *testing.T) {
	// Save original values
	origAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	origAllowBilling := spawnAllowAPIBilling
	defer func() {
		if origAnthropicKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", origAnthropicKey)
		}
		spawnAllowAPIBilling = origAllowBilling
	}()

	// Set API key, flag is false, but config allows it
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	spawnAllowAPIBilling = false

	cfg := &config.Config{
		Spawn: config.SpawnConfig{
			AllowAPIBilling: true,
		},
	}

	err := checkAPIKeyBilling(cfg)
	if err != nil {
		t.Errorf("Expected no error when config.spawn.allow_api_billing is true, got: %v", err)
	}
}

func TestCheckAPIKeyBilling_ErrorMessageContent(t *testing.T) {
	// Save original values
	origAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	origAllowBilling := spawnAllowAPIBilling
	defer func() {
		if origAnthropicKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", origAnthropicKey)
		}
		spawnAllowAPIBilling = origAllowBilling
	}()

	// Set API key and ensure flag is false
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	spawnAllowAPIBilling = false

	err := checkAPIKeyBilling(nil)
	if err == nil {
		t.Fatal("Expected error")
	}

	errMsg := err.Error()

	// Check that error message contains key information
	expectedPhrases := []string{
		"Pay-per-token API key detected",
		"ANTHROPIC_API_KEY",
		"--allow-api-billing",
		"unset",
		"OAuth",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(errMsg, phrase) {
			t.Errorf("Error message should contain %q, got: %v", phrase, errMsg)
		}
	}
}

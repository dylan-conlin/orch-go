// Package model provides model resolution and alias mapping for OpenCode.
package model

import "strings"

// ModelSpec represents a resolved model specification.
type ModelSpec struct {
	Provider string // e.g., "anthropic", "google"
	ModelID  string // e.g., "claude-sonnet-4-5-20250929", "gemini-2.5-flash"
}

// Format returns the provider/model format string.
func (m ModelSpec) Format() string {
	return m.Provider + "/" + m.ModelID
}

// DefaultModel is used when no model is specified.
// Sonnet is the default (Flash has TPM limits that make it unusable for real work).
var DefaultModel = ModelSpec{
	Provider: "anthropic",
	ModelID:  "claude-sonnet-4-5-20250929",
}

// Aliases maps short names to full model specs.
// Designed for quick switching between common models.
var Aliases = map[string]ModelSpec{
	// Anthropic models (Claude)
	"opus":       {Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
	"opus-4.5":   {Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
	"opus-4-5":   {Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"},
	"sonnet":     {Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
	"sonnet-4.5": {Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
	"sonnet-4-5": {Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
	"haiku":      {Provider: "anthropic", ModelID: "claude-haiku-4-5-20251001"},
	"haiku-4.5":  {Provider: "anthropic", ModelID: "claude-haiku-4-5-20251001"},
	"haiku-4-5":  {Provider: "anthropic", ModelID: "claude-haiku-4-5-20251001"},

	// Google models (Gemini)
	"flash":     {Provider: "google", ModelID: "gemini-3-flash-preview"},
	"flash-2.5": {Provider: "google", ModelID: "gemini-2.5-flash"},
	"flash3":    {Provider: "google", ModelID: "gemini-3-flash-preview"},
	"flash-3":   {Provider: "google", ModelID: "gemini-3-flash-preview"},
	"pro":       {Provider: "google", ModelID: "gemini-2.5-pro"},
	"pro-2.5":   {Provider: "google", ModelID: "gemini-2.5-pro"},

	// OpenAI models (GPT) - IDs from models.dev
	"gpt5":        {Provider: "openai", ModelID: "gpt-5"},
	"gpt-5":       {Provider: "openai", ModelID: "gpt-5"},
	"gpt5-latest": {Provider: "openai", ModelID: "gpt-5.2"},
	"gpt5-mini":   {Provider: "openai", ModelID: "gpt-5-mini"},
	"gpt-5-mini":  {Provider: "openai", ModelID: "gpt-5-mini"},
	"o3":          {Provider: "openai", ModelID: "o3"},
	"o3-mini":     {Provider: "openai", ModelID: "o3-mini"},

	// DeepSeek models (IDs from models.dev: deepseek-chat, deepseek-reasoner)
	"deepseek":      {Provider: "deepseek", ModelID: "deepseek-chat"},
	"deepseek-chat": {Provider: "deepseek", ModelID: "deepseek-chat"},
	"deepseek-r1":   {Provider: "deepseek", ModelID: "deepseek-reasoner"},
	"reasoning":     {Provider: "deepseek", ModelID: "deepseek-reasoner"},
}

// Resolve resolves a model specification to a full ModelSpec.
// Accepts:
//   - Empty string: returns DefaultModel
//   - Alias: "opus", "sonnet", "haiku", "flash", etc.
//   - Provider/model format: "anthropic/claude-opus-4-5-20251101", "google/gemini-2.5-flash"
//   - Model ID only (assumes anthropic for claude, google for gemini): "claude-opus-4-5-20251101"
func Resolve(spec string) ModelSpec {
	if spec == "" {
		return DefaultModel
	}

	// Normalize to lowercase for alias lookup
	specLower := strings.ToLower(spec)

	// Check aliases first
	if resolved, ok := Aliases[specLower]; ok {
		return resolved
	}

	// Check for provider/model format
	if idx := strings.Index(spec, "/"); idx > 0 {
		return ModelSpec{
			Provider: spec[:idx],
			ModelID:  spec[idx+1:],
		}
	}

	// Infer provider from model ID
	if strings.Contains(specLower, "claude") {
		return ModelSpec{Provider: "anthropic", ModelID: spec}
	}
	if strings.Contains(specLower, "gemini") {
		return ModelSpec{Provider: "google", ModelID: spec}
	}
	if strings.Contains(specLower, "gpt") {
		return ModelSpec{Provider: "openai", ModelID: spec}
	}
	if strings.Contains(specLower, "deepseek") {
		return ModelSpec{Provider: "deepseek", ModelID: spec}
	}

	// Default to anthropic for unknown models
	return ModelSpec{Provider: "anthropic", ModelID: spec}
}

// ListAliases returns a formatted list of available model aliases.
func ListAliases() []string {
	return []string{
		"Anthropic: opus, sonnet, haiku (also -4.5 variants)",
		"Google: flash, flash-2.5, flash3, flash-3, pro, pro-2.5",
		"OpenAI: gpt-5, gpt5-latest (5.2), gpt-5-mini, o3, o3-mini",
		"DeepSeek: deepseek, deepseek-chat, deepseek-r1, reasoning (alias for reasoner)",
	}
}

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
// Flash is the default (cheapest/fastest, as Opus is restricted to Claude Code as of Jan 2026).
var DefaultModel = ModelSpec{
	Provider: "google",
	ModelID:  "gemini-3-flash-preview",
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
	"flash-3.0": {Provider: "google", ModelID: "gemini-3-flash-preview"},
	"pro":       {Provider: "google", ModelID: "gemini-2.5-pro"},
	"pro-2.5":   {Provider: "google", ModelID: "gemini-2.5-pro"},
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

	// Default to anthropic for unknown models
	return ModelSpec{Provider: "anthropic", ModelID: spec}
}

// ListAliases returns a formatted list of available model aliases.
func ListAliases() []string {
	// Group by provider
	anthropic := []string{}
	google := []string{}

	for alias, spec := range Aliases {
		if spec.Provider == "anthropic" {
			anthropic = append(anthropic, alias)
		} else if spec.Provider == "google" {
			google = append(google, alias)
		}
	}

	return append(
		[]string{"Anthropic: opus, sonnet, haiku (also -4.5 variants)"},
		"Google: flash, flash-2.5, flash3, flash-3, pro, pro-2.5",
	)
}

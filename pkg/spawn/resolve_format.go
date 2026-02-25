package spawn

import (
	"fmt"
	"strings"
)

// FormatResolvedSpawnSettings returns a markdown bullet list of resolved settings.
// Returns an empty string when no resolved settings are present.
func FormatResolvedSpawnSettings(settings ResolvedSpawnSettings) string {
	if !hasResolvedSettings(settings) {
		return ""
	}

	lines := []string{
		formatResolvedSetting("Backend", settings.Backend, "unknown"),
		formatResolvedSetting("Model", settings.Model, "unknown"),
		formatResolvedSetting("Tier", settings.Tier, "unknown"),
		formatResolvedSetting("Spawn Mode", settings.SpawnMode, "unknown"),
		formatResolvedSetting("MCP", settings.MCP, "none"),
		formatResolvedSetting("Mode", settings.Mode, "unknown"),
		formatResolvedSetting("Validation", settings.Validation, "unknown"),
		formatResolvedSetting("Account", settings.Account, "none"),
	}

	return strings.Join(lines, "\n")
}

func hasResolvedSettings(settings ResolvedSpawnSettings) bool {
	if settings.Backend.Value != "" || settings.Backend.Source != "" {
		return true
	}
	if settings.Model.Value != "" || settings.Model.Source != "" {
		return true
	}
	if settings.Tier.Value != "" || settings.Tier.Source != "" {
		return true
	}
	if settings.SpawnMode.Value != "" || settings.SpawnMode.Source != "" {
		return true
	}
	if settings.MCP.Value != "" || settings.MCP.Source != "" {
		return true
	}
	if settings.Mode.Value != "" || settings.Mode.Source != "" {
		return true
	}
	if settings.Validation.Value != "" || settings.Validation.Source != "" {
		return true
	}
	if settings.Account.Value != "" || settings.Account.Source != "" {
		return true
	}
	return len(settings.Warnings) > 0
}

func formatResolvedSetting(label string, setting ResolvedSetting, emptyValue string) string {
	value := strings.TrimSpace(setting.Value)
	if value == "" {
		value = emptyValue
	}
	return fmt.Sprintf("- %s: %s (source: %s)", label, value, formatSettingSource(setting))
}

func formatSettingSource(setting ResolvedSetting) string {
	source := strings.TrimSpace(string(setting.Source))
	if source == "" {
		source = "unknown"
	}
	if strings.TrimSpace(setting.Detail) != "" {
		return fmt.Sprintf("%s (%s)", source, strings.TrimSpace(setting.Detail))
	}
	return source
}

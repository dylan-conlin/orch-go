package spawn

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// skillContentData holds the data context for processing skill content templates.
// Skill content files (SKILL.md) can contain Go template variables that need to be
// replaced with spawn-specific values before injection into SPAWN_CONTEXT.md.
type skillContentData struct {
	BeadsID string // The beads issue ID for progress tracking
	Tier    string // Spawn tier: "light" or "full"
}

// ProcessSkillContentTemplate processes Go template variables in skill content.
// Skill content (from SKILL.md files) may contain template variables like {{.BeadsID}}
// and conditionals like {{if eq .Tier "light"}}. This function processes those
// templates using the spawn-specific data context before the skill content is
// injected into SPAWN_CONTEXT.md.
//
// If template parsing or execution fails, returns the original content unchanged
// (fail-open behavior to avoid breaking spawns for minor template issues).
func ProcessSkillContentTemplate(content string, beadsID string, tier string) string {
	if content == "" {
		return content
	}

	// Quick check: if content doesn't contain template syntax, skip processing
	if !strings.Contains(content, "{{") {
		return content
	}

	tmpl, err := template.New("skill_content").Parse(content)
	if err != nil {
		// Template parse error - return original content
		// This can happen if skill content has malformed templates
		return content
	}

	data := skillContentData{
		BeadsID: beadsID,
		Tier:    tier,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Template execution error - return original content
		return content
	}

	return buf.String()
}

// StripBeadsInstructions removes beads-specific instructions from skill content.
// This is used when NoTrack=true to avoid confusing agents with beads commands
// that won't work (since there's no beads issue to track against).
//
// Removes:
// - Code blocks containing `bd comment` or `bd close` commands
// - "Report via Beads" sections
// - Lines containing `<beads-id>` placeholders
// - Completion criteria mentioning beads reporting
func StripBeadsInstructions(content string) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	var result []string
	inBeadsCodeBlock := false
	skipUntilNextSection := false
	inCodeBlockDuringSkip := false // Track if we entered a code block while skipping

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track code block state even while skipping
		// This prevents us from ending skip mode inside a code block
		if strings.HasPrefix(trimmedLine, "```") {
			if skipUntilNextSection {
				inCodeBlockDuringSkip = !inCodeBlockDuringSkip
			}
		}

		// Check if we're starting a beads-related section
		if regexBeadsSectionHeader.MatchString(line) {
			skipUntilNextSection = true
			inCodeBlockDuringSkip = false // Reset code block tracking
			continue
		}

		// Check if we've reached a new section (exit beads section)
		// But ONLY if we're not inside a code block
		if skipUntilNextSection && !inCodeBlockDuringSkip && regexNextSectionHeader.MatchString(line) && !regexBeadsSectionHeader.MatchString(line) {
			skipUntilNextSection = false
			// Include this line (the new section header)
		}

		// Skip lines while in beads section
		if skipUntilNextSection {
			continue
		}

		_ = i // Silence unused variable warning

		// Check for code blocks containing beads commands
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inBeadsCodeBlock {
				// End of beads code block - skip the closing ```
				inBeadsCodeBlock = false
				continue
			}
			// Look ahead to see if this code block contains beads commands
			hasBeadsCommand := false
			for j := i + 1; j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "```"); j++ {
				if regexBeadsIDPlaceholder.MatchString(lines[j]) {
					hasBeadsCommand = true
					break
				}
			}
			if hasBeadsCommand {
				inBeadsCodeBlock = true
				continue
			}
		}

		// Skip lines inside beads code blocks
		if inBeadsCodeBlock {
			continue
		}

		// Skip individual lines with beads completion criteria
		if regexBeadsReportedCriteria.MatchString(line) {
			continue
		}

		// Skip lines that are just beads commands with <beads-id>
		if regexBeadsIDPlaceholder.MatchString(line) && strings.TrimSpace(line) != "" {
			// Only skip if it's a standalone command line (not part of documentation)
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "bd ") || strings.HasPrefix(trimmed, "- `bd ") {
				continue
			}
		}

		result = append(result, line)
	}

	// Clean up excessive blank lines that may result from stripping
	output := strings.Join(result, "\n")
	// Replace 3+ consecutive newlines with 2
	output = regexMultiNewline.ReplaceAllString(output, "\n\n")

	return output
}

// WriteSkillPromptFile writes compiled skill content to SKILL_PROMPT.md in the workspace directory.
// This file is used by --append-system-prompt "$(cat SKILL_PROMPT.md)" for system-level injection.
// The file path is set on cfg.SystemPromptFile for use by BuildClaudeLaunchCommand.
// Returns nil if cfg.SkillContent is empty (no skill to write).
func WriteSkillPromptFile(cfg *Config) error {
	if cfg.SkillContent == "" {
		return nil
	}

	skillContent := cfg.SkillContent
	// Strip beads instructions when NoTrack is true
	if cfg.NoTrack {
		skillContent = StripBeadsInstructions(skillContent)
	}
	// Process template variables (e.g., {{.BeadsID}})
	skillContent = ProcessSkillContentTemplate(skillContent, cfg.BeadsID, cfg.Tier)

	promptPath := filepath.Join(cfg.WorkspacePath(), "SKILL_PROMPT.md")
	if err := os.WriteFile(promptPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("failed to write SKILL_PROMPT.md: %w", err)
	}

	cfg.SystemPromptFile = promptPath
	return nil
}

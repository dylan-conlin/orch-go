package spawn

import (
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for skill content cleanup.
var (
	regexBeadsSectionHeader    = regexp.MustCompile(`(?i)^#+\s*(report\s+(via|to)\s+beads|beads\s+(progress\s+)?tracking)`)
	regexNextSectionHeader     = regexp.MustCompile(`^#{1,6}\s+[A-Z]`)
	regexBeadsReportedCriteria = regexp.MustCompile(`(?i)\*\*Reported\*\*.*bd\s+comment`)
	regexBeadsIDPlaceholder    = regexp.MustCompile(`bd\s+(comment|close|show)\s+<beads-id>`)
	regexMultiNewline          = regexp.MustCompile(`\n{3,}`)
)

func prepareSkillContent(cfg *Config) string {
	skillContent := cfg.SkillContent
	if cfg.NoTrack && skillContent != "" {
		return StripBeadsInstructions(skillContent)
	}
	return skillContent
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

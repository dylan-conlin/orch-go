package spawn

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/episodic"
)

const episodicSectionLimit = 3

// GenerateRecentValidatedEpisodesSection renders accepted episodic entries for spawn context.
func GenerateRecentValidatedEpisodesSection(cfg *Config) string {
	if cfg == nil || cfg.NoTrack {
		return ""
	}
	if strings.TrimSpace(cfg.BeadsID) == "" {
		return ""
	}

	store := episodic.NewStore("")
	entries, err := store.QueryByBeadsID(cfg.BeadsID, 20)
	if err != nil || len(entries) == 0 {
		return ""
	}

	validated := episodic.ValidateEpisodesForReuse(entries, episodic.ValidateOptions{
		Now:           time.Now().UTC(),
		AutoInjection: true,
		MinConfidence: 0.7,
		Scope: episodic.Scope{
			Project:    scopeProject(cfg),
			BeadsID:    cfg.BeadsID,
			ProjectDir: cfg.ProjectDir,
		},
	})

	accepted := []episodic.ValidatedEpisode{}
	for _, item := range validated {
		if item.State != episodic.ValidationStateAccepted {
			continue
		}
		accepted = append(accepted, item)
	}

	if len(accepted) == 0 {
		return ""
	}

	if len(accepted) > episodicSectionLimit {
		accepted = accepted[:episodicSectionLimit]
	}

	var sb strings.Builder
	sb.WriteString("## RECENT VALIDATED EPISODES\n\n")
	for _, item := range accepted {
		source := evidenceSource(item.Episode)
		action := item.Episode.Action.Name
		if strings.TrimSpace(action) == "" {
			action = item.Episode.Action.Type
		}
		if strings.TrimSpace(action) == "" {
			action = "action"
		}

		summary := strings.TrimSpace(item.Summary)
		if summary == "" {
			summary = strings.TrimSpace(item.Episode.Outcome.Summary)
		}

		sb.WriteString(fmt.Sprintf("- %s: %s (confidence %.2f, source %s)\n", action, summary, item.Episode.Confidence, source))
	}
	sb.WriteString("\n")

	return sb.String()
}

func scopeProject(cfg *Config) string {
	if strings.TrimSpace(cfg.Project) != "" {
		return strings.TrimSpace(cfg.Project)
	}
	if strings.TrimSpace(cfg.ProjectDir) == "" {
		return ""
	}
	return filepath.Base(cfg.ProjectDir)
}

func evidenceSource(ep episodic.Episode) string {
	if strings.TrimSpace(ep.Evidence.Pointer) == "" {
		return "unknown"
	}
	first := strings.TrimSpace(ep.Evidence.Pointer)
	if idx := strings.Index(first, "#"); idx >= 0 {
		first = first[:idx]
	}
	if first == "" {
		return "unknown"
	}
	return filepath.Base(first)
}

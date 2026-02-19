package main

import (
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

type untrackedSession struct {
	Session       opencode.Session
	Category      string
	Role          string
	BeadsID       string
	Tier          string
	SpawnMode     string
	Skill         string
	Model         string
	WorkspacePath string
}

func listUntrackedSessions(client *opencode.Client, currentProjectDir string) ([]untrackedSession, error) {
	sessions, err := listSessionsAcrossProjects(client, currentProjectDir)
	if err != nil {
		return nil, err
	}

	result := make([]untrackedSession, 0, len(sessions))
	for _, session := range sessions {
		category, meta := classifyUntrackedSession(session)
		if category == "" {
			continue
		}
		result = append(result, untrackedSession{
			Session:       session,
			Category:      category,
			Role:          meta.Role,
			BeadsID:       meta.BeadsID,
			Tier:          meta.Tier,
			SpawnMode:     meta.SpawnMode,
			Skill:         meta.Skill,
			Model:         meta.Model,
			WorkspacePath: meta.WorkspacePath,
		})
	}

	return result, nil
}

type untrackedSessionMeta struct {
	Role          string
	BeadsID       string
	Tier          string
	SpawnMode     string
	Skill         string
	Model         string
	WorkspacePath string
}

func classifyUntrackedSession(session opencode.Session) (string, untrackedSessionMeta) {
	meta := untrackedSessionMeta{}
	if session.Metadata != nil {
		meta.Role = strings.TrimSpace(session.Metadata["role"])
		meta.BeadsID = strings.TrimSpace(session.Metadata["beads_id"])
		meta.Tier = strings.TrimSpace(session.Metadata["tier"])
		meta.SpawnMode = strings.TrimSpace(session.Metadata["spawn_mode"])
		meta.Skill = strings.TrimSpace(session.Metadata["skill"])
		meta.Model = strings.TrimSpace(session.Metadata["model"])
		meta.WorkspacePath = strings.TrimSpace(session.Metadata["workspace_path"])
	}

	roleLower := strings.ToLower(meta.Role)
	skillLower := strings.ToLower(meta.Skill)
	if roleLower == "" && (skillLower == "orchestrator" || skillLower == "meta-orchestrator") {
		roleLower = skillLower
		meta.Role = meta.Skill
	}

	if roleLower == "orchestrator" || roleLower == "meta-orchestrator" {
		return "orchestrator", meta
	}

	noTrack := false
	if session.Metadata != nil {
		noTrack = parseMetadataBool(session.Metadata["no_track"])
	}
	if noTrack {
		return "no-track", meta
	}

	if meta.BeadsID == "" {
		return "ad-hoc", meta
	}

	return "", meta
}

func parseMetadataBool(value string) bool {
	if value == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func formatSessionTime(epochMillis int64) string {
	if epochMillis <= 0 {
		return ""
	}
	return time.Unix(epochMillis/1000, 0).Format(time.RFC3339)
}

func (s untrackedSession) toOutput() UntrackedSessionOutput {
	return UntrackedSessionOutput{
		ID:            s.Session.ID,
		Title:         s.Session.Title,
		Category:      s.Category,
		Role:          s.Role,
		BeadsID:       s.BeadsID,
		Tier:          s.Tier,
		SpawnMode:     s.SpawnMode,
		Skill:         s.Skill,
		Model:         s.Model,
		WorkspacePath: s.WorkspacePath,
		ProjectDir:    s.Session.Directory,
		CreatedAt:     formatSessionTime(s.Session.Time.Created),
		UpdatedAt:     formatSessionTime(s.Session.Time.Updated),
	}
}

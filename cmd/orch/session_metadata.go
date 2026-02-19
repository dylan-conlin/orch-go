package main

import (
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func workspacePathFromSession(session *opencode.Session) string {
	if session == nil || session.Metadata == nil {
		return ""
	}
	if workspacePath, ok := session.Metadata["workspace_path"]; ok {
		return strings.TrimSpace(workspacePath)
	}
	return ""
}

func workspaceNameFromSession(session *opencode.Session) string {
	if session == nil {
		return ""
	}
	if workspacePath := workspacePathFromSession(session); workspacePath != "" {
		return filepath.Base(workspacePath)
	}
	return session.Title
}

func beadsIDFromSession(session *opencode.Session) string {
	if session == nil {
		return ""
	}
	if session.Metadata != nil {
		if beadsID, ok := session.Metadata["beads_id"]; ok && beadsID != "" {
			return beadsID
		}
	}
	return extractBeadsIDFromTitle(session.Title)
}

func projectDirFromWorkspacePath(workspacePath string) string {
	if workspacePath == "" {
		return ""
	}
	cleaned := filepath.Clean(workspacePath)
	marker := string(filepath.Separator) + ".orch" + string(filepath.Separator) + "workspace"
	idx := strings.Index(cleaned, marker)
	if idx <= 0 {
		return ""
	}
	return cleaned[:idx]
}

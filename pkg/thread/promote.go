package thread

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PromotionCandidate represents a converged thread ready for promotion.
type PromotionCandidate struct {
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Updated    string `json:"updated"`
	EntryCount int    `json:"entry_count"`
}

// Promote transitions a converged thread to promoted status, setting its
// promoted_to field to the target artifact path. It also propagates the
// promotion to ancestor threads whose resolved_to points at this thread.
func Promote(threadsDir, slug, artifactType, targetPath string) error {
	path, _ := findThreadBySlug(threadsDir, slug)
	if path == "" {
		return fmt.Errorf("thread %q not found", slug)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	thread, err := ParseThread(string(data))
	if err != nil {
		return err
	}

	if thread.Status != StatusConverged {
		return fmt.Errorf("thread %q has status %q, must be %q to promote", slug, thread.Status, StatusConverged)
	}

	content := string(data)
	today := time.Now().Format("2006-01-02")

	// Update status to promoted
	content = updateFrontmatter(content, "status", StatusPromoted)
	content = updateFrontmatter(content, "updated", today)

	// Set promoted_to — need to insert if field doesn't exist
	if strings.Contains(content, "promoted_to:") {
		content = updateFrontmatterQuoted(content, "promoted_to", targetPath)
	} else {
		content = insertFrontmatterField(content, "promoted_to", "\""+targetPath+"\"")
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	// Propagate to ancestors: update resolved_to on threads that point at this thread
	propagatePromotionToAncestors(threadsDir, slug, targetPath)

	return nil
}

// PromotionReady returns converged threads that have no promoted_to set.
func PromotionReady(threadsDir string) ([]PromotionCandidate, error) {
	all, err := List(threadsDir)
	if err != nil {
		return nil, err
	}

	var ready []PromotionCandidate
	for _, s := range all {
		if s.Status == StatusConverged && s.PromotedTo == "" {
			ready = append(ready, PromotionCandidate{
				Slug:       s.Name,
				Title:      s.Title,
				Updated:    s.Updated,
				EntryCount: s.EntryCount,
			})
		}
	}

	return ready, nil
}

// propagatePromotionToAncestors scans threads whose resolved_to points at
// the promoted thread (by slug) and updates it to point at the new artifact.
func propagatePromotionToAncestors(threadsDir, promotedSlug, targetPath string) {
	entries, err := os.ReadDir(threadsDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		path := filepath.Join(threadsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		thread, err := ParseThread(string(data))
		if err != nil {
			continue
		}

		// Check if resolved_to references the promoted thread slug
		if thread.ResolvedTo == "" {
			continue
		}
		// Match by slug name (threads reference each other by slug)
		if thread.ResolvedTo == promotedSlug || strings.HasSuffix(thread.ResolvedTo, "/"+promotedSlug) {
			content := string(data)
			content = updateFrontmatterQuoted(content, "resolved_to", targetPath)
			today := time.Now().Format("2006-01-02")
			content = updateFrontmatter(content, "updated", today)
			_ = os.WriteFile(path, []byte(content), 0644)
		}
	}
}

// insertFrontmatterField inserts a new field before the closing --- of frontmatter.
func insertFrontmatterField(content, field, value string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			// Insert before closing ---
			result := make([]string, 0, len(lines)+1)
			result = append(result, lines[:i]...)
			result = append(result, field+": "+value)
			result = append(result, lines[i:]...)
			return strings.Join(result, "\n")
		}
	}
	return content
}

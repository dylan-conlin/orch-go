package thread

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackPropResult holds the result of a back-propagation on a single thread.
type BackPropResult struct {
	Slug     string
	FilePath string
}

// BackPropagateCompletion scans all threads in threadsDir for ones that reference
// beadsID in their active_work list. For each match, it removes the beadsID from
// active_work and adds it to resolved_by, and updates the 'updated' field.
func BackPropagateCompletion(threadsDir, beadsID string) ([]BackPropResult, error) {
	entries, err := os.ReadDir(threadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []BackPropResult
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

		// Check if this thread has the beads ID in active_work
		found := false
		for _, aw := range thread.ActiveWork {
			if aw == beadsID {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		// Remove from active_work, add to resolved_by
		if err := removeFrontmatterListItem(path, "active_work", beadsID); err != nil {
			return nil, fmt.Errorf("removing from active_work in %s: %w", e.Name(), err)
		}
		if err := addFrontmatterListItem(path, "resolved_by", beadsID); err != nil {
			return nil, fmt.Errorf("adding to resolved_by in %s: %w", e.Name(), err)
		}

		// Update the 'updated' field
		today := time.Now().Format("2006-01-02")
		data, _ = os.ReadFile(path)
		content := updateFrontmatter(string(data), "updated", today)
		_ = os.WriteFile(path, []byte(content), 0644)

		slug := extractSlug(e.Name())
		results = append(results, BackPropResult{
			Slug:     slug,
			FilePath: path,
		})
	}

	return results, nil
}

// addFrontmatterListItem adds an item to a YAML list field in a thread's frontmatter.
// If the field doesn't exist, it creates it. Deduplicates entries.
func addFrontmatterListItem(path, field, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	thread, err := ParseThread(content)
	if err != nil {
		return err
	}

	// Check for duplicates
	var existing []string
	switch field {
	case "spawned":
		existing = thread.Spawned
	case "active_work":
		existing = thread.ActiveWork
	case "resolved_by":
		existing = thread.ResolvedBy
	}
	for _, v := range existing {
		if v == value {
			return nil // already present
		}
	}

	lines := strings.Split(content, "\n")
	inFrontmatter := false
	fmEnd := -1
	fieldLine := -1
	lastListItem := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			fmEnd = i
			break
		}
		if inFrontmatter {
			if strings.HasPrefix(trimmed, field+":") {
				fieldLine = i
				lastListItem = i
			} else if fieldLine >= 0 && strings.HasPrefix(trimmed, "- ") {
				lastListItem = i
			} else if fieldLine >= 0 {
				break
			}
		}
	}

	newItem := fmt.Sprintf("  - \"%s\"", value)

	if fieldLine >= 0 {
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:lastListItem+1]...)
		result = append(result, newItem)
		result = append(result, lines[lastListItem+1:]...)
		return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
	}

	if fmEnd < 0 {
		return fmt.Errorf("no frontmatter found")
	}
	result := make([]string, 0, len(lines)+2)
	result = append(result, lines[:fmEnd]...)
	result = append(result, field+":")
	result = append(result, newItem)
	result = append(result, lines[fmEnd:]...)
	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
}

// removeFrontmatterListItem removes an item from a YAML list field in a thread's frontmatter.
func removeFrontmatterListItem(path, field, item string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	inFrontmatter := false
	inField := false
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				inField = false
			}
			inFrontmatter = !inFrontmatter
			result = append(result, line)
			continue
		}
		if inFrontmatter {
			if strings.HasPrefix(trimmed, field+":") {
				inField = true
				result = append(result, line)
				continue
			}
			if inField && strings.HasPrefix(trimmed, "- ") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				val = strings.Trim(val, "\"")
				if val == item {
					continue // skip this item
				}
				result = append(result, line)
				continue
			}
			if inField {
				inField = false
			}
		}
		result = append(result, line)
	}

	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
}

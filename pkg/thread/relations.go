// Package thread relations provides thread-to-thread and thread-to-work linkage.
package thread

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CreateWithParent creates a new thread with a spawned_from reference to a parent thread.
// It also adds the new thread's slug to the parent's spawned list (best-effort).
func CreateWithParent(threadsDir, title, entry, parentSlug string) (*Result, error) {
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating threads dir: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	slug := Slugify(title)
	filename := today + "-" + slug + ".md"
	path := filepath.Join(threadsDir, filename)

	content := fmt.Sprintf(`---
title: "%s"
status: open
created: %s
updated: %s
resolved_to: ""
spawned_from: "%s"
---

# %s

## %s

%s
`, title, today, today, parentSlug, title, today, entry)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("writing thread: %w", err)
	}

	// Update parent's spawned list (best-effort — parent may not exist)
	if parentPath, _ := findThreadBySlug(threadsDir, parentSlug); parentPath != "" {
		_ = addFrontmatterListItem(parentPath, "spawned", slug)
	}

	return &Result{
		Created:    true,
		EntryCount: 1,
		FilePath:   path,
	}, nil
}

// LinkWork adds a beads ID to a thread's active_work list.
func LinkWork(threadsDir, slug, beadsID string) error {
	path, _ := findThreadBySlug(threadsDir, slug)
	if path == "" {
		return fmt.Errorf("thread %q not found", slug)
	}
	return addFrontmatterListItem(path, "active_work", beadsID)
}

// AddSpawned adds a child thread slug to a parent thread's spawned list.
func AddSpawned(threadsDir, parentSlug, childSlug string) error {
	path, _ := findThreadBySlug(threadsDir, parentSlug)
	if path == "" {
		return fmt.Errorf("thread %q not found", parentSlug)
	}
	return addFrontmatterListItem(path, "spawned", childSlug)
}

// Package thread provides living threads — multi-session accumulating
// knowledge artifacts for forming comprehension. Threads capture insight
// as it crystallizes mid-session and accumulate dated entries across sessions.
package thread

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Thread represents a parsed thread file.
type Thread struct {
	Title      string
	Status     string // open, resolved
	Created    string // YYYY-MM-DD
	Updated    string // YYYY-MM-DD
	ResolvedTo string
	Entries    []Entry
	Content    string // raw file content
	Slug       string // filename slug (without date prefix and .md)
	Filename   string // full filename
}

// Entry represents a single dated entry within a thread.
type Entry struct {
	Date string // YYYY-MM-DD
	Text string // entry content (trimmed)
}

// ThreadSummary is a compact representation for listing.
type ThreadSummary struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	ResolvedTo  string `json:"resolved_to,omitempty"`
	LatestEntry string `json:"latest_entry"`
	EntryCount  int    `json:"entry_count"`
	Filename    string `json:"filename"`
}

// ThreadEntry represents a thread entry for debrief aggregation.
type ThreadEntry struct {
	ThreadName  string `json:"thread_name"`
	ThreadTitle string `json:"thread_title"`
	Text        string `json:"text"`
}

// Result represents the outcome of a CreateOrAppend operation.
type Result struct {
	Created    bool
	EntryCount int
	FilePath   string
}

// stopWords are removed from slugs.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"but": true, "in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true, "is": true,
	"it": true, "how": true, "what": true, "when": true, "where": true,
	"why": true, "whether": true, "should": true, "be": true,
}

var dateHeadingRe = regexp.MustCompile(`^## (\d{4}-\d{2}-\d{2})$`)

// Slugify converts a title or name to a URL-safe slug.
// Removes stop words and limits to ~5 significant words.
func Slugify(input string) string {
	input = strings.ToLower(input)

	// Replace non-alphanumeric with hyphens
	var b strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	slug := b.String()

	// Split into words, remove stop words
	parts := strings.Split(slug, "-")
	var words []string
	for _, p := range parts {
		if p == "" || stopWords[p] {
			continue
		}
		words = append(words, p)
	}

	// Limit to 5 significant words
	if len(words) > 5 {
		words = words[:5]
	}

	return strings.Join(words, "-")
}

// CreateOrAppend creates a new thread file or appends to an existing one.
// The nameOrTitle parameter can be either a slug (to find an existing thread)
// or a full title (to create a new one).
func CreateOrAppend(threadsDir, nameOrTitle, entry string) (*Result, error) {
	if err := os.MkdirAll(threadsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating threads dir: %w", err)
	}

	today := time.Now().Format("2006-01-02")

	// Try to find existing thread by slug
	slug := Slugify(nameOrTitle)
	existingPath, existingFilename := findThreadBySlug(threadsDir, slug)

	if existingPath != "" {
		return appendToThread(existingPath, existingFilename, today, entry)
	}

	// Create new thread — nameOrTitle is the title
	return createThread(threadsDir, nameOrTitle, slug, today, entry)
}

// createThread creates a new thread file.
func createThread(dir, title, slug, today, entry string) (*Result, error) {
	filename := today + "-" + slug + ".md"
	path := filepath.Join(dir, filename)

	content := fmt.Sprintf(`---
title: "%s"
status: open
created: %s
updated: %s
resolved_to: ""
---

# %s

## %s

%s
`, title, today, today, title, today, entry)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("writing thread: %w", err)
	}

	return &Result{
		Created:    true,
		EntryCount: 1,
		FilePath:   path,
	}, nil
}

// appendToThread appends an entry to an existing thread file.
func appendToThread(path, filename, today, entry string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading thread: %w", err)
	}

	content := string(data)

	// Update frontmatter 'updated' field
	content = updateFrontmatter(content, "updated", today)

	// Check if today's date heading already exists
	todayHeading := "## " + today
	if strings.Contains(content, todayHeading) {
		// Append to existing today section
		content = appendToDateSection(content, todayHeading, entry)
	} else {
		// Add new dated section at the end
		content = strings.TrimRight(content, "\n") + "\n\n" + todayHeading + "\n\n" + entry + "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("writing thread: %w", err)
	}

	// Count entries
	thread, _ := ParseThread(content)
	entryCount := 0
	if thread != nil {
		entryCount = len(thread.Entries)
	}

	return &Result{
		Created:    false,
		EntryCount: entryCount,
		FilePath:   path,
	}, nil
}

// appendToDateSection appends text to an existing dated section.
func appendToDateSection(content, heading, entry string) string {
	idx := strings.Index(content, heading)
	if idx < 0 {
		return content
	}

	// Find the next ## heading or end of file
	afterHeading := idx + len(heading)
	rest := content[afterHeading:]

	nextHeadingIdx := -1
	lines := strings.Split(rest, "\n")
	pos := 0
	for i, line := range lines {
		if i > 0 && dateHeadingRe.MatchString(strings.TrimSpace(line)) {
			nextHeadingIdx = pos
			break
		}
		pos += len(line) + 1
	}

	if nextHeadingIdx >= 0 {
		// Insert before next heading
		insertPoint := afterHeading + nextHeadingIdx
		before := strings.TrimRight(content[:insertPoint], "\n")
		after := content[insertPoint:]
		return before + "\n\n" + entry + "\n\n" + after
	}

	// Append at end
	return strings.TrimRight(content, "\n") + "\n\n" + entry + "\n"
}

// updateFrontmatter updates a field in the YAML frontmatter.
func updateFrontmatter(content, field, value string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break // end of frontmatter
		}
		if inFrontmatter && strings.HasPrefix(trimmed, field+":") {
			lines[i] = field + ": " + value
		}
	}
	return strings.Join(lines, "\n")
}

// List returns all threads sorted by updated date (most recent first).
func List(threadsDir string) ([]ThreadSummary, error) {
	entries, err := os.ReadDir(threadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var summaries []ThreadSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(threadsDir, e.Name()))
		if err != nil {
			continue
		}

		thread, err := ParseThread(string(data))
		if err != nil {
			continue
		}

		slug := extractSlug(e.Name())
		latestEntry := ""
		if len(thread.Entries) > 0 {
			last := thread.Entries[len(thread.Entries)-1]
			latestEntry = truncate(strings.TrimSpace(last.Text), 80)
		}

		summaries = append(summaries, ThreadSummary{
			Name:        slug,
			Title:       thread.Title,
			Status:      thread.Status,
			Created:     thread.Created,
			Updated:     thread.Updated,
			ResolvedTo:  thread.ResolvedTo,
			LatestEntry: latestEntry,
			EntryCount:  len(thread.Entries),
			Filename:    e.Name(),
		})
	}

	// Sort by updated date descending
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Updated > summaries[j].Updated
	})

	return summaries, nil
}

// Show returns the full thread content for a given slug.
func Show(threadsDir, slug string) (*Thread, error) {
	path, filename := findThreadBySlug(threadsDir, slug)
	if path == "" {
		return nil, fmt.Errorf("thread %q not found", slug)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	thread, err := ParseThread(string(data))
	if err != nil {
		return nil, err
	}

	thread.Content = string(data)
	thread.Slug = slug
	thread.Filename = filename
	return thread, nil
}

// Resolve marks a thread as resolved with an optional target artifact path.
func Resolve(threadsDir, slug, resolvedTo string) error {
	path, _ := findThreadBySlug(threadsDir, slug)
	if path == "" {
		return fmt.Errorf("thread %q not found", slug)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	today := time.Now().Format("2006-01-02")

	content = updateFrontmatter(content, "status", "resolved")
	content = updateFrontmatter(content, "updated", today)
	content = updateFrontmatterQuoted(content, "resolved_to", resolvedTo)

	return os.WriteFile(path, []byte(content), 0644)
}

// updateFrontmatterQuoted updates a quoted field in the YAML frontmatter.
func updateFrontmatterQuoted(content, field, value string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}
		if inFrontmatter && strings.HasPrefix(trimmed, field+":") {
			lines[i] = field + ": \"" + value + "\""
		}
	}
	return strings.Join(lines, "\n")
}

// TodaysEntries returns thread entries from the given date for debrief aggregation.
// Only includes open (non-resolved) threads.
func TodaysEntries(threadsDir, date string) ([]ThreadEntry, error) {
	entries, err := os.ReadDir(threadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []ThreadEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(threadsDir, e.Name()))
		if err != nil {
			continue
		}

		thread, err := ParseThread(string(data))
		if err != nil {
			continue
		}

		// Skip resolved threads
		if thread.Status == "resolved" {
			continue
		}

		// Find entry for the given date
		for _, entry := range thread.Entries {
			if entry.Date == date {
				slug := extractSlug(e.Name())
				result = append(result, ThreadEntry{
					ThreadName:  slug,
					ThreadTitle: thread.Title,
					Text:        strings.TrimSpace(entry.Text),
				})
			}
		}
	}

	return result, nil
}

// ActiveThreads returns open threads updated within maxAge days.
func ActiveThreads(threadsDir string, maxAge int) ([]ThreadSummary, error) {
	all, err := List(threadsDir)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().AddDate(0, 0, -maxAge).Format("2006-01-02")

	var active []ThreadSummary
	for _, s := range all {
		if s.Status == "resolved" {
			continue
		}
		if s.Updated < cutoff {
			continue
		}
		active = append(active, s)
	}

	return active, nil
}

// ParseThread parses a thread file's content into a Thread struct.
func ParseThread(content string) (*Thread, error) {
	thread := &Thread{}

	lines := strings.Split(content, "\n")

	// Parse frontmatter
	inFrontmatter := false
	fmEnd := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			fmEnd = i + 1
			break
		}
		if inFrontmatter {
			parseFrontmatterLine(thread, trimmed)
		}
	}

	// Parse entries (## YYYY-MM-DD sections)
	var currentEntry *Entry
	for i := fmEnd; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if m := dateHeadingRe.FindStringSubmatch(trimmed); m != nil {
			if currentEntry != nil {
				currentEntry.Text = strings.TrimSpace(currentEntry.Text)
				thread.Entries = append(thread.Entries, *currentEntry)
			}
			currentEntry = &Entry{Date: m[1]}
			continue
		}
		if currentEntry != nil {
			currentEntry.Text += lines[i] + "\n"
		}
	}
	if currentEntry != nil {
		currentEntry.Text = strings.TrimSpace(currentEntry.Text)
		thread.Entries = append(thread.Entries, *currentEntry)
	}

	return thread, nil
}

// parseFrontmatterLine parses a single YAML frontmatter line.
func parseFrontmatterLine(thread *Thread, line string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	value = strings.Trim(value, "\"")

	switch key {
	case "title":
		thread.Title = value
	case "status":
		thread.Status = value
	case "created":
		thread.Created = value
	case "updated":
		thread.Updated = value
	case "resolved_to":
		thread.ResolvedTo = value
	}
}

// findThreadBySlug finds a thread file by matching the slug portion of the filename.
func findThreadBySlug(dir, slug string) (path string, filename string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fileSlug := extractSlug(e.Name())
		if fileSlug == slug {
			return filepath.Join(dir, e.Name()), e.Name()
		}
	}
	return "", ""
}

// extractSlug extracts the slug from a thread filename.
// "2026-03-05-enforcement-comprehension.md" -> "enforcement-comprehension"
func extractSlug(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	// Remove date prefix (YYYY-MM-DD-)
	if len(name) > 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
		return name[11:]
	}
	return name
}

// truncate limits a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

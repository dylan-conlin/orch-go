// Package sessions provides search and listing capabilities for OpenCode session history.
// It walks the OpenCode disk storage and uses the API to fetch message content for searching.
package sessions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/execution"
)

// Pre-compiled regex patterns for sessions.go
var regexMultipleNewlines = regexp.MustCompile(`\n+`)

// DefaultStoragePath returns the default OpenCode storage path.
func DefaultStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "storage")
}

// StoredSession represents a session stored on disk.
type StoredSession struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectID"`
	Directory string    `json:"directory"`
	Title     string    `json:"title"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
	Summary   struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Files     int `json:"files"`
	} `json:"summary"`
}

// SearchResult represents a search match in a session.
type SearchResult struct {
	Session    StoredSession `json:"session"`
	MatchCount int           `json:"match_count"`
	Snippets   []string      `json:"snippets"`
}

// DiskSession represents the JSON structure of a session file on disk.
type DiskSession struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	ProjectID string `json:"projectID"`
	Directory string `json:"directory"`
	Title     string `json:"title"`
	Time      struct {
		Created int64 `json:"created"`
		Updated int64 `json:"updated"`
	} `json:"time"`
	Summary struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Files     int `json:"files"`
	} `json:"summary"`
}

// ListOptions configures session listing.
type ListOptions struct {
	StoragePath string
	Directory   string
	After       *time.Time
	Before      *time.Time
	Limit       int
}

// SearchOptions configures session searching.
type SearchOptions struct {
	StoragePath   string
	Query         string
	UseRegex      bool
	CaseSensitive bool
	Directory     string
	After         *time.Time
	Before        *time.Time
	Limit         int
	Client        execution.SessionClient
}

// Store provides access to OpenCode session storage.
type Store struct {
	storagePath string
	client      execution.SessionClient
}

// NewStore creates a new session store.
func NewStore(storagePath string, client execution.SessionClient) *Store {
	if storagePath == "" {
		storagePath = DefaultStoragePath()
	}
	return &Store{
		storagePath: storagePath,
		client:      client,
	}
}

// List returns all sessions matching the given options.
func (s *Store) List(opts ListOptions) ([]StoredSession, error) {
	if opts.StoragePath == "" {
		opts.StoragePath = s.storagePath
	}

	sessionDir := filepath.Join(opts.StoragePath, "session")
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil, nil
	}

	var sessions []StoredSession

	projectDirs, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}

		projectPath := filepath.Join(sessionDir, projectDir.Name())
		sessionFiles, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".json") {
				continue
			}

			filePath := filepath.Join(projectPath, sessionFile.Name())
			session, err := s.readSessionFile(filePath)
			if err != nil {
				continue
			}

			if opts.Directory != "" && session.Directory != opts.Directory {
				continue
			}
			if opts.After != nil && session.Created.Before(*opts.After) {
				continue
			}
			if opts.Before != nil && session.Created.After(*opts.Before) {
				continue
			}

			sessions = append(sessions, session)
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Updated.After(sessions[j].Updated)
	})

	if opts.Limit > 0 && len(sessions) > opts.Limit {
		sessions = sessions[:opts.Limit]
	}

	return sessions, nil
}

// Search finds sessions containing the given query.
func (s *Store) Search(opts SearchOptions) ([]SearchResult, error) {
	if opts.StoragePath == "" {
		opts.StoragePath = s.storagePath
	}

	client := opts.Client
	if client == nil {
		client = s.client
	}

	sessions, err := s.List(ListOptions{
		StoragePath: opts.StoragePath,
		Directory:   opts.Directory,
		After:       opts.After,
		Before:      opts.Before,
	})
	if err != nil {
		return nil, err
	}

	var pattern *regexp.Regexp
	if opts.UseRegex {
		flags := ""
		if !opts.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err = regexp.Compile(flags + opts.Query)
		if err != nil {
			return nil, err
		}
	}

	var results []SearchResult

	for _, session := range sessions {
		if client == nil {
			continue
		}

		messages, err := client.GetMessages(context.Background(), execution.SessionHandle(session.ID))
		if err != nil {
			continue
		}

		matchCount := 0
		var snippets []string

		for _, msg := range messages {
			for _, part := range msg.Parts {
				if part.Type != "text" || part.Text == "" {
					continue
				}

				var matches [][]int
				text := part.Text

				if opts.UseRegex {
					matches = pattern.FindAllStringIndex(text, -1)
				} else {
					query := opts.Query
					searchText := text
					if !opts.CaseSensitive {
						query = strings.ToLower(query)
						searchText = strings.ToLower(text)
					}
					start := 0
					for {
						idx := strings.Index(searchText[start:], query)
						if idx == -1 {
							break
						}
						matches = append(matches, []int{start + idx, start + idx + len(query)})
						start = start + idx + 1
					}
				}

				if len(matches) > 0 {
					matchCount += len(matches)
					if len(snippets) < 3 {
						snippet := extractSnippet(text, matches[0][0], matches[0][1], 100)
						snippets = append(snippets, snippet)
					}
				}
			}
		}

		if matchCount > 0 {
			results = append(results, SearchResult{
				Session:    session,
				MatchCount: matchCount,
				Snippets:   snippets,
			})
		}

		if opts.Limit > 0 && len(results) >= opts.Limit {
			break
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].MatchCount > results[j].MatchCount
	})

	return results, nil
}

// Show returns full session details including message content.
func (s *Store) Show(sessionID string) (*StoredSession, []execution.Message, error) {
	if s.client == nil {
		return nil, nil, os.ErrNotExist
	}

	ctx := context.Background()
	apiSession, err := s.client.GetSession(ctx, execution.SessionHandle(sessionID))
	if err != nil {
		return nil, nil, err
	}

	session := StoredSession{
		ID:        apiSession.ID,
		Directory: apiSession.Directory,
		Title:     apiSession.Title,
		Created:   apiSession.Created,
		Updated:   apiSession.Updated,
	}
	session.Summary.Additions = apiSession.Summary.Additions
	session.Summary.Deletions = apiSession.Summary.Deletions
	session.Summary.Files = apiSession.Summary.Files

	messages, err := s.client.GetMessages(ctx, execution.SessionHandle(sessionID))
	if err != nil {
		return &session, nil, err
	}

	return &session, messages, nil
}

func (s *Store) readSessionFile(path string) (StoredSession, error) {
	var session StoredSession

	data, err := os.ReadFile(path)
	if err != nil {
		return session, err
	}

	var disk DiskSession
	if err := json.Unmarshal(data, &disk); err != nil {
		return session, err
	}

	session.ID = disk.ID
	session.ProjectID = disk.ProjectID
	session.Directory = disk.Directory
	session.Title = disk.Title
	session.Created = time.Unix(disk.Time.Created/1000, 0)
	session.Updated = time.Unix(disk.Time.Updated/1000, 0)
	session.Summary = disk.Summary

	return session, nil
}

func extractSnippet(text string, start, end, contextLen int) string {
	snippetStart := start - contextLen
	if snippetStart < 0 {
		snippetStart = 0
	}
	snippetEnd := end + contextLen
	if snippetEnd > len(text) {
		snippetEnd = len(text)
	}

	for snippetStart > 0 && text[snippetStart] != ' ' && text[snippetStart] != '\n' {
		snippetStart--
	}
	for snippetEnd < len(text) && text[snippetEnd] != ' ' && text[snippetEnd] != '\n' {
		snippetEnd++
	}

	snippet := strings.TrimSpace(text[snippetStart:snippetEnd])

	prefix := ""
	suffix := ""
	if snippetStart > 0 {
		prefix = "..."
	}
	if snippetEnd < len(text) {
		suffix = "..."
	}

	snippet = regexMultipleNewlines.ReplaceAllString(snippet, " ")

	return prefix + snippet + suffix
}

package opencode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ListSessionsOpts configures optional server-side filtering for session listing.
type ListSessionsOpts struct {
	Start     int64
	Limit     int
	Search    string
	RootsOnly bool
}

// CreateSessionRequest represents the request body for creating a new session.
type CreateSessionRequest struct {
	Title     string `json:"title,omitempty"`
	Directory string `json:"directory,omitempty"`
	Model     string `json:"model,omitempty"`
	Variant   string `json:"variant,omitempty"`
}

// CreateSessionResponse represents the response from creating a new session.
type CreateSessionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Directory string `json:"directory,omitempty"`
}

// ListSessions fetches all sessions from the OpenCode API.
func (c *Client) ListSessions(directory string) ([]Session, error) {
	return c.ListSessionsWithOpts(directory, nil)
}

// ListSessionsWithOpts fetches sessions with optional server-side filtering.
func (c *Client) ListSessionsWithOpts(directory string, opts *ListSessionsOpts) ([]Session, error) {
	u := c.ServerURL + "/session"

	params := make([]string, 0)
	if opts != nil {
		if opts.Start > 0 {
			params = append(params, fmt.Sprintf("start=%d", opts.Start))
		}
		if opts.Limit > 0 {
			params = append(params, fmt.Sprintf("limit=%d", opts.Limit))
		}
		if opts.Search != "" {
			params = append(params, "search="+opts.Search)
		}
		if opts.RootsOnly {
			params = append(params, "roots=true")
		}
	}
	if len(params) > 0 {
		u += "?" + strings.Join(params, "&")
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if directory != "" {
		req.Header.Set("x-opencode-directory", directory)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}

// GetSession fetches a single session by ID from the OpenCode API.
func (c *Client) GetSession(sessionID string) (*Session, error) {
	req, err := http.NewRequest("GET", c.ServerURL+"/session/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// SessionExists checks if a session exists in OpenCode (in-memory).
func (c *Client) SessionExists(sessionID string) bool {
	req, err := http.NewRequest("GET", c.ServerURL+"/session/"+sessionID, nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// IsSessionActive checks if a session is actively running (updated within maxIdleTime).
func (c *Client) IsSessionActive(sessionID string, maxIdleTime time.Duration) bool {
	session, err := c.GetSession(sessionID)
	if err != nil {
		return false
	}
	updatedAt := time.Unix(session.Time.Updated/1000, 0)
	return time.Since(updatedAt) <= maxIdleTime
}

// IsSessionProcessing checks if a session is actively processing.
func (c *Client) IsSessionProcessing(sessionID string) bool {
	messages, err := c.GetMessages(sessionID)
	if err != nil || len(messages) == 0 {
		return false
	}

	lastMsg := messages[len(messages)-1]

	if lastMsg.Info.Role == "assistant" {
		return lastMsg.Info.Finish == "" && lastMsg.Info.Time.Completed == 0
	}

	if lastMsg.Info.Role == "user" {
		createdAt := time.Unix(lastMsg.Info.Time.Created/1000, 0)
		return time.Since(createdAt) < 30*time.Second
	}

	return false
}

// CreateSession creates a new OpenCode session via HTTP API.
func (c *Client) CreateSession(title, directory, model, variant string, isWorker bool) (*CreateSessionResponse, error) {
	payload := CreateSessionRequest{
		Title:     title,
		Directory: directory,
		Model:     model,
		Variant:   variant,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.ServerURL+"/session", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if directory != "" {
		req.Header.Set("x-opencode-directory", directory)
	}

	if isWorker {
		req.Header.Set("x-opencode-env-ORCH_WORKER", "1")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteSession deletes an OpenCode session by ID.
func (c *Client) DeleteSession(sessionID string) error {
	req, err := http.NewRequest("DELETE", c.ServerURL+"/session/"+sessionID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete session: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateSessionTitle updates the title of an existing OpenCode session.
func (c *Client) UpdateSessionTitle(sessionID, newTitle string) error {
	payload := map[string]string{"title": newTitle}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/sessions/%s", c.ServerURL, sessionID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// ListDiskSessions lists all sessions stored on disk for a given directory.
func (c *Client) ListDiskSessions(directory string) ([]Session, error) {
	if directory == "" {
		return nil, fmt.Errorf("directory is required for ListDiskSessions")
	}

	req, err := http.NewRequest("GET", c.ServerURL+"/session", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-opencode-directory", directory)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch disk sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}

// FindRecentSession finds the most recent session for a given project directory.
func (c *Client) FindRecentSession(projectDir string) (string, error) {
	req, err := http.NewRequest("GET", c.ServerURL+"/session", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-opencode-directory", projectDir)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return "", err
	}

	var mostRecent *Session
	now := time.Now().UnixMilli()
	for i := range sessions {
		s := &sessions[i]
		if s.Directory != projectDir {
			continue
		}
		if now-s.Time.Created > 30*1000 {
			continue
		}
		if mostRecent == nil || s.Time.Created > mostRecent.Time.Created {
			mostRecent = s
		}
	}

	if mostRecent == nil {
		return "", fmt.Errorf("no sessions found for directory: %s (within last 30s)", projectDir)
	}

	return mostRecent.ID, nil
}

// FindRecentSessionWithRetry retries finding a recent session with exponential backoff.
func (c *Client) FindRecentSessionWithRetry(projectDir string, maxAttempts int, initialDelay time.Duration) (string, error) {
	delay := initialDelay
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		sessionID, err := c.FindRecentSession(projectDir)
		if err == nil {
			return sessionID, nil
		}
		lastErr = err

		if attempt < maxAttempts {
			time.Sleep(delay)
			delay = delay * 2
		}
	}

	return "", lastErr
}

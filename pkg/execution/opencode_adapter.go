package execution

import (
	"context"
	"io"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// OpenCodeAdapter wraps an opencode.Client to implement SessionClient.
// This is the bridge layer that allows existing OpenCode-based code to work
// through the backend-agnostic interface.
type OpenCodeAdapter struct {
	client *opencode.Client
}

// Compile-time interface check.
var _ SessionClient = (*OpenCodeAdapter)(nil)

// NewOpenCodeAdapter creates a new adapter wrapping an OpenCode client.
func NewOpenCodeAdapter(serverURL string) *OpenCodeAdapter {
	return &OpenCodeAdapter{client: opencode.NewClient(serverURL)}
}

// NewOpenCodeAdapterFromClient wraps an existing opencode.Client.
func NewOpenCodeAdapterFromClient(client *opencode.Client) *OpenCodeAdapter {
	return &OpenCodeAdapter{client: client}
}

// Underlying returns the wrapped opencode.Client for cases where callers
// need direct access during the migration period. This should be removed
// once all callers use the SessionClient interface.
func (a *OpenCodeAdapter) Underlying() *opencode.Client {
	return a.client
}

func (a *OpenCodeAdapter) CreateSession(_ context.Context, req SessionRequest) (SessionHandle, error) {
	resp, err := a.client.CreateSession(req.Title, req.Directory, req.Model, req.Metadata, req.TimeTTL)
	if err != nil {
		return "", err
	}
	return SessionHandle(resp.ID), nil
}

func (a *OpenCodeAdapter) SendPrompt(_ context.Context, handle SessionHandle, prompt, model string) error {
	return a.client.SendPrompt(string(handle), prompt, model)
}

func (a *OpenCodeAdapter) DeleteSession(_ context.Context, handle SessionHandle) error {
	return a.client.DeleteSession(string(handle))
}

func (a *OpenCodeAdapter) GetSession(_ context.Context, handle SessionHandle) (*SessionInfo, error) {
	session, err := a.client.GetSession(string(handle))
	if err != nil {
		return nil, err
	}
	return convertSession(session), nil
}

func (a *OpenCodeAdapter) ListSessions(_ context.Context, directory string) ([]SessionInfo, error) {
	sessions, err := a.client.ListSessions(directory)
	if err != nil {
		return nil, err
	}
	result := make([]SessionInfo, len(sessions))
	for i := range sessions {
		result[i] = *convertSession(&sessions[i])
	}
	return result, nil
}

func (a *OpenCodeAdapter) GetMessages(_ context.Context, handle SessionHandle) ([]Message, error) {
	messages, err := a.client.GetMessages(string(handle))
	if err != nil {
		return nil, err
	}
	return convertMessages(messages), nil
}

func (a *OpenCodeAdapter) GetSessionTokens(_ context.Context, handle SessionHandle) (*TokenStats, error) {
	stats, err := a.client.GetSessionTokens(string(handle))
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, nil
	}
	return &TokenStats{
		InputTokens:     stats.InputTokens,
		OutputTokens:    stats.OutputTokens,
		ReasoningTokens: stats.ReasoningTokens,
		CacheReadTokens: stats.CacheReadTokens,
		TotalTokens:     stats.TotalTokens,
	}, nil
}

func (a *OpenCodeAdapter) GetLastActivity(_ context.Context, handle SessionHandle) (*LastActivity, error) {
	activity, err := a.client.GetLastActivity(string(handle))
	if err != nil {
		return nil, err
	}
	if activity == nil {
		return nil, nil
	}
	return &LastActivity{
		Text:      activity.Text,
		Timestamp: time.Unix(activity.Timestamp/1000, 0),
	}, nil
}

func (a *OpenCodeAdapter) GetSessionStatus(_ context.Context, handle SessionHandle) (*SessionStatusInfo, error) {
	status, err := a.client.GetSessionStatusByID(string(handle))
	if err != nil {
		return nil, err
	}
	if status == nil {
		return &SessionStatusInfo{Type: "idle"}, nil
	}
	return &SessionStatusInfo{
		Type:    status.Type,
		Message: status.Message,
	}, nil
}

func (a *OpenCodeAdapter) GetAllSessionStatus(_ context.Context) (map[string]SessionStatusInfo, error) {
	statuses, err := a.client.GetAllSessionStatus()
	if err != nil {
		return nil, err
	}
	result := make(map[string]SessionStatusInfo, len(statuses))
	for id, s := range statuses {
		result[id] = SessionStatusInfo{
			Type:    s.Type,
			Message: s.Message,
		}
	}
	return result, nil
}

func (a *OpenCodeAdapter) IsReachable(_ context.Context) bool {
	return a.client.IsReachable()
}

func (a *OpenCodeAdapter) WaitForIdle(_ context.Context, handle SessionHandle) error {
	return a.client.WaitForSessionIdle(string(handle))
}

func (a *OpenCodeAdapter) SendMessageWithStreaming(_ context.Context, handle SessionHandle, content string, streamTo io.Writer) error {
	return a.client.SendMessageWithStreaming(string(handle), content, streamTo)
}

func (a *OpenCodeAdapter) FindRecentSession(_ context.Context, directory string) (SessionHandle, error) {
	id, err := a.client.FindRecentSession(directory)
	if err != nil {
		return "", err
	}
	return SessionHandle(id), nil
}

func (a *OpenCodeAdapter) FindRecentSessionWithRetry(_ context.Context, directory string, maxAttempts int, initialDelay time.Duration) (SessionHandle, error) {
	id, err := a.client.FindRecentSessionWithRetry(directory, maxAttempts, initialDelay)
	if err != nil {
		return "", err
	}
	return SessionHandle(id), nil
}

func (a *OpenCodeAdapter) ExportTranscript(_ context.Context, handle SessionHandle) (string, error) {
	return a.client.ExportSessionTranscript(string(handle))
}

func (a *OpenCodeAdapter) SendMessageAsync(_ context.Context, handle SessionHandle, content, model string) error {
	return a.client.SendMessageAsync(string(handle), content, model)
}

func (a *OpenCodeAdapter) SendMessageInDirectory(_ context.Context, handle SessionHandle, content, model, directory string) error {
	return a.client.SendMessageInDirectory(string(handle), content, model, directory)
}

func (a *OpenCodeAdapter) SessionExists(_ context.Context, handle SessionHandle) bool {
	return a.client.SessionExists(string(handle))
}

func (a *OpenCodeAdapter) IsSessionActive(_ context.Context, handle SessionHandle, maxIdleTime time.Duration) bool {
	return a.client.IsSessionActive(string(handle), maxIdleTime)
}

func (a *OpenCodeAdapter) IsSessionProcessing(_ context.Context, handle SessionHandle) bool {
	return a.client.IsSessionProcessing(string(handle))
}

func (a *OpenCodeAdapter) GetSessionStatusByIDs(_ context.Context, sessionIDs []string) (map[string]SessionStatusInfo, error) {
	statuses, err := a.client.GetSessionStatusByIDs(sessionIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]SessionStatusInfo, len(statuses))
	for id, s := range statuses {
		result[id] = SessionStatusInfo{
			Type:    s.Type,
			Message: s.Message,
		}
	}
	return result, nil
}

func (a *OpenCodeAdapter) ListDiskSessions(_ context.Context, directory string) ([]SessionInfo, error) {
	sessions, err := a.client.ListDiskSessions(directory)
	if err != nil {
		return nil, err
	}
	result := make([]SessionInfo, len(sessions))
	for i := range sessions {
		result[i] = *convertSession(&sessions[i])
	}
	return result, nil
}

func (a *OpenCodeAdapter) GetLastMessage(_ context.Context, handle SessionHandle) (*Message, error) {
	msg, err := a.client.GetLastMessage(string(handle))
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	msgs := convertMessages([]opencode.Message{*msg})
	return &msgs[0], nil
}

func (a *OpenCodeAdapter) SetSessionMetadata(_ context.Context, handle SessionHandle, metadata map[string]string) error {
	return a.client.SetSessionMetadata(string(handle), metadata)
}

func (a *OpenCodeAdapter) WaitForSessionError(_ context.Context, handle SessionHandle, timeout time.Duration) (string, error) {
	return a.client.WaitForSessionError(string(handle), timeout)
}

func (a *OpenCodeAdapter) VerifySessionAfterPrompt(_ context.Context, handle SessionHandle, directory string, timeout time.Duration) error {
	return a.client.VerifySessionAfterPrompt(string(handle), directory, timeout)
}

func (a *OpenCodeAdapter) GetSessionInDirectory(_ context.Context, handle SessionHandle, directory string) (*SessionInfo, error) {
	session, err := a.client.GetSessionInDirectory(string(handle), directory)
	if err != nil {
		return nil, err
	}
	return convertSession(session), nil
}

// --- Conversion helpers ---

func convertSession(s *opencode.Session) *SessionInfo {
	return &SessionInfo{
		ID:        s.ID,
		Directory: s.Directory,
		Title:     s.Title,
		ParentID:  s.ParentID,
		Created:   time.Unix(s.Time.Created/1000, 0),
		Updated:   time.Unix(s.Time.Updated/1000, 0),
		Metadata:  s.Metadata,
		Summary: ChangeSummary{
			Additions: s.Summary.Additions,
			Deletions: s.Summary.Deletions,
			Files:     s.Summary.Files,
		},
	}
}

func convertMessages(msgs []opencode.Message) []Message {
	result := make([]Message, len(msgs))
	for i, m := range msgs {
		msg := Message{
			ID:        m.Info.ID,
			SessionID: m.Info.SessionID,
			Role:      m.Info.Role,
			Created:   time.Unix(m.Info.Time.Created/1000, 0),
			Finish:    m.Info.Finish,
			Cost:      m.Info.Cost,
		}
		if m.Info.Time.Completed > 0 {
			msg.Completed = time.Unix(m.Info.Time.Completed/1000, 0)
		}
		if m.Info.Tokens != nil {
			msg.Tokens = &TokenCount{
				Input:     m.Info.Tokens.Input,
				Output:    m.Info.Tokens.Output,
				Reasoning: m.Info.Tokens.Reasoning,
			}
			if m.Info.Tokens.Cache != nil {
				msg.Tokens.CacheRead = m.Info.Tokens.Cache.Read
			}
		}
		msg.Parts = make([]MessagePart, len(m.Parts))
		for j, p := range m.Parts {
			part := MessagePart{
				Type:   p.Type,
				Text:   p.Text,
				Tool:   p.Tool,
				CallID: p.CallID,
			}
			if p.State != nil {
				part.State = &ToolState{
					Status:   p.State.Status,
					Input:    p.State.Input,
					Output:   p.State.Output,
					Title:    p.State.Title,
					Metadata: p.State.Metadata,
				}
			}
			msg.Parts[j] = part
		}
		result[i] = msg
	}
	return result
}

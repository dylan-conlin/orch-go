package execution

import (
	"context"
	"io"
	"time"
)

// SessionClient defines the backend-agnostic interface for session operations.
// Implementations exist for OpenCode (current) and will exist for OpenClaw (future).
type SessionClient interface {
	// Session lifecycle
	CreateSession(ctx context.Context, req SessionRequest) (SessionHandle, error)
	SendPrompt(ctx context.Context, handle SessionHandle, prompt, model string) error
	DeleteSession(ctx context.Context, handle SessionHandle) error

	// Session queries
	GetSession(ctx context.Context, handle SessionHandle) (*SessionInfo, error)
	ListSessions(ctx context.Context, directory string) ([]SessionInfo, error)
	GetMessages(ctx context.Context, handle SessionHandle) ([]Message, error)
	GetSessionTokens(ctx context.Context, handle SessionHandle) (*TokenStats, error)
	GetLastActivity(ctx context.Context, handle SessionHandle) (*LastActivity, error)

	// Session status
	GetSessionStatus(ctx context.Context, handle SessionHandle) (*SessionStatusInfo, error)
	GetAllSessionStatus(ctx context.Context) (map[string]SessionStatusInfo, error)
	IsReachable(ctx context.Context) bool

	// Blocking operations
	WaitForIdle(ctx context.Context, handle SessionHandle) error
	SendMessageWithStreaming(ctx context.Context, handle SessionHandle, content string, streamTo io.Writer) error

	// Session discovery
	FindRecentSession(ctx context.Context, directory string) (SessionHandle, error)
	FindRecentSessionWithRetry(ctx context.Context, directory string, maxAttempts int, initialDelay time.Duration) (SessionHandle, error)

	// Transcript
	ExportTranscript(ctx context.Context, handle SessionHandle) (string, error)
}

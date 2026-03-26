package execution

import (
	"context"
	"io"
	"time"
)

// DefaultServerURL is the default execution backend server URL.
const DefaultServerURL = "http://127.0.0.1:4096"

// SessionClient defines the backend-agnostic interface for session operations.
// Implementations exist for OpenCode (current) and will exist for OpenClaw (future).
type SessionClient interface {
	// Session lifecycle
	CreateSession(ctx context.Context, req SessionRequest) (SessionHandle, error)
	SendPrompt(ctx context.Context, handle SessionHandle, prompt, model string) error
	SendMessageAsync(ctx context.Context, handle SessionHandle, content, model string) error
	SendMessageInDirectory(ctx context.Context, handle SessionHandle, content, model, directory string) error
	DeleteSession(ctx context.Context, handle SessionHandle) error

	// Session queries
	GetSession(ctx context.Context, handle SessionHandle) (*SessionInfo, error)
	GetSessionInDirectory(ctx context.Context, handle SessionHandle, directory string) (*SessionInfo, error)
	ListSessions(ctx context.Context, directory string) ([]SessionInfo, error)
	ListDiskSessions(ctx context.Context, directory string) ([]SessionInfo, error)
	GetMessages(ctx context.Context, handle SessionHandle) ([]Message, error)
	GetLastMessage(ctx context.Context, handle SessionHandle) (*Message, error)
	GetSessionTokens(ctx context.Context, handle SessionHandle) (*TokenStats, error)
	GetLastActivity(ctx context.Context, handle SessionHandle) (*LastActivity, error)

	// Session status
	GetSessionStatus(ctx context.Context, handle SessionHandle) (*SessionStatusInfo, error)
	GetSessionStatusByIDs(ctx context.Context, sessionIDs []string) (map[string]SessionStatusInfo, error)
	GetAllSessionStatus(ctx context.Context) (map[string]SessionStatusInfo, error)
	IsReachable(ctx context.Context) bool
	SessionExists(ctx context.Context, handle SessionHandle) bool
	IsSessionActive(ctx context.Context, handle SessionHandle, maxIdleTime time.Duration) bool
	IsSessionProcessing(ctx context.Context, handle SessionHandle) bool

	// Session metadata
	SetSessionMetadata(ctx context.Context, handle SessionHandle, metadata map[string]string) error

	// Blocking operations
	WaitForIdle(ctx context.Context, handle SessionHandle) error
	WaitForSessionError(ctx context.Context, handle SessionHandle, timeout time.Duration) (string, error)
	VerifySessionAfterPrompt(ctx context.Context, handle SessionHandle, directory string, timeout time.Duration) error
	SendMessageWithStreaming(ctx context.Context, handle SessionHandle, content string, streamTo io.Writer) error

	// Session discovery
	FindRecentSession(ctx context.Context, directory string) (SessionHandle, error)
	FindRecentSessionWithRetry(ctx context.Context, directory string, maxAttempts int, initialDelay time.Duration) (SessionHandle, error)

	// Transcript
	ExportTranscript(ctx context.Context, handle SessionHandle) (string, error)
}

// Package beads provides abstraction for beads issue tracking operations.
// The BeadsClient interface enables decoupling from the bd CLI, supporting
// both RPC daemon and CLI backends, and enabling easy mocking for tests.
package beads

// BeadsClient defines the interface for beads issue tracking operations.
// This abstraction allows for different implementations:
// - Client: RPC-based client connecting to the beads daemon
// - CLIClient: Shell-based client using bd CLI commands
// - MockClient: Mock implementation for testing
//
// Future: This interface could support an HTTP API backend when beads
// provides one.
type BeadsClient interface {
	// Ready retrieves issues that are ready for work.
	Ready(args *ReadyArgs) ([]Issue, error)

	// Show retrieves a single issue by ID.
	Show(id string) (*Issue, error)

	// List retrieves issues matching the given criteria.
	List(args *ListArgs) ([]Issue, error)

	// Stats retrieves beads statistics.
	Stats() (*Stats, error)

	// Comments retrieves comments for an issue.
	Comments(id string) ([]Comment, error)

	// AddComment adds a comment to an issue.
	// The author parameter is used by RPC client; CLI client ignores it.
	AddComment(id, author, text string) error

	// CloseIssue closes an issue with an optional reason.
	CloseIssue(id, reason string) error

	// Create creates a new issue.
	Create(args *CreateArgs) (*Issue, error)

	// Update updates an existing issue.
	Update(args *UpdateArgs) (*Issue, error)

	// AddLabel adds a label to an issue.
	AddLabel(id, label string) error

	// RemoveLabel removes a label from an issue.
	RemoveLabel(id, label string) error

	// ResolveID resolves a partial issue ID to a full ID.
	ResolveID(partialID string) (string, error)

	// FindByTitle finds an open issue with the exact given title.
	// Returns nil if no matching issue is found.
	// Used for deduplication before creating new issues.
	FindByTitle(title string) (*Issue, error)
}

// Ensure Client implements BeadsClient.
var _ BeadsClient = (*Client)(nil)

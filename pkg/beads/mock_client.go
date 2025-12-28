package beads

import (
	"fmt"
	"sync"
)

// MockClient implements BeadsClient for testing purposes.
// It stores issues and comments in memory and allows inspection of calls.
type MockClient struct {
	mu sync.RWMutex

	// Issues is the in-memory issue store, keyed by ID.
	Issues map[string]*Issue

	// CommentsStore is the in-memory comment store, keyed by issue ID.
	CommentsStore map[string][]Comment

	// Errors allows injecting errors for specific operations.
	// Key format: "<operation>" or "<operation>:<id>"
	// Examples: "ready", "show:issue-123", "comments:issue-456"
	Errors map[string]error

	// CallLog records all method calls for verification.
	CallLog []MockCall

	// nextCommentID tracks the next comment ID to assign.
	nextCommentID int
}

// MockCall represents a recorded method call.
type MockCall struct {
	Method string
	Args   []interface{}
}

// NewMockClient creates a new MockClient with empty stores.
func NewMockClient() *MockClient {
	return &MockClient{
		Issues:        make(map[string]*Issue),
		CommentsStore: make(map[string][]Comment),
		Errors:        make(map[string]error),
		CallLog:       nil,
		nextCommentID: 1,
	}
}

// recordCall records a method call for later verification.
func (m *MockClient) recordCall(method string, args ...interface{}) {
	m.CallLog = append(m.CallLog, MockCall{Method: method, Args: args})
}

// getError returns an injected error for the operation, if any.
func (m *MockClient) getError(operation string, id string) error {
	// Check for specific error first
	if id != "" {
		if err, ok := m.Errors[operation+":"+id]; ok {
			return err
		}
	}
	// Check for general error
	if err, ok := m.Errors[operation]; ok {
		return err
	}
	return nil
}

// Ready retrieves issues that are ready for work.
func (m *MockClient) Ready(args *ReadyArgs) ([]Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.recordCall("Ready", args)

	if err := m.getError("ready", ""); err != nil {
		return nil, err
	}

	var issues []Issue
	for _, issue := range m.Issues {
		// Filter by open/in_progress status (simplified ready logic)
		if issue.Status == "open" || issue.Status == "in_progress" {
			issues = append(issues, *issue)
		}
	}
	return issues, nil
}

// Show retrieves a single issue by ID.
func (m *MockClient) Show(id string) (*Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.recordCall("Show", id)

	if err := m.getError("show", id); err != nil {
		return nil, err
	}

	issue, ok := m.Issues[id]
	if !ok {
		return nil, fmt.Errorf("issue not found: %s", id)
	}
	return issue, nil
}

// List retrieves issues matching the given criteria.
func (m *MockClient) List(args *ListArgs) ([]Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.recordCall("List", args)

	if err := m.getError("list", ""); err != nil {
		return nil, err
	}

	var issues []Issue
	for _, issue := range m.Issues {
		// Apply status filter if provided
		if args != nil && args.Status != "" && issue.Status != args.Status {
			continue
		}
		issues = append(issues, *issue)
	}
	return issues, nil
}

// Stats retrieves beads statistics.
func (m *MockClient) Stats() (*Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.recordCall("Stats")

	if err := m.getError("stats", ""); err != nil {
		return nil, err
	}

	// Calculate stats from stored issues
	var total, open, closed, inProgress int
	for _, issue := range m.Issues {
		total++
		switch issue.Status {
		case "open":
			open++
		case "closed":
			closed++
		case "in_progress":
			inProgress++
		}
	}

	return &Stats{
		Summary: StatsSummary{
			TotalIssues:      total,
			OpenIssues:       open,
			ClosedIssues:     closed,
			InProgressIssues: inProgress,
		},
	}, nil
}

// Comments retrieves comments for an issue.
func (m *MockClient) Comments(id string) ([]Comment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.recordCall("Comments", id)

	if err := m.getError("comments", id); err != nil {
		return nil, err
	}

	comments, ok := m.CommentsStore[id]
	if !ok {
		return []Comment{}, nil
	}
	return comments, nil
}

// AddComment adds a comment to an issue.
func (m *MockClient) AddComment(id, author, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("AddComment", id, author, text)

	if err := m.getError("addcomment", id); err != nil {
		return err
	}

	comment := Comment{
		ID:      m.nextCommentID,
		IssueID: id,
		Author:  author,
		Text:    text,
	}
	m.nextCommentID++

	m.CommentsStore[id] = append(m.CommentsStore[id], comment)
	return nil
}

// CloseIssue closes an issue with an optional reason.
func (m *MockClient) CloseIssue(id, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("CloseIssue", id, reason)

	if err := m.getError("close", id); err != nil {
		return err
	}

	issue, ok := m.Issues[id]
	if !ok {
		return fmt.Errorf("issue not found: %s", id)
	}

	issue.Status = "closed"
	issue.CloseReason = reason
	return nil
}

// Create creates a new issue.
func (m *MockClient) Create(args *CreateArgs) (*Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("Create", args)

	if err := m.getError("create", ""); err != nil {
		return nil, err
	}

	if args == nil {
		return nil, fmt.Errorf("create args required")
	}

	// Generate ID if not provided
	id := args.ID
	if id == "" {
		id = fmt.Sprintf("mock-%d", len(m.Issues)+1)
	}

	issue := &Issue{
		ID:          id,
		Title:       args.Title,
		Description: args.Description,
		IssueType:   args.IssueType,
		Priority:    args.Priority,
		Labels:      args.Labels,
		Status:      "open",
	}

	m.Issues[id] = issue
	return issue, nil
}

// Update updates an existing issue.
func (m *MockClient) Update(args *UpdateArgs) (*Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("Update", args)

	if err := m.getError("update", args.ID); err != nil {
		return nil, err
	}

	if args == nil {
		return nil, fmt.Errorf("update args required")
	}

	issue, ok := m.Issues[args.ID]
	if !ok {
		return nil, fmt.Errorf("issue not found: %s", args.ID)
	}

	if args.Title != nil {
		issue.Title = *args.Title
	}
	if args.Description != nil {
		issue.Description = *args.Description
	}
	if args.Status != nil {
		issue.Status = *args.Status
	}
	if args.Priority != nil {
		issue.Priority = *args.Priority
	}

	// Handle label operations
	for _, label := range args.RemoveLabels {
		for i, l := range issue.Labels {
			if l == label {
				issue.Labels = append(issue.Labels[:i], issue.Labels[i+1:]...)
				break
			}
		}
	}
	issue.Labels = append(issue.Labels, args.AddLabels...)

	return issue, nil
}

// AddLabel adds a label to an issue.
func (m *MockClient) AddLabel(id, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("AddLabel", id, label)

	if err := m.getError("addlabel", id); err != nil {
		return err
	}

	issue, ok := m.Issues[id]
	if !ok {
		return fmt.Errorf("issue not found: %s", id)
	}

	issue.Labels = append(issue.Labels, label)
	return nil
}

// RemoveLabel removes a label from an issue.
func (m *MockClient) RemoveLabel(id, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("RemoveLabel", id, label)

	if err := m.getError("removelabel", id); err != nil {
		return err
	}

	issue, ok := m.Issues[id]
	if !ok {
		return fmt.Errorf("issue not found: %s", id)
	}

	for i, l := range issue.Labels {
		if l == label {
			issue.Labels = append(issue.Labels[:i], issue.Labels[i+1:]...)
			break
		}
	}
	return nil
}

// ResolveID resolves a partial issue ID to a full ID.
func (m *MockClient) ResolveID(partialID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.recordCall("ResolveID", partialID)

	if err := m.getError("resolveid", partialID); err != nil {
		return "", err
	}

	// First try exact match
	if _, ok := m.Issues[partialID]; ok {
		return partialID, nil
	}

	// Try prefix match
	var matches []string
	for id := range m.Issues {
		if len(id) >= len(partialID) && id[:len(partialID)] == partialID {
			matches = append(matches, id)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no issue found matching: %s", partialID)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous ID: %s matches %d issues", partialID, len(matches))
	}

	return matches[0], nil
}

// Reset clears all stored data and call logs.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Issues = make(map[string]*Issue)
	m.CommentsStore = make(map[string][]Comment)
	m.Errors = make(map[string]error)
	m.CallLog = nil
	m.nextCommentID = 1
}

// GetCalls returns all recorded calls for a specific method.
func (m *MockClient) GetCalls(method string) []MockCall {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var calls []MockCall
	for _, call := range m.CallLog {
		if call.Method == method {
			calls = append(calls, call)
		}
	}
	return calls
}

// Ensure MockClient implements BeadsClient.
var _ BeadsClient = (*MockClient)(nil)

package main

import (
	"sync"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

type countingBeadsClient struct {
	mu sync.Mutex

	listCalls  map[string]int
	readyCalls map[string]int
	statsCalls int
}

func newCountingBeadsClient() *countingBeadsClient {
	return &countingBeadsClient{
		listCalls:  make(map[string]int),
		readyCalls: make(map[string]int),
	}
}

func (m *countingBeadsClient) Ready(args *beads.ReadyArgs) ([]beads.Issue, error) {
	key := cacheKey(args)

	m.mu.Lock()
	m.readyCalls[key]++
	m.mu.Unlock()

	return []beads.Issue{{ID: "orch-go-ready", Title: "Ready issue", Priority: 1, Status: "open", IssueType: "task"}}, nil
}

func (m *countingBeadsClient) Show(id string) (*beads.Issue, error) {
	return &beads.Issue{ID: id}, nil
}

func (m *countingBeadsClient) List(args *beads.ListArgs) ([]beads.Issue, error) {
	key := cacheKey(args)

	m.mu.Lock()
	m.listCalls[key]++
	m.mu.Unlock()

	status := ""
	if args != nil {
		status = args.Status
	}

	return []beads.Issue{{
		ID:        "orch-go-" + status,
		Title:     "Issue " + status,
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	}}, nil
}

func (m *countingBeadsClient) Stats() (*beads.Stats, error) {
	m.mu.Lock()
	m.statsCalls++
	m.mu.Unlock()

	return &beads.Stats{}, nil
}

func (m *countingBeadsClient) Comments(id string) ([]beads.Comment, error) {
	return nil, nil
}

func (m *countingBeadsClient) AddComment(id, author, text string) error {
	return nil
}

func (m *countingBeadsClient) CloseIssue(id, reason string) error {
	return nil
}

func (m *countingBeadsClient) Create(args *beads.CreateArgs) (*beads.Issue, error) {
	return nil, nil
}

func (m *countingBeadsClient) Update(args *beads.UpdateArgs) (*beads.Issue, error) {
	return nil, nil
}

func (m *countingBeadsClient) AddLabel(id, label string) error {
	return nil
}

func (m *countingBeadsClient) RemoveLabel(id, label string) error {
	return nil
}

func (m *countingBeadsClient) ResolveID(partialID string) (string, error) {
	return partialID, nil
}

func (m *countingBeadsClient) listCallCount(args *beads.ListArgs) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listCalls[cacheKey(args)]
}

func (m *countingBeadsClient) readyCallCount(args *beads.ReadyArgs) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readyCalls[cacheKey(args)]
}

func TestRequestScopedBeadsClient_CachesListPerArgs(t *testing.T) {
	base := newCountingBeadsClient()
	client := newRequestScopedBeadsClient(base)

	openArgs := &beads.ListArgs{Status: "open", Limit: 0}
	for i := 0; i < 5; i++ {
		if _, err := client.List(openArgs); err != nil {
			t.Fatalf("List(open) failed: %v", err)
		}
	}

	if got := base.listCallCount(openArgs); got != 1 {
		t.Fatalf("expected 1 underlying List(open) call, got %d", got)
	}

	inProgressArgs := &beads.ListArgs{Status: "open,in_progress", Limit: 0}
	if _, err := client.List(inProgressArgs); err != nil {
		t.Fatalf("List(open,in_progress) failed: %v", err)
	}
	if got := base.listCallCount(inProgressArgs); got != 1 {
		t.Fatalf("expected 1 underlying List(open,in_progress) call, got %d", got)
	}
}

func TestRequestScopedBeadsClient_CachesReadyAndStats(t *testing.T) {
	base := newCountingBeadsClient()
	client := newRequestScopedBeadsClient(base)

	readyArgs := &beads.ReadyArgs{Limit: 0}
	for i := 0; i < 4; i++ {
		if _, err := client.Ready(readyArgs); err != nil {
			t.Fatalf("Ready failed: %v", err)
		}
		if _, err := client.Stats(); err != nil {
			t.Fatalf("Stats failed: %v", err)
		}
	}

	if got := base.readyCallCount(readyArgs); got != 1 {
		t.Fatalf("expected 1 underlying Ready call, got %d", got)
	}

	base.mu.Lock()
	statsCalls := base.statsCalls
	base.mu.Unlock()
	if statsCalls != 1 {
		t.Fatalf("expected 1 underlying Stats call, got %d", statsCalls)
	}
}

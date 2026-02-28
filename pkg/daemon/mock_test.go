package daemon

import (
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// mockIssueQuerier implements IssueQuerier for tests.
// Each method delegates to a function field if set, otherwise returns zero values.
type mockIssueQuerier struct {
	ListReadyIssuesFunc     func() ([]Issue, error)
	GetIssueStatusFunc      func(beadsID string) (string, error)
	ListEpicChildrenFunc    func(epicID string) ([]Issue, error)
	ListIssuesWithLabelFunc func(label string) ([]Issue, error)
	CreateExtractionIssueFunc func(task, parentIssueID string) (string, error)
}

func (m *mockIssueQuerier) ListReadyIssues() ([]Issue, error) {
	if m.ListReadyIssuesFunc != nil {
		return m.ListReadyIssuesFunc()
	}
	return nil, nil
}

func (m *mockIssueQuerier) GetIssueStatus(beadsID string) (string, error) {
	if m.GetIssueStatusFunc != nil {
		return m.GetIssueStatusFunc(beadsID)
	}
	// Default to "open" so tests that don't set GetIssueStatusFunc
	// pass the fresh-status-check dedup gate in spawnIssue().
	return "open", nil
}

func (m *mockIssueQuerier) ListEpicChildren(epicID string) ([]Issue, error) {
	if m.ListEpicChildrenFunc != nil {
		return m.ListEpicChildrenFunc(epicID)
	}
	return nil, nil
}

func (m *mockIssueQuerier) ListIssuesWithLabel(label string) ([]Issue, error) {
	if m.ListIssuesWithLabelFunc != nil {
		return m.ListIssuesWithLabelFunc(label)
	}
	return nil, nil
}

func (m *mockIssueQuerier) CreateExtractionIssue(task, parentIssueID string) (string, error) {
	if m.CreateExtractionIssueFunc != nil {
		return m.CreateExtractionIssueFunc(task, parentIssueID)
	}
	return "", nil
}

// mockIssueUpdater implements IssueUpdater for tests.
type mockIssueUpdater struct {
	UpdateStatusFunc func(beadsID, status string) error
}

func (m *mockIssueUpdater) UpdateStatus(beadsID, status string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(beadsID, status)
	}
	return nil
}

// mockSpawner implements Spawner for tests.
type mockSpawner struct {
	SpawnWorkFunc func(beadsID, model, workdir string) error
}

func (m *mockSpawner) SpawnWork(beadsID, model, workdir string) error {
	if m.SpawnWorkFunc != nil {
		return m.SpawnWorkFunc(beadsID, model, workdir)
	}
	return nil
}

// mockCompletionFinder implements CompletionFinder for tests.
type mockCompletionFinder struct {
	ListCompletedAgentsFunc func(config CompletionConfig) ([]CompletedAgent, error)
}

func (m *mockCompletionFinder) ListCompletedAgents(config CompletionConfig) ([]CompletedAgent, error) {
	if m.ListCompletedAgentsFunc != nil {
		return m.ListCompletedAgentsFunc(config)
	}
	return nil, nil
}

// mockReflector implements Reflector for tests.
type mockReflector struct {
	ReflectFunc     func(createIssues bool) (*ReflectResult, error)
	ReflectOpenFunc func() error
}

func (m *mockReflector) Reflect(createIssues bool) (*ReflectResult, error) {
	if m.ReflectFunc != nil {
		return m.ReflectFunc(createIssues)
	}
	return &ReflectResult{}, nil
}

func (m *mockReflector) ReflectOpen() error {
	if m.ReflectOpenFunc != nil {
		return m.ReflectOpenFunc()
	}
	return nil
}

// mockModelDriftStore implements ModelDriftStore for tests.
type mockModelDriftStore struct {
	ReadStalenessEventsFunc func(path string) ([]spawn.StalenessEvent, error)
	LoadMetadataFunc        func(modelPath string) (ModelDriftMetadata, error)
	CountCommitsFunc        func(projectDir, lastUpdated string, files []string) (int, error)
	CreateIssueFunc         func(args ModelDriftIssueCreateArgs) (string, error)
}

func (m *mockModelDriftStore) ReadStalenessEvents(path string) ([]spawn.StalenessEvent, error) {
	if m.ReadStalenessEventsFunc != nil {
		return m.ReadStalenessEventsFunc(path)
	}
	return nil, nil
}

func (m *mockModelDriftStore) LoadMetadata(modelPath string) (ModelDriftMetadata, error) {
	if m.LoadMetadataFunc != nil {
		return m.LoadMetadataFunc(modelPath)
	}
	return ModelDriftMetadata{}, nil
}

func (m *mockModelDriftStore) CountCommits(projectDir, lastUpdated string, files []string) (int, error) {
	if m.CountCommitsFunc != nil {
		return m.CountCommitsFunc(projectDir, lastUpdated, files)
	}
	return 0, nil
}

func (m *mockModelDriftStore) CreateIssue(args ModelDriftIssueCreateArgs) (string, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(args)
	}
	return "", nil
}

// mockKnowledgeHealthService implements KnowledgeHealthService for tests.
type mockKnowledgeHealthService struct {
	CheckFunc       func() (*KnowledgeHealthResult, error)
	CreateIssueFunc func(result *KnowledgeHealthResult) error
}

func (m *mockKnowledgeHealthService) Check() (*KnowledgeHealthResult, error) {
	if m.CheckFunc != nil {
		return m.CheckFunc()
	}
	return &KnowledgeHealthResult{}, nil
}

func (m *mockKnowledgeHealthService) CreateIssue(result *KnowledgeHealthResult) error {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(result)
	}
	return nil
}

// mockSessionCleaner implements SessionCleaner for tests.
type mockSessionCleaner struct {
	CleanupFunc func(config Config) (int, string, error)
}

func (m *mockSessionCleaner) Cleanup(config Config) (int, string, error) {
	if m.CleanupFunc != nil {
		return m.CleanupFunc(config)
	}
	return 0, "", nil
}

// mockActiveCounter implements ActiveCounter for tests.
type mockActiveCounter struct {
	CountFunc func() int
}

func (m *mockActiveCounter) Count() int {
	if m.CountFunc != nil {
		return m.CountFunc()
	}
	return 0
}

// mockAgentDiscoverer implements AgentDiscoverer for tests.
type mockAgentDiscoverer struct {
	GetActiveAgentsFunc    func() ([]ActiveAgent, error)
	HasExistingSessionFunc func(beadsID string) bool
}

func (m *mockAgentDiscoverer) GetActiveAgents() ([]ActiveAgent, error) {
	if m.GetActiveAgentsFunc != nil {
		return m.GetActiveAgentsFunc()
	}
	return nil, nil
}

func (m *mockAgentDiscoverer) HasExistingSession(beadsID string) bool {
	if m.HasExistingSessionFunc != nil {
		return m.HasExistingSessionFunc(beadsID)
	}
	return false
}

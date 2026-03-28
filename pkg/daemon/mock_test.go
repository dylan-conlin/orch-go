package daemon

import (
	"github.com/dylan-conlin/orch-go/pkg/artifactsync"
)

// Test helpers

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockIssueQuerier implements IssueQuerier for tests.
// Each method delegates to a function field if set, otherwise returns zero values.
type mockIssueQuerier struct {
	ListReadyIssuesFunc       func() ([]Issue, error)
	GetIssueStatusFunc        func(beadsID string) (string, error)
	ListEpicChildrenFunc      func(epicID string) ([]Issue, error)
	ListIssuesWithLabelFunc   func(label string) ([]Issue, error)
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
	SpawnWorkFunc func(beadsID, skill, model, workdir, account string) error
}

func (m *mockSpawner) SpawnWork(beadsID, skill, model, workdir, account string) error {
	if m.SpawnWorkFunc != nil {
		return m.SpawnWorkFunc(beadsID, skill, model, workdir, account)
	}
	return nil
}

// mockBoundaryTransitioner implements BoundaryTransitioner for tests.
type mockBoundaryTransitioner struct {
	TransitionFunc func(beadsID, feedback, workdir string) error
}

func (m *mockBoundaryTransitioner) Transition(beadsID, feedback, workdir string) error {
	if m.TransitionFunc != nil {
		return m.TransitionFunc(beadsID, feedback, workdir)
	}
	return nil
}

// mockWorkspaceVerifier implements WorkspaceVerifier for tests.
type mockWorkspaceVerifier struct {
	ExistsFunc func(beadsID, projectDir string) bool
}

func (m *mockWorkspaceVerifier) Exists(beadsID, projectDir string) bool {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(beadsID, projectDir)
	}
	return true // default: workspace exists (backward compatible)
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

// mockAgreementCheckService implements AgreementCheckService for tests.
type mockAgreementCheckService struct {
	CheckFunc        func() (*AgreementCheckResult, error)
	CreateIssueFunc  func(failure AgreementFailureDetail) error
	HasOpenIssueFunc func(agreementID string) (bool, error)
}

func (m *mockAgreementCheckService) Check() (*AgreementCheckResult, error) {
	if m.CheckFunc != nil {
		return m.CheckFunc()
	}
	return &AgreementCheckResult{}, nil
}

func (m *mockAgreementCheckService) CreateIssue(failure AgreementFailureDetail) error {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(failure)
	}
	return nil
}

func (m *mockAgreementCheckService) HasOpenIssue(agreementID string) (bool, error) {
	if m.HasOpenIssueFunc != nil {
		return m.HasOpenIssueFunc(agreementID)
	}
	return false, nil
}

// mockArtifactSyncService implements ArtifactSyncService for tests.
type mockArtifactSyncService struct {
	AnalyzeFunc                   func(projectDir string) (*ArtifactSyncResult, error)
	HasOpenIssueFunc              func() (bool, error)
	CreateIssueFunc               func(report *artifactsync.DriftReport) (string, error)
	SpawnSyncAgentFunc            func(report *artifactsync.DriftReport) error
	SpawnBudgetAwareSyncAgentFunc func(report *artifactsync.DriftReport, currentLines, budget int) error
	CLAUDEMDLineCountFunc         func(projectDir string) (int, error)
}

func (m *mockArtifactSyncService) Analyze(projectDir string) (*ArtifactSyncResult, error) {
	if m.AnalyzeFunc != nil {
		return m.AnalyzeFunc(projectDir)
	}
	return &ArtifactSyncResult{}, nil
}

func (m *mockArtifactSyncService) HasOpenIssue() (bool, error) {
	if m.HasOpenIssueFunc != nil {
		return m.HasOpenIssueFunc()
	}
	return false, nil
}

func (m *mockArtifactSyncService) CreateIssue(report *artifactsync.DriftReport) (string, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(report)
	}
	return "", nil
}

func (m *mockArtifactSyncService) SpawnSyncAgent(report *artifactsync.DriftReport) error {
	if m.SpawnSyncAgentFunc != nil {
		return m.SpawnSyncAgentFunc(report)
	}
	return nil
}

func (m *mockArtifactSyncService) SpawnBudgetAwareSyncAgent(report *artifactsync.DriftReport, currentLines, budget int) error {
	if m.SpawnBudgetAwareSyncAgentFunc != nil {
		return m.SpawnBudgetAwareSyncAgentFunc(report, currentLines, budget)
	}
	return nil
}

func (m *mockArtifactSyncService) CLAUDEMDLineCount(projectDir string) (int, error) {
	if m.CLAUDEMDLineCountFunc != nil {
		return m.CLAUDEMDLineCountFunc(projectDir)
	}
	return 0, nil
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
	GetActiveAgentsFunc           func() ([]ActiveAgent, error)
	HasExistingSessionFunc        func(beadsID string) bool
	HasExistingSessionOrErrorFunc func(beadsID string) (bool, error)
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

func (m *mockAgentDiscoverer) HasExistingSessionOrError(beadsID string) (bool, error) {
	if m.HasExistingSessionOrErrorFunc != nil {
		return m.HasExistingSessionOrErrorFunc(beadsID)
	}
	// Default: delegate to the bool-only version (no error)
	return m.HasExistingSession(beadsID), nil
}

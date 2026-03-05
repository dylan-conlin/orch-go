// Package daemon provides autonomous overnight processing capabilities.
// This file defines the dependency interfaces used by the Daemon struct.
// Each interface replaces one or more mock function fields with a typed contract.
package daemon


// IssueQuerier reads beads issues for the daemon spawn pipeline.
type IssueQuerier interface {
	ListReadyIssues() ([]Issue, error)
	GetIssueStatus(beadsID string) (string, error)
	ListEpicChildren(epicID string) ([]Issue, error)
	ListIssuesWithLabel(label string) ([]Issue, error)
	CreateExtractionIssue(task, parentIssueID string) (string, error)
}

// IssueUpdater updates beads issue status.
type IssueUpdater interface {
	UpdateStatus(beadsID, status string) error
}

// Spawner spawns agent work.
type Spawner interface {
	SpawnWork(beadsID, model, workdir string) error
}

// CompletionFinder finds completed agents.
type CompletionFinder interface {
	ListCompletedAgents(config CompletionConfig) ([]CompletedAgent, error)
}

// Reflector runs knowledge reflection.
type Reflector interface {
	Reflect(createIssues bool) (*ReflectResult, error)
	ReflectOpen() error
}

// KnowledgeHealthService provides knowledge health operations.
type KnowledgeHealthService interface {
	Check() (*KnowledgeHealthResult, error)
	CreateIssue(result *KnowledgeHealthResult) error
}

// SessionCleaner cleans up stale sessions.
type SessionCleaner interface {
	Cleanup(config Config) (int, string, error)
}

// ActiveCounter counts active agents.
type ActiveCounter interface {
	Count() int
}

// AgentDiscoverer discovers agents for orphan detection and recovery.
type AgentDiscoverer interface {
	GetActiveAgents() ([]ActiveAgent, error)
	HasExistingSession(beadsID string) bool
	// HasExistingSessionOrError is the error-aware version of HasExistingSession.
	// Returns (found, nil) on success, or (false, err) when session checks fail.
	// Used by orphan detector to fail-closed on infrastructure errors.
	HasExistingSessionOrError(beadsID string) (bool, error)
}

// --- Default implementations ---

// defaultIssueQuerier is the production IssueQuerier backed by beads CLI.
type defaultIssueQuerier struct {
	// registry is consulted for multi-project issue listing.
	// When non-nil, ListReadyIssues uses ListReadyIssuesMultiProject.
	registry *ProjectRegistry
}

func (q *defaultIssueQuerier) ListReadyIssues() ([]Issue, error) {
	if q.registry != nil {
		return ListReadyIssuesMultiProject(q.registry)
	}
	return ListReadyIssues()
}

func (q *defaultIssueQuerier) GetIssueStatus(beadsID string) (string, error) {
	return GetBeadsIssueStatus(beadsID)
}

func (q *defaultIssueQuerier) ListEpicChildren(epicID string) ([]Issue, error) {
	return ListEpicChildren(epicID)
}

func (q *defaultIssueQuerier) ListIssuesWithLabel(label string) ([]Issue, error) {
	return ListIssuesWithLabel(label)
}

func (q *defaultIssueQuerier) CreateExtractionIssue(task, parentIssueID string) (string, error) {
	return DefaultCreateExtractionIssue(task, parentIssueID)
}

// defaultIssueUpdater is the production IssueUpdater backed by beads CLI.
type defaultIssueUpdater struct{}

func (u *defaultIssueUpdater) UpdateStatus(beadsID, status string) error {
	return UpdateBeadsStatus(beadsID, status)
}

// defaultSpawner is the production Spawner.
type defaultSpawner struct{}

func (s *defaultSpawner) SpawnWork(beadsID, model, workdir string) error {
	return SpawnWork(beadsID, model, workdir)
}

// defaultCompletionFinder is the production CompletionFinder.
// When registry is set, it populates config.ProjectDirs for cross-project scanning.
type defaultCompletionFinder struct {
	registry *ProjectRegistry
}

func (f *defaultCompletionFinder) ListCompletedAgents(config CompletionConfig) ([]CompletedAgent, error) {
	// If registry is available and ProjectDirs not already set, populate from registry
	if f.registry != nil && len(config.ProjectDirs) == 0 {
		for _, entry := range f.registry.Projects() {
			config.ProjectDirs = append(config.ProjectDirs, entry.Dir)
		}
	}
	return ListCompletedAgentsDefault(config)
}

// defaultReflector is the production Reflector.
type defaultReflector struct{}

func (r *defaultReflector) Reflect(createIssues bool) (*ReflectResult, error) {
	return DefaultRunReflection(createIssues)
}

func (r *defaultReflector) ReflectOpen() error {
	return RunOpenReflection()
}

// defaultKnowledgeHealthService is the production KnowledgeHealthService.
type defaultKnowledgeHealthService struct{}

func (s *defaultKnowledgeHealthService) Check() (*KnowledgeHealthResult, error) {
	return DefaultKnowledgeHealthCheck()
}

func (s *defaultKnowledgeHealthService) CreateIssue(result *KnowledgeHealthResult) error {
	return DefaultCreateKnowledgeHealthIssue(result)
}

// defaultAgreementCheckService is the production AgreementCheckService.
type defaultAgreementCheckService struct{}

func (s *defaultAgreementCheckService) Check() (*AgreementCheckResult, error) {
	return DefaultAgreementCheck()
}

func (s *defaultAgreementCheckService) CreateIssue(failure AgreementFailureDetail) error {
	return DefaultCreateAgreementIssue(failure)
}

func (s *defaultAgreementCheckService) HasOpenIssue(agreementID string) (bool, error) {
	return DefaultHasOpenAgreementIssue(agreementID)
}

// defaultSessionCleaner is the production SessionCleaner.
type defaultSessionCleaner struct{}

func (c *defaultSessionCleaner) Cleanup(config Config) (int, string, error) {
	return defaultCleanup(config)
}

// defaultActiveCounter is the production ActiveCounter.
type defaultActiveCounter struct{}

func (c *defaultActiveCounter) Count() int {
	return BeadsActiveCount()
}

// defaultAgentDiscoverer is the production AgentDiscoverer.
type defaultAgentDiscoverer struct{}

func (d *defaultAgentDiscoverer) GetActiveAgents() ([]ActiveAgent, error) {
	return GetActiveAgents()
}

func (d *defaultAgentDiscoverer) HasExistingSession(beadsID string) bool {
	return HasExistingSessionForBeadsID(beadsID)
}

func (d *defaultAgentDiscoverer) HasExistingSessionOrError(beadsID string) (bool, error) {
	return HasExistingSessionForBeadsIDWithError(beadsID)
}

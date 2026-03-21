package daemon

import "time"

// Task name constants for the periodic scheduler.
const (
	TaskReflect              = "reflect"
	TaskModelDriftReflect    = "model_drift_reflect"
	TaskKnowledgeHealth      = "knowledge_health"
	TaskCleanup              = "cleanup"
	TaskRecovery             = "recovery"
	TaskOrphanDetection      = "orphan_detection"
	TaskPhaseTimeout         = "phase_timeout"
	TaskQuestionDetection    = "question_detection"
	TaskAgreementCheck       = "agreement_check"
	TaskBeadsHealth          = "beads_health"
	TaskFrictionAccumulation = "friction_accumulation"
	TaskArtifactSync         = "artifact_sync"
	TaskRegistryRefresh      = "registry_refresh"
	TaskSynthesisAutoCreate  = "synthesis_auto_create"
	TaskLearningRefresh      = "learning_refresh"
	TaskPlanStaleness        = "plan_staleness"
	TaskProactiveExtraction  = "proactive_extraction"
	TaskAccretionResponse    = "accretion_response"
	TaskTriggerScan          = "trigger_scan"
	TaskTriggerExpiry        = "trigger_expiry"
	TaskDigest               = "digest"
	TaskInvestigationOrphan             = "investigation_orphan"
	TaskVerificationFailedEscalation   = "verification_failed_escalation"
	TaskLightweightCleanup             = "lightweight_cleanup"
	TaskClaimProbeGeneration           = "claim_probe_generation"
	TaskTensionClusterScan             = "tension_cluster_scan"
	TaskCapacityPoll                   = "capacity_poll"
	TaskAuditSelect                    = "audit_select"
)

// periodicTask holds the scheduling state for a single periodic task.
type periodicTask struct {
	enabled  bool
	interval time.Duration
	lastRun  time.Time
}

// PeriodicScheduler manages timing for multiple named periodic tasks.
// It replaces the pattern of individual last* fields and ShouldRun*/Last*Time/Next*Time
// methods that were previously duplicated for each periodic task on the Daemon struct.
type PeriodicScheduler struct {
	tasks map[string]*periodicTask
}

// NewPeriodicScheduler creates a new scheduler with no tasks registered.
func NewPeriodicScheduler() *PeriodicScheduler {
	return &PeriodicScheduler{
		tasks: make(map[string]*periodicTask),
	}
}

// Register adds a named periodic task with the given enabled state and interval.
// If the task already exists, its configuration is updated. No-op if scheduler is nil.
func (s *PeriodicScheduler) Register(name string, enabled bool, interval time.Duration) {
	if s == nil {
		return
	}
	s.tasks[name] = &periodicTask{
		enabled:  enabled,
		interval: interval,
	}
}

// IsDue returns true if the named task should run now.
// Returns false if the scheduler is nil, the task is unregistered, disabled,
// has zero interval, or hasn't waited long enough since its last run.
func (s *PeriodicScheduler) IsDue(name string) bool {
	if s == nil {
		return false
	}
	task, ok := s.tasks[name]
	if !ok {
		return false
	}
	if !task.enabled || task.interval <= 0 {
		return false
	}
	if task.lastRun.IsZero() {
		return true
	}
	return time.Since(task.lastRun) >= task.interval
}

// MarkRun records that the named task just completed successfully.
// No-op if the scheduler is nil or the task is not registered.
func (s *PeriodicScheduler) MarkRun(name string) {
	if s == nil {
		return
	}
	if task, ok := s.tasks[name]; ok {
		task.lastRun = time.Now()
	}
}

// SetLastRun sets the last run time for a named task to a specific time.
// Useful for tests and for restoring state. No-op if scheduler is nil or task unregistered.
func (s *PeriodicScheduler) SetLastRun(name string, t time.Time) {
	if s == nil {
		return
	}
	if task, ok := s.tasks[name]; ok {
		task.lastRun = t
	}
}

// LastRunTime returns when the named task was last run.
// Returns zero time if the scheduler is nil, the task is unregistered, or has never run.
func (s *PeriodicScheduler) LastRunTime(name string) time.Time {
	if s == nil {
		return time.Time{}
	}
	task, ok := s.tasks[name]
	if !ok {
		return time.Time{}
	}
	return task.lastRun
}

// NewSchedulerFromConfig creates a PeriodicScheduler with all daemon tasks
// registered from the given config.
func NewSchedulerFromConfig(cfg Config) *PeriodicScheduler {
	s := NewPeriodicScheduler()
	s.Register(TaskReflect, cfg.ReflectEnabled, cfg.ReflectInterval)
	s.Register(TaskModelDriftReflect, cfg.ReflectModelDriftEnabled, cfg.ReflectModelDriftInterval)
	s.Register(TaskKnowledgeHealth, cfg.KnowledgeHealthEnabled, cfg.KnowledgeHealthInterval)
	s.Register(TaskCleanup, cfg.CleanupEnabled, cfg.CleanupInterval)
	s.Register(TaskRecovery, cfg.RecoveryEnabled, cfg.RecoveryInterval)
	s.Register(TaskOrphanDetection, cfg.OrphanDetectionEnabled, cfg.OrphanDetectionInterval)
	s.Register(TaskPhaseTimeout, cfg.PhaseTimeoutEnabled, cfg.PhaseTimeoutInterval)
	s.Register(TaskQuestionDetection, cfg.PhaseTimeoutEnabled, cfg.PhaseTimeoutInterval) // shares config with phase timeout
	s.Register(TaskAgreementCheck, cfg.AgreementCheckEnabled, cfg.AgreementCheckInterval)
	s.Register(TaskBeadsHealth, cfg.BeadsHealthEnabled, cfg.BeadsHealthInterval)
	s.Register(TaskFrictionAccumulation, cfg.FrictionAccumulationEnabled, cfg.FrictionAccumulationInterval)
	s.Register(TaskArtifactSync, cfg.ArtifactSyncEnabled, cfg.ArtifactSyncInterval)
	s.Register(TaskRegistryRefresh, cfg.RegistryRefreshEnabled, cfg.RegistryRefreshInterval)
	s.Register(TaskSynthesisAutoCreate, cfg.SynthesisAutoCreateEnabled, cfg.SynthesisAutoCreateInterval)
	s.Register(TaskLearningRefresh, cfg.LearningRefreshEnabled, cfg.LearningRefreshInterval)
	s.Register(TaskPlanStaleness, cfg.PlanStalenessEnabled, cfg.PlanStalenessInterval)
	s.Register(TaskProactiveExtraction, cfg.ProactiveExtractionEnabled, cfg.ProactiveExtractionInterval)
	s.Register(TaskAccretionResponse, cfg.AccretionResponseEnabled, cfg.AccretionResponseInterval)
	s.Register(TaskTriggerScan, cfg.TriggerScanEnabled, cfg.TriggerScanInterval)
	s.Register(TaskTriggerExpiry, cfg.TriggerExpiryEnabled, cfg.TriggerExpiryInterval)
	s.Register(TaskDigest, cfg.DigestEnabled, cfg.DigestInterval)
	s.Register(TaskInvestigationOrphan, cfg.InvestigationOrphanEnabled, cfg.InvestigationOrphanInterval)
	s.Register(TaskVerificationFailedEscalation, cfg.VerificationFailedEscalationEnabled, cfg.VerificationFailedEscalationInterval)
	s.Register(TaskLightweightCleanup, cfg.LightweightCleanupEnabled, cfg.LightweightCleanupInterval)
	s.Register(TaskClaimProbeGeneration, cfg.ClaimProbeGenerationEnabled, cfg.ClaimProbeGenerationInterval)
	s.Register(TaskTensionClusterScan, cfg.TensionClusterScanEnabled, cfg.TensionClusterScanInterval)
	s.Register(TaskCapacityPoll, cfg.CapacityPollEnabled, cfg.CapacityPollInterval)
	s.Register(TaskAuditSelect, cfg.AuditSelectEnabled, cfg.AuditSelectInterval)
	return s
}

// NextRunTime returns when the named task is next scheduled to run.
// Returns zero time if the scheduler is nil, the task is unregistered, disabled, or has zero interval.
func (s *PeriodicScheduler) NextRunTime(name string) time.Time {
	if s == nil {
		return time.Time{}
	}
	task, ok := s.tasks[name]
	if !ok {
		return time.Time{}
	}
	if !task.enabled || task.interval <= 0 {
		return time.Time{}
	}
	if task.lastRun.IsZero() {
		return time.Now()
	}
	return task.lastRun.Add(task.interval)
}

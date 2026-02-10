package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/account"
)

func saveSpawnPipelineGlobals() func() {
	oldSpawnBypassTriage := spawnBypassTriage
	oldSpawnMaxAgents := spawnMaxAgents
	oldSpawnModel := spawnModel
	oldSpawnWorkdir := spawnWorkdir
	oldSpawnForce := spawnForce
	oldSpawnAcknowledgeHotspot := spawnAcknowledgeHotspot
	oldSpawnAutoInit := spawnAutoInit
	oldSpawnNoTrack := spawnNoTrack
	oldSpawnIssue := spawnIssue
	oldSpawnSkipArtifactCheck := spawnSkipArtifactCheck
	oldSpawnLight := spawnLight
	oldSpawnFull := spawnFull
	oldSpawnValidation := spawnValidation
	oldSpawnBackendFlag := spawnBackendFlag
	oldSpawnAccount := spawnAccount
	oldSpawnOpus := spawnOpus
	oldSpawnInfra := spawnInfra
	oldSpawnVariant := spawnVariant
	oldSpawnMCP := spawnMCP
	oldSpawnMode := spawnMode
	oldSpawnPhases := spawnPhases
	oldSpawnDesignWorkspace := spawnDesignWorkspace
	oldSpawnContextBudget := spawnContextBudget

	return func() {
		spawnBypassTriage = oldSpawnBypassTriage
		spawnMaxAgents = oldSpawnMaxAgents
		spawnModel = oldSpawnModel
		spawnWorkdir = oldSpawnWorkdir
		spawnForce = oldSpawnForce
		spawnAcknowledgeHotspot = oldSpawnAcknowledgeHotspot
		spawnAutoInit = oldSpawnAutoInit
		spawnNoTrack = oldSpawnNoTrack
		spawnIssue = oldSpawnIssue
		spawnSkipArtifactCheck = oldSpawnSkipArtifactCheck
		spawnLight = oldSpawnLight
		spawnFull = oldSpawnFull
		spawnValidation = oldSpawnValidation
		spawnBackendFlag = oldSpawnBackendFlag
		spawnAccount = oldSpawnAccount
		spawnOpus = oldSpawnOpus
		spawnInfra = oldSpawnInfra
		spawnVariant = oldSpawnVariant
		spawnMCP = oldSpawnMCP
		spawnMode = oldSpawnMode
		spawnPhases = oldSpawnPhases
		spawnDesignWorkspace = oldSpawnDesignWorkspace
		spawnContextBudget = oldSpawnContextBudget
	}
}

func TestNewSpawnPipelineInitializesInputs(t *testing.T) {
	p := newSpawnPipeline("http://127.0.0.1:4096", "feature-impl", "add happy-path tests", true, false, true, false, true)

	if p.client == nil {
		t.Fatal("expected client to be initialized")
	}
	if p.serverURL != "http://127.0.0.1:4096" {
		t.Fatalf("serverURL = %q, want %q", p.serverURL, "http://127.0.0.1:4096")
	}
	if p.skillName != "feature-impl" {
		t.Fatalf("skillName = %q, want %q", p.skillName, "feature-impl")
	}
	if p.task != "add happy-path tests" {
		t.Fatalf("task = %q, want %q", p.task, "add happy-path tests")
	}
	if !p.inline || p.headless || !p.tmux || p.attach || !p.daemonDriven {
		t.Fatalf("unexpected mode flags: inline=%v headless=%v tmux=%v attach=%v daemonDriven=%v", p.inline, p.headless, p.tmux, p.attach, p.daemonDriven)
	}
}

func TestRunPreFlightValidationHappyPathNonAnthropic(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	spawnBypassTriage = false
	spawnMaxAgents = 0 // disable concurrency checks with external dependencies
	spawnModel = "google/gemini-2.5-pro"
	spawnWorkdir = t.TempDir()
	spawnForce = false

	p := newSpawnPipeline("http://127.0.0.1:4096", "feature-impl", "add coverage", false, false, false, false, true)

	if err := p.runPreFlightValidation(); err != nil {
		t.Fatalf("runPreFlightValidation() error = %v", err)
	}
	if p.resolvedModel.Provider != "google" {
		t.Fatalf("resolved provider = %q, want %q", p.resolvedModel.Provider, "google")
	}
	if p.usageCheckResult == nil {
		t.Fatal("expected usageCheckResult to be initialized for non-anthropic path")
	}
}

func TestRunPreFlightValidationRequiresBypassForTrackedManualSpawns(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()
	t.Setenv(triageBypassEnvVar, "")

	spawnBypassTriage = false
	spawnNoTrack = false
	spawnMaxAgents = 0
	spawnModel = "google/gemini-2.5-pro"
	spawnWorkdir = t.TempDir()

	p := newSpawnPipeline("http://127.0.0.1:4096", "feature-impl", "manual tracked spawn", false, false, false, false, false)

	err := p.runPreFlightValidation()
	if err == nil {
		t.Fatal("expected runPreFlightValidation() to require triage bypass for tracked manual spawn")
	}
	if !strings.Contains(err.Error(), "triage bypass required") {
		t.Fatalf("expected triage bypass error, got %v", err)
	}
}

func TestRunPreFlightValidationAllowsTrackedManualSpawnsWithSessionBypassEnv(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()
	t.Setenv(triageBypassEnvVar, "1")

	spawnBypassTriage = false
	spawnNoTrack = false
	spawnMaxAgents = 0
	spawnModel = "google/gemini-2.5-pro"
	spawnWorkdir = t.TempDir()

	p := newSpawnPipeline("http://127.0.0.1:4096", "feature-impl", "manual tracked spawn", false, false, false, false, false)

	if err := p.runPreFlightValidation(); err != nil {
		t.Fatalf("runPreFlightValidation() error = %v", err)
	}
}

func TestRunPreFlightValidationAllowsNoTrackManualSpawnsWithoutBypass(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	spawnBypassTriage = false
	spawnNoTrack = true
	spawnMaxAgents = 0
	spawnModel = "google/gemini-2.5-pro"
	spawnWorkdir = t.TempDir()

	p := newSpawnPipeline("http://127.0.0.1:4096", "feature-impl", "manual no-track spawn", false, false, false, false, false)

	if err := p.runPreFlightValidation(); err != nil {
		t.Fatalf("runPreFlightValidation() error = %v", err)
	}
}

func TestResolveProjectUsesExplicitWorkdir(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	workdir := t.TempDir()
	if err := os.Mkdir(filepath.Join(workdir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads: %v", err)
	}

	spawnWorkdir = workdir
	spawnAutoInit = false
	spawnNoTrack = false

	p := &spawnPipeline{}
	if err := p.resolveProject(); err != nil {
		t.Fatalf("resolveProject() error = %v", err)
	}

	if p.projectDir != workdir {
		t.Fatalf("projectDir = %q, want %q", p.projectDir, workdir)
	}
	if p.projectName != filepath.Base(workdir) {
		t.Fatalf("projectName = %q, want %q", p.projectName, filepath.Base(workdir))
	}
}

func TestLoadSkillGeneratesWorkspaceAndLoadsContent(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	home := t.TempDir()
	t.Setenv("HOME", home)

	skillDir := filepath.Join(home, ".claude", "skills", "worker", "feature-impl")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	content := `---
name: feature-impl
skill-type: procedure
---

# Feature Impl
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test skill: %v", err)
	}

	p := &spawnPipeline{
		skillName:    "feature-impl",
		projectName:  "orch-go",
		task:         "add pipeline tests",
		daemonDriven: true,
	}

	if err := p.loadSkill(); err != nil {
		t.Fatalf("loadSkill() error = %v", err)
	}
	if p.workspaceName == "" {
		t.Fatal("expected workspaceName to be set")
	}
	if p.skillContent == "" {
		t.Fatal("expected skillContent to be loaded")
	}
	if !strings.Contains(p.skillContent, "# Feature Impl") {
		t.Fatalf("expected skill content marker, got %q", p.skillContent)
	}
	if p.isOrchestrator {
		t.Fatal("expected feature-impl not to be marked as orchestrator")
	}
}

func TestSetupIssueTrackingNoTrackHappyPath(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	spawnNoTrack = true
	spawnIssue = ""
	spawnForce = false

	p := &spawnPipeline{
		projectName: "orch-go",
		skillName:   "feature-impl",
		task:        "add tests",
	}

	if err := p.setupIssueTracking(); err != nil {
		t.Fatalf("setupIssueTracking() error = %v", err)
	}
	if !strings.Contains(p.beadsID, "orch-go-untracked-") {
		t.Fatalf("beadsID = %q, want untracked id", p.beadsID)
	}
	if p.skipBeadsForOrchestrator {
		t.Fatal("expected skipBeadsForOrchestrator to be false for feature-impl")
	}
}

func TestGatherContextSkipArtifactCheck(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	spawnSkipArtifactCheck = true

	p := &spawnPipeline{}
	if err := p.gatherContext(); err != nil {
		t.Fatalf("gatherContext() error = %v", err)
	}
	if p.kbContext != "" {
		t.Fatalf("kbContext = %q, want empty", p.kbContext)
	}
	if p.gapAnalysis != nil {
		t.Fatal("expected gapAnalysis to remain nil when artifact check is skipped")
	}
}

func TestBuildSpawnConfigHappyPathNoTrack(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	spawnNoTrack = true
	spawnLight = false
	spawnFull = false
	spawnValidation = "tests"
	spawnBackendFlag = "opencode"
	spawnOpus = false
	spawnInfra = false
	spawnModel = "sonnet"
	spawnVariant = ""
	spawnMCP = ""
	spawnMode = "tdd"
	spawnPhases = ""
	spawnDesignWorkspace = ""
	spawnContextBudget = 9000

	p := &spawnPipeline{
		task:          "add critical-path tests",
		skillName:     "feature-impl",
		projectName:   "orch-go",
		projectDir:    t.TempDir(),
		workspaceName: "og-feat-critical-path-tests",
		skillContent:  "# test skill",
	}

	if err := p.buildSpawnConfig(); err != nil {
		t.Fatalf("buildSpawnConfig() error = %v", err)
	}
	if p.cfg == nil {
		t.Fatal("expected cfg to be built")
	}
	if p.cfg.SpawnMode != "opencode" {
		t.Fatalf("SpawnMode = %q, want %q", p.cfg.SpawnMode, "opencode")
	}
	if !p.cfg.NoTrack {
		t.Fatal("expected NoTrack=true")
	}
	if p.cfg.Validation != "tests" {
		t.Fatalf("Validation = %q, want %q", p.cfg.Validation, "tests")
	}
	if p.cfg.ContextBudget != 9000 {
		t.Fatalf("ContextBudget = %d, want %d", p.cfg.ContextBudget, 9000)
	}
	if p.cfg.WorkspaceName != p.workspaceName {
		t.Fatalf("WorkspaceName = %q, want %q", p.cfg.WorkspaceName, p.workspaceName)
	}
}

func TestBuildSpawnConfigSetsClaudeConfigDirForAutoSwitchedNonPrimary(t *testing.T) {
	restore := saveSpawnPipelineGlobals()
	defer restore()

	oldLoad := loadSpawnAccounts
	oldHome := spawnUserHomeDir
	defer func() {
		loadSpawnAccounts = oldLoad
		spawnUserHomeDir = oldHome
	}()

	loadSpawnAccounts = func() (*account.Config, error) {
		return &account.Config{Default: "personal"}, nil
	}
	spawnUserHomeDir = func() (string, error) {
		return "/tmp/home", nil
	}

	spawnNoTrack = true
	spawnLight = false
	spawnFull = false
	spawnValidation = "tests"
	spawnBackendFlag = "claude"
	spawnOpus = false
	spawnInfra = false
	spawnModel = "opus"
	spawnVariant = ""
	spawnMCP = ""
	spawnMode = "tdd"
	spawnPhases = ""
	spawnDesignWorkspace = ""
	spawnContextBudget = 12000
	spawnAccount = ""

	p := &spawnPipeline{
		task:             "fix rate limit behavior",
		skillName:        "feature-impl",
		projectName:      "orch-go",
		projectDir:       t.TempDir(),
		workspaceName:    "og-feat-rate-limit-fix",
		skillContent:     "# test skill",
		usageCheckResult: &UsageCheckResult{Switched: true, SwitchedToAccount: "work"},
	}

	if err := p.buildSpawnConfig(); err != nil {
		t.Fatalf("buildSpawnConfig() error = %v", err)
	}

	want := filepath.Join("/tmp/home", ".claude-work")
	if p.cfg.ClaudeConfigDir != want {
		t.Fatalf("ClaudeConfigDir = %q, want %q", p.cfg.ClaudeConfigDir, want)
	}
}

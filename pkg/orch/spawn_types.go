package orch

import (
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// SpawnInput holds all input parameters for spawn operation.
type SpawnInput struct {
	ServerURL    string
	SkillName    string
	Task         string
	IssueID      string // Beads issue ID (if spawning with --issue)
	Inline       bool
	Headless     bool
	Tmux         bool
	Attach       bool
	DaemonDriven bool
}

// SpawnContext holds all computed context for spawn operation.
type SpawnContext struct {
	Task               string
	OrientationFrame   string
	IntentType         string
	SkillName          string
	ProjectDir         string
	ProjectName        string
	WorkspaceName      string
	SkillContent       string
	BeadsID            string
	IsOrchestrator     bool
	IsMetaOrchestrator bool
	ResolvedModel      model.ModelSpec
	ResolvedSettings   spawn.ResolvedSpawnSettings
	KBContext          string
	GapAnalysis        *spawn.GapAnalysis
	HasInjectedModels  bool
	PrimaryModelPath   string
	CrossRepoModelDir  string
	IsBug              bool
	ReproSteps         string
	ReworkFeedback     string
	ReworkNumber       int
	PriorSynthesis     string
	PriorWorkspace     string
	UsageInfo          *spawn.UsageInfo
	Account            string
	AccountConfigDir   string
	SpawnBackend       string
	Tier               string
	VerifyLevel        string
	ReviewTier         string
	IssueType          string
	Scope              string
	HotspotArea          bool
	HotspotFiles         []string
	HotspotDefectClasses []string
	DesignMockupPath   string
	DesignPromptPath   string
	DesignNotes        string
	BeadsDir           string
	PriorCompletions   string
	MaxTurns           int
	Settings           string
	Explore            bool
	ExploreBreadth     int
	ExploreParentSkill string
}

// ResolvedSpawnResult holds resolved spawn settings and the parsed model spec.
type ResolvedSpawnResult struct {
	Settings spawn.ResolvedSpawnSettings
	Model    model.ModelSpec
}

// GapCheckResult contains the results of a pre-spawn gap check.
type GapCheckResult struct {
	Context      string
	GapAnalysis  *spawn.GapAnalysis
	Blocked      bool
	BlockReason  string
	FormatResult *spawn.KBContextFormatResult
}

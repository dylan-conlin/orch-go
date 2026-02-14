# spawn_cmd.go Decomposition Plan

## Goal
Decompose runSpawnWithSkillInternal (478 lines) into a clean pipeline pattern following complete_cmd.go's approach.

## Pattern from complete_cmd.go

```go
func runComplete(identifier, workdir string) error {
    // 1. Validate flags
    skipConfig := getSkipConfig()
    if err := validateSkipFlags(skipConfig); err != nil {
        return err
    }
    
    // 2. Determine workspace/beads ID
    workspacePath, beadsID := determineWorkspaceAndBeadsID(...)
    
    // 3. Get issue details
    issue, isClosed := getIssueDetails(beadsID)
    
    // 4. Verify completion
    result := verifyCompletion(beadsID, workspacePath, skipConfig)
    
    // 5. Close issue and cleanup
    closeIssueAndCleanup(beadsID, workspacePath, result)
    
    return nil
}
```

## Proposed Decomposition for spawn_cmd.go

### Helper Types (add to top of file)

```go
// SpawnInput holds all input parameters for spawn operation
type SpawnInput struct {
    ServerURL     string
    SkillName     string
    Task          string
    Inline        bool
    Headless      bool
    Tmux          bool
    Attach        bool
    DaemonDriven  bool
}

// SpawnContext holds all computed context for spawn operation
type SpawnContext struct {
    ProjectDir         string
    ProjectName        string
    WorkspaceName      string
    SkillContent       string
    BeadsID            string
    IsOrchestrator     bool
    IsMetaOrchestrator bool
    ResolvedModel      model.ModelSpec
    KBContext          string
    GapAnalysis        *spawn.GapAnalysis
    HasInjectedModels  bool
    PrimaryModelPath   string
    IsBug              bool
    ReproSteps         string
    UsageInfo          *spawn.UsageInfo
    SpawnBackend       string
    Tier               string
    DesignMockupPath   string
    DesignPromptPath   string
    DesignNotes        string
}
```

### Helper Functions

#### 1. runPreFlightChecks (lines 470-512)
```go
func runPreFlightChecks(input *SpawnInput, projectDir string) (*gates.RateLimitCheckResult, error)
```
Responsibilities:
- Check triage bypass
- Check concurrency limit
- Check rate limits
- Check hotspots
Returns: usage check result, error

#### 2. resolveProjectDirectory (lines 514-538)
```go
func resolveProjectDirectory() (projectDir, projectName string, err error)
```
Responsibilities:
- Get workdir (flag or current dir)
- Validate directory exists
- Extract project name from path
Returns: projectDir, projectName, error

#### 3. loadSkillAndGenerateWorkspace (lines 540-578)
```go
func loadSkillAndGenerateWorkspace(skillName, projectName, task string, projectDir string) (
    skillContent, workspaceName string,
    isOrchestrator, isMetaOrchestrator bool,
    err error)
```
Responsibilities:
- Ensure orch scaffolding
- Load skill content (raw + with dependencies)
- Detect orchestrator type
- Generate workspace name
Returns: skillContent, workspaceName, isOrchestrator, isMetaOrchestrator, error

#### 4. setupBeadsTracking (lines 580-659)
```go
func setupBeadsTracking(skillName, task, projectName, beadsID string, isOrchestrator, isMetaOrchestrator bool, serverURL string) (string, error)
```
Responsibilities:
- Determine beads ID (from flag, create new, or skip)
- Check retry patterns
- Check for duplicate spawns
- Update issue status to in_progress
Returns: finalBeadsID, error

#### 5. resolveAndValidateModel (lines 661-678)
```go
func resolveAndValidateModel(modelFlag string) (model.ModelSpec, error)
```
Responsibilities:
- Resolve model aliases
- Validate flash model (block if flash)
Returns: resolvedModel, error

#### 6. gatherSpawnContext (lines 680-743)
```go
func gatherSpawnContext(skillContent, task, beadsID, projectDir string) (
    kbContext string,
    gapAnalysis *spawn.GapAnalysis,
    hasInjectedModels bool,
    primaryModelPath string,
    err error)
```
Responsibilities:
- Parse skill requirements
- Gather KB context (skill-driven or default)
- Check gap gating
- Record gaps for learning
- Log gap bypass if used
Returns: kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, error

#### 7. extractBugReproInfo (lines 745-759)
```go
func extractBugReproInfo(beadsID string, noTrack bool) (isBug bool, reproSteps string)
```
Responsibilities:
- Get reproduction info for bug issues
- Print bug detection message
Returns: isBug, reproSteps

#### 8. buildUsageInfo (lines 761-771)
```go
func buildUsageInfo(usageCheckResult *gates.RateLimitCheckResult) *spawn.UsageInfo
```
Responsibilities:
- Convert rate limit check result to UsageInfo struct
Returns: usageInfo

#### 9. determineSpawnBackend (lines 773-838)
```go
func determineSpawnBackend(resolvedModel model.ModelSpec, task, beadsID, projectDir string) (string, error)
```
Responsibilities:
- Apply backend selection priority logic
- Validate mode+model combination
- Log infrastructure detection
Returns: spawnBackend, error

#### 10. loadDesignArtifacts (lines 840-851)
```go
func loadDesignArtifacts(designWorkspace, projectDir string) (mockupPath, promptPath, notes string)
```
Responsibilities:
- Read design artifacts if --design-workspace provided
- Print handoff summary
Returns: mockupPath, promptPath, notes

#### 11. buildSpawnConfig (lines 853-885)
```go
func buildSpawnConfig(ctx *SpawnContext) *spawn.Config
```
Responsibilities:
- Build spawn.Config struct from SpawnContext
Returns: cfg

#### 12. validateAndWriteContext (lines 887-915)
```go
func validateAndWriteContext(cfg *spawn.Config) (minimalPrompt string, err error)
```
Responsibilities:
- Validate context size
- Warn about large contexts
- Check workspace exists
- Write SPAWN_CONTEXT.md
- Record spawn in session
- Generate minimal prompt
Returns: minimalPrompt, error

#### 13. dispatchSpawn (lines 920-946)
```go
func dispatchSpawn(input *SpawnInput, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task, serverURL string) error
```
Responsibilities:
- Dispatch to appropriate spawn mode (inline, headless, claude, tmux)
Returns: error

### Refactored runSpawnWithSkillInternal

```go
func runSpawnWithSkillInternal(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool, daemonDriven bool) error {
    input := &SpawnInput{
        ServerURL:    serverURL,
        SkillName:    skillName,
        Task:         task,
        Inline:       inline,
        Headless:     headless,
        Tmux:         tmux,
        Attach:       attach,
        DaemonDriven: daemonDriven,
    }
    
    // 1. Pre-flight checks
    usageCheckResult, err := runPreFlightChecks(input, "")
    if err != nil {
        return err
    }
    
    // 2. Resolve project directory
    projectDir, projectName, err := resolveProjectDirectory()
    if err != nil {
        return err
    }
    
    // 3. Load skill and generate workspace
    skillContent, workspaceName, isOrchestrator, isMetaOrchestrator, err := loadSkillAndGenerateWorkspace(skillName, projectName, task, projectDir)
    if err != nil {
        return err
    }
    
    // 4. Setup beads tracking
    beadsID, err := setupBeadsTracking(skillName, task, projectName, spawnIssue, isOrchestrator, isMetaOrchestrator, serverURL)
    if err != nil {
        return err
    }
    
    // 5. Resolve and validate model
    resolvedModel, err := resolveAndValidateModel(spawnModel)
    if err != nil {
        return err
    }
    
    // 6. Gather spawn context
    kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, err := gatherSpawnContext(skillContent, task, beadsID, projectDir)
    if err != nil {
        return err
    }
    
    // 7. Extract bug reproduction info
    isBug, reproSteps := extractBugReproInfo(beadsID, spawnNoTrack || isOrchestrator || isMetaOrchestrator)
    
    // 8. Build usage info
    usageInfo := buildUsageInfo(usageCheckResult)
    
    // 9. Determine spawn backend
    spawnBackend, err := determineSpawnBackend(resolvedModel, task, beadsID, projectDir)
    if err != nil {
        return err
    }
    
    // 10. Load design artifacts
    designMockupPath, designPromptPath, designNotes := loadDesignArtifacts(spawnDesignWorkspace, projectDir)
    
    // 11. Build spawn context
    ctx := &SpawnContext{
        ProjectDir:         projectDir,
        ProjectName:        projectName,
        WorkspaceName:      workspaceName,
        SkillContent:       skillContent,
        BeadsID:            beadsID,
        IsOrchestrator:     isOrchestrator,
        IsMetaOrchestrator: isMetaOrchestrator,
        ResolvedModel:      resolvedModel,
        KBContext:          kbContext,
        GapAnalysis:        gapAnalysis,
        HasInjectedModels:  hasInjectedModels,
        PrimaryModelPath:   primaryModelPath,
        IsBug:              isBug,
        ReproSteps:         reproSteps,
        UsageInfo:          usageInfo,
        SpawnBackend:       spawnBackend,
        Tier:               determineSpawnTier(skillName, spawnLight, spawnFull),
        DesignMockupPath:   designMockupPath,
        DesignPromptPath:   designPromptPath,
        DesignNotes:        designNotes,
    }
    
    // 12. Build spawn config
    cfg := buildSpawnConfig(ctx)
    
    // 13. Validate and write context
    minimalPrompt, err := validateAndWriteContext(cfg)
    if err != nil {
        return err
    }
    
    // 14. Dispatch spawn
    return dispatchSpawn(input, cfg, minimalPrompt, beadsID, skillName, task, serverURL)
}
```

## Implementation Order

1. Create helper types (SpawnInput, SpawnContext) at top of file
2. Extract helper functions in order (1-13)
3. Refactor runSpawnWithSkillInternal to use pipeline pattern
4. Test spawn with various flags/modes
5. Commit

## Testing Strategy

After each major extraction (groups of 3-4 functions), test:
- `orch spawn --bypass-triage feature-impl "test task" --light`
- `orch spawn --bypass-triage --tmux investigation "test task"`
- `orch spawn --bypass-triage --model opus investigation "test task"`

Final comprehensive test:
- Headless mode spawn
- Tmux mode spawn
- Inline mode spawn
- Claude backend spawn
- With and without --no-track
- With and without --workdir
- With and without --design-workspace

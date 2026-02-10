# Investigation: orch spawn --bypass-triage Ceremony UX

**Date:** 2026-02-09  
**Status:** Active  
**Issue:** pw-8955  

## Question

What ceremony does `orch spawn --bypass-triage` require, why does it add friction in interactive sessions, and how can we make manual spawns smoother (auto-detect interactive orchestrator sessions or reduce ceremony)?

## Findings

### Finding 1: orch binary location

**Evidence:**
- `orch` binary is symlinked from `/Users/dylanconlin/bin/orch` → `/Users/dylanconlin/Documents/personal/orch-go/build/orch`
- Source code located at `/Users/dylanconlin/Documents/personal/orch-go`
- Written in Go

**Source:** `which orch` and `ls -la $(which orch)`

**Significance:** Need to explore the orch-go codebase to understand spawn implementation and --bypass-triage flag

---

### Finding 2: The --bypass-triage ceremony

**Evidence:**
From `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_validation.go:219-249`:

The ceremony occurs when:
- Spawning WITH issue tracking (not `--no-track`)
- Without `--bypass-triage` flag or `ORCH_BYPASS_TRIAGE=1` env var

The system displays this warning box:
```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ⚠️  TRIAGE BYPASS REQUIRED                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  Tracked manual spawn requires --bypass-triage flag.                        │
│                                                                             │
│  The preferred workflow is daemon-driven triage:                            │
│    1. Create issue: bd create "task" --type task -l triage:ready            │
│    2. Daemon auto-spawns: orch daemon run                                   │
│                                                                             │
│  Manual spawn is for exceptions only:                                       │
│    - Single urgent item requiring immediate attention                       │
│    - Complex/ambiguous task needing custom context                          │
│    - Skill selection requires orchestrator judgment                         │
│                                                                             │
│  For ad-hoc work without issue tracking, use --no-track.                    │
│                                                                             │
│  To proceed with manual spawn, use one of:                                  │
│    1. One-off: orch spawn --bypass-triage ...                               │
│    2. Session: export ORCH_BYPASS_TRIAGE=1                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Source:** `spawn_validation.go:showTriageBypassRequired()`

**Significance:** This creates friction by requiring extra ceremony on every manual spawn. The intent is to push users toward daemon-driven workflow.

---

### Finding 3: Current bypass mechanisms

**Evidence:**
From `spawn_validation.go:189-202` (`resolveTriageBypass()`):

Two bypass sources:
1. **Flag**: `--bypass-triage` (one-off)
2. **Env var**: `ORCH_BYPASS_TRIAGE=1` (session-level)

The env var accepts: "1", "true", "yes", "on" (case-insensitive)

**Source:** `spawn_validation.go:resolveTriageBypass()`

**Significance:** Session-level bypass exists but requires manual export. No auto-detection of interactive orchestrator sessions.

---

### Finding 4: The friction points

**Evidence:**
- User must add `--bypass-triage` to every manual spawn command
- OR remember to `export ORCH_BYPASS_TRIAGE=1` at session start
- Warning box interrupts flow even when user understands the rationale
- Interactive orchestrators (like the one that spawned this agent) repeatedly hit this friction when delegating work

**Significance:** The ceremony adds valuable guardrails for human-initiated spawns but creates unnecessary friction for orchestrator-initiated spawns that are already doing triage.

---

## Synthesis

The `--bypass-triage` ceremony is **working as designed** to create friction that encourages daemon-driven workflow. However, the friction is **indiscriminate** - it applies equally to:
1. Human ad-hoc spawns (where friction is valuable)
2. Orchestrator-delegated spawns (where friction is redundant - triage already happened)

The task asks to "auto-detect interactive orchestrator sessions" - this would mean:
- Detecting when `orch spawn` is called from within an OpenCode/Claude session that is itself an orchestrator
- Automatically setting `ORCH_BYPASS_TRIAGE=1` for these sessions
- Preserving the ceremony for direct human invocations

**Key insight:** The ceremony is a gate with no awareness of context. An orchestrator delegating to a worker has already done triage - requiring `--bypass-triage` is redundant ceremony in that context.

### Finding 5: The daemonDriven flag mechanism

**Evidence:**
From `spawn_cmd.go:307-324`:

```go
func runSpawnWithSkill(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool) error {
    return runSpawnWithSkillInternal(serverURL, skillName, task, inline, headless, tmux, attach, false)
}

// When daemonDriven is true, the triage bypass check is skipped (issue already triaged).
func runSpawnWithSkillInternal(serverURL, skillName, task string, inline bool, headless bool, tmux bool, attach bool, daemonDriven bool) error {
    p := newSpawnPipeline(serverURL, skillName, task, inline, headless, tmux, attach, daemonDriven)
    // ...
}
```

In `spawn_pipeline.go:99-110`:
```go
// Check for triage bypass (required for tracked manual spawns).
// Daemon-driven spawns and ad-hoc --no-track spawns skip this check.
bypass, source := resolveTriageBypass()
if !p.daemonDriven && !spawnNoTrack && !bypass {
    return showTriageBypassRequired(p.skillName, p.task)
}
```

**Source:** `spawn_cmd.go` and `spawn_pipeline.go`

**Significance:** The mechanism to bypass triage already exists! `daemonDriven=true` skips the ceremony. The problem is it's hardcoded to `false` for manual `orch spawn` invocations.

---

### Finding 6: How work command uses daemonDriven

**Evidence:**
From `work_cmd.go:99-101`:

```go
// Work command is daemon-driven (issue already created and triaged)
// Pass daemonDriven=true to skip triage bypass check
return runSpawnWithSkillInternal(serverURL, skillName, task, inline, true, false, false, true)
```

The `work` command (used by daemon) sets `daemonDriven=true` because the issue has already been triaged by the daemon.

**Source:** `work_cmd.go`

**Significance:** This proves that skipping the ceremony is intentional and safe when triage has already happened upstream.

---

## Open Questions

1. How can we detect if `orch spawn` is being called from within an orchestrator session?
   - Check for OpenCode session env vars?
   - Check for Claude CLI env vars?
   - Check parent process?
   - Look for marker files in workspace?
2. What signals are available to distinguish orchestrator-initiated vs human-initiated spawns?
3. Should the auto-detection be:
   - Environment-based (check for orchestrator session env vars)?
   - Process-based (check parent process)?
   - Config-based (explicit orchestrator mode setting)?
4. What are the risks of auto-bypassing triage in orchestrator contexts?

## Proposed Solution

### Option A: Detect orchestrator skill type early (RECOMMENDED)

**Approach:** Move orchestrator skill type detection before the triage bypass check, then set `daemonDriven=true` for orchestrator skills.

**Rationale:**
- Orchestrator spawns are inherently different from worker spawns
- Orchestrators ARE doing triage - they're making decisions about what work to spawn
- The triage bypass ceremony is redundant for orchestrators
- Orchestrator skills already skip beads tracking (`skipBeadsForOrchestrator`)

**Implementation:**
1. Load skill type in `runSpawnWithSkill()` before calling `runSpawnWithSkillInternal()`
2. If skill is orchestrator/meta-orchestrator, set `daemonDriven=true` automatically
3. This preserves the ceremony for worker spawns while removing it for orchestrators

**Benefits:**
- Zero ceremony for orchestrators (auto-detected)
- Preserves ceremony for human-initiated worker spawns (valuable friction)
- No need for orchestrators to remember flags or env vars
- Aligns with existing `skipBeadsForOrchestrator` pattern

**Risks:**
- Minimal: Orchestrators are policy/coordination skills that inherently do triage

### Option B: Environment variable approach

Set `ORCH_ORCHESTRATOR_SESSION=1` in orchestrator spawn contexts, then check for this var alongside `ORCH_BYPASS_TRIAGE`.

**Benefits:** Simple, explicit
**Drawbacks:** Requires orchestrators to set env var manually, easy to forget

### Option C: Reduce ceremony verbosity

Make the warning shorter/less prominent while keeping the gate.

**Benefits:** Less interruption
**Drawbacks:** Doesn't solve the core friction problem

## Implementation

### Solution Implemented: Option A (Orchestrator Skill Auto-Bypass)

**Changes made to orch-go:**

1. **Added `shouldAutoBypassTriage()` function** (`spawn_validation.go`)
   - Detects orchestrator and meta-orchestrator skills
   - Returns `(bool, string)` indicating bypass and reason

2. **Modified `runSpawnWithSkill()`** (`spawn_cmd.go`)
   - Checks skill type before calling `runSpawnWithSkillInternal()`
   - Auto-sets `daemonDriven=true` for orchestrator skills
   - Displays informative message: "ℹ️ Auto-bypassing triage ceremony (orchestrator skill performs triage)"

3. **Added test coverage** (`spawn_validation_test.go`)
   - `TestShouldAutoBypassTriageForOrchestrator` verifies bypass logic
   - Tests orchestrator, meta-orchestrator, and worker skills
   - All tests passing

**Verification:**
- Manual testing confirmed:
  - `orch spawn orchestrator "test"` → Auto-bypasses, no ceremony
  - `orch spawn investigation "test"` → Requires --bypass-triage (ceremony preserved)
- Test suite: `go test ./cmd/orch -run TestShouldAutoBypassTriageForOrchestrator` → PASS
- All existing tests continue to pass

**Benefits:**
- Zero ceremony for orchestrator-initiated spawns
- Preserves valuable friction for human-initiated worker spawns
- Aligns with existing `skipBeadsForOrchestrator` pattern
- No need for orchestrators to remember flags or env vars

**Commit:** `b25efd18` in orch-go repository

## Conclusion

The triage bypass ceremony was adding redundant friction for orchestrator-initiated spawns. Orchestrators inherently perform triage as part of their coordination work, so requiring `--bypass-triage` was unnecessary ceremony.

The implemented solution auto-detects orchestrator skills and bypasses the ceremony while preserving it for worker skills. This removes friction where it doesn't add value while maintaining the valuable guardrails for human-initiated spawns.

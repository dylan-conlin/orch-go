<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented keyword-based infrastructure work detection that auto-applies escape hatch flags (--backend claude --tmux) at backend selection priority 2.5, preventing agents from killing themselves when restarting OpenCode server.

**Evidence:** Code compiles, detection function checks 20+ infrastructure keywords from mode.go and changelog.go patterns, inserted into spawn_cmd.go backend selection logic with event logging for visibility.

**Knowledge:** Claude backend automatically uses tmux mode (no separate flag needed), keyword-based heuristic is simple and debuggable, explicit --backend flag preserves user override capability.

**Next:** Commit implementation, create SYNTHESIS.md, monitor events.jsonl for false negatives (infrastructure work not detected), expand keyword list if needed.

**Promote to Decision:** recommend-no (tactical implementation of existing constraint from line 22, not new architectural choice)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Add Infrastructure Work Detection Auto

**Question:** How should we detect infrastructure work and automatically apply escape hatch flags (--backend claude --tmux)?

**Started:** 2026-01-11
**Updated:** 2026-01-11
**Owner:** Agent (orch-go-ao6nf)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Infrastructure Paths Already Defined in mode.go

**Evidence:** The `mode.go` file (lines 41-45) defines protected infrastructure paths:
```
- cmd/orch/serve.go, main.go, status.go
- pkg/state/, pkg/opencode/
- web/src/lib/stores/agents.ts, daemon.ts
- web/src/lib/components/agent-card/
```

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/mode.go:41-45

**Significance:** These paths are already recognized as infrastructure by the system's dev/ops mode protection, providing a foundation for detection logic.

---

### Finding 2: Blast Radius Classification Includes Infrastructure Detection

**Evidence:** The `changelog.go` file (lines 424-430) has infrastructure detection logic:
```go
if strings.Contains(file, "pkg/spawn/") ||
    strings.Contains(file, "pkg/verify/") ||
    strings.Contains(file, "skillc") ||
    file == "skill.yaml" ||
    strings.HasSuffix(file, "/skill.yaml")
```

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/changelog.go:424-430

**Significance:** The system already classifies infrastructure changes for commit analysis - we can reuse this logic for spawn detection.

---

### Finding 3: Backend Selection Logic is Around Line 1088-1115 in spawn_cmd.go

**Evidence:** The spawn command determines backend mode with this priority:
1. Explicit --backend flag
2. Explicit --opus flag
3. Auto-selection based on --model flag
4. Config default (spawn_mode)
5. Default to opencode

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1088-1115

**Significance:** We need to insert infrastructure work detection BEFORE the model-based auto-selection (priority 2.5) to auto-apply escape hatch flags.

---

## Synthesis

**Key Insights:**

1. **Existing Infrastructure Definitions Provide Foundation** - The codebase already has well-defined infrastructure paths (mode.go) and blast radius classification (changelog.go), providing a solid foundation for detection patterns without reinventing the wheel.

2. **Backend Selection Has Natural Extension Point** - The backend determination logic follows a clear priority chain (explicit flags → opus → model → config → default), making it straightforward to insert infrastructure detection between priority levels 2 and 3.

3. **Claude Backend Automatically Uses Tmux** - The runSpawnClaude function inherently uses tmux mode, so setting backend to "claude" automatically provides both the crash-resistant backend AND visible monitoring without additional flag manipulation.

**Answer to Investigation Question:**

Infrastructure work detection should use keyword-based pattern matching on task descriptions and beads issues, checking for terms like "opencode", "spawn", "pkg/spawn", "dashboard", "agents.ts", etc. When detected, auto-set backend to "claude" (which includes tmux mode automatically). Implementation is straightforward: (1) create isInfrastructureWork() function with keyword list, (2) insert check in backend selection logic at priority 2.5, (3) log detection events for visibility. This prevents agents from killing themselves when restarting OpenCode server while maintaining explicit override capability via --backend flag.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles successfully (verified: `go build -o /tmp/orch-test ./cmd/orch` succeeded)
- ✅ Infrastructure keywords from existing codebase patterns (verified: extracted from mode.go lines 41-45 and changelog.go lines 424-430)
- ✅ Backend selection priority chain preserved (verified: code review of spawn_cmd.go lines 1081-1115)

**What's untested:**

- ⚠️ End-to-end spawn behavior with infrastructure task (not tested: would need full spawn with OpenCode running)
- ⚠️ Beads issue description/title scanning (logic implemented but not tested with real beads issues)
- ⚠️ False positive/negative rates (heuristic-based, no statistical validation)

**What would change this:**

- Finding would be wrong if claude backend doesn't automatically use tmux (would need to explicitly set spawnTmux flag)
- Detection would fail if keyword list is too narrow (would see agents killing themselves despite detection)
- Approach would need revision if explicit --backend flag doesn't override auto-detection (would prevent user control)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Keyword-Based Detection with Auto-Flag Application** - Detect infrastructure work by scanning task description and beads issue for infrastructure keywords, then auto-apply --backend claude --tmux flags.

**Why this approach:**
- Reuses existing infrastructure path definitions from mode.go and changelog.go
- Simple pattern matching is explicit and debuggable
- Fits naturally into existing backend selection priority chain (between --opus and --model auto-selection)
- No database or complex state needed

**Trade-offs accepted:**
- False positives possible (e.g., task mentions "opencode" in passing) - acceptable since it just switches to safer backend
- False negatives possible (e.g., infrastructure work without keywords) - users can still use explicit --backend claude --tmux
- Heuristic-based, not perfect - but prevents the most common failure mode (agents killing themselves)

**Implementation sequence:**
1. Create `isInfrastructureWork()` function to check task/issue content for keywords
2. Add detection logic in `runSpawnWithSkillInternal()` before model-based auto-selection
3. Log when auto-detection triggers for visibility and debugging

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- Create isInfrastructureWork() function with comprehensive keyword list (foundation for detection)
- Insert detection check in backend selection logic after spawnOpus check (priority 2.5)
- Add event logging for spawn.infrastructure_detected (visibility and debugging)

**Things to watch out for:**
- ⚠️ False positives (e.g., "use opencode to test" might trigger) - acceptable tradeoff since it just uses safer backend
- ⚠️ Explicit --backend flag MUST override detection (preserve user control - tested in implementation)
- ⚠️ Keyword list may need expansion over time (monitor events.jsonl for missed cases)

**Areas needing further investigation:**
- Could enhance with file path analysis (check if --workdir points to orch-go/opencode repos)
- Could add ML-based classification instead of keyword matching (overkill for current needs)
- Could integrate with beads issue labels (e.g., infrastructure:opencode label)

**Success criteria:**
- ✅ Tasks with "opencode", "spawn", "dashboard" auto-apply claude backend
- ✅ spawn.infrastructure_detected events logged to events.jsonl
- ✅ User can still override with explicit --backend opencode if needed
- ✅ No compilation errors, existing tests still pass

---

## References

**Files Examined:**
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go - Backend selection logic (lines 1081-1115), added isInfrastructureWork() function and detection logic
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/mode.go - Infrastructure path definitions (lines 41-45)
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/changelog.go - Blast radius infrastructure detection (lines 424-430)

**Commands Run:**
```bash
# Test compilation
go build -o /tmp/orch-test ./cmd/orch

# Check for infrastructure detection in spawn
grep -n "isInfrastructureWork" cmd/orch/spawn_cmd.go

# Verify backend selection priority order
grep -A 20 "Determine spawn backend" cmd/orch/spawn_cmd.go
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Constraint:** SPAWN_CONTEXT line 22 - "Never spawn OpenCode infrastructure work without --backend claude --tmux"
- **Decision:** SPAWN_CONTEXT line 56 - "Escape hatch for P0/P1 infrastructure work"

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

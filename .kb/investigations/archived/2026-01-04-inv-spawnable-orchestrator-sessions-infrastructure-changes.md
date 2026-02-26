<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawnable orchestrator sessions require primarily skill-type detection in spawn_cmd.go and ORCHESTRATOR_CONTEXT.md template - existing infrastructure can be reused with targeted extensions.

**Evidence:** Analyzed spawn_cmd.go, session.go, complete_cmd.go, skills/loader.go, and verify/check.go - found existing skill metadata parsing supports skill-type field, and spawn/complete flows can be extended without new subcommands.

**Knowledge:** Orchestrator-type skills differ from worker skills in: (1) no beads tracking, (2) SESSION_HANDOFF.md instead of SYNTHESIS.md, (3) default tmux visibility, (4) session goal focus. Infrastructure changes are additive, not replacing existing patterns.

**Next:** Implement skill-type detection in spawn_cmd.go to route orchestrator-type skills to different context template and completion verification.

---

# Investigation: Spawnable Orchestrator Sessions Infrastructure Changes

**Question:** What infrastructure changes are needed to `orch spawn` and `orch complete` for orchestrator-type skills?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Design Session Agent
**Phase:** Synthesizing
**Next Step:** Present findings to orchestrator for decision on approach
**Status:** Complete

**Related-From:** `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md`
**Related-From:** `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md`

---

## Findings

### Finding 1: Skill Metadata Already Supports skill-type Field

**Evidence:** The `pkg/skills/loader.go:21-30` already parses a `SkillType` field from skill YAML frontmatter:

```go
type SkillMetadata struct {
    Name         string   `yaml:"name"`
    SkillType    string   `yaml:"skill-type"`  // Already exists!
    Audience     string   `yaml:"audience"`
    Spawnable    bool     `yaml:"spawnable"`
    ...
}
```

The meta-orchestrator skill already has frontmatter (verified in SPAWN_CONTEXT.md):
```yaml
---
name: orchestrator
skill-type: policy
...
---
```

**Source:** `pkg/skills/loader.go:21-30`, SPAWN_CONTEXT.md skill guidance section

**Significance:** No new field needed. We can use existing `skill-type: orchestrator` or `skill-type: meta` to detect orchestrator-type skills. The routing decision happens in `spawn_cmd.go` after skill loading.

---

### Finding 2: Spawn Context Template is Data-Driven via Config

**Evidence:** The spawn context is generated from `pkg/spawn/context.go:26-272` using a Go template with data from `spawn.Config`. Key fields:

```go
type Config struct {
    Tier         string  // "light" or "full"
    NoTrack      bool    // Opt-out of beads tracking
    SkillContent string  // Full skill SKILL.md
    ...
}
```

The template already handles NoTrack mode with different instructions (lines 58-97). This pattern can be extended for orchestrator mode.

**Source:** `pkg/spawn/context.go:26-272`, `pkg/spawn/config.go:66-131`

**Significance:** Adding ORCHESTRATOR_CONTEXT.md is straightforward - add a new template or extend the existing template with orchestrator-specific sections, controlled by a new Config field like `IsOrchestrator bool`.

---

### Finding 3: Completion Verification is Tier-Aware and Extensible

**Evidence:** The `pkg/verify/check.go:161-209` shows verification is already tier-aware:

```go
// Light tier spawns skip the SYNTHESIS.md requirement
if workspacePath != "" && tier != "light" {
    ok, err := VerifySynthesis(workspacePath)
    ...
}
```

For orchestrators, we need SESSION_HANDOFF.md instead of SYNTHESIS.md. This can be implemented by:
1. Adding a new tier "orchestrator" OR
2. Adding a config field `ExpectedArtifact string` (defaults to SYNTHESIS.md)

**Source:** `pkg/verify/check.go:156-209`

**Significance:** Verification is extensible. We can add orchestrator-specific verification (SESSION_HANDOFF.md exists) without changing the core verification flow.

---

### Finding 4: Spawn Mode Defaults Can Be Skill-Specific

**Evidence:** The spawn command already has mode selection logic in `spawn_cmd.go:767-780`:

```go
// Spawn mode: inline (blocking TUI), tmux (opt-in), or headless (default)
if inline {
    return runSpawnInline(...)
}
if tmux || attach {
    return runSpawnTmux(...)
}
// Default: Headless mode
return runSpawnHeadless(...)
```

For orchestrator-type skills, we want the opposite default: tmux visibility for interaction, not headless.

**Source:** `cmd/orch/spawn_cmd.go:767-780`

**Significance:** The mode selection is a simple conditional. We can add `if isOrchestratorSkill { tmux = true }` after skill loading to change the default without adding new subcommands.

---

### Finding 5: Session Commands Already Track Orchestrator State

**Evidence:** The `cmd/orch/session.go` provides `orch session start/status/end` which tracks:
- Goal (focus)
- Start time
- Spawns made during session

This is orchestrator-specific tracking. Spawnable orchestrator sessions would need similar tracking but inverted: the SESSION is the spawnable unit, not the spawns within it.

**Source:** `cmd/orch/session.go:80-115` (session start), `cmd/orch/session.go:300-356` (session end)

**Significance:** Orchestrator sessions need a different kind of tracking. Instead of spawning agents FROM a session, we're spawning the session itself. This suggests reusing the session state mechanism but triggered by `orch spawn meta-orchestrator "goal"` instead of `orch session start "goal"`.

---

## Synthesis

**Key Insights:**

1. **Skill-type detection is the key routing decision** - The skill loader already parses `skill-type` from frontmatter. Orchestrator-type skills (skill-type: policy/orchestrator) can be detected at spawn time and routed to different context templates and spawn modes.

2. **Infrastructure is additive, not replacing** - All the pieces exist (skill loading, context templates, verification, spawn modes). We're adding conditional branches for orchestrator-type skills, not creating parallel infrastructure.

3. **The three differences between worker and orchestrator spawns are:**
   - **Context template**: SPAWN_CONTEXT.md → ORCHESTRATOR_CONTEXT.md (or extended SPAWN_CONTEXT with orchestrator sections)
   - **Default spawn mode**: Headless → Tmux (orchestrators need interactive visibility)
   - **Completion artifact**: SYNTHESIS.md → SESSION_HANDOFF.md
   - **Beads tracking**: Required by default → Optional/different (orchestrators manage sessions, not issues)

4. **No new subcommand needed** - `orch spawn meta-orchestrator "goal"` can work with the existing spawn command by detecting the skill type and adjusting behavior. This is simpler than `orch spawn-orchestrator`.

**Answer to Investigation Question:**

The infrastructure changes needed for spawnable orchestrator sessions are:

1. **spawn_cmd.go**: Add skill-type detection after skill loading to:
   - Default to tmux mode (visible interaction)
   - Use orchestrator context template
   - Skip or modify beads tracking

2. **pkg/spawn/context.go**: Create ORCHESTRATOR_CONTEXT.md template (or extend SpawnContextTemplate with orchestrator mode):
   - Remove beads progress tracking
   - Add SESSION_HANDOFF.md instead of SYNTHESIS.md requirement
   - Include session goal and duration expectations
   - Reference `orch session end` instead of `/exit`

3. **pkg/verify/check.go**: Add orchestrator verification mode:
   - Check for SESSION_HANDOFF.md instead of SYNTHESIS.md
   - Verify session end reported (vs Phase: Complete)
   - Skip beads-dependent checks

4. **No new subcommand required**: `orch spawn <orchestrator-skill> "goal"` works with skill-type detection.

---

## Structured Uncertainty

**What's tested:**

- ✅ Skill loader parses skill-type field (verified: read loader.go code)
- ✅ Spawn context is template-driven with data config (verified: read context.go)
- ✅ Verification is tier-aware and extensible (verified: read check.go)
- ✅ Spawn mode selection is conditional (verified: read spawn_cmd.go)

**What's untested:**

- ⚠️ Whether skill-type detection is sufficient vs needing explicit --orchestrator flag
- ⚠️ Whether tmux visibility is actually better than headless for orchestrators
- ⚠️ Whether SESSION_HANDOFF.md structure works for spawned orchestrators
- ⚠️ Whether orchestrators need beads tracking at all

**What would change this:**

- If orchestrator sessions need issue tracking → add orchestrator-specific beads issue type
- If tmux visibility creates friction → keep headless default with --tmux opt-in
- If skill-type detection is unreliable → add explicit --orchestrator flag

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Skill-Type Detection with Conditional Behavior** - Detect orchestrator-type skills via frontmatter and conditionally modify spawn/complete behavior within existing commands.

**Why this approach:**
- Uses existing skill metadata infrastructure (no new parsing)
- Single code path with branches (simpler than parallel infrastructure)
- Gradual rollout possible (start with one orchestrator skill, extend later)

**Trade-offs accepted:**
- Skills must have correct frontmatter (skill-type: policy/orchestrator)
- Can't easily spawn a worker skill "as orchestrator" (but why would you?)

**Implementation sequence:**

1. **Phase 1: Skill-type detection in spawn_cmd.go**
   - After LoadSkillWithDependencies, parse metadata
   - Add `isOrchestratorSkill := metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"`
   - Use this flag to modify defaults (tmux mode, tracking behavior)

2. **Phase 2: ORCHESTRATOR_CONTEXT.md template**
   - Create new template in pkg/spawn/orchestrator_context.go
   - Include: session goal, SESSION_HANDOFF.md requirement, no beads instructions
   - Add Config field `IsOrchestrator bool` to route to this template

3. **Phase 3: Complete verification for orchestrators**
   - Add orchestrator verification in pkg/verify/check.go
   - Check SESSION_HANDOFF.md instead of SYNTHESIS.md
   - Verify session ended properly

4. **Phase 4: Dashboard/visibility updates** (optional)
   - Show orchestrator sessions differently than worker agents
   - Track orchestrator session goal/duration

### Alternative Approaches Considered

**Option B: New `orch spawn-orchestrator` subcommand**
- **Pros:** Explicit, no detection needed
- **Cons:** More code paths, harder to maintain, mental overhead for users
- **When to use instead:** If skill-type detection proves unreliable

**Option C: Extend `orch session` commands**
- **Pros:** Session commands already exist for orchestrators
- **Cons:** Conflates interactive sessions with spawnable sessions
- **When to use instead:** If we want ALL orchestrator sessions to be spawnable (not just skill-driven)

**Rationale for recommendation:** Option A (skill-type detection) is simplest because it leverages existing infrastructure and keeps a single spawn command. The complexity is in the spawn_cmd.go conditional, not in the user interface.

---

### Implementation Details

**What to implement first:**
- Skill-type detection in spawn_cmd.go (gate everything else on this)
- Create minimal ORCHESTRATOR_CONTEXT.md template
- Test with meta-orchestrator skill

**Things to watch out for:**
- ⚠️ Beads tracking: orchestrators may still want tracking, just different (session-level vs issue-level)
- ⚠️ Template complexity: don't over-engineer the orchestrator template initially
- ⚠️ Verification timing: SESSION_HANDOFF.md may need different verification (session end vs phase complete)

**Areas needing further investigation:**
- Should orchestrators get their own beads issue type?
- How should cross-project orchestrator sessions be tracked?
- Should orchestrator sessions have their own workspace path pattern?

**Success criteria:**
- ✅ `orch spawn meta-orchestrator "goal"` spawns in tmux by default
- ✅ ORCHESTRATOR_CONTEXT.md generated with appropriate instructions
- ✅ `orch complete` verifies SESSION_HANDOFF.md for orchestrator spawns
- ✅ No regression in worker skill spawning

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - Spawn command implementation, mode selection, skill loading
- `cmd/orch/session.go` - Session management, session start/end
- `cmd/orch/complete_cmd.go` - Completion verification, artifact checking
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template generation
- `pkg/spawn/config.go` - Spawn configuration, tier defaults
- `pkg/skills/loader.go` - Skill loading, metadata parsing
- `pkg/verify/check.go` - Completion verification, tier-aware checks

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` - Frame shift concept
- **Investigation:** `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md` - Prior architecture investigation

---

## Investigation History

**2026-01-04 10:00:** Investigation started
- Initial question: What infrastructure changes are needed for spawnable orchestrator sessions?
- Context: Prior investigation found orchestrators are structurally spawnable, this investigates concrete changes

**2026-01-04 10:30:** Analyzed spawn and session infrastructure
- Found skill-type field already supported in metadata
- Identified key differences between worker and orchestrator spawns

**2026-01-04 11:00:** Investigation complete
- Status: Complete
- Key outcome: Recommend skill-type detection with conditional behavior in spawn_cmd.go, no new subcommand needed

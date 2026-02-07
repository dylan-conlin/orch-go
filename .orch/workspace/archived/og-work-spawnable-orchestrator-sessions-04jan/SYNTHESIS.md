# Session Synthesis

**Agent:** og-work-spawnable-orchestrator-sessions-04jan
**Issue:** orch-go-d3nt
**Duration:** 2026-01-04 10:00 → 2026-01-04 11:30
**Outcome:** success

---

## TLDR

Investigated infrastructure changes needed for spawnable orchestrator sessions. Found that existing skill-type metadata parsing supports detection, and changes are additive: skill-type detection in spawn_cmd.go, ORCHESTRATOR_CONTEXT.md template, SESSION_HANDOFF.md verification. No new subcommand needed.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md` - Investigation answering the five key questions about orchestrator spawning infrastructure

### Files Modified
- None (investigation only)

### Commits
- None yet (investigation deliverable only)

---

## Evidence (What Was Observed)

- `pkg/skills/loader.go:21-30`: SkillMetadata already has `SkillType string` field that can be used for detection
- `pkg/spawn/context.go:26-272`: Template is data-driven, can be extended with orchestrator mode via Config field
- `pkg/verify/check.go:161-209`: Verification is tier-aware, can add orchestrator tier for SESSION_HANDOFF.md
- `cmd/orch/spawn_cmd.go:767-780`: Spawn mode selection is simple conditional, easy to change defaults
- `cmd/orch/session.go`: Session commands already track orchestrator-specific state

### Key Structural Observations

| Aspect | Worker Skill | Orchestrator Skill | Infrastructure Change |
|--------|-------------|-------------------|----------------------|
| Spawn mode | Headless (default) | Tmux (default) | Conditional in spawn_cmd.go |
| Context template | SPAWN_CONTEXT.md | ORCHESTRATOR_CONTEXT.md | New template file |
| Completion artifact | SYNTHESIS.md | SESSION_HANDOFF.md | Tier-aware check |
| Tracking | Beads issue | Session state | Skip/modify beads |

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md` - Answers five key questions about spawnable orchestrator infrastructure

### Decisions Made
- **Skill-type detection over new subcommand**: Using existing skill-type frontmatter field is simpler than creating `orch spawn-orchestrator`. The routing decision happens after skill loading.
- **Additive infrastructure**: All changes are conditional branches in existing code, not parallel systems

### Constraints Discovered
- Skills must have correct frontmatter (skill-type: policy/orchestrator) for detection
- Verification needs to handle two different artifacts (SYNTHESIS.md vs SESSION_HANDOFF.md)
- Beads tracking may still be useful for orchestrators, just at session level vs issue level

### Questions Answered

1. **Special subcommand vs skill-type detection?**
   → Skill-type detection. Use existing `skill-type: policy` from frontmatter. No new subcommand needed.

2. **ORCHESTRATOR_CONTEXT.md template design?**
   → Separate template or extended SpawnContextTemplate with orchestrator mode. Key differences: no beads tracking, SESSION_HANDOFF.md requirement, session goal focus.

3. **orch complete changes for SESSION_HANDOFF.md verification?**
   → Add orchestrator tier check in verify/check.go. Check SESSION_HANDOFF.md instead of SYNTHESIS.md.

4. **Default tmux visibility for orchestrators?**
   → Yes, change default in spawn_cmd.go: `if isOrchestratorSkill { tmux = true }`.

5. **Beads tracking differences?**
   → Orchestrators may skip beads or use session-level tracking instead of issue-level tracking.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement spawnable orchestrator sessions infrastructure
**Skill:** feature-impl
**Context:**
```
Implement skill-type detection in spawn_cmd.go for orchestrator-type skills.
Create ORCHESTRATOR_CONTEXT.md template with session-focused instructions.
Add orchestrator verification mode in verify/check.go.
Reference: .kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md
```

### Implementation Phases
1. **Phase 1**: Skill-type detection in spawn_cmd.go (~2h)
2. **Phase 2**: ORCHESTRATOR_CONTEXT.md template (~1h)
3. **Phase 3**: Complete verification for orchestrators (~1h)
4. **Phase 4**: Dashboard/visibility updates (optional, ~2h)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orchestrators get their own beads issue type for tracking?
- How should cross-project orchestrator sessions be tracked?
- Should orchestrator sessions have their own workspace path pattern (e.g., `.orch/orchestrator/{name}/`)?

**Areas worth exploring further:**
- How orchestrator skills interact with meta-orchestrator role (Dylan)
- Whether spawned orchestrators should have reduced authority vs interactive orchestrators

**What remains unclear:**
- Whether SESSION_HANDOFF.md structure is sufficient for spawned orchestrators
- Whether tmux visibility actually improves orchestrator spawns (vs headless with dashboard)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-work-spawnable-orchestrator-sessions-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md`
**Beads:** `bd show orch-go-d3nt`

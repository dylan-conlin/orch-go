# Session Synthesis

**Agent:** og-arch-design-backend-agnostic-14feb-2c53
**Issue:** orch-go-he8
**Duration:** 2026-02-14T10:00:00-08:00 → 2026-02-14T11:30:00-08:00
**Outcome:** success

---

## TLDR

Designed the backend-agnostic session contract for orch-go's spawn backends. AGENT_MANIFEST.json (already exists, already written by both OpenCode and Claude CLI backends) IS the universal session contract. Individual dotfiles (.beads_id, .tier, .spawn_time, .spawn_mode) are redundant and can be deprecated via a phased read-with-fallback migration, removing ~300 lines. The `.session_id` file stays separate as an infrastructure handle.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-14-design-backend-agnostic-session-contract.md` - Full design investigation with 6 forks navigated, substrate traces, implementation-ready checklist
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-14-backend-agnostic-session-contract.md` - Probe confirming model's state vs infrastructure distinction holds for both backends

### Files Modified
- None (design-only task)

### Commits
- (pending) architect: backend-agnostic session contract design

---

## Evidence (What Was Observed)

- `pkg/spawn/claude.go:12-70` — SpawnClaude does NOT write .session_id (no OpenCode session exists). Confirmed that Claude CLI creates tmux window but has no session object.
- `cmd/orch/spawn_cmd.go:1657-1717` — runSpawnClaude skips WriteSessionID, unlike runSpawnHeadless and runSpawnInline
- `pkg/spawn/session.go:161-196` — AgentManifest struct already contains all fields from dotfiles (beads_id, tier, spawn_time, spawn_mode) plus workspace_name, skill, project_dir, git_baseline, model
- `pkg/spawn/context.go:629-650` — WriteAgentManifest called for ALL backends
- `pkg/state/reconcile.go:128-136` — checkOpenCodeSession reads .session_id; when absent (Claude CLI), returns false gracefully
- `pkg/verify/backend.go:24-39` — VerifyBackendDeliverables routes via spawn_mode to either OpenCode or tmux verification
- All 7 consumers of .session_id handle missing values gracefully via fallthrough
- AGENT_MANIFEST.json is already the canonical source — dotfiles are legacy redundancy

### Verification Contract

The investigation file contains an implementation-ready checklist with:
- File targets (10 files to migrate)
- Acceptance criteria (6 conditions)
- Phasing (4 phases over ~2 weeks)
- VERIFICATION_SPEC not created (design-only task; implementation agent will create verification spec)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-14-design-backend-agnostic-session-contract.md` - Design with recommendation
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-14-backend-agnostic-session-contract.md` - Model probe

### Decisions Made
- **AGENT_MANIFEST.json is the session contract** — because it already exists, is written by all backends, and contains all state
- **Keep .session_id separate** — infrastructure handle ≠ immutable state; different write timing (post-spawn with retries)
- **Reject SESSION_STATE.json** — would duplicate AGENT_MANIFEST.json
- **Reject abstract SessionStore interface** — over-engineering for 2 backends, filesystem IS the interface

### Constraints Discovered
- AGENT_MANIFEST.json is immutable post-spawn — infrastructure handles (session_id, tmux window) have different write timing and should remain separate
- Tmux window targets are NOT persisted (discovered at runtime via FindWindowByBeadsID) — asymmetry with .session_id persistence is acceptable because both are infrastructure, not state

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + probe)
- [x] Design is implementation-ready (file targets, acceptance criteria, phasing)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-he8`

### Follow-up Implementation
**Issue:** Migrate session state consumers to ReadAgentManifest
**Skill:** feature-impl
**Context:**
```
Consolidate workspace state reads from individual dotfiles to AGENT_MANIFEST.json.
See .kb/investigations/2026-02-14-design-backend-agnostic-session-contract.md for
full design, file targets, and phasing. Phase 1: add ReadAgentState() fallback,
migrate status/complete/clean consumers.
```

---

## Unexplored Questions

**Questions that emerged during this session:**
- Should AGENT_MANIFEST.json include a `version` field for future schema evolution?
- When OpenCode metadata API ships (Phase 5 Step 2), should AGENT_MANIFEST.json gain a `session_id` field for parity, or keep the separation?
- Could AGENT_MANIFEST.json be extended with post-spawn fields (e.g., completion_time, outcome) or should that remain in beads only?

**What remains unclear:**
- Exact line count reduction (~300 is estimated from consumer analysis, actual may vary)
- Whether any consumers depend on dotfile write-order semantics (seems unlikely but not verified)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-backend-agnostic-14feb-2c53/`
**Investigation:** `.kb/investigations/2026-02-14-design-backend-agnostic-session-contract.md`
**Probe:** `.kb/models/agent-lifecycle-state-model/probes/2026-02-14-backend-agnostic-session-contract.md`
**Beads:** `bd show orch-go-he8`

# Session Synthesis

**Agent:** og-arch-architect-atomic-spawn-19feb-de89
**Issue:** orch-go-1083
**Outcome:** success

---

## Plain-Language Summary

Designed the architecture for atomic spawn — the mechanism that ensures every spawn either fully succeeds (beads tagged, workspace written, session created) or fully fails (nothing left behind). The current spawn is fire-and-forget: if session creation fails after workspace writes, you get a half-spawned agent that appears in the workspace but has no running session. This was the root cause of the 238-dead-agents incident.

The key insight is that atomic spawn must be two-phase because orch has 4 spawn backends (headless, inline, tmux, claude) that create sessions through fundamentally different mechanisms. Phase 1 (common) handles beads `orch:agent` tagging and workspace manifest writes with a rollback function. Phase 2 (per-backend) handles session creation and updates the manifest with the session ID. The existing `AGENT_MANIFEST.json` only needs a `SessionID` field added — it's already 90% of the "workspace manifest" the ADR describes.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

Key outcomes:
- Architecture recommendation: two-phase atomic spawn in `pkg/spawn/atomic.go`
- 5 decision forks navigated with substrate reasoning
- All existing infrastructure assessed (beads labels, manifest, session metadata)
- Implementation plan: 4 phases, ~575 lines of new/modified code
- 2 discovered work issues created (orch-go-1089, orch-go-1090)

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-19-design-atomic-spawn-workspace-manifest-orch-agent.md` — Full architecture investigation with 5 forks navigated
- `.kb/models/spawn-architecture/probes/2026-02-19-atomic-spawn-architecture-readiness.md` — Probe confirming/extending model claims about spawn infrastructure
- `.orch/workspace/og-arch-architect-atomic-spawn-19feb-de89/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-architect-atomic-spawn-19feb-de89/VERIFICATION_SPEC.yaml` — Verification specification

---

## Evidence (What Was Observed)

- `spawn_cmd.go` is 798 lines — under 1,500 threshold, no extraction needed
- `AGENT_MANIFEST.json` already written at spawn with most ADR-specified fields (missing: SessionID)
- OpenCode session metadata already written for headless/inline modes (beads_id, workspace_path, tier, spawn_mode)
- `DeleteSession` API exists in `pkg/opencode/client.go` — rollback feasible
- `AddLabel`/`RemoveLabel` fully implemented in `pkg/beads/` (RPC + CLI fallback)
- `ListArgs.Labels` supports filtering — `orch:agent` queryable via existing API
- No `SetSessionMetadata` client method exists — needed for tmux backend (discovered work: orch-go-1089)
- Claude backend has no OpenCode session — SessionID must be optional in manifest

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-19-design-atomic-spawn-workspace-manifest-orch-agent.md` — Architecture design
- `.kb/models/spawn-architecture/probes/2026-02-19-atomic-spawn-architecture-readiness.md` — Infrastructure readiness probe

### Decisions Made
- Two-phase atomic: Phase 1 (common beads+workspace) + Phase 2 (backend-specific session) — because 4 backends create sessions through fundamentally different mechanisms
- SessionID optional in manifest — because claude backend has no OpenCode session (escape hatch principle)
- Add to existing AgentManifest struct, don't rename — because it's 90% of what's needed
- `orch:agent` applied inside atomic function, not at issue creation — because the label means "agent is running," not "someone planned to spawn"
- New file `pkg/spawn/atomic.go`, not inline in extraction.go — because extraction.go is already 1,984 lines

### Constraints Discovered
- Claude backend cannot participate in session metadata (no OpenCode session)
- Tmux session metadata requires `SetSessionMetadata` client method (doesn't exist yet)
- Manifest must be written twice: initial (without SessionID), then updated (with SessionID) — because session creation happens after workspace writes

### Discovered Work
- orch-go-1089: Add SetSessionMetadata to OpenCode client (prerequisite for tmux metadata injection)
- orch-go-1090: Tmux backend: inject session metadata after session discovery (depends on 1089)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] Architecture investigation complete with all forks navigated
- [x] Implementation plan with 4 phases and file-level detail
- [x] Probe file with model impact analysis
- [x] Discovered work tracked (orch-go-1089, orch-go-1090)
- [x] Ready for `orch complete orch-go-1083`

**Implementation order:**
1. orch-go-1089 (SetSessionMetadata client method) — prerequisite
2. orch-go-1083 implementation — atomic spawn + manifest evolution + orch:agent tagging
3. orch-go-1090 (tmux metadata injection) — depends on 1089
4. orch-go-1085 (queryTrackedAgents) — consumes atomic spawn's outputs
5. orch-go-1087 (orch sessions for untracked) — Lane 2

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-architect-atomic-spawn-19feb-de89/`
**Investigation:** `.kb/investigations/2026-02-19-design-atomic-spawn-workspace-manifest-orch-agent.md`
**Beads:** `bd show orch-go-1083`

# Session Synthesis

**Agent:** og-arch-evaluate-lifecycle-management-13feb-d001
**Issue:** orch-go-i2h
**Duration:** 2026-02-13T10:05 → 2026-02-13T11:15
**Outcome:** success

---

## TLDR

Evaluated orch's lifecycle management (~8,800 lines) and found it's primarily compensating for OpenCode's missing session management features. Proposed a three-bucket ownership model (Own/Accept/Lobby) that can reduce complexity by ~40% through immediate simplification (merge 3 ghost types, reduce 7 clean flags to 3, eliminate registry) while filing upstream issues for long-term reduction.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-13-inv-evaluate-lifecycle-management-orch-own.md` - Full investigation with 5 findings, three-bucket model, implementation roadmap
- `.orch/workspace/og-arch-evaluate-lifecycle-management-13feb-d001/SYNTHESIS.md` - This file

### Files Modified
- None (analysis-only session)

### Commits
- Single commit with investigation and synthesis artifacts

---

## Evidence (What Was Observed)

- **8,800+ lines of lifecycle code** across 9 files (clean_cmd.go, complete_cmd.go, abandon_cmd.go, serve_agents.go, registry.go, client.go, sse.go, check.go, shared.go)
- **7 clean flags** for different state layers (windows, phantoms, ghosts, verify-opencode, investigations, stale, sessions)
- **3 ghost types** (phantom, ghost, orphan) all caused by one root cause: OpenCode sessions persist indefinitely
- **Registry methods deprecated with zero callers** - Abandon(), Complete(), Remove() exist but are never called
- **11 verification gates** in complete_cmd.go represent genuine orchestration value
- **Four-layer state model** conflates two concerns: state (beads, workspace) and infrastructure (OpenCode sessions, tmux)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-13-inv-evaluate-lifecycle-management-orch-own.md` - Three-bucket ownership model for lifecycle management

### Decisions Made
- Lifecycle responsibilities should follow three-bucket model: OWN (verification, phase tracking, workspace, beads), ACCEPT (session persistence, SSE-only completion), LOBBY (session TTL, metadata API, state endpoint)
- The four-layer state model should be refined to distinguish state (beads + workspace) from infrastructure (OpenCode + tmux)
- Registry should be eliminated — workspace `.session_id` files serve the same lookup function

### Constraints Discovered
- OpenCode sessions never expire (no TTL) — drives all ghost/phantom/orphan detection code
- OpenCode doesn't expose session state (busy/idle) via HTTP — forces SSE for completion detection
- OpenCode doesn't store arbitrary metadata — forces registry and workspace metadata files
- Ghost types are symptoms of one root cause — three algorithms for one problem is Coherence Over Patches smell

### Key Distinction Surfaced
- **State vs Infrastructure**: Beads and workspace files represent business meaning (what work was done). OpenCode sessions and tmux windows represent execution resources. Treating infrastructure as state creates the reconciliation burden. This is the "Evolve by Distinction" the system needed.

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, ready for orchestrator to promote to decision)

### If Close
- [x] Investigation file has `**Phase:** Complete`
- [x] SYNTHESIS.md created
- [x] All findings documented with evidence
- [ ] Ready for `orch complete orch-go-i2h`

### Follow-up Work (for orchestrator to triage)

1. **Promote to decision:** Create `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` from investigation recommendations
2. **Phase 1 implementation:** Simplify clean to 3 modes (feature-impl, 1-2 days)
3. **Phase 2 implementation:** Eliminate registry (feature-impl, 2-3 days)
4. **Phase 3 upstream:** File Session TTL, Metadata API, State endpoint issues on OpenCode GitHub
5. **Phase 4 model update:** Revise agent-lifecycle-state-model.md with state vs infrastructure distinction

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does `serve_agents.go`'s Priority Cascade simplify if we formally distinguish state from infrastructure?
- Can the 11 verification gates in complete_cmd.go benefit from the completed pipeline refactoring pattern?
- Would OpenCode accept these upstream contributions? What's their stance on session lifecycle management?

**Areas worth exploring further:**
- Cross-project spawn workspace files — do they reliably exist in the expected project directory for all edge cases?
- Whether `orch status` can work purely from beads + workspace files (no registry, no OpenCode API) for a "degraded but functional" mode

**What remains unclear:**
- OpenCode's design philosophy on session persistence — is "sessions never expire" a feature or an oversight?
- Whether Dylan wants to invest in OpenCode upstream contributions vs accepting the compensation code long-term

---

## Verification Contract

**Investigation:** `.kb/investigations/2026-02-13-inv-evaluate-lifecycle-management-orch-own.md`

**Key outcomes to verify:**
1. Investigation file exists with Phase: Complete
2. Three-bucket model clearly defines Own/Accept/Lobby boundaries
3. Implementation roadmap has 4 phases with authority classification
4. Findings are evidence-based (line counts, code references, decision references)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-evaluate-lifecycle-management-13feb-d001/`
**Investigation:** `.kb/investigations/2026-02-13-inv-evaluate-lifecycle-management-orch-own.md`
**Beads:** `bd show orch-go-i2h`

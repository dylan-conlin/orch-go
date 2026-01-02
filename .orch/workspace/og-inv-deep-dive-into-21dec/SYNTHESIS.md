# Session Synthesis

**Agent:** og-inv-deep-dive-into-21dec
**Issue:** orch-go-kqjm
**Duration:** ~1.5 hours
**Outcome:** success

---

## TLDR

Deep investigation into inter-agent communication architecture revealed the system's "thrashing" between tmux and HTTP is actually coherent evolution toward dual-mode architecture: tmux for visual access, HTTP API for programmatic state, beads comments as lifecycle authority, and registry reduced to session_id caching only.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-deep-dive-inter-agent-communication.md` - Comprehensive chronicle-style investigation answering 5 key architecture questions

### Files Modified
- None (investigation only)

### Commits
- (Investigation file will be committed)

---

## Evidence (What Was Observed)

- **55 kn entries** traced operational decisions over 3 days showing coherent evolution
- **52+ investigations** documented specific architectural decisions and their rationale
- **100+ commits** in 3 days showed deliberate movement toward dual-mode architecture
- **Four-layer state problem** identified: tmux windows, OpenCode in-memory, OpenCode on-disk, beads comments
- **Registry drift** caused by being a fifth layer caching four independent systems
- **Beads comments** are the ONLY authoritative source for lifecycle (Phase: Complete)
- **tmux provides irreplaceable visual access** for orchestrating parallel agents

### Key Findings

| Question | Answer |
|----------|--------|
| Source of truth for agent state? | Beads comments (lifecycle), OpenCode API (sessions), tmux (visual) |
| Why return to tmux? | Visual access for parallel agents - HTTP gives data, not visibility |
| Hidden tmux dependencies? | Window-per-agent isolation, full TUI experience, real-time visual monitoring |
| Is beads comments right for lifecycle? | Yes - session status (busy/idle) cannot reliably indicate completion |
| Clean architecture? | Dual-mode: tmux for visual, HTTP for programmatic, beads for lifecycle |

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-deep-dive-inter-agent-communication.md` - Comprehensive architecture investigation

### Decisions Made
- Dual-mode architecture (tmux + HTTP) is correct because each serves distinct, irreplaceable needs
- Registry should be reduced to session_id caching only (not state management)
- Beads comments are the definitive lifecycle record (Phase: Complete)

### Constraints Discovered
- Four-layer state (tmux/OpenCode mem/OpenCode disk/beads) has independent lifecycles
- Registry as fifth layer can never stay synchronized with four independent systems
- Session status (busy/idle) cannot reliably indicate agent completion (kn-bef2d9)

### Externalized via `kn`
- `kn decide "Dual-mode architecture (tmux for visual, HTTP for programmatic) is the correct design" --reason "Investigation confirmed each mode serves distinct, irreplaceable needs"` - kn-26c3e5

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation file has comprehensive findings and recommendations
- [x] Ready for `orch complete orch-go-kqjm`

### Follow-up Work (Optional)
If implementation is desired based on this investigation:

1. **Complete workspace-local session_id storage** - Per Phase 3 investigation recommendation
2. **Update `orch status` to query OpenCode API directly** - Remove registry dependency
3. **Reduce registry to session_id cache only** - Or remove entirely

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does attach mode handle OpenCode server restarts mid-session?
- What happens to on-disk sessions when orphaned from tmux windows?
- Should there be an on-demand "reconcile" command instead of automatic reconciliation?

**Areas worth exploring further:**
- Performance of direct-query vs registry-cache with 10+ concurrent agents
- Whether workspace-local session_id files introduce new failure modes
- Production stability of dual-mode with high agent counts

**What remains unclear:**
- Exact latency characteristics of OpenCode API under load
- Edge cases around session restoration after server restart
- Whether any external tooling depends on registry format

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-deep-dive-into-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-deep-dive-inter-agent-communication.md`
**Beads:** `bd show orch-go-kqjm`

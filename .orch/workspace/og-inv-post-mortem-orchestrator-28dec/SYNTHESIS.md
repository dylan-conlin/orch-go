# Session Synthesis

**Agent:** og-inv-post-mortem-orchestrator-28dec
**Issue:** orch-go-3i8u
**Duration:** ~45 minutes
**Outcome:** success

---

## TLDR

Post-mortem of 4 orchestrator sessions revealed three failure modes: (1) stale binary inheritance caused 30+ min debugging already-fixed bugs, (2) documentation drift left 12+ orch features undiscovered, (3) orchestrators lack runtime context that workers receive. Root cause: features exist but aren't surfaced at session start. Recommended three-tier prevention: SessionStart staleness warning, orchestrator server context injection, and documentation drift linting.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-post-mortem-orchestrator-session-inefficiency.md` - Full post-mortem with findings, timeline, and recommendations

### Files Modified
- None (analysis investigation only)

### Commits
- (pending) Post-mortem investigation file

---

## Evidence (What Was Observed)

- **Stale binary timeline verified:** Session A fixed serve.go at 12:23:58 (d948e5d6), Session B started at 12:43:14 with pre-fix binary, spent 22+ min on phantom problem
- **12+ commands undocumented:** `orch servers init/up/down`, `orch kb ask`, `orch sessions search`, `orch doctor --fix`, etc.
- **Port mismatch confirmed:** Orchestrator skill had 3333, `orch serve` runs on 3348
- **Three-layer divergence documented:** `orch status` showed 6 agents, API showed 3, dashboard showed 1
- **Context asymmetry verified:** `GenerateServerContext()` exists in spawn code but orchestrators never receive it

### Tests Run
```bash
# Timeline reconstruction
git log --oneline --since="2025-12-28 12:00" --until="2025-12-28 14:00"
# Confirmed: d948e5d6 at 12:23:58, Session B started at 12:43:14 with stale binary

# Staleness detection exists
orch version --source
# Output: status: ✓ UP TO DATE (mechanism works, just not surfaced)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-post-mortem-orchestrator-session-inefficiency.md` - Full post-mortem

### Decisions Made
- Three-tier prevention is the recommended approach (SessionStart + context injection + lint)

### Constraints Discovered
- **Stale binary is self-hiding:** Running stale binary can't show you the fix exists
- **Documentation drift is systematic:** Features added without updating CLAUDE.md/skill
- **Workers get context orchestrators don't:** GenerateServerContext only for spawned agents

### Patterns Identified

**The Circular Debug Pattern:**
```
Session A fixes bug → Doesn't deploy fix
         ↓
Session B starts with stale binary → Sees symptoms of unfixed bug
         ↓
Session B debugs for 30 min → Discovers same root cause
         ↓
Session B fixes + deploys → Problem finally resolved
```

**The Meta-Gap Pattern:**
```
Feature built → Not added to session context
         ↓
Orchestrator hits problem → Doesn't know feature exists
         ↓
Orchestrator builds workaround → Feature still not surfaced
```

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] D.E.K.N. summary filled
- [x] Investigation file has `**Status:** Complete`
- [x] Priority-ranked action items documented
- [ ] Ready for `orch complete orch-go-3i8u`

### Priority Action Items for Orchestrator

| Priority | Action | Effort |
|----------|--------|--------|
| **P0** | Add stale binary warning to SessionStart hook | 30 min |
| **P0** | Unify status determination in `pkg/state/reconcile.go` | 2h |
| **P1** | Add server context to orchestrator SessionStart | 1h |
| **P1** | Document 12+ missing commands in CLAUDE.md/skill | 2h |
| **P2** | Add `orch lint --skills --check-commands` | 2h |

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why didn't pre-commit hook prevent Session B staleness? (Hook might not run on session start)
- Should `orch spawn` refuse to run with stale binary? (Aggressive but prevents problem)
- How common is this pattern across Dylan's other projects? (May need cross-project audit)

**Areas worth exploring further:**
- Session overlap detection - warn when starting in project with recent uncommitted changes
- Unified "orchestrator session context" that matches what workers get
- Automatic documentation from `--help` output

**What remains unclear:**
- Whether launchd daemon is affected by staleness (PATH suggests yes)
- Whether SessionStart latency (~100ms) is acceptable

---

## Session Metadata

**Skill:** investigation
**Model:** (inherited from spawn)
**Workspace:** `.orch/workspace/og-inv-post-mortem-orchestrator-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-post-mortem-orchestrator-session-inefficiency.md`
**Beads:** `bd show orch-go-3i8u`

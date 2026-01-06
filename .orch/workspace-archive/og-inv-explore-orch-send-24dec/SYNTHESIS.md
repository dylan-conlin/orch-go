# Session Synthesis

**Agent:** og-inv-explore-orch-send-24dec
**Issue:** orch-go-imi5
**Duration:** 08:48 → 09:15
**Outcome:** success

---

## TLDR

Investigated `orch send` vs spawn boundaries. Found sessions have no TTL (persist indefinitely), completed agents accept Q&A, and large sessions (51k chars) maintain full context. The send vs spawn decision should be based purely on task relatedness, not technical constraints.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-24-inv-explore-orch-send-vs-spawn.md` - Complete investigation with D.E.K.N. summary, 4 findings, synthesis, and recommendations

### Files Modified
- None

### Commits
- (Pending) Investigation file to be committed

---

## Evidence (What Was Observed)

- Sessions from Nov 27 (27 days ago) exist on disk at `~/.local/share/opencode/storage/session/`
- Successfully sent message to completed session (orch-go-99lk, Phase: Complete, 44 min idle) - received coherent response
- Successfully sent message to 2-day old session - agent correctly recalled its task
- Session with 239 messages / 51,347 characters responded with full context preserved
- Cross-project sessions are inaccessible (error: "session not found in OpenCode")

### Tests Run
```bash
# Sent to completed session - SUCCESS
orch send orch-go-99lk "What was the main finding?"
# Response: coherent summary of investigation findings

# Sent to 2-day old session - SUCCESS  
orch send ses_4bb0a1dcdffeoLsSbI7A1n1Jyd "What task were you working on?"
# Response: accurate recall of "implementing FAILURE_REPORT.md template"

# Sent to large session (51k chars) - SUCCESS
orch send ses_4b93901e9ffeQb37JlzsFl83Wf "What was the main task?"
# Response: accurate recall of "migrating skills to skillc-managed structure"
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-24-inv-explore-orch-send-vs-spawn.md` - Complete investigation on send vs spawn boundaries

### Decisions Made
- Decision: Send vs spawn heuristic should be based on task relatedness, not session age/size - because technical constraints (TTL, context degradation) are non-issues in practice

### Constraints Discovered
- Sessions are scoped per-project directory - cannot send to sessions from other projects without directory switching
- Cross-project session access requires explicit `x-opencode-directory` header

### Externalized via `kn`
- (See below - to be run before completion)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests performed (sent messages to 4 different sessions, observed responses)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-imi5`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What happens if you send a work request (not just Q&A) to a Phase: Complete agent? Does it need to re-report phase?
- At what point does context saturation actually occur? (Tested 51k chars, Claude supports ~200k tokens)

**Areas worth exploring further:**
- Adding context-size indicator to `orch status` for visibility
- Automatic suggestions when context is approaching limits

**What remains unclear:**
- Whether completed agents should re-enter phase reporting if given new work (vs just Q&A)

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus
**Workspace:** `.orch/workspace/og-inv-explore-orch-send-24dec/`
**Investigation:** `.kb/investigations/2025-12-24-inv-explore-orch-send-vs-spawn.md`
**Beads:** `bd show orch-go-imi5`

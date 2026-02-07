# Session Synthesis

**Agent:** og-arch-review-design-coaching-11jan-f74a
**Issue:** orch-go-rcah9
**Duration:** 2026-01-11 18:36 → 2026-01-11 18:50
**Outcome:** success

---

## TLDR

Investigated why the coaching plugin injection system has 8 bugs and 2 abandonments, finding that the root cause is architectural coupling between observation (metric collection) and intervention (coaching injection). Recommended separating injection into an independent daemon to eliminate the entire class of restart/state bugs.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md` - Architectural review with findings and recommendations

### Files Modified
- None (architectural review only, no code changes)

### Commits
- `eeab1142` - architect: Review design of coaching plugin injection system

---

## Evidence (What Was Observed)

- **Metrics are persistent** (plugins/coaching.ts:391-399) - written to ~/.orch/coaching-metrics.jsonl, survive restart
- **Session state is ephemeral** (plugins/coaching.ts:942) - sessions Map is in-memory, lost on restart
- **Injection coupled to observation** (plugins/coaching.ts:1287, 1297) - flushMetrics only called from tool.execute.after hook
- **8+ commits in 2 days** (git log --grep="coaching") - all tactical fixes, no architectural changes
- **2 abandoned attempts** (.kb/investigations/*coaching*.md) - investigation files are empty templates, never completed
- **Dashboard reads persistent metrics** (serve_coaching.go:36-76) - independent of plugin state
- **Plugin writes persistent metrics** (coaching.ts:391-399) - but injection requires ephemeral state

### Tests Run
```bash
# No tests run - architectural analysis only
# Verified code paths via grep and read
grep -n "flushMetrics\(" plugins/coaching.ts
grep -n "injectCoachingMessage" plugins/coaching.ts
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md` - Architectural review documenting root cause and recommendation

### Decisions Made
- **Coherence Over Patches applies**: 8 bugs in same area = redesign needed, not 8 more fixes
- **Separation of Concerns violated**: observation (passive) and intervention (active) should have separate code paths and lifecycles
- **Daemon is the right architecture**: independent process reading persistent metrics eliminates coupling and restart brittleness

### Constraints Discovered
- Plugin runs in OpenCode server process - can't easily persist state
- Metrics file is append-only JSONL - daemon can read this
- OpenCode API provides client.session.prompt() - daemon can inject
- Worker sessions must not receive injections - daemon needs detection logic

### Externalized via `kb`
- None - architectural review doesn't produce quick knowledge entries
- Recommendation is to promote this investigation to a decision record

---

## Next (What Should Happen)

**Recommendation:** escalate

### Escalation Details

**Question:** Should we implement the architectural separation (injection daemon) or continue patching symptoms?

**Options:**
1. **Implement daemon architecture (recommended)**
   - **Pros:** Eliminates entire class of bugs, aligns with Coherence Over Patches principle, makes system conceptually simpler
   - **Cons:** Adds operational complexity (new process to manage), initial implementation cost

2. **Continue tactical fixes**
   - **Pros:** Smaller incremental changes, no new infrastructure
   - **Cons:** Doesn't address root cause, bugs will keep emerging, violates Coherence Over Patches principle

**My Recommendation:** Option 1 (daemon). This is a canonical case for architectural redesign. The investigation found that the current design has THREE different lifecycles (metrics file, plugin state, OpenCode sessions) with no coordination, and injection depends on the wrong one (ephemeral plugin state instead of persistent metrics). The daemon approach eliminates this mismatch and prevents future bugs in this class.

**Next Steps if Approved:**
1. Create beads epic for daemon implementation
2. Break into phases: (1) daemon skeleton, (2) injection logic, (3) state tracking, (4) integration test, (5) cleanup plugin
3. Spawn feature-impl agent with daemon design from investigation file

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should daemon inject once per pattern or continuously until resolved? (impacts user experience - once = less spam, continuous = persistent reminder)
- How to handle multiple orchestrator sessions running simultaneously? (inject into all, or only "active" one based on recency?)
- Should daemon have its own dashboard section showing injection history? (useful for debugging, but adds scope)

**Areas worth exploring further:**
- Could the same "separate observation from intervention" pattern apply to other behavioral monitoring features?
- Is there a generalized "behavioral monitoring framework" we should extract from this?
- Should the daemon be merged with existing `orch daemon` or be a separate process?

**What remains unclear:**
- Polling frequency trade-off (30s matches dashboard, but is that optimal for injection?)
- Whether daemon should use injection-state.jsonl or in-memory state for tracking what it already injected
- Integration with overmind (dev) vs systemd (prod) for lifecycle management

---

## Session Metadata

**Skill:** architect
**Model:** sonnet
**Workspace:** `.orch/workspace/og-arch-review-design-coaching-11jan-f74a/`
**Investigation:** `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md`
**Beads:** `bd show orch-go-rcah9`

# Session Synthesis

**Agent:** og-feat-orchestrator-coaching-plugin-10jan-2249
**Issue:** orch-go-zyuik
**Duration:** 2026-01-10 01:57 → 2026-01-10 02:06
**Outcome:** success

---

## TLDR

Implemented minimal orchestrator coaching plugin prototype: OpenCode plugin tracks behavioral patterns (context ratio, action ratio, analysis paralysis) → JSONL storage → API endpoint → dashboard metrics card. Hypothesis: do quantified metrics drive orchestrator behavior change?

---

## Delta (What Changed)

### Files Created
- `plugins/coaching.ts` (293 lines) - OpenCode plugin tracking tool usage patterns
- `cmd/orch/serve_coaching.go` (236 lines) - API endpoint aggregating coaching metrics
- `web/src/lib/stores/coaching.ts` (59 lines) - Svelte store for coaching data
- `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` (322 lines) - Design document
- `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-prototype.md` - Investigation findings

### Files Modified
- `cmd/orch/serve.go:350` - Registered `/api/coaching` endpoint
- `web/src/routes/+page.svelte` - Added coaching metrics card UI (60 lines)

### Commits
- `d6d1f720` - feat: add orchestrator coaching plugin prototype

---

## Evidence (What Was Observed)

### Investigation Findings
- OpenCode plugins can only hook tool calls (Read, Edit, Bash), not LLM responses
- Must use behavioral proxies: tool usage patterns as signals for Level 1→2 detection
- Existing `action-log.ts` and `evidence-hierarchy.ts` provide proven JSONL + plugin patterns

### Design Decisions
- **Behavioral Proxies Chosen:**
  - Context ratio: `kb context` calls per spawn (strategic reasoning proxy)
  - Action ratio: Edit/Write per Read/Grep (option theater proxy)
  - Analysis paralysis: Tool repetition sequences (3+ same tool)
- **Storage:** JSONL at `~/.orch/coaching-metrics.jsonl` (follows action-log pattern)
- **Thresholds:**
  - Context ratio: >0.7 good, 0.4-0.7 warning, <0.4 poor
  - Action ratio: >0.5 good, 0.3-0.5 warning, <0.3 poor
  - Analysis paralysis: 0 good, 1-2 warning, 3+ poor

### Implementation Evidence
- Plugin deployed via symlink: `~/.config/opencode/plugin/coaching.ts`
- API compiles: `make install` succeeded
- Endpoint responds: `curl https://localhost:3348/api/coaching` returns JSON
- Dashboard integrated: Coaching section added after StatsBar

### Tests Run
```bash
# Build verification
make install
# Output: ✓ Installed to ~/bin/orch

# API endpoint test
curl -k https://localhost:3348/api/coaching
# Output: {"session":{},"metrics":{},"coaching":[]} (empty as expected, no metrics yet)

# Plugin deployment
ls -la ~/.config/opencode/plugin/coaching.ts
# Output: symlink → /Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts
```

---

## Knowledge (What Was Learned)

### New Artifacts
- Design doc: `docs/designs/2026-01-10-orchestrator-coaching-plugin.md`
- Investigation: `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-prototype.md`

### Constraints Discovered
- OpenCode plugins have no access to LLM response text, only tool call metadata
- Detection must work from tool sequences, not text analysis
- Cannot directly detect "option theater" phrases - must infer from tool behavior

### Pattern Applied
- **Plugin → JSONL → API → Dashboard** (follows existing action-log pattern)
- **Tool hook tracking** (follows evidence-hierarchy pattern)
- **Behavioral proxies** (ratios and sequences as signals for qualitative patterns)

### Externalized via kb quick
- No new kb quick entries needed - follows established patterns from prior plugins

---

## Next (What Should Happen)

**Recommendation:** close

### Visual Verification Required
- [ ] **UI Verification:** Since `web/` files were modified, manual browser verification needed
- [ ] Restart servers: `orch servers restart orch-go`
- [ ] Open dashboard: http://localhost:5188
- [ ] Verify coaching card appears (will be empty until metrics generated)
- [ ] Check 666px width constraint (half MacBook Pro screen)

### Hypothesis Testing
- Plugin will start tracking metrics once orchestrator uses tools
- Dashboard will update every 30 seconds with new metrics
- Next step: Run for 1 week, measure if behavior changes

### If Close
- [x] All deliverables complete (plugin + API + dashboard + docs)
- [x] Code compiles and API responds
- [x] Commits pushed: `d6d1f720`
- [x] Investigation file marked Complete
- [ ] UI visual verification (must do before closing)
- [x] Ready for `orch complete orch-go-zyuik`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to measure actual behavior change? Need baseline metrics before/after comparison.
- Should coaching messages be injected into session (like evidence-hierarchy) vs just dashboard?
- What's the false positive rate for these behavioral proxies?

**Areas worth exploring further:**
- Integration with `action-log.ts` - both track tool usage, could deduplicate
- More sophisticated pattern detection (ML-based vs simple thresholds)
- Session-level coaching reports (aggregate over time, not just current session)

**What remains unclear:**
- Optimal threshold values (chose based on intuition, need real-world tuning)
- Effectiveness of behavioral proxies (do they correlate with actual Level 1→2 patterns?)
- Dashboard visibility impact (will orchestrators actually check it?)

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude
**Workspace:** `.orch/workspace/og-feat-orchestrator-coaching-plugin-10jan-2249/`
**Investigation:** `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-prototype.md`
**Design:** `docs/designs/2026-01-10-orchestrator-coaching-plugin.md`
**Beads:** `bd show orch-go-zyuik`
**Commit:** `d6d1f720` - feat: add orchestrator coaching plugin prototype

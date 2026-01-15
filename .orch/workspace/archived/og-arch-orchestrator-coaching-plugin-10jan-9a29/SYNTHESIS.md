# Session Synthesis

**Agent:** og-arch-orchestrator-coaching-plugin-10jan-9a29
**Issue:** Ad-hoc (no beads tracking)
**Duration:** 2026-01-10 ~02:00 → ~03:00
**Outcome:** success

---

## TLDR

Explored technical implementation options for orchestrator coaching plugin. Found backend infrastructure (plugin + API) is 100% complete; only missing worker filtering in plugin and dashboard UI. Recommended incremental approach: add filtering + UI, test behavioral proxies for 1 week, defer session correlation until validated.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md` - Technical design investigation with trade-offs and recommendations

### Files Modified
- None (investigation-only session)

### Commits
- Not yet committed (to be committed before /exit)

---

## Evidence (What Was Observed)

### OpenCode Plugin API Surface (Finding 1)
- **Hooks available:** `config`, `event`, `tool.execute.before`, `tool.execute.after` (verified in plugins/orchestrator-session.ts, event-test.ts, coaching.ts)
- **No LLM response access:** Plugins see tool calls only, not free-text responses (fundamental constraint)
- **Source:** plugins/coaching.ts:1-297, plugins/orchestrator-session.ts:1-218

### Backend Infrastructure Complete (Finding 2)
- **Plugin exists:** plugins/coaching.ts implements tool tracking, JSONL persistence, flush logic
- **API exists:** cmd/orch/serve_coaching.go:1-243 implements /api/coaching endpoint
- **API registered:** cmd/orch/serve.go:352-353 registers coaching endpoint
- **Source:** Verified via grep and file reads

### Session Management (Finding 3)
- **Two session concepts:** (1) OpenCode session ID (ephemeral, per-agent), (2) Orchestrator session in ~/.orch/session.json (persistent, goal-oriented)
- **Metrics indexed by:** OpenCode session ID (see plugins/coaching.ts:222-242)
- **Orchestrator session tracked:** pkg/session/session.go implements session store with goal, start time, spawns
- **Source:** pkg/session/session.go:1-437, plugins/coaching.ts:109-117

### Worker Detection (Finding 4)
- **Three signals:** (1) `ORCH_WORKER=1` env var, (2) `SPAWN_CONTEXT.md` exists, (3) path contains `.orch/workspace/`
- **Implementation:** plugins/orchestrator-session.ts:76-100 shows `isWorker()` function
- **Env var set by:** cmd/orch/spawn_cmd.go:1323-1324 when spawning agents
- **Source:** Verified via grep and file reads

### Tests Run
None (investigation-only session, no code changes)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md` - Complete technical design investigation with 4 findings, synthesis, and implementation recommendations

### Decisions Made
- **Incremental approach recommended:** Add worker filtering + dashboard UI first, defer session correlation until behavioral proxies validate (test hypothesis first, optimize second principle)
- **Worker filtering is critical:** Without it, agent tool usage pollutes orchestrator metrics (Finding 4)
- **Behavioral proxies only option:** OpenCode plugins cannot analyze LLM response text, must use tool patterns as proxy (fundamental constraint, Finding 1)

### Constraints Discovered
- **OpenCode plugin limitation:** No access to LLM free-text responses, only tool calls (Finding 1)
- **Session mismatch:** OpenCode session IDs are ephemeral (reset on restart), orchestrator sessions persist across multiple OpenCode sessions (Finding 3)
- **Dashboard pattern established:** All stores follow same pattern (fetch on mount, poll for updates) - see beads.ts, usage.ts

### Externalized via `kb`
- Not applicable (investigation findings documented in investigation file, not kb quick entries)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created with trade-offs and recommendations)
- [x] Tests passing (N/A - investigation only)
- [x] Investigation file has `**Phase:** Complete` (verified)
- [x] Ready for `orch complete` (no beads tracking, ad-hoc spawn)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- **How to optimize session correlation?** - If behavioral proxies validate, what's the best way to map OpenCode session IDs to orchestrator session goals? (Deferred to post-validation)
- **Threshold tuning methodology?** - Current thresholds (context_ratio >0.7 = good) are guesses. What's the process for tuning based on real usage? (Needs data collection first)
- **Coaching message effectiveness?** - Are the generated messages actionable? Do they drive behavior change? (Hypothesis to test after deployment)

**Areas worth exploring further:**
- **Text analysis via OpenCode API** - Could poll message history API for "option theater" text patterns as secondary validation (not primary detection)
- **Cross-project metric aggregation** - Should coaching metrics aggregate across projects or stay project-specific?

**What remains unclear:**
- **Behavioral proxy correlation** - Do tool patterns actually predict Level 1→2 behaviors? Hypothesis is untested (requires 1-week deployment)

---

## Session Metadata

**Skill:** architect
**Model:** sonnet (default)
**Workspace:** `.orch/workspace/og-arch-orchestrator-coaching-plugin-10jan-9a29/`
**Investigation:** `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md`
**Beads:** N/A (ad-hoc spawn, no tracking)

# Session Synthesis

**Agent:** og-inv-model-selection-issue-23dec
**Issue:** orch-go-cyzr
**Duration:** 2025-12-23 11:00 → 2025-12-23 12:00
**Outcome:** success

---

## TLDR

Investigated why headless spawns ignore --model opus flag and use sonnet instead. Root cause: OpenCode HTTP API doesn't support model selection in POST /session endpoint. Fix: Use CLI mode (opencode run --format json) for headless spawns instead of pure HTTP API.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md` - Investigation documenting API limitation and fix approach

### Files Modified
- None (investigation only, no implementation)

### Commits
- Investigation file to be committed at completion

---

## Evidence (What Was Observed)

- Created 3 test sessions via `curl -X POST /session` with model parameter set to opus
- All test sessions used modelID: "claude-sonnet-4-5-20250929" despite opus being requested
- Traced orch-go code from `spawnModel` flag → `model.Resolve()` → `CreateSession()` - all correct
- Verified CLI mode (--inline, --tmux) uses `opencode run --model` which works correctly
- Confirmed via `opencode run --help` that --model flag exists and is documented

### Tests Run
```bash
# Test 1: API with model parameter
curl -X POST http://127.0.0.1:4096/session \
  -d '{"model":"anthropic/claude-opus-4-5-20251101"}'
# Result: Session created but used sonnet

# Test 2: Check actual model used
curl -s http://127.0.0.1:4096/session/{id}/message | jq '.[-1].info.modelID'
# Result: "claude-sonnet-4-5-20250929" (NOT opus)

# Test 3: Verify CLI supports --model
opencode run --help | grep -A 3 "model"
# Result: -m, --model flag confirmed
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md` - Documents OpenCode API limitation and recommended fix

### Decisions Made
- Decision: This is an OpenCode API bug, not an orch-go bug
  - Rationale: Traced code confirms orch-go correctly passes model; API tests confirm API ignores it
- Decision: Recommend CLI-based headless spawn as fix
  - Rationale: Reuses existing BuildSpawnCommand infrastructure, proven to work in inline/tmux modes

### Constraints Discovered
- OpenCode HTTP API doesn't support model selection (POST /session ignores model field)
- Model selection only works via CLI flags (opencode run --model)
- Subprocess management required for model selection in headless mode

### Externalized via `kn`
- None (investigation findings captured in .kb/investigations/ which is the correct artifact)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix headless spawn model selection via CLI mode
**Skill:** feature-impl
**Context:**
```
Modify runSpawnHeadless in cmd/orch/main.go to use BuildSpawnCommand (like inline mode) 
instead of CreateSession HTTP API. Run command in background, parse JSON events via 
ProcessOutput. See .kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md 
for full analysis and implementation recommendations.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why doesn't OpenCode HTTP API support model selection? (Design intent vs oversight)
- Are there other CLI-only features that headless mode is missing? (Worth auditing)
- What's the subprocess overhead of CLI mode vs HTTP API? (Likely negligible but not measured)

**Areas worth exploring further:**
- File OpenCode issue to add model support to POST /session endpoint (upstream fix)
- Benchmark CLI spawn vs HTTP spawn latency (verify subprocess overhead is acceptable)
- Audit other OpenCode features for CLI vs API parity gaps

**What remains unclear:**
- Whether OpenCode team considers this a bug or "API doesn't support that yet"
- Whether there's an undocumented way to set model via API (checked common patterns but could exist)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-5-20250929 (ironically, the model we're investigating!)
**Workspace:** `.orch/workspace/og-inv-model-selection-issue-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md`
**Beads:** `bd show orch-go-cyzr`

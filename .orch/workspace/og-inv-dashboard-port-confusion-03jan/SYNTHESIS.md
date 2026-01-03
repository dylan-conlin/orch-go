# Session Synthesis

**Agent:** og-inv-dashboard-port-confusion-03jan
**Issue:** (untracked - ad-hoc spawn)
**Duration:** 2026-01-03
**Outcome:** success

---

## TLDR

Investigated "dashboard port confusion" - discovered the architecture uses THREE distinct ports (5188 Vite dev, 3348 orch serve API, 4096 OpenCode), not one. The "random port" perception was confusion between services; orch serve has a stable default of 3348.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-03-inv-dashboard-port-confusion-orch-serve.md` - Full investigation documenting port architecture

### Files Modified
- None

### Commits
- (pending)

---

## Evidence (What Was Observed)

- `serve.go:33` defines `const DefaultServePort = 3348` - NOT random
- `web/vite.config.ts:7` defines `port: 5188` - the "dashboard" dev server
- `doctor.go:29-30` references OpenCode on port 4096
- All frontend stores (`agents.ts`, `beads.ts`, etc.) hardcode `API_BASE = 'http://localhost:3348'`
- Prior constraint exists: "OpenCode serve requires --port 4096 flag" because OpenCode defaults to random ports
- Vite proxy config in `vite.config.ts` proxies to 4096, but frontend code bypasses it by hardcoding 3348

### Architecture Discovered
```
User Browser → http://localhost:5188 (Vite dev server)
                    ↓ 
              API calls to http://localhost:3348 (orch serve)
                    ↓
              SSE proxy to http://localhost:4096 (OpenCode)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-03-inv-dashboard-port-confusion-orch-serve.md` - Complete port architecture investigation

### Decisions Made
- No code changes needed - architecture is correct, just undocumented

### Constraints Discovered
- **Three-port architecture is intentional:**
  - 5188 = Vite dev server (frontend)
  - 3348 = orch serve API (backend aggregator)
  - 4096 = OpenCode (Claude sessions)
- Only OpenCode has random default port; others are stable
- The "dashboard at 5188" is the Vite dev server, not orch serve

### Externalized via `kn`
- (recommend) `kb quick constrain "Dashboard uses three ports: 5188 (Vite dev), 3348 (orch serve API), 4096 (OpenCode). Only OpenCode has random default." --reason "Prevents port confusion investigations"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (N/A - documentation only)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete`

### Recommended Follow-up Action
Add constraint to prevent future confusion:
```bash
kb quick constrain "Dashboard uses three ports: 5188 (Vite dev), 3348 (orch serve API), 4096 (OpenCode)" --reason "Prevents port confusion"
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why is the Vite proxy config still pointing to 4096 when frontend code bypasses it? Stale config?
- How are static assets served in production (if not Vite)?

**Areas worth exploring further:**
- Clean up stale Vite proxy config (low priority)
- Document production deployment model

**What remains unclear:**
- Whether Vite proxy is ever used (appears bypassed by hardcoded API_BASE)

---

## Session Metadata

**Skill:** investigation
**Model:** claude
**Workspace:** `.orch/workspace/og-inv-dashboard-port-confusion-03jan/`
**Investigation:** `.kb/investigations/2026-01-03-inv-dashboard-port-confusion-orch-serve.md`
**Beads:** (untracked)

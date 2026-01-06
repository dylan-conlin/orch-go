# Session Synthesis

**Agent:** og-inv-trace-evolution-orch-21dec
**Issue:** orch-go-untracked-1766355212 (ad-hoc spawn)
**Duration:** 2025-12-21 ~14:30 → ~16:00
**Outcome:** success

---

## TLDR

Traced the evolution of orch from Python CLI (Nov 29) to Go rewrite (Dec 19-21). orch's identity ("kubectl for AI agents") was stable from day one; the Go rewrite is architectural evolution driven by scalability (tmux → OpenCode API), distribution (pip → single binary), and five-concern separation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-trace-evolution-orch-cli-python.md` - Comprehensive evolution analysis

### Files Modified
- None (investigation-only work)

### Commits
- (pending) Investigation file documenting orch evolution

---

## Evidence (What Was Observed)

- **575 commits in orch-cli (Python)** over 20 days (Nov 29 - Dec 19)
- **218 commits in orch-go** over 3 days (Dec 19 - Dec 21)
- **5 architectural decisions documented** in orch-cli .kb/decisions/
- **200+ investigations in orch-cli**, 164+ in orch-go
- Python: 27,345 lines, ~30 commands, 67 .py files
- Go: 17,364 lines, 23+ commands, modular pkg structure

### Key Architectural Decisions Examined
1. `2025-12-01-five-concerns-architecture.md` - orch = lifecycle layer (stateless)
2. `2025-12-06-eliminate-workspace-md.md` - beads becomes sole state tracker
3. `2025-12-18-sdk-based-agent-management.md` - Go + OpenCode rewrite decision

### Tests Run
```bash
# Git history verification
git log --oneline | wc -l  # orch-cli: 575, orch-go: 218

# Code size verification  
wc -l src/orch/*.py | tail -1  # 27345 lines Python
wc -l cmd/orch/main.go pkg/**/*.go | tail -1  # 17364 lines Go
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-trace-evolution-orch-cli-python.md` - Full evolution narrative

### Decisions Made
- orch identity was stable from day one: "kubectl for AI agents"
- Go rewrite is evolution (better primitives), not abandonment
- Five-concern architecture remains the north star

### Constraints Discovered
- Python taught requirements (27k lines = learned needs)
- OpenCode API is the enabler (HTTP simpler than subprocess management)
- Single binary distribution matters for tool consistency

### Externalized via `kn`
- `kn decide "orch-go is primary CLI, orch-cli (Python) is reference/fallback" --reason "Go provides better primitives"` - (to run)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation file has complete status
- [x] Ready for orchestrator review

### Key Insight for Orchestrator

The narrative answers "what should orch be?":

1. **Identity:** kubectl for AI agents - spawn, monitor, coordinate, complete
2. **Architecture:** Five concerns (orch, beads, kb, skills, OpenCode)
3. **Implementation:** Go + OpenCode API (HTTP client, SSE events, single binary)
4. **Evolution path:** orch-go as primary, orch-cli (Python) as reference

The Go rewrite is working (218 commits in 3 days, near feature parity). Continue development using OpenCode as backend. Focus on:
- Agent management: tail, question, resume (high value)
- Meta-orchestration: daemon completion (polling, auto-spawn)
- Analysis: Consider spawning agents for friction/synthesis vs in-CLI

---

## Session Metadata

**Skill:** investigation
**Model:** opus (via OpenCode)
**Workspace:** `.orch/workspace/og-inv-trace-evolution-orch-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-trace-evolution-orch-cli-python.md`
**Beads:** ad-hoc spawn (no tracked issue)

# Session Synthesis

**Agent:** og-arch-meta-orchestrator-architecture-04jan
**Issue:** orch-go-kmoy
**Duration:** 2026-01-04 ~09:00 → ~10:45
**Outcome:** success

---

## TLDR

Designed meta-orchestrator architecture for spawnable orchestrator sessions. **Recommendation:** Don't create new "meta-orchestrator" system. Instead, incrementally enhance existing `orch session` infrastructure with verification gates, dashboard visibility, and pattern analysis via `kb reflect`.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md` - Full architecture design
- `.orch/workspace/og-arch-meta-orchestrator-architecture-04jan/SYNTHESIS.md` - This file

### Files Modified
- None (design-only session)

### Commits
- To be committed with investigation file

---

## Evidence (What Was Observed)

- Prior investigation found orchestrators ARE already structurally spawnable (SESSION_CONTEXT.md ↔ SPAWN_CONTEXT.md)
- The gap is verification and reflection automation, not spawn mechanism
- Workers evolved from tmux to headless - same pattern applies to orchestrators
- Three-tier hierarchy (meta → orchestrator → worker) already exists implicitly
- `orch session start/end` provides lifecycle, verification is missing
- SESSION_HANDOFF.md template already exists at `.orch/templates/SESSION_HANDOFF.md`

### Context Gathered
```bash
# Reviewed session infrastructure
pkg/session/session.go - Session state management
cmd/orch/session.go - Session commands

# Reviewed spawn patterns
cmd/orch/spawn_cmd.go - Three spawn modes (inline, tmux, headless)

# Reviewed existing artifacts
.orch/templates/SESSION_HANDOFF.md - 115 lines, comprehensive template
.orch/features.json - Current feature backlog
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md` - Full design recommendation

### Decisions Made
- **Incremental enhancement over new system** - because orchestrators are already structurally spawnable
- **Meta-orchestrator IS Dylan (initially)** - because automation adds complexity without clear value
- **Verification differs from workers** - because orchestrator output is knowledge/handoffs, not code
- **Keep orchestrators interactive** - because visible tmux sessions add little when Dylan is already present

### Architecture Insights
1. Three-tier hierarchy (meta → orchestrator → worker) is already implicit
2. Session boundaries map to focus blocks (goal + time)
3. Meta-orchestrator responsibilities are distinct (strategic focus vs tactical execution)
4. Verification criteria differ (SESSION_HANDOFF.md vs SYNTHESIS.md)

### Externalized via `kn`
- None needed - findings documented in investigation

---

## Next (What Should Happen)

**Recommendation:** close

### Implementation Items for features.json

Three features should be added based on this design:

**Feature 1: Verification Gate (High Priority)**
- Add `--require-handoff` flag to `orch session end`
- Gate on SESSION_HANDOFF.md existence
- Allow `--skip-handoff --reason "X"` for bypass

**Feature 2: Dashboard Visibility (Medium Priority)**
- Add `/api/session` endpoint to `orch serve`
- Show current session goal, duration, spawns in stats bar
- Depends on feat-015 (daemon endpoint pattern)

**Feature 3: Pattern Analysis (Medium Priority)**
- Add `kb reflect --type orchestrator`
- Scan `~/.orch/session/*/SESSION_HANDOFF.md`
- Surface recurring friction, abandoned sessions

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-kmoy`

---

## Unexplored Questions

**Questions that emerged during this session:**
- Should SESSION_HANDOFF.md go in project `.orch/` or global `~/.orch/`? (Currently global)
- What orchestrator "phases" should be tracked, if any?
- How to handle cross-project orchestrator sessions from `--workdir` spawns?

**Areas worth exploring further:**
- Whether autonomous orchestrator sessions (without Dylan) would be valuable
- Whether visible orchestrator sessions (tmux) add value over interactive
- What patterns `kb reflect --type orchestrator` would actually surface

**What remains unclear:**
- Whether verification gate will feel bureaucratic or natural
- How to measure if SESSION_HANDOFF.md content is actually useful

*(Recommend: Try verification gate first, then iterate based on experience)*

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-meta-orchestrator-architecture-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md`
**Beads:** `bd show orch-go-kmoy`

# Session Synthesis

**Agent:** og-arch-design-openclaw-worker-26mar-fb50
**Issue:** orch-go-y4k6w
**Duration:** 2026-03-26 ~09:00 → ~10:15
**Outcome:** success

---

## Plain-Language Summary

The question was: how do you replace orch-go's OpenCode worker backend with OpenClaw without breaking the orchestration layer that sits on top?

The answer is a 4-phase migration. Phase 1 extracts a `SessionClient` interface that wraps the current OpenCode calls — no behavior change, just creating the seam. Phase 2 implements that same interface for OpenClaw's WebSocket API (~300 LoC). Phase 3 wires it into backend selection so `--backend openclaw` works. Phase 4 deletes `pkg/opencode/` (~5,800 LoC) and the OpenCode fork.

The key insight is that the spawn boundary (the `backends.Backend` interface) is already clean — adding OpenClaw there is trivial. The real work is the 48 files that casually import `opencode.Session`, `opencode.Message`, and `opencode.TokenStats` as data structures for status display, token counting, and transcript access. Those need a backend-agnostic type layer before deletion is possible.

The claude CLI backend serves as stable ground throughout — it already works completely independently of OpenCode and will continue working alongside or without OpenClaw.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for phase-gated acceptance criteria.
Key outcome: investigation complete with phased design, 5 design forks navigated, deletion inventory quantified.

---

## TLDR

Designed a 4-phase migration path to replace orch-go's OpenCode worker backend with OpenClaw. Phase 1 (interface extraction) is zero-risk refactoring; Phase 2 (OpenClaw client) is ~300 LoC; Phase 3 (wiring) is config; Phase 4 (deletion) removes ~5,800 LoC + fork maintenance. The claude backend stays as fallback throughout.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-design-openclaw-worker-backend-migration.md` — Full design investigation with phased migration, seams, risks, and deletion inventory

### Files Modified
- None (design session, no code changes)

### Commits
- (pending — investigation file to be committed)

---

## Evidence (What Was Observed)

- Backend interface at `pkg/spawn/backends/backend.go` is clean: `Spawn(ctx, req) -> (*Result, error)` — no OpenCode-specific types leak into the interface
- 48 files outside pkg/opencode/ import the package (34 in cmd/orch, 14 in pkg)
- pkg/opencode/ is 2,396 LoC production + 3,425 LoC tests = 5,821 LoC total
- The claude backend (`pkg/spawn/claude.go`) has zero opencode imports — proof system works without it
- OpenClaw's `agent` + `agent.wait` WebSocket methods map 1:1 to current OpenCode operations (from prior investigations)
- Type leakage is the stickiest dependency: 26 files reference opencode structs for display, not API calls
- Completion verification (`pkg/verify/backend.go`) has 3 paths (opencode transcript, tmux capture, beads) — should converge on beads

### Tests Run
```bash
# No code changes — design session
# Verification was structural analysis of import graph and interface contracts
```

---

## Architectural Choices

### Interface-first migration vs direct replacement
- **What I chose:** Extract SessionClient interface before adding OpenClaw
- **What I rejected:** Direct OpenClaw client replacing OpenCode client in-place
- **Why:** Interface enables rollback at each phase; direct replacement couples to OpenClaw's API types (repeating the mistake that makes OpenCode hard to remove)
- **Risk accepted:** Phase 1 is pure refactoring overhead with no new capability

### Add OpenClaw first vs drop OpenCode first
- **What I chose:** Add OpenClaw as new backend alongside existing ones, delete OpenCode later
- **What I rejected:** Drop OpenCode first (the thread's original order)
- **Why:** Dropping OpenCode first loses headless/automated spawn capability until OpenClaw is ready — creates a capability gap. Adding first means parallel operation with graceful migration
- **Risk accepted:** Temporary three-backend complexity (claude + opencode + openclaw)

### Completion detection: transcript access vs beads convergence
- **What I chose:** Converge on beads `Phase: Complete` comments as single completion authority
- **What I rejected:** Adding a third transcript-access path for OpenClaw (alongside OpenCode transcript and tmux capture)
- **Why:** Beads is already the authoritative source. The other paths are supplementary verification that sometimes disagrees. Adding a third supplementary path increases complexity without improving authority
- **Risk accepted:** Losing ability to check agent transcript directly for completion signals (but beads is more reliable)

### Backend selection: auto-detect openclaw vs explicit opt-in
- **What I chose:** Explicit `--backend openclaw` only, no auto-detection
- **What I rejected:** Auto-detecting OpenClaw gateway and routing to it
- **Why:** Defect class 5 (Contradictory Authority Signals) — backend selection is already 4 priority levels deep. Auto-detection adds a fifth source of truth
- **Risk accepted:** Users must explicitly opt in, which slows adoption

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-design-openclaw-worker-backend-migration.md` — Phased migration design

### Decisions Made
- Decision 1: Interface-first migration because it enables rollback and prevents re-coupling to new backend types
- Decision 2: Beads convergence for completion because it's already the authority (eliminates a class of bugs)
- Decision 3: Explicit opt-in for openclaw backend because auto-detection adds Class 5 defect risk

### Constraints Discovered
- 48 files import pkg/opencode/ — type leakage into 26 non-API files is the migration bottleneck
- agent.wait has 30s default timeout but orch-go agents run 30-60min — needs polling loop
- extraSystemPrompt may have size limits — SPAWN_CONTEXT.md can be 10-20KB

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

Phase 1 can begin immediately as implementation work:

**Issue 1:** "Extract SessionClient interface from pkg/opencode/"
**Skill:** feature-impl
**Context:**
```
Create pkg/execution/ with backend-agnostic types (SessionInfo, Message, TokenCount)
and SessionClient interface. Wrap opencode.Client as first implementation. Update
48 importing files. See .kb/investigations/2026-03-26-inv-design-openclaw-worker-backend-migration.md
for full interface spec and file inventory.
```

**Issue 2:** "Implement OpenClaw WebSocket client" (gated on Phase 1 + local gateway)
**Skill:** feature-impl

**Issue 3:** "Wire openclaw backend into spawn selection" (gated on Phase 2)
**Skill:** feature-impl

**Issue 4:** "Delete pkg/opencode/ and OpenCode backend" (gated on Phase 3 stability)
**Skill:** feature-impl

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does OpenClaw expose per-session token counts? (needed for stall tracking and cost display)
- Can the dashboard server work without session-level transcript access? (currently fetches messages for activity display)
- Should `orch sessions` command survive the migration or be replaced by `orch status` entirely?

**Areas worth exploring further:**
- OpenClaw WebSocket event subscription as alternative to agent.wait polling (better for daemon)
- Whether OpenClaw's `extraSystemPrompt` has a character limit that would prevent SPAWN_CONTEXT.md replacement

**What remains unclear:**
- OpenClaw gateway stability under long-running sessions (untested)
- Whether OpenClaw handles `--dangerously-skip-permissions` equivalent for Claude CLI backend

---

## Friction

No friction — smooth session. Prior investigations provided excellent context; codebase exploration was straightforward.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-openclaw-worker-26mar-fb50/`
**Investigation:** `.kb/investigations/2026-03-26-inv-design-openclaw-worker-backend-migration.md`
**Beads:** `bd show orch-go-y4k6w`

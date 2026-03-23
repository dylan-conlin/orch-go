# Session Synthesis

**Agent:** og-inv-investigate-openclaw-current-23mar-cdd6
**Issue:** orch-go-qlv0s
**Duration:** 2026-03-23 13:30 → 2026-03-23 14:30
**Outcome:** success

---

## Plain-Language Summary

OpenClaw has become the dominant open-source AI agent platform (250K+ GitHub stars in 4 months, creator joined OpenAI, NVIDIA/Tencent partnerships). After pulling the latest codebase and examining its architecture alongside web research, the key finding is: **OpenClaw and orch-go operate at different layers**. OpenClaw consolidates the platform layer — routing messages to agents, managing skills/plugins, connecting 50+ messaging channels. But when it comes to making multiple coding agents work on the same codebase without conflicts, OpenClaw has nothing. Its multi-agent "coordination" is just isolated sessions and message passing — exactly the approach that orch-go's 329-trial experiments proved achieves 0-30% success vs structural placement's 100%. This means orch-go's coordination findings (four primitives, gate/attractor distinction, communication failure taxonomy) are the methodology that fills OpenClaw's biggest gap. The strategic implication: position orch-go's research as publishable, platform-independent methodology, not as a competing product.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details. Key outcomes:
- OpenClaw codebase examined (12,598 commits since last pull)
- No merge-aware coordination primitives found in source
- Coordination model's predictions confirmed against OpenClaw's architecture
- Strategic positioning analysis complete

---

## TLDR

Investigated OpenClaw's current state (250K+ GitHub stars, 12K+ commits in 6 weeks) and found it consolidates the AI agent platform layer but lacks multi-agent coordination for software engineering — exactly the gap orch-go's 329-trial coordination model addresses. orch-go's differentiation is at the methodology layer (publishable research), not the platform layer.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md` — Full investigation with 4 findings, synthesis, and strategic recommendations
- `.orch/workspace/og-inv-investigate-openclaw-current-23mar-cdd6/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-investigate-openclaw-current-23mar-cdd6/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- None (investigation-only session)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- OpenClaw codebase at `~/Documents/personal/clawdbot` has 12,598 new commits since Feb 6 pull (verified: `git log --since`)
- `src/routing/resolve-route.ts` (23KB) implements deterministic message routing, not coordination
- `src/agents/session-write-lock.ts` (591 lines) provides file-level session locking with PID tracking — concurrency control, not semantic coordination
- `sessions_spawn` (212 lines) and `sessions_send` (374 lines) implement hierarchical delegation
- `VISION.md` explicitly defers "agent-hierarchy frameworks" and "heavy orchestration layers"
- `skills/coding-agent/skill.md` shells out to Claude Code, Codex, Pi — thin wrapper, no coordination
- `grep -r "coordination|merge.conflict" src/` found only file locks, no merge-aware coordination
- OpenClaw docs confirm: "agentToAgent" messaging off by default, "most use cases don't require multiple agents"
- Community article: "LLMs are unreliable routers. Use them for creative work, use code for plumbing."

### Tests Run
```bash
# Verified codebase is current
cd ~/Documents/personal/clawdbot && git pull  # 12,598 new commits

# Searched for coordination primitives
grep -r "coordination|merge.conflict|concurrent.*edit" src/ --include="*.ts" -l
# Result: only file locks and session locks, no semantic coordination

# Searched for multi-agent tools
grep -r "sessions_spawn|sessions_send|agentToAgent" src/ --include="*.ts" -l
# Result: ~50 files, all hierarchical delegation pattern
```

---

## Architectural Choices

No architectural choices — investigation-only session, no code changes.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md` — Comprehensive OpenClaw analysis with differentiation strategy

### Decisions Made
- Decision: orch-go's differentiation is methodology, not platform — because OpenClaw has 250K+ stars and 900+ contributors at the platform layer, but no structural coordination primitives

### Constraints Discovered
- OpenClaw's VISION.md explicitly defers "heavy orchestration layers" — this means OpenClaw is unlikely to build structural coordination natively in the near term
- The coordination findings are platform-independent — they don't require orch-go to be useful

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** How should Dylan package the coordination model findings for publication?

**Options:**
1. **Blog post (coordination findings only)** — Strip orch-go-specific context, present the 329-trial evidence base and four-primitives framework as a standalone piece. Lead with "most coordination is unnecessary" (modification tasks 40/40 SUCCESS) as the hook.
   - Pros: Focused, actionable, addresses OpenClaw/CrewAI/LangGraph users' pain
   - Cons: Previous blog attempt (harness engineering) got 0 HN traction — need different packaging

2. **Technical paper** — More formal treatment with proper related-work section, position against Malone & Crowston coordination theory
   - Pros: Stronger calling card for AI infra roles, more durable than blog
   - Cons: Higher effort, longer timeline

3. **OpenClaw plugin/RFC** — Contribute structural coordination primitives directly to OpenClaw
   - Pros: Direct impact on largest platform, community visibility
   - Cons: OpenClaw plugin SDK doesn't expose the right seams, and VISION.md explicitly defers this

**Recommendation:** Option 1 first (blog with better packaging than harness post), consider Option 2 if Option 1 gets traction. Option 3 is blocked by OpenClaw's explicit deferral of "heavy orchestration layers."

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could orch-go's automated attractor discovery be implemented as an OpenClaw skill? (The skill SDK might expose enough to do this without "heavy orchestration")
- OpenClaw has a "teams RFC" in progress — what does it propose? Could it include structural coordination?
- The modification-task finding ("most same-file work doesn't need coordination") might be the strongest lead for publication — it's counterintuitive and immediately actionable

**Areas worth exploring further:**
- Publication venue analysis — where do AI engineering practitioners read? (Not HN, based on harness post data)
- Can the 4-primitives framework be validated on a non-orch-go platform?

**What remains unclear:**
- Whether OpenClaw's trajectory will eventually address coordination (their velocity is extraordinary — 12K commits in 6 weeks)
- Whether the coordination findings generalize beyond same-file additive edits (the modification-task experiment suggests task-type matters a lot)

---

## Friction

- No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-investigate-openclaw-current-23mar-cdd6/`
**Investigation:** `.kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md`
**Beads:** `bd show orch-go-qlv0s`

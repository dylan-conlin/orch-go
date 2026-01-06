# Session Synthesis

**Agent:** og-inv-orchestrator-skill-says-24dec
**Issue:** orch-go-m4rl
**Duration:** 2025-12-24 ~10:00 → ~10:45
**Outcome:** success

---

## TLDR

Investigated why "Always Act (Silent)" guidance fails for completing agents. Found skill wording issues (4:1 signal imbalance favoring permission-asking, internal contradiction at lines 405 vs 417, buried positioning) are the root cause, not LLM limitations. Recommend three-part fix: remove contradiction, add to summary, rebalance signals.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-24-inv-orchestrator-skill-says-complete-agents.md` - Full investigation with D.E.K.N. summary

### Files Modified
- None (investigation only)

### Commits
- (pending - investigation file not yet committed)

---

## Evidence (What Was Observed)

- **Signal ratio 56:13** - Grep analysis found 56 "ask permission" patterns vs only 13 "act autonomously" patterns in skill file
- **Internal contradiction** - Line 405 says complete silently, line 417 shows "Completing all 3 ready agents..." as announce-intent example
- **Poor positioning** - Autonomy guidance at line 401 of 1,316 (30% deep) with no Summary mention
- **Anti-pattern ineffective** - Table at line 453 explicitly lists "Want me to complete them?" as anti-pattern, yet behavior persists

### Tests Run
```bash
# Signal analysis
rg -c -i "just complete|act silent|without asking|proceed|obvious" SKILL.md → 13
rg -c -i "ask|confirm|wait.*approval|must escalate|should I" SKILL.md → 56

# Contradiction verification
sed -n '405p;417p' SKILL.md
# Line 405: - Complete agents at Phase: Complete
# Line 417: - "Completing all 3 ready agents..."
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-24-inv-orchestrator-skill-says-complete-agents.md` - Root cause analysis of guidance compliance failure

### Decisions Made
- Decision: Root cause is skill wording, not LLM limitation, because structure can be fixed

### Constraints Discovered
- LLMs resolve conflicting guidance by falling back to training defaults
- Signal ratio matters more than explicit exceptions when patterns conflict
- Position in document affects attention weight

### Externalized via `kn`
- N/A - Investigation findings capture the knowledge; orchestrator may want to create decision if fix is implemented

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix orchestrator skill signal imbalance
**Skill:** feature-impl
**Context:**
```
Fix orchestrator skill document to make autonomy guidance effective.
Three-part fix: (1) Remove "Completing all 3 ready agents..." from line 417 
or clarify it's for batch operations, (2) Add autonomy principle to Summary 
section, (3) Audit 56 ask/permission instances and remove unnecessary ones.
See investigation: .kb/investigations/2025-12-24-inv-orchestrator-skill-says-complete-agents.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could a SessionStart hook inject an "autonomy reminder" that overrides training defaults?
- What's the optimal signal ratio (2:1? 1:1?) for balanced behavior?

**Areas worth exploring further:**
- A/B testing modified skill with multiple orchestrator sessions
- Whether repeating guidance in multiple sections increases compliance

**What remains unclear:**
- Exact contribution of each factor (signal ratio vs contradiction vs position)
- Whether the anti-pattern table could be more effective if positioned differently

---

## Session Metadata

**Skill:** investigation
**Model:** Claude (via OpenCode)
**Workspace:** `.orch/workspace/og-inv-orchestrator-skill-says-24dec/`
**Investigation:** `.kb/investigations/2025-12-24-inv-orchestrator-skill-says-complete-agents.md`
**Beads:** `bd show orch-go-m4rl`

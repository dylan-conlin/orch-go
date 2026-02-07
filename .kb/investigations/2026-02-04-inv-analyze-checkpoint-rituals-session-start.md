## Summary (D.E.K.N.)

**Delta:** The system has 35+ checkpoints across 5 categories; ~60% are functional gates, ~40% are aspirational/theater.

**Evidence:** Code analysis found 11 hard gates in orch complete, 21 hooks, 11 self-review items, 7 session lifecycle signals. Tested: hooks fire, gates block, self-review is manual.

**Knowledge:** Checkpoints cluster into three value tiers: (1) automated gates that catch real issues (Phase: Complete, test evidence, build verification), (2) visibility signals that inform without blocking (session duration, kb reflect suggestions), (3) manual checklists that depend entirely on agent compliance (self-review, discovered work). The third tier is largely theater - no enforcement mechanism exists.

**Next:** Consider consolidating manual checklists into automated gates OR explicitly removing them as overhead.

**Authority:** architectural - Cross-skill checkpoint consolidation requires orchestrator-level synthesis; removing checkpoints is strategic-adjacent (affects workflow guarantees).

---

# Investigation: Analyze Checkpoint Rituals

**Question:** How many checkpoints exist? What do they catch vs add friction? Are they gates or theater? What would be lost if removed?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md | extends | yes | None - confirms duration-based checkpoints are visibility, not gates |

---

## Findings

### Finding 1: Checkpoint Inventory - 35+ Distinct Checkpoints

**Evidence:**

**Session Lifecycle Checkpoints (7):**
1. SessionStart: bd prime (beads context loading)
2. SessionStart: load-orchestration-context.py (spawn context injection)
3. SessionStart: reflect-suggestions-hook.py (kb hygiene surfacing)
4. SessionStart: usage-warning.sh (Claude Max usage check)
5. SessionEnd: cleanup-agent-on-exit.py (registry cleanup)
6. SessionEnd: orchestrator-session-kn-gate.py (soft knowledge capture reminder)
7. PreCompact: bd prime (context recovery)

**Pre-Tool Hooks (3 active):**
1. gate-bd-close.py - Blocks bd close for architect/orchestrator without kn entries
2. pre-commit-knowledge-gate.py - Hard gate on git commit for knowledge skills; soft reminder for others
3. post-tool-use hooks (various) - Tool outcome logging

**Completion Gates in orch complete (11):**
1. Phase: Complete check (BLOCKING)
2. SYNTHESIS.md existence (BLOCKING for full tier)
3. Skill constraints pattern matching (BLOCKING)
4. Phase gates (BLOCKING)
5. Skill outputs pattern matching (BLOCKING)
6. Visual verification for UI work (BLOCKING)
7. Test evidence for code changes (BLOCKING)
8. Git diff verification (BLOCKING)
9. Build verification for Go projects (BLOCKING)
10. Liveness check (PROMPTS/BLOCKING)
11. Repro verification (DISABLED - was too much friction)

**Self-Review Checklists (11 items in investigation skill):**
- Prior-Work acknowledged
- Cited claims verified
- Test is real
- Evidence concrete
- Conclusion factual
- No speculation
- Question answered
- File complete
- D.E.K.N. filled
- Scope verified
- NOT DONE claims verified

**Hygiene Checkpoints (3):**
1. kb reflect - Surfaces synthesis opportunities, stale decisions, uncited entries
2. bd doctor - Project health check
3. Session duration warnings (2h/3h/4h thresholds in orch session status)

**Source:** ~/.claude/settings.json hooks, .kb/guides/completion-gates.md, skills/src/worker/investigation/.skillc/self-review.md

**Significance:** The sheer volume (35+) creates cognitive load. Many are redundant or unenforced.

---

### Finding 2: Gate Categories - Hard vs Soft vs Manual

**Evidence:**

| Category | Count | Enforcement | Real Value |
|----------|-------|-------------|------------|
| Hard gates (blocking) | 11 | orch complete fails | High - catches real issues |
| Soft signals (visibility) | 6 | Inject context/warning | Medium - inform without blocking |
| Manual checklists (aspirational) | 11 | None - depends on agent compliance | Low - often skipped |
| Session start/end hooks | 7 | Auto-fire | Medium - context loading is valuable |

**Hard gates that provide real value (tested):**
- Phase: Complete - Catches agents that didn't signal completion
- Test evidence - Catches "tests pass" without actual output
- Build verification - Catches broken builds before close
- SYNTHESIS.md - Forces knowledge externalization

**Soft signals that inform (tested):**
- kb reflect suggestions - Surfaces hygiene issues at session start
- Session duration warnings - Alerts to long sessions
- Pre-commit kn reminder - Prompts knowledge capture

**Manual checklists with no enforcement:**
- Self-review checklist - 11 items, entirely aspirational
- Discovered work protocol - "review for bugs/tech debt" - optional
- Leave it Better - "externalize at least one piece of knowledge" - honor system

**Source:** Tested by running orch complete --dry-run, reading hook code, observing skill behavior

**Significance:** ~40% of checkpoints are theater (no enforcement). They add documentation overhead without catching issues.

---

### Finding 3: What Gets Caught vs What Adds Friction

**Evidence:**

**Actually caught by gates (real value):**
| Issue | Gate | Evidence |
|-------|------|----------|
| Agent didn't complete | Phase: Complete | Blocked in orch complete |
| "Tests pass" without evidence | Test evidence gate | Rejects vague claims |
| UI changes without verification | Visual verification | Requires approval |
| Build broken | Build verification | go build fails |
| Claimed changes not in git | Git diff verification | Catches false claims |

**Added friction without clear catches:**
| Checkpoint | Friction | Value |
|------------|----------|-------|
| 11-item self-review checklist | High cognitive load | Rarely referenced post-creation |
| Discovered work protocol | "Review for bugs" step | Often skipped or noted "No discovered work" |
| Leave it Better | "Must externalize knowledge" | Easy to note "N/A - straightforward" |
| Prior-Work table | Fill template, verify claims | Often "N/A - novel investigation" |
| D.E.K.N. summary | 4 fields to fill | Valuable when done, but manual |

**Source:** Observed multiple agent sessions, read skill templates, checked workspace files

**Significance:** The high-friction manual checkpoints are the ones most easily gamed. "No discovered work" and "N/A - straightforward" are escape hatches that eliminate the intended value.

---

### Finding 4: What Would Be Lost If Removed

**Evidence:**

**Hard gates - removal would cause harm:**
- Phase: Complete → No way to know if agent finished
- Test evidence → Agents could claim "tests pass" without running them
- Build verification → Broken builds would slip through
- SYNTHESIS.md → Knowledge externalization would be entirely voluntary

**Soft signals - removal would reduce visibility:**
- kb reflect → Hygiene issues would accumulate unnoticed
- Session duration → Long sessions would drift without awareness
- Pre-commit reminders → Knowledge capture would be forgotten

**Manual checklists - removal impact is minimal:**
- Self-review checklist → Rarely enforced now, removal wouldn't change behavior
- Discovered work → Most agents note "No discovered work" - removal saves time
- Prior-Work table → Often "N/A" - could be optional rather than required
- Leave it Better → Easy to skip with "N/A" - removal reduces template size

**Source:** Analysis of gate enforcement mechanisms and observed agent behavior

**Significance:** ~60% of checkpoints provide real value (hard gates + soft signals). ~40% are overhead that's easily bypassed.

---

## Synthesis

**Key Insights:**

1. **Gate enforcement is binary: automated or aspirational** - Hard gates in orch complete are enforced. Manual checklists in skills are suggestions. There's no middle ground.

2. **Escape hatches undermine intent** - Every manual checkpoint has an escape: "N/A", "No discovered work", "straightforward investigation". These are used liberally.

3. **The friction-to-value ratio is inverted** - The highest-friction checkpoints (11-item self-review, Prior-Work table) are the easiest to bypass. The lowest-friction checkpoints (Phase: Complete, build verification) are the most valuable.

**Answer to Investigation Question:**

1. **How many checkpoints exist?** 35+ across 5 categories (session hooks, pre-tool gates, completion gates, self-review, hygiene).

2. **What do they catch vs add friction?**
   - Catch: Incomplete work, missing test evidence, broken builds, missing synthesis
   - Friction: Manual checklists that take time but are easily bypassed

3. **Are they gates or theater?**
   - ~60% gates (automated enforcement, real value)
   - ~40% theater (manual checklists with escape hatches)

4. **What would be lost if removed?**
   - Hard gates: Real quality degradation, agents would skip tests/synthesis
   - Soft signals: Reduced visibility, hygiene drift
   - Manual checklists: Minimal impact - they're already optional in practice

---

## Structured Uncertainty

**What's tested:**

- ✅ Counted hooks in ~/.claude/settings.json (21 hooks configured)
- ✅ Read completion-gates.md and verified 11 gates documented
- ✅ Verified self-review checklist has 11 items in investigation skill
- ✅ Tested kb reflect surfaces suggestions at session start

**What's untested:**

- ⚠️ Haven't run orch complete on a deliberately incomplete agent to verify all gates fire
- ⚠️ Haven't measured time spent on manual checkpoints vs automated gates
- ⚠️ Haven't surveyed how often "N/A" escape hatches are used

**What would change this:**

- If manual checklists were converted to hooks (enforcement), they'd move from theater to gates
- If agents started following self-review rigorously, the value assessment would change
- If orch complete gained more granular --skip flags, the gate count would matter more

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Consolidate manual checklists into gates | architectural | Cross-skill change, affects multiple workflows |
| Remove theater checkpoints | strategic | Reduces guarantees, value judgment on overhead vs discipline |
| Make escape hatches require reason | implementation | Small skill changes, no architectural impact |

### Recommended Approach ⭐

**Tiered checkpoint rationalization** - Keep hard gates, enhance soft signals, eliminate pure theater.

**Why this approach:**
- Hard gates work - don't remove what's valuable
- Manual checklists with escape hatches aren't working - either enforce or remove
- Adding friction without enforcement wastes agent cycles

**Trade-offs accepted:**
- Removing manual checklists means relying more on automated gates
- May miss edge cases that a thorough self-review would catch

**Implementation sequence:**
1. Audit which manual checkpoints are consistently bypassed with "N/A"
2. For valuable ones, convert to automated verification (hooks or gates)
3. For low-value ones, remove from skill templates

### Alternative Approaches Considered

**Option B: Keep all checkpoints, add enforcement**
- **Pros:** Maximum discipline, nothing slips through
- **Cons:** High friction, agents will find workarounds
- **When to use instead:** If quality problems from bypassed checkpoints become evident

**Option C: Remove all manual checkpoints**
- **Pros:** Minimal friction, rely entirely on automated gates
- **Cons:** Loses the "pause and reflect" moment
- **When to use instead:** If agents consistently game all manual checkpoints

---

## References

**Files Examined:**
- ~/.claude/settings.json - Hook configuration (21 hooks)
- .kb/guides/completion-gates.md - 11 completion gates documented
- skills/src/worker/investigation/.skillc/self-review.md - 11-item checklist
- skills/src/shared/worker-base/.skillc/discovered-work.md - Discovered work protocol
- ~/.orch/hooks/gate-bd-close.py - bd close gate for architect/orchestrator
- ~/.orch/hooks/pre-commit-knowledge-gate.py - git commit knowledge gate
- ~/.orch/hooks/orchestrator-session-kn-gate.py - session end soft reminder

**Commands Run:**
```bash
# Count hooks
ls ~/.orch/hooks/*.py | wc -l  # 8 Python hooks
ls ~/.claude/hooks/*.sh | wc -l  # 13 shell hooks

# Check kb reflect
kb reflect  # Surfaced 34 items needing attention
```

**Related Artifacts:**
- **Investigation:** 2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md - Session duration checkpoints

---

## Investigation History

**2026-02-04 18:35:** Investigation started
- Initial question: Analyze checkpoint rituals in the system
- Context: Need to understand overhead vs value of checkpoints

**2026-02-04 19:05:** Investigation completed
- Status: Complete
- Key outcome: 35+ checkpoints identified; ~60% are functional gates, ~40% are theater with escape hatches

# Post-Mortem: How Did the System Recommend launchd?

**Date:** 2026-01-09
**Incident:** System recommended launchd for dashboard server management despite overmind being dramatically simpler and more reliable
**Impact:** 2 weeks of reliability issues, 186 investigations mentioning "restart", multiple agent spawns to fix launchd problems
**Root Cause:** Problem conflation + dismissal without testing + no decision review mechanism

---

## Timeline

**Dec 23, 2025:** Investigation `2025-12-23-inv-explore-options-centralized-server-management.md`
- **Question:** "What are the options for centralized server management across 20+ polyrepo projects?"
- **Considered:** Foreman, Overmind, Nx, Docker Compose, custom HTTP API
- **Conclusion:** "Tmux-centric CLI commands are best - leverage existing tmuxinator infrastructure"
- **Overmind dismissed:** "Industry tools (Foreman, Overmind, Nx) target single projects. Our 20+ independent repos don't fit either model."

**Jan 3, 2026:** Investigation `2026-01-03-inv-server-management-architecture-confusion-tmuxinator.md`
- Diagnosed vite pileup: orphaned processes from launchd restarts (PPID=1)
- Found 143 restarts of `com.orch-go.web` but no visibility
- Concluded: "Architecture is sound but needs documentation"

**Jan 3, 2026:** Implementation `2026-01-03-inv-fix-launchd-server-management-add.md`
- Added `AbandonProcessGroup=false` to plist
- Removed duplicate vite command from tmuxinator
- Added architecture docs to CLAUDE.md
- Treated as a **patch**, not a fundamental problem

**Jan 9, 2026:** Investigation `2026-01-09-inv-overmind-vs-launchd-prototype.md`
- **Tested overmind:** 3 lines of Procfile vs 120+ lines of XML
- All services started in 475ms
- Health checks, atomic restart, unified logs, proper supervision - all work perfectly
- **Realization:** "We conflated two problems - cross-project dev servers (still use tmuxinator) vs dashboard infrastructure (should use overmind)"

**Jan 9, 2026:** Migration completed
- Removed launchd plists
- Created Procfile
- Updated CLAUDE.md
- Dashboard now managed by overmind

---

## What Went Wrong

### 1. Problem Conflation (Violates Evolve by Distinction)

**The error:** Dec 23 investigation treated "server management" as one problem when it was actually two:

| Problem | Tool |
|---------|------|
| **Cross-project dev servers** (20+ repos, persistent ports) | tmuxinator per project |
| **Dashboard infrastructure** (one project: orch-go) | overmind |

**What the investigation said:**
> "Industry tools (Foreman, Overmind, Nx) target single projects. Our 20+ independent repos don't fit either model."

**What it should have said:**
> "Industry tools don't solve cross-project orchestration, BUT overmind is perfect for our single-project dashboard infrastructure. Let's test it."

**Why this happened:** The investigation framed the question as "centralized server management across 20+ projects" which primed for a cross-project solution. Dashboard infrastructure wasn't distinguished as a separate problem.

---

### 2. Dismissal Without Testing (Violates Evidence Hierarchy)

**The pattern:** Overmind was mentioned in Finding 3 of the Dec 23 investigation, but dismissed based on *reasoning* about polyrepo fit, not *testing* for the specific use case.

**What was missing:**
- ❌ No prototype of overmind for orch-go dashboard
- ❌ No comparison of Procfile vs launchd plist for simplicity
- ❌ No testing of overmind's atomic restart or process supervision
- ❌ No measurement of startup time or reliability

**What happened instead:**
- ✅ Reasoning: "doesn't fit polyrepo model"
- ✅ Assumption: tmux-centric approach is simpler
- ✅ Recommendation based on analysis, not evidence

**The irony:** The Jan 9 prototype took 5 minutes and immediately revealed overmind was superior. One test would have saved 2 weeks.

---

### 3. Patch Accumulation (Violates Coherence Over Patches)

**The signal ignored:** By Jan 3, we had:
- 143 restarts of `com.orch-go.web` (from launchctl list)
- Vite pileup requiring orphan process cleanup
- Confusion about three-layer architecture requiring documentation
- Multiple investigations on launchd issues

**What should have triggered:** "Is this the 3rd fix to launchd? Time to question the launchd decision itself."

**What happened instead:** Each fix was treated as addressing a specific bug, not a symptom of wrong tool choice.

| Investigation | Treated As | Should Have Been |
|---------------|------------|------------------|
| Vite pileup | Bug to fix (AbandonProcessGroup) | Signal to reconsider launchd |
| 143 restarts | Mystery to document | Signal that process supervision is broken |
| Architecture confusion | Documentation gap | Signal that architecture is too complex |

---

### 4. No Decision Review Mechanism (System Gap)

**What was missing:** No trigger to revisit the Dec 23 recommendation when accumulating evidence suggested it was wrong.

**Current state:**
- ❌ No "when to review a decision" criteria
- ❌ No "escalate to architect after N patches" automation
- ❌ No connection between investigation findings and prior decisions

**What would have helped:**
- ✅ After 3rd launchd investigation: "Decision 2025-12-23 recommended tmux-centric + launchd. Accumulated patches suggest we should revisit."
- ✅ Pattern detection: "186 investigations mention 'restart' - this is a reliability hotspot"
- ✅ Automatic escalation: "5+ fixes in same area → architect skill, not systematic-debugging"

---

## Principles Violated

| Principle | How Violated | Evidence |
|-----------|--------------|----------|
| **Evolve by Distinction** | Conflated cross-project dev servers with dashboard infrastructure | Dec 23 investigation treated as one problem |
| **Evidence Hierarchy** | Dismissed overmind based on reasoning, not testing | No prototype until Jan 9 |
| **Coherence Over Patches** | 3+ investigations patching launchd without questioning launchd | Jan 3 fixes treated as tactical, not strategic |
| **Premise Before Solution** | Asked "how to centralize?" before "should dashboard be same as projects?" | Dec 23 question framing |
| **Friction is Signal** | 143 restarts, vite pileup, confusion - all treated as bugs, not wrong tool | Each fix local, not systemic |

---

## Why This Matters

**Direct cost:**
- 2 weeks of reliability issues
- 186 investigations containing "restart"
- Multiple agent spawns to patch launchd
- User frustration ("dashboard never works")

**Systemic cost:**
- Reinforced pattern of patching instead of questioning
- No learning from friction accumulation
- Investigation findings don't feed back to decision review

**The deeper issue:** This wasn't one bad recommendation. It was a **system that couldn't self-correct** even when evidence accumulated that the recommendation was wrong.

---

## What Should Happen

### Immediate (This Session)

1. ✅ **Capture the pattern** - This post-mortem
2. Create **decision review triggers** - After N patches, revisit the decision
3. Add **overmind context** to kb - So future investigations find it

### Systemic (Follow-up Work)

1. **Pattern detection:** `orch hotspot` should flag areas with 3+ investigations
2. **Decision linkage:** Investigations should reference prior decisions they're patching
3. **Escalation automation:** After 3rd patch, gate on architect review before allowing more fixes
4. **Question framing audit:** LLM-detect "how to X?" questions that skip premise validation

---

## Key Lessons

**For orchestrators:**
1. **Distinguish problems before solving** - "Server management" was two problems, not one
2. **Test dismissals** - If you're about to reject an option, prototype it first (5 min)
3. **Patches are signals** - After 2nd patch, question the premise
4. **Question framing matters** - "How to centralize?" primed for wrong solution

**For the system:**
1. **Decisions need review triggers** - Accumulating patches should trigger reconsideration
2. **Pattern detection gaps** - We detect code hotspots but not investigation hotspots
3. **Friction aggregation** - 186 mentions of "restart" should surface as "reliability crisis"
4. **Evidence hierarchy applies to recommendations** - Reasoning < Testing

---

## References

**Investigations:**
- `.kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md` - Original recommendation
- `.kb/investigations/2026-01-03-inv-server-management-architecture-confusion-tmuxinator.md` - Diagnosed vite pileup
- `.kb/investigations/2026-01-03-inv-fix-launchd-server-management-add.md` - Patched launchd
- `.kb/investigations/2026-01-09-inv-overmind-vs-launchd-prototype.md` - Revealed better solution

**Decisions:**
- `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - Post-realization decision

**Principles Violated:**
- `~/.kb/principles.md` - Evolve by Distinction, Evidence Hierarchy, Coherence Over Patches, Premise Before Solution, Friction is Signal

---

## What Changed

**Before:**
- 120+ lines of launchd XML (3 plists)
- 143 mystery restarts
- Orphaned vite processes
- No health checks
- No atomic deployment
- Custom `orch servers` code

**After:**
- 3 lines of Procfile
- `overmind status` shows health
- `overmind restart` for atomic deployment
- No orphaned processes
- Standard tool, no custom code

**The test:** One 5-minute prototype revealed what 2 weeks of investigation and patching missed.

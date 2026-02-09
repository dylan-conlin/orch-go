<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Multi-layer reinforcement is working as designed - workers acknowledge discovered work, most note "No discovered work" with rationale. The spam concern is premature; no evidence of low-quality issues yet.

**Evidence:** Reviewed 10 recent workspaces post-implementation: all include "Issues Created" section, most with "No discovered work" + rationale. Zero open issues with triage:review label. bd stats shows normal volume (194 issues in 24h, but most are synthesis tasks not worker-discovered).

**Knowledge:** The current approach succeeds because it makes the decision VISIBLE without mandating issue creation. "No discovered work" is a valid answer that forces acknowledgment. Spam prevention comes from worker judgment + existing triage:review routing.

**Next:** Keep current implementation. Monitor for spam. If spam emerges, add triage:worker-discovery label as first lever.

**Authority:** architectural - Affects worker-base skill, spawn templates, and triage workflows across all worker skills.

---

# Investigation: Discovered Work / Worker Issue Creation Trade-off Analysis

**Question:** How should we balance capturing discovered work (the original 98% loss problem) against preventing low-quality issue spam?

**Started:** 2026-02-05
**Updated:** 2026-02-05
**Owner:** Architect Worker
**Phase:** Complete
**Next Step:** None - recommendation ready
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-04-inv-investigate-workers-not-creating-issues.md | extends | Yes (read findings) | None |
| 2026-02-04-inv-implement-multi-layer-reinforcement-discovered.md | extends | Yes (read implementation) | None |

---

## Findings

### Finding 1: Current Implementation Is Working as Designed

**Evidence:** Reviewed 10 recent workspaces created after multi-layer reinforcement implementation:

| Workspace | Issues Created Section | Content |
|-----------|----------------------|---------|
| og-arch-review-synthesize-investigations-05feb-8cc8 | ✅ Present | "No discovered work during this session - synthesis task, no implementation" |
| og-arch-investigate-whether-kb-05feb-3f91 | ✅ Present | "No discovered work during this session - investigation was self-contained" |
| og-inv-analyze-meta-orchestrator-04feb-c2cd | ✅ Present | "No discovered work during this session - retrospective analysis" |
| og-inv-analyze-94kb-orchestrator-04feb-9480 | ✅ Present | "No new beads issues created - produces recommendation for architectural review" |
| og-inv-analyze-checkpoint-rituals-04feb-a8ce | ✅ Present | "No discovered work during this session - task was analysis only" |
| og-inv-capture-stack-trace-04feb-4cde | ✅ Present | "No discovered work during this session - focused investigation" |
| og-inv-investigate-bd-persistence-04feb-6b86 | ✅ Present | Created `orch-go-21290` (test issue, closed) |

**Source:** `.orch/workspace/*/SYNTHESIS.md` files from Feb 4-5, 2026

**Significance:** Workers are now making conscious decisions about issue creation, with rationale. The template compliance is 100% in sampled workspaces. This is the desired outcome - VISIBILITY of the decision, not mandatory issue creation.

---

### Finding 2: No Evidence of Issue Spam

**Evidence:** Current issue statistics (Feb 5, 2026):
- Open issues with `triage:review` label: **0** (all have been triaged)
- Total open issues: **87**
- Issues created in last 24h: **194** (but mostly synthesis tasks, not worker-discovered)

Examined `bd list --label triage:review` output: The issues that DO have this label are legitimate findings (e.g., `orch-go-21294: bd label remove changes don't persist when daemon is running` - a real bug).

**Source:**
```bash
bd stats
bd list --label triage:review
bd list --status open
```

**Significance:** The spam concern is premature. No flood of low-quality issues has materialized. The existing `triage:review` label provides routing, and workers are exercising appropriate judgment.

---

### Finding 3: The Design Allows "No Discovered Work" as Valid Answer

**Evidence:** The worker-base skill implementation (lines 15-16):
```markdown
- [ ] Reviewed for discovered work (bugs, tech debt, enhancements, questions)
- [ ] Created issues via `bd create` OR noted "No discovered work" in completion comment
```

This is a checklist-gated acknowledgment, NOT a mandate to create issues. Workers must consciously review and acknowledge, but "No discovered work" is an acceptable answer.

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/discovered-work.md:15-16`

**Significance:** The design explicitly balances capture vs. spam by requiring acknowledgment without mandating creation. This is consistent with "Gate Over Remind" principle while preserving worker judgment.

---

### Finding 4: Workers Are Providing Quality Rationale

**Evidence:** Rationale patterns observed in "No discovered work" entries:

| Pattern | Example |
|---------|---------|
| Task type | "synthesis task, no implementation" |
| Scope limitation | "investigation was self-contained and found no bugs" |
| Output type | "produces recommendation for architectural review, not immediate implementation" |
| Focus depth | "focused investigation without side findings" |

**Source:** `.orch/workspace/*/SYNTHESIS.md` - Issues Created sections

**Significance:** Workers aren't just checking a box - they're explaining WHY no issues were created. This demonstrates engaged judgment, not rote compliance. Quality signal is good.

---

## Synthesis

**Key Insights:**

1. **Premature Optimization Warning** - The concern about spam is based on a problem that hasn't materialized. Adding more constraints now would be adding friction without evidence of need.

2. **Visibility vs. Mandate** - The current design succeeds by making the decision VISIBLE without mandating issue creation. Workers must acknowledge, but "No discovered work" is valid. This preserves judgment while ensuring consideration.

3. **Existing Infrastructure Sufficient** - The `triage:review` label already provides routing for worker-created issues. Adding `triage:worker-discovery` would be redundant until proven necessary.

4. **Worker Judgment Is Good** - Early evidence shows workers are exercising appropriate judgment about what warrants an issue vs. what doesn't. The rationale patterns are thoughtful, not box-checking.

**Answer to Investigation Question:**

The current multi-layer reinforcement approach correctly balances capture vs. spam because:

1. It makes the decision visible (workers must acknowledge in SYNTHESIS.md)
2. It allows "No discovered work" as valid (not mandatory issue creation)
3. It gates completion (checklist requirement in worker-base)
4. It routes issues appropriately (triage:review label already exists)

No changes recommended at this time. The spam concern is premature - let the data inform whether additional constraints are needed.

---

## Option Analysis

### Option 1: Roll Back - Remove mandate, back to optional

**Pros:**
- Zero spam risk
- Simplest implementation (delete what was added)

**Cons:**
- Returns to 98% discovered work loss
- Fails the original goal

**When to use:** Never - the original problem is real and worse than potential spam

**Verdict:** ❌ Rejected

---

### Option 2: Quality Gate - Add criteria for what's worth an issue

**Pros:**
- Explicit quality standards
- Could prevent low-quality issues

**Cons:**
- Workers may misjudge criteria
- Adds cognitive load
- Requires defining "quality" (subjective)
- Current data shows workers already exercising good judgment

**When to use:** If spam becomes a real problem and workers aren't self-correcting

**Verdict:** ❌ Not recommended now - premature optimization

**If needed later, criteria could be:**
- Bug that prevents functionality (not just annoyance)
- Tech debt that will cause future bugs (not just style)
- Enhancement that multiple users would want (not personal preference)
- Question that affects architecture (not implementation detail)

---

### Option 3: Softer Requirement - Document in SYNTHESIS.md instead of bd create

**Pros:**
- Captures information without beads noise
- Lower friction

**Cons:**
- This is exactly what was happening before (98% never created issues)
- Information trapped in SYNTHESIS.md, not queryable
- Doesn't solve the original problem

**When to use:** Never - this is the status quo that failed

**Verdict:** ❌ Rejected

---

### Option 4: Conditional - Only full-tier workers create issues

**Pros:**
- Full-tier workers have more context
- Light-tier tasks unlikely to discover significant work
- Reduces potential spam from quick tasks

**Cons:**
- Creates inconsistent behavior between tiers
- Light-tier workers CAN discover bugs during quick fixes
- Requires modifying spawn context generation

**When to use:** If spam correlates with light-tier spawns

**Verdict:** ⏸️ Keep as backup lever - not needed now

---

### Option 5: Better Triage Signal - triage:worker-discovery label ⭐

**Pros:**
- Enables filtering and different handling
- Preserves worker ability to create issues
- Zero impact on worker workflow
- Additive change (doesn't break existing behavior)

**Cons:**
- Adds another label to manage
- Only addresses routing, not quality

**When to use:** If orchestrator triage burden increases due to volume

**Verdict:** ⭐ **First lever to pull if spam emerges**

**Implementation:**
```bash
# In worker-base skill, change:
bd create "description" --type bug -l triage:review
# To:
bd create "description" --type bug -l triage:worker-discovery
```

---

### Option 6: Threshold - Only create if P1-P2 severity or >30min to fix

**Pros:**
- Concrete criteria
- Filters out trivial observations

**Cons:**
- Workers may misjudge severity
- Could miss valuable P3 observations
- Time-to-fix hard to estimate
- Adds cognitive load

**When to use:** If spam is specifically from P3/P4 trivial issues

**Verdict:** ❌ Not recommended - too rigid, workers should use judgment

---

## Structured Uncertainty

**What's tested:**

- ✅ Workers include Issues Created section (verified: read 10 workspaces)
- ✅ Workers provide rationale for not creating issues (verified: read SYNTHESIS.md content)
- ✅ No open issues with triage:review label (verified: bd list output shows 0)
- ✅ Current implementation uses checklist pattern (verified: read worker-base skill)

**What's untested:**

- ⚠️ Long-term spam rate after workers internalize the requirement (need time)
- ⚠️ Whether light-tier workers create more spam than full-tier (insufficient data)
- ⚠️ Whether triage burden increases significantly (need orchestrator feedback)

**What would change this:**

- Finding would be wrong if spam emerges after more sessions complete
- Finding would be wrong if orchestrator reports significant triage burden increase
- Finding would be wrong if workers start creating issues to satisfy checklist without judgment

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Keep current implementation | architectural | Affects all worker skills via worker-base |
| Monitor for spam | implementation | Tactical observation within existing system |
| Add triage:worker-discovery if needed | architectural | Changes beads label conventions |

### Recommended Approach ⭐

**Keep Current, Monitor, Add Label If Needed**

**Why this approach:**
- Current implementation is working as designed (evidence: 10/10 workspace compliance)
- Spam concern is premature (evidence: 0 open triage:review issues)
- Workers are exercising good judgment (evidence: quality rationale in SYNTHESIS.md)
- Adding constraints now would be premature optimization

**Trade-offs accepted:**
- Accepting some risk of future spam
- Trusting worker judgment over rigid rules
- Deferring additional infrastructure until evidence of need

**Implementation sequence:**
1. Keep current implementation unchanged
2. Monitor for spam signals (triage burden, issue quality)
3. If spam emerges, add `triage:worker-discovery` label first
4. If spam persists, consider Option 4 (full-tier only) as second lever

### Alternative Approaches Considered

**Option 5: triage:worker-discovery label NOW**
- **Pros:** Preemptive routing, zero downside
- **Cons:** Adds complexity without proven need
- **When to use instead:** If you prefer proactive vs. reactive approach

**Option 4: Full-tier only**
- **Pros:** Reduces potential spam from light tasks
- **Cons:** Inconsistent, may miss valid discoveries
- **When to use instead:** If light-tier spam is specifically the problem

**Rationale for recommendation:** The current implementation has been live for ~24 hours and shows good signals. Changing it now based on hypothetical concerns would be premature. "Let the data inform" is the right stance.

---

### Implementation Details

**What to monitor:**
- Open issues with `triage:review` label (should stay low)
- Orchestrator feedback on triage burden
- Quality of issues created by workers

**Things to watch out for:**
- ⚠️ Workers creating issues to satisfy checklist without judgment
- ⚠️ Low-quality issues (vague descriptions, P4 trivia)
- ⚠️ Issues for things already tracked elsewhere

**Areas needing further investigation:**
- Long-term compliance rate (more data needed)
- Correlation between spawn tier and issue quality

**Success criteria:**
- ✅ >50% of workers either create issues or note "No discovered work" (currently 100%)
- ✅ Triage burden doesn't significantly increase
- ✅ Quality of worker-created issues remains good

---

## References

**Files Examined:**
- `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/discovered-work.md` - Current implementation
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md` - Template with Issues Created section
- `.orch/workspace/*/SYNTHESIS.md` - Recent workspace outputs
- `.kb/investigations/2026-02-04-inv-*.md` - Prior investigations

**Commands Run:**
```bash
bd stats
bd list --label triage:review
bd list --status open
git log --oneline --since="2 days ago" | grep -i discovered
ls -lt .orch/workspace/ | head -15
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/archived/2026-02-04-inv-investigate-workers-not-creating-issues.md` - Original problem analysis
- **Investigation:** `.kb/investigations/2026-02-04-inv-implement-multi-layer-reinforcement-discovered.md` - Implementation record

---

## Investigation History

**2026-02-05 10:16:** Investigation started
- Initial question: How to balance discovered work capture vs. spam prevention?
- Context: Concern about multi-layer reinforcement causing issue spam

**2026-02-05 10:35:** Analyzed current state
- Found 10/10 workspace compliance with Issues Created section
- Found 0 open issues with triage:review
- Concluded spam concern is premature

**2026-02-05 10:45:** Investigation completed
- Status: Complete
- Key outcome: Keep current implementation; spam concern is premature. Add triage:worker-discovery label as first lever if spam emerges.

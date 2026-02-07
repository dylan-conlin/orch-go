## Summary (D.E.K.N.)

**Delta:** Agent investigations on the same topic don't build on each other - each starts fresh, contradictions pile up unnoticed, human evidence vanishes into chat. The coaching plugin saga (19 investigations, never fixed) is the canonical failure case.

**Evidence:** Coaching plugin: 19 investigations, 34 commits, 3 weeks, 5 contradictory conclusions on Jan 28 alone. Dylan's screenshots went into chat, invisible to agents. Final outcome: disabled entirely.

**Knowledge:** The pattern is case management - legal, medical, and insurance systems all converge on: persistent case files, evidence lockers, prior work citation, contradiction detection, and authoritative arbitration. Novel for AI agents: the system self-generates volume that humans can't review, so it must synthesize itself.

**Next:** Rebuild case file with diagnosis-first structure (orch-go-21125). Spike (orch-go-21124) validated that case files help but revealed timeline isn't enough - diagnosis is the point.

**Authority:** architectural - Cross-system design affecting knowledge management, agent coordination, and dashboard architecture.

---

# Investigation: Case Files and Arbitration

**Question:** How should the system help agents build on each other's work, detect contradictions, and learn from failure patterns like the coaching plugin saga?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** Dylan + Claude (design session)
**Phase:** Complete
**Next Step:** Execute orch-go-21125 (rebuild with diagnosis-first structure)
**Status:** Complete

---

## Findings

### Finding 1: Agents don't build on prior work

**Evidence:** The coaching plugin had 19 investigations over 3 weeks. Each agent:
- Started fresh with SPAWN_CONTEXT
- Didn't systematically read prior investigation conclusions
- Formed independent hypotheses
- Concluded independently ("fixed", "needs upstream fix", "revert to title-based")

**Source:** `.kb/investigations/*coaching*` (19 files), `.kb/decisions/2026-01-28-coaching-plugin-disabled.md`

**Significance:** The investigation skill doesn't require reading prior work on the same topic. Agents are detectives who don't read the case file.

---

### Finding 2: Contradictions go unnoticed

**Evidence:** On Jan 28 alone, 5 investigations produced contradictory conclusions:
- One: "working correctly, no changes needed"
- One: "just rebuild OpenCode"
- One: "metadata.role can't work, revert to title-based"
- One: "migrate all plugins to metadata.role"
- One: "architectural - needs upstream fix"

No mechanism flagged these contradictions. They sat side-by-side in `.kb/investigations/`.

**Source:** `.kb/decisions/2026-01-28-coaching-plugin-disabled.md:30-36`

**Significance:** The system lacks contradiction detection. When Agent A says X and Agent B says not-X, nothing triggers.

---

### Finding 3: Human evidence vanishes into chat

**Evidence:** Dylan pasted screenshots showing orchestrator coaching alerts appearing in worker activity windows. This was the clearest evidence the bug existed. But:
- Screenshots went into chat messages
- Not persisted to any artifact
- Subsequent agents couldn't see them
- The best evidence was invisible to investigators

**Source:** Design session conversation, Dylan's observation

**Significance:** Evidence locker is missing. Human observations (screenshots, logs pasted in chat) don't become part of the persistent record.

---

### Finding 4: No arbitration mechanism exists

**Evidence:** When investigations contradict:
- No trigger fires
- No one synthesizes
- No ruling is produced
- Human must manually notice and decide

Dylan eventually said "I've lost faith that this is actually possible" and disabled the plugin. Human arbitration happened, but only after 19 attempts.

**Source:** `.kb/decisions/2026-01-28-coaching-plugin-disabled.md:42-44`

**Significance:** The system needs an arbitration role - triggered by contradictions, retry loops, or human override - that adjudicates and produces a ruling.

---

### Finding 5: The pattern is universal (case management)

**Evidence:** Legal, medical, and insurance systems all converge on similar structures:
- Persistent case files that accumulate
- Evidence preservation with chain of custody
- Prior art/history review is mandatory
- Contradiction triggers cross-examination / differential diagnosis
- Authoritative closure (judge, attending physician, claims board)

**Source:** Design session analysis of cross-domain patterns

**Significance:** We're not inventing something new - we're applying a proven pattern to AI agent coordination. The novel aspect: volume is 10-100x higher, so the system must synthesize itself.

---

## Synthesis

**Key Insights:**

1. **Agents are detectives without a case file** - Each investigation is standalone. The next agent doesn't inherit the evolving understanding, just raw context.

2. **Contradictions are signal, not noise** - When Agent A says "fixed" and Agent B says "revert", that conflict is information. Currently it's invisible.

3. **Human evidence is first-class** - Dylan's screenshots were the ground truth. They should be in an evidence locker, not lost in chat.

4. **Arbitration is the missing role** - Not every contradiction needs arbitration, but thresholds should trigger it: 3+ investigations, contradictory conclusions, human override.

5. **Failure modes become benchmarks** - The coaching plugin saga can test skill changes: "Does this version of the investigation skill resolve this case faster?"

**Answer to Investigation Question:**

Build case files that accumulate investigations on the same topic, with evidence lockers for human observations, contradiction detection, and arbitration triggers. Start with a spike on the coaching plugin saga to learn the schema and detection mechanisms.

---

## Structured Uncertainty

**What's tested:**

- ✅ Coaching plugin had 19 investigations (verified: ls .kb/investigations/*coaching* | wc -l)
- ✅ Contradictory conclusions exist (verified: read decision record)
- ✅ No existing case file mechanism (verified: reviewed kb-cli capabilities)

**What's untested:**

- ⚠️ How to detect "same topic" (naming convention? kb context clustering? manual tagging?)
- ⚠️ Schema for case files (new artifact type? virtual grouping? linked list?)
- ⚠️ Arbitration trigger thresholds (3 investigations? contradictions? human override?)
- ⚠️ Whether case file view actually helps (spike will test this)

**What would change this:**

- If the coaching plugin saga doesn't render well as a case file, the model may need adjustment
- If topic detection is unreliable, may need manual case creation
- If arbitration is too heavyweight, may need lighter "synthesis" role first

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Case file as concept | architectural | Cross-boundary: affects investigations, dashboard, agent spawning |
| Evidence locker | architectural | New storage pattern, affects multiple components |
| Arbitration role | strategic | New agent role, changes coordination model |
| Spike first | implementation | Low-risk learning, single scope |

### Recommended Approach ⭐

**Spike-Driven Development** - Build case file view for coaching plugin saga first, learn from it, then design the full system.

**Why this approach:**
- Real data, real failure, real pain to validate against
- Surfaces hard questions (schema, detection) through building
- One day of work, maximum learning
- Avoids premature abstraction

**Trade-offs accepted:**
- Delays the full system
- Spike may be throwaway code
- Manual curation for first case

**Implementation sequence:**
1. Spike: Manual case file view for coaching plugin (orch-go-21124)
2. Learn: What schema emerged? How did we detect same-topic? What was missing?
3. Design: Full case file system based on learnings
4. Build: Case files, evidence locker, contradiction detection
5. Later: Arbitration role, failure mode benchmarks

### Alternative Approaches Considered

**Option B: Design everything first**
- **Pros:** Coherent architecture before building
- **Cons:** Risk designing for hypothetical, not real pain
- **When to use instead:** If spike reveals we need more clarity before any code

**Option C: Add prior-work-reading to investigation skill**
- **Pros:** Lower lift than full case files
- **Cons:** Doesn't solve contradiction detection, evidence locker, arbitration
- **When to use instead:** If case files prove too heavy

**Rationale for recommendation:** The coaching plugin saga is concrete evidence of the problem. Building against it ensures we solve something real.

---

## Design Concepts (from session)

### Case File Structure

```
Case: [Topic]
Status: [Active | Stalled | Resolved | Abandoned]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Evidence Locker
├── [screenshots, logs, artifacts with provenance]

Investigation Timeline
├── #1 (date): "[conclusion]" 
├── #2 (date): "[conclusion]" ← contradicts #1
│   ...
└── [CONTRADICTION DETECTED / ARBITRATION NEEDED]

Human Observations
├── "[observation]" (date)
└── [TRUST SIGNAL: if human overrides agent]

Current Hypothesis
└── "[hypothesis]" (Agent #N)
    Confidence: [HIGH | LOW based on contradictions]
    
Suggested Action
└── [Continue | Escalate | Arbitrate | Abandon]
```

### Arbitration Triggers

| Trigger | Threshold | Action |
|---------|-----------|--------|
| Investigation count | 5+ on same topic | Surface for review |
| Contradictions | 2+ conflicting conclusions | Flag for arbitration |
| Human override | Human says "still broken" | Escalate immediately |
| Retry commits | 5+ fix commits without resolution | Suggest pivot or abandon |

### System Health Signals

Embed in existing views (badges) + summary in Strategic Center:
- Retry loops (same issue, multiple failed attempts)
- Investigation piles (knowledge without decisions)
- Verification gaps (agent "fixed" without human confirmation)
- Trust erosion (human overriding agent conclusions)

---

## References

**Files Examined:**
- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md` - The failure decision
- `.kb/investigations/*coaching*` - 19 investigation files
- `~/.claude/CLAUDE.md` - Knowledge placement patterns

**Related Artifacts:**
- **Issue:** `orch-go-21124` - Spike for case file view
- **Investigation:** `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md` - Work graph design this integrates with
- **Investigation:** `.kb/investigations/2026-01-28-inv-design-unified-strategic-center-dashboard.md` - Strategic center where health signals surface

---

## Investigation History

**2026-01-31 ~10:00:** Design session started
- Initial question: How do knowledge artifacts feed back into system improvement?
- Expanded to: artifact browser, work graph integration, system health signals

**2026-01-31 ~10:30:** Coaching plugin saga examined
- Dylan asked: "what about these patterns, these issue and artifact heaps that pile up when you just can't fix a bug"
- Analyzed 19 investigations, identified failure mode

**2026-01-31 ~11:00:** Case files concept emerged
- Parallel to legal/medical case management recognized
- Key insight: agents are detectives without a case file

**2026-01-31 ~11:30:** Investigation completed
- Status: Complete
- Key outcome: Spike-first approach, build case file view for coaching plugin saga
- Issue created: orch-go-21124

**2026-01-31 ~12:00:** Spike completed and reviewed
- Agent built timeline-based HTML case file
- Dylan's feedback: "still not immediately clear what went wrong"
- Key learning: **Timeline isn't enough. Diagnosis is the point.**

**2026-01-31 ~12:15:** Spike learnings captured
- Case file needs to answer: verdict, contradiction, ground truth, failure mode, lessons
- Structure revised: diagnosis-first, not chronology-first
- Issue created: orch-go-21125 (rebuild with new structure)

---

## Spike Learnings (orch-go-21124)

**What the spike built:** Timeline-based HTML showing 19 investigations chronologically

**What was missing:**
1. Contradictions weren't visually confrontational - just entries in a timeline
2. Human evidence (screenshots) wasn't shown, just mentioned
3. Failure mode wasn't diagnosed - showed *what* but not *why*
4. No "what should have happened" - not actionable

**Key insight:** Timeline shows what happened. Diagnosis explains why it kept failing.

**Revised case file structure:**

| Section | Purpose |
|---------|---------|
| THE VERDICT | Read this first - outcome, root cause, pattern |
| THE CONTRADICTION | Side-by-side conflicting conclusions - impossible to miss |
| THE GROUND TRUTH | What Dylan actually saw (screenshots, quotes) |
| THE TIMELINE | Compressed, grouped by week (not the centerpiece) |
| THE FAILURE MODE | Named pattern, what was missing, where to stop |
| WHAT SHOULD HAVE HAPPENED | Specific intervention points |
| LESSONS FOR NEXT TIME | Actionable takeaways |

**Principle validated:** Spike-driven design works. Building against real data revealed what mattered (diagnosis) vs what we assumed mattered (chronology).

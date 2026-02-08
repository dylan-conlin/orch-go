## Summary (D.E.K.N.)

**Delta:** Case Files are valid (fill real gap), but three architectural choices exist: (A) views-not-types per chronicle precedent, (B) types-with-dependencies per investigation failures, (C) defer until active churn exists.

**Evidence:** Two architect reviews reached different conclusions from same evidence - one applied chronicle precedent (kn-160dc9: views-over-types), other identified kb tooling gaps blocking type adoption. Coaching plugin saga validates gap (19 investigations, contradictions, human evidence lost), but was one-time event (no active churn).

**Knowledge:** The chronicle precedent is strong (minimal taxonomy, synthesized narratives), but Case Files have different structure/purpose than chronicles. Tooling gaps are real (no `kb create case-file`, no lifecycle, no templates). Meta-learning value is clear but priority depends on whether investigation churn is active problem.

**Next:** Recommend Option C (defer) - acknowledge gap exists, keep spike as example, revisit when investigation churn recurs. If Dylan wants immediate adoption, Option A (views) is lower lift than Option B (types with tooling work).

**Authority:** architectural - Knowledge taxonomy addition, cross-boundary decision

---

# Investigation: Architect Synthesis on Case File Artifact Type

**Question:** Should Case Files be added to Knowledge Placement table, and if so, as what: formal artifact type, generated view, or deferred concept?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** og-feat-architect-review-case-31jan-e298
**Phase:** Complete
**Next Step:** Dylan decides between three options
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Potential addition to artifact taxonomy

---

## Context

Design session (orch-go-21124, orch-go-21125) proposed Case Files as new artifact type for multi-attempt investigation failures. Spawn context asked architect to review:

1. Is it distinct from existing artifacts?
2. Are trigger signals right?
3. Does structure capture what matters?
4. How integrate with kb tooling?
5. Auto-detect vs manual trigger?
6. What's the lifecycle?

Two parallel architect reviews occurred (og-feat-architect-review-case-31jan-e298 and og-arch-review-case-file-31jan-97c8), reaching different architectural stances. This synthesis addresses all six questions with three options for decision.

---

## Findings

### Finding 1: Case Files ARE distinct from existing artifacts

**Evidence:**
- **Investigation** (`.kb/investigations/`) - Single-attempt, question → findings → answer
- **Post-mortem** (`.kb/post-mortems/`) - Production incidents, principle violations
- **Decision** (`.kb/decisions/`) - Architectural choices
- **Case Files** (proposed) - Multi-attempt investigation failures with contradictions, human evidence, diagnosis

Coaching plugin evidence: 19 investigations, 5 contradictory conclusions on Jan 28, Dylan's screenshots lost in chat, each agent started fresh.

**Source:** 
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` (existing taxonomy)
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:30-88`
- `.kb/case-files/coaching-plugin-worker-detection.html` (spike)

**Significance:** Gap is real. Not investigation (multi-attempt not single), not post-mortem (agent failure not production), not decision (failure capture not choice).

---

### Finding 2: Trigger signals validated but detection mechanism undefined

**Evidence:** Proposed triggers from design session:
- 3+ investigations on same topic ✅ (coaching: 19)
- Contradictory conclusions ✅ (coaching: 5 on Jan 28)
- Human override ("still broken") ✅ (Dylan repeatedly said "still getting alerts")
- Issue reopened ⚠️ (unclear in beads workflow)

But detection mechanism unspecified:
- Manual: Orchestrator recognizes pattern, creates case file
- Automated: `kb reflect --type contradiction` (not built, per SPAWN_CONTEXT.md:123)

**Source:** 
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:228-236`
- `.kb/case-files/coaching-plugin-worker-detection.html:562-618`

**Significance:** Triggers grounded in real failure, but "how to detect?" unanswered. Manual is realistic near-term, automated is aspirational.

---

### Finding 3: Structure is diagnosis-first and validated by spike

**Evidence:** Spike learnings (orch-go-21124 → orch-go-21125):
- Timeline-first FAILED: "still not immediately clear what went wrong"
- Diagnosis-first WORKED: 8 sections (Verdict, Contradiction, Ground Truth, Timeline, Failure Mode, Evidence, What Should Have Happened, Lessons)
- Key insight: "Timeline shows what happened. Diagnosis explains why it kept failing."

**Source:**
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:292-316`
- `.kb/case-files/coaching-plugin-worker-detection.html` (991-line HTML demonstrating structure)

**Significance:** Structure is battle-tested against real failure. Diagnosis-first is THE value, not timeline.

---

### Finding 4: Chronicle precedent argues for views-not-types

**Evidence:** Prior decision kn-160dc9 (2025-12-21):
> "Chronicle should be a view over existing artifacts, not new artifact type. Reason: Minimal taxonomy principle; source data already exists; value is in narrative synthesis."

Chronicles and Case Files share DNA:
- Both synthesize existing data (investigations, decisions, beads)
- Both are narratives, not new source data
- Both answer meta-questions ("how did X evolve?" vs "why did X fail?")

Current artifact taxonomy: Only 2 types in `pkg/kb/artifacts.go` (Investigation, Decision). Intentionally minimal.

**Source:**
- `.kb/investigations/2026-01-31-inv-review-case-file-artifact-type.md:30-45` (chronicle analysis)
- Decision kn-160dc9
- `pkg/kb/artifacts.go:16-19`

**Significance:** Strong precedent exists for "views over types." Minimal taxonomy is intentional design, not laziness.

---

### Finding 5: kb tooling gaps block type adoption

**Evidence:** If Case Files become artifact type, need:
- ❌ `kb create case-file` command (doesn't exist)
- ❌ Template in kb system (doesn't exist)
- ❌ Lifecycle states (Active/Resolved/Abandoned - undefined)
- ❌ Promotion workflow (investigation → case file - undefined)
- ❌ `kb reflect --type contradiction` (not built)

Current state: Manual HTML in `.kb/case-files/` with README.

**Source:**
- `kb --help` output (no case-file commands)
- `.kb/case-files/` directory (manual spike, no tooling)
- SPAWN_CONTEXT.md:123 ("kb reflect: manual discipline until ships")

**Significance:** Without tooling, Case Files won't be created consistently. Type adoption blocked by implementation work.

---

### Finding 6: Case Files are meta-learning, not operational

**Evidence:** Proposed use: "Coaching plugin saga can test skill changes: 'Does this version resolve faster?'"

8 sections focus on retrospective:
- Verdict (what went wrong)
- Failure Mode (why it kept failing)
- What Should Have Happened (intervention points)
- Lessons (actionable takeaways)

This is system improvement, not immediate problem-solving.

**Source:**
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md:119`
- `.kb/case-files/coaching-plugin-worker-detection.html:911-980` (Lessons section)

**Significance:** Value is real but not urgent. Without active investigation churn, creating case files is premature. Coaching plugin was one-time event.

---

## Synthesis

**Three Architectural Options:**

### Option A: Views (per chronicle precedent) ⭐ if immediate adoption desired

**Approach:** Case Files remain generated HTML reports in `.kb/case-files/`. Optional: add `kb case-file generate <topic>` for automation.

**Pros:**
- Aligns with chronicle decision (kn-160dc9)
- No artifact taxonomy changes
- Lower implementation lift
- Working proof (coaching plugin HTML)

**Cons:**
- Not queryable like investigations
- No first-class status
- Manual unless automation built

**When:** If Dylan wants case files available now, this is fastest path.

---

### Option B: Types (with dependencies)

**Approach:** Add CaseFile to ArtifactType enum, build kb tooling, define lifecycle.

**Pros:**
- First-class artifact status
- Queryable, standard lifecycle
- Fits with "investigation failure" mental model

**Cons:**
- Violates minimal taxonomy principle
- Requires kb tooling work (create, templates, lifecycle)
- Precedent (chronicle) argues against

**When:** If investigation churn is active problem AND worth tooling investment.

---

### Option C: Defer ⭐⭐ recommended

**Approach:** Acknowledge gap, keep spike as example, add to Knowledge Placement as "future artifact," revisit when churn recurs.

**Pros:**
- No premature tooling investment
- Spike validated concept
- Can adopt later when need is active
- Avoids taxonomy expansion without clear ongoing need

**Cons:**
- Gap remains unfilled
- If churn happens, will need manual case files

**When:** If coaching plugin was one-time event, not active pattern. Wait for second case to validate recurring need.

---

## Answers to Spawn Context Questions

**1. Is Case Files distinct from existing artifacts?**
→ Yes. Multi-attempt investigation failures with contradictions are not covered by investigation/post-mortem/decision. (Finding 1)

**2. Are trigger signals right?**
→ Yes, grounded in coaching plugin evidence (3+ investigations, contradictions, human override). Detection mechanism needs clarification (manual vs automated). (Finding 2)

**3. Does structure capture what matters?**
→ Yes. Diagnosis-first structure validated by spike. Timeline-first failed, diagnosis-first worked. (Finding 3)

**4. How integrate with kb tooling?**
→ Two paths: (A) Views = no kb changes, optional automation. (B) Types = requires `kb create case-file`, templates, lifecycle. (Findings 4, 5)

**5. Auto-detect vs manual trigger?**
→ Manual realistic near-term (orchestrator recognizes pattern). Automated aspirational (kb reflect not built). (Finding 2)

**6. What's the lifecycle?**
→ Undefined. Need states (Active/Resolved/Abandoned) and rules for creation/update/closure before type adoption. (Finding 5)

---

## Recommendation

**Recommended: Option C (Defer) until investigation churn recurs.**

**Rationale:**
1. Coaching plugin was one-time event (Jan 10-28), not active pattern
2. No investigation churn since (27 days)
3. Meta-learning value real but not urgent (Finding 6)
4. Spike validated concept - keep as example
5. If churn recurs, revisit with two data points

**Knowledge Placement table entry (deferred):**

| You have... | Put it in... | Trigger |
|-------------|--------------|---------|
| Multi-attempt investigation failure with contradictions | `.kb/case-files/` (manual HTML spike, future: `kb case-file generate`) | 3+ investigations on same topic + contradictions + human override (deferred until churn recurs) |

**If Dylan wants immediate adoption:**
→ Option A (Views) is lower lift. Add to Knowledge Placement, keep as HTML, optional `kb case-file generate` later.

---

## Structured Uncertainty

**What's tested:**
- ✅ Chronicle precedent exists and applies (verified kn-160dc9)
- ✅ Coaching plugin saga validates gap (verified 19 investigations, contradictions)
- ✅ Spike structure works (verified 991-line HTML)
- ✅ kb tooling gaps exist (verified: no case-file commands in `kb --help`)

**What's untested:**
- ⚠️ Is coaching plugin one-time or recurring pattern? (need second data point)
- ⚠️ Would automated case file generation be used? (automation not built)
- ⚠️ Do case files improve investigation skill? (benchmark not run)

**What would change this:**
- Second investigation churn event → reconsider Option B or A
- Active investigation failures piling up → urgency increases
- Dylan says "this is recurring pain" → immediate adoption justified

---

## Implementation if Option A chosen

1. Add to Knowledge Placement table in `~/.claude/CLAUDE.md`
2. Update `.kb/case-files/README.md` with when/how to create
3. Optional later: `kb case-file generate <topic>` command

## Implementation if Option B chosen

1. Add CaseFile to `pkg/kb/artifacts.go` ArtifactType enum
2. Create case file template
3. Add `kb create case-file` command
4. Define lifecycle states (Active/Resolved/Abandoned)
5. Build contradiction detection (or manual workflow)

## Implementation if Option C chosen

1. Document in Knowledge Placement as "deferred until churn recurs"
2. Keep coaching plugin spike as example
3. Revisit when second investigation churn event occurs

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md` - Original design session
- `.kb/case-files/coaching-plugin-worker-detection.html` - Validated spike
- `.kb/investigations/2026-01-31-inv-review-case-file-artifact-type.md` - Chronicle precedent analysis
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Existing taxonomy
- `~/.claude/CLAUDE.md` - Knowledge Placement table
- `pkg/kb/artifacts.go` - Artifact type definitions
- Decision kn-160dc9 - Chronicle precedent

**Related Artifacts:**
- **Epic:** orch-go-21124 (spike), orch-go-21125 (rebuild)
- **Decision:** `.kb/decisions/2026-01-28-coaching-plugin-disabled.md`
- **Investigation:** `.kb/investigations/2026-01-31-design-case-files-and-arbitration.md`

---

## Investigation History

**2026-01-31 10:32:** Investigation started
- Context: Two parallel architect reviews with different stances
- Goal: Synthesize perspectives, answer spawn context questions

**2026-01-31 10:45:** Synthesis complete
- Option A: Views (fast, aligns with precedent)
- Option B: Types (requires tooling, violates minimal taxonomy)
- Option C: Defer (wait for recurring need)
- Recommendation: Option C unless Dylan wants immediate adoption (then A)

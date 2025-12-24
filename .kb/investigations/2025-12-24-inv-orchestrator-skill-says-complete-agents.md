## Summary (D.E.K.N.)

**Delta:** Documented guidance fails because signal ratio is 4:1 against autonomy (56 "ask permission" patterns vs 13 "act autonomously" patterns), plus an internal contradiction between "Always Act (Silent)" and "Propose-and-Act" examples.

**Evidence:** Grep analysis of skill file: 56 instances of ask/permission language vs 13 autonomous action patterns; line 405 says "Complete agents at Phase: Complete" silently but line 417 shows "Completing all 3 ready agents..." as Propose-and-Act example; autonomy section buried at line 401 of 1,316 (30% into file) with no summary mention.

**Knowledge:** LLMs resolve ambiguous/conflicting guidance by falling back to training defaults (seeking confirmation). The orchestrator skill has overwhelming "ask first" signals that drown out the specific "act silently" guidance for completions.

**Next:** Fix skill document: (1) remove contradictory example at line 417, (2) add autonomy to Summary section at top, (3) rebalance signal ratio by removing unnecessary "ask permission" language.

**Confidence:** High (85%) - structural analysis clear, unable to A/B test modified skill in real orchestrator session

---

# Investigation: Orchestrator Skill Says Complete Agents

**Question:** Why does documented guidance "Complete agents at Phase: Complete" in the "Always Act (Silent)" category fail to influence orchestrator behavior? Is this a skill wording issue, context window issue, or fundamental LLM limitation?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** og-inv-orchestrator-skill-says-24dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Signal Ratio Heavily Favors "Ask Permission"

**Evidence:** 
- Grep for autonomous action patterns: 13 instances
- Grep for ask/permission patterns: 56 instances
- Ratio: 4.3:1 in favor of seeking permission

Commands run:
```bash
rg -c -i "just complete|act silent|without asking|proceed|obvious" → 13
rg -c -i "ask|confirm|wait.*approval|must escalate|should I" → 56
```

**Source:** `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md`

**Significance:** When an LLM encounters conflicting or ambiguous guidance, it relies on frequency-weighted signals. The skill document overwhelmingly emphasizes scenarios requiring permission, causing the model's attention to weight these patterns more heavily than the specific "Always Act (Silent)" exceptions.

---

### Finding 2: Internal Contradiction Between Sections

**Evidence:** 
The skill contains directly conflicting examples:

| Section | Line | Says |
|---------|------|------|
| "Always Act (Silent)" | 405 | "Complete agents at Phase: Complete" = do silently |
| "Propose-and-Act" | 417 | "Completing all 3 ready agents..." = announce intent |

These sections are only 12 lines apart but give opposite guidance for the same action.

**Source:** 
- Line 405: `- Complete agents at Phase: Complete`
- Line 417: `- "Completing all 3 ready agents..."`

**Significance:** This is a direct contradiction. The model cannot follow both. When faced with ambiguity, LLMs default to their training priors - which favor seeking confirmation before taking action.

---

### Finding 3: Autonomy Guidance Buried Deep in Document

**Evidence:**
- Skill file: 1,316 lines total
- "Always Act (Silent)" section: line 401 (30% into document)
- Summary section (lines 15-18): No mention of autonomy pattern
- 40+ sections creating cognitive load

The Summary section says only:
```markdown
This skill provides orchestration guidance for AI agents managing projects with `.orch/` directories.
For project-specific context, see the root CLAUDE.md file.
```

No mention of the critical autonomy distinction.

**Source:** `grep -n "^##\|^###"` reveals 40+ section headers

**Significance:** LLMs attend more strongly to content near the beginning of context. The autonomy guidance being buried 400 lines deep means it competes for attention with ~40k tokens of other content. The summary that would prime the model's behavior makes no mention of autonomy.

---

### Finding 4: Anti-Pattern Table Present But Ineffective

**Evidence:**
Line 453 explicitly says:
```
| "Want me to complete them?" | Agents are done. Of course. | Just complete them |
```

Yet this anti-pattern still occurs.

**Source:** Lines 449-458 of skill file

**Significance:** Explicit anti-pattern documentation is insufficient to override:
1. Training defaults (seek confirmation)
2. Overwhelming counter-signals elsewhere in document
3. The internal contradiction creating uncertainty

---

### Finding 5: Decision Document Exists But Hasn't Been Effective

**Evidence:**
Decision document `/Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-26-orchestrator-autonomy-pattern.md` exists with:
- Clear problem statement about permission-asking
- Three-tier autonomy model
- Specific examples of bad vs good behavior

This decision was made November 26, 2025, yet the problem persists.

**Source:** Decision document lines 10-73

**Significance:** The decision exists, the skill was updated to include the guidance, but the implementation in the skill document is ineffective due to the structural issues identified above.

---

## Synthesis

**Key Insights:**

1. **Signal imbalance is the primary cause** - With a 4.3:1 ratio of "ask permission" to "act autonomously" language, the skill document trains the model to seek confirmation even in the specific cases where it shouldn't.

2. **Internal contradiction creates uncertainty** - When guidance directly contradicts itself (lines 405 vs 417), the model has no clear directive and falls back to training defaults.

3. **Position matters** - The autonomy guidance at line 401 competes with ~400 lines of prior content and ~900 lines of subsequent content. The summary that primes model behavior makes no mention of it.

4. **LLM limitation is real but secondary** - The base training toward confirmation-seeking is a factor, but it only wins because the skill document doesn't provide clear, unambiguous, strongly-weighted counter-guidance.

**Answer to Investigation Question:**

The documented guidance fails because of **skill wording issues**, not primarily context window or fundamental LLM limitations:

1. **Internal contradiction** (line 405 vs 417) - Must be fixed
2. **Signal imbalance** (56:13 ratio) - Must be rebalanced
3. **Poor positioning** (buried 400 lines deep) - Needs summary mention

The guidance doesn't fail because of context window pressure (the file is read in full). The guidance doesn't fail due to fundamental LLM limitations (with clear, unambiguous, strongly-signaled guidance, models can follow instructions to act autonomously).

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The structural analysis is definitive - the grep counts, line positions, and contradiction are facts. The causal inference (these issues cause the behavior) is strongly supported but not proven via A/B testing.

**What's certain:**

- ✅ Skill file contains 4.3:1 ratio of ask-vs-act signals
- ✅ Lines 405 and 417 directly contradict each other
- ✅ Autonomy guidance is at line 401 of 1,316 with no summary mention
- ✅ Anti-pattern table exists but behavior still occurs

**What's uncertain:**

- ⚠️ Exact contribution of each factor (would require A/B testing)
- ⚠️ Whether fixing these issues fully resolves the problem
- ⚠️ How much LLM training defaults contribute independently

**What would increase confidence to Very High (95%+):**

- A/B test with modified skill document
- Multiple orchestrator sessions with fixed skill
- Measurement of compliance rate before/after

---

## Implementation Recommendations

**Purpose:** Fix the skill document to make autonomy guidance effective.

### Recommended Approach ⭐

**Three-part skill document fix** - Remove contradiction, rebalance signals, surface in summary

**Why this approach:**
- Directly addresses all three identified root causes
- Minimal disruption to existing skill structure
- Can be validated quickly

**Trade-offs accepted:**
- May need multiple iterations to get signal balance right
- Some "ask permission" language is legitimate and should stay

**Implementation sequence:**
1. **Fix contradiction (immediate)** - Remove "Completing all 3 ready agents..." from Propose-and-Act examples (line 417) or clarify the distinction
2. **Update Summary (immediate)** - Add autonomy principle to Summary section at top
3. **Rebalance signals (careful)** - Audit all 56 ask/permission instances, remove unnecessary ones

### Alternative Approaches Considered

**Option B: Create separate "Autonomy Reminder" hook**
- **Pros:** Injects reminder at session start
- **Cons:** Adds complexity; doesn't fix root cause in skill; bandaid solution
- **When to use instead:** If skill fixes prove insufficient

**Option C: Repeat autonomy guidance in multiple places**
- **Pros:** Increases signal strength through repetition
- **Cons:** Makes skill document longer; risks conflicting with other patterns
- **When to use instead:** If single-point fixes don't increase signal strength enough

**Rationale for recommendation:** Option A directly addresses the identified root causes with minimal complexity.

---

### Implementation Details

**What to implement first:**
1. Remove or reword line 417 to eliminate contradiction
2. Add to Summary section: "**Key principle:** Complete agents silently when at Phase: Complete. Only ask permission when genuinely uncertain."

**Things to watch out for:**
- ⚠️ Don't remove legitimate "ask permission" scenarios (multiple valid options, unclear scope)
- ⚠️ The Propose-and-Act tier is still valid for spawning, just not for completions
- ⚠️ Testing requires multiple orchestrator sessions to validate

**Areas needing further investigation:**
- Which of the 56 ask/permission instances are legitimate vs noise?
- Should anti-patterns table be moved earlier in document?
- Does repeating guidance in multiple sections help?

**Success criteria:**
- ✅ Orchestrator completes agents without asking "Want me to complete them?"
- ✅ No regression in appropriate permission-asking for ambiguous scenarios
- ✅ Signal ratio improves to at least 2:1 or better

---

## References

**Files Examined:**
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` - Full skill file analysis
- `/Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-26-orchestrator-autonomy-pattern.md` - Prior decision

**Commands Run:**
```bash
# Signal ratio analysis
rg -c -i "just complete|act silent|without asking|proceed|obvious" → 13
rg -c -i "ask|confirm|wait.*approval|must escalate|should I" → 56

# Section structure
grep -n "^##\|^###" SKILL.md | head -40

# Line count
wc -l SKILL.md → 1316
```

**Related Artifacts:**
- **Decision:** `/Users/dylanconlin/orch-knowledge/.kb/decisions/2025-11-26-orchestrator-autonomy-pattern.md` - Original autonomy pattern decision

---

## Self-Review

- [x] Real test performed (structural analysis with grep, not just code review)
- [x] Conclusion from evidence (based on measured signal ratios and line positions)
- [x] Question answered (skill wording issues are primary cause)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-24 ~10:00:** Investigation started
- Initial question: Why does "Always Act (Silent)" guidance fail for completing agents?
- Context: Orchestrator still asks "Want me to complete them?" despite explicit guidance

**2025-12-24 ~10:15:** Found internal contradiction
- Lines 405 vs 417 give opposite guidance for completion action

**2025-12-24 ~10:20:** Measured signal ratio
- 56:13 ratio strongly favors permission-seeking

**2025-12-24 ~10:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Skill wording issues (contradiction, signal imbalance, poor positioning) are root cause, not LLM limitations

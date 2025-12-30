## Summary (D.E.K.N.)

**Delta:** All three "gaps" are already addressed by existing principles or pragmatic implementations: (1) cost/efficiency has `MaxInvestigationsInContext=3` and Progressive Disclosure, (2) trust dynamics are explicitly covered in orchestrator skill's "Orchestrator Autonomy" section, (3) principle failure modes are prevented by the 4-criteria test for new principles and Evolve by Distinction pattern.

**Evidence:** Found `MaxInvestigationsInContext = 3` in kbcontext.go limiting context; "Orchestrator Autonomy" section spans 80+ lines defining when to act vs ask; principles.md lines 416-421 establish 4 strict criteria (tested, generative, non-derivable, has teeth) that prevent over-principling.

**Knowledge:** The system has implicit guards against these concerns but they're not visible as "principles" - they're embedded in implementations and skill documentation. Making them explicit principles would violate the "must have teeth" criterion since they haven't caused real failures.

**Next:** Close - no new principles recommended. Suggest adding brief "Practical Limits" section to principles.md acknowledging these implicit guards exist.

---

# Investigation: Three Gaps in Meta-Orchestration Principles

**Question:** Are there real gaps in the principles around (1) cost/efficiency, (2) human-AI trust dynamics, and (3) principle failure modes? Should new principles be added?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-inv-investigate-three-gaps-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Cost/Efficiency is Handled Pragmatically, Not Principally

**Evidence:** The codebase already contains explicit cost/efficiency guards:

1. **`MaxInvestigationsInContext = 3`** (pkg/spawn/kbcontext.go:59)
   - Hard limit on investigations surfaced in spawn context
   - Rationale documented: "With 500+ investigations, including all matches would explode context"

2. **Token validation** (pkg/spawn/tokens.go)
   - `ValidateContextSize()` checks if spawn context is within limits
   - `ContextTooLargeError` returned when exceeding size limits
   - Warnings at 80% of safe limit

3. **Progressive Disclosure principle** already addresses this:
   > "TLDR first. Key sections next. Full details available."
   > "Why: Context windows are finite. Attention is limited. Front-load the signal."

4. **Surfacing Over Browsing principle** explicitly mentions cost:
   > "Agents lack persistent memory and spatial intuition. Every file read costs context."

**Source:** 
- `pkg/spawn/kbcontext.go:57-59`
- `pkg/spawn/tokens.go:148-171`
- `~/.kb/principles.md:104-127`

**Significance:** Cost/efficiency is handled through IMPLEMENTATION (hard limits, validation) rather than PRINCIPLE (guidance). This is correct - principles guide decisions where judgment is needed; fixed limits don't need judgment.

---

### Finding 2: Trust Dynamics are Extensively Documented in Orchestrator Skill

**Evidence:** The orchestrator skill contains an entire section "Orchestrator Autonomy (Proactive Execution)" spanning 80+ lines that explicitly addresses trust dynamics:

**Three-tier autonomy model:**
1. **Always Act (Silent)** - Completing agents, monitoring, status checks
2. **Propose-and-Act** - "Spawning for X..." (can be interrupted with "wait")
3. **Actually Ask** - Only for genuine ambiguity, multiple valid tradeoffs

**The Mind-Reading Test:**
> "If Dylan were doing this himself, would he pause to ask himself this question?" If no → act. If yes → ask.

**Explicit anti-patterns:**
- "Want me to complete them?" → Just do it
- "Option A it is" → Wait for approval after presenting options
- "Runaway Orchestrator" pattern → Self-select → implement (forbidden)

**Reference:** `~/.claude/skills/meta/orchestrator/SKILL.md:614-695`

**Significance:** The "gap" around human-AI trust dynamics is actually well-covered - just not as a principle, because it's role-specific guidance, not a universal constraint. The orchestrator skill is the right location for this, not principles.md.

---

### Finding 3: Principle Failure Modes are Prevented by Strict Criteria

**Evidence:** The principles.md file includes explicit criteria for new principles:

> **When adding new principles:**
> - Must be tested (emerged from actual problems)
> - Must be generative (guides future decisions)
> - Must not be derivable from existing principles
> - Must have teeth (violation causes real problems)

And the final test:
> "The test for new principles: Can you trace it to a specific failure? If not, it's not a principle yet."

Additionally, the **Evolve by Distinction** meta-principle provides a mechanism for principles to evolve without accumulating:
> "When problems recur, ask 'what are we conflating?' Make the distinction explicit."

**Source:** 
- `~/.kb/principles.md:416-421`
- `~/.kb/principles.md:278-296`

**Significance:** The concern about "Gate Over Remind becoming bureaucratic" or "Session Amnesia leading to over-documentation" is addressed by requiring every principle to trace to a specific failure. Principles without real teeth would fail the test and shouldn't be added.

---

### Finding 4: The Investigation Question Contains a Premise Violation

**Evidence:** The investigation prompt asked about gaps in three areas but framed them as "no guidance on X." Testing reveals:

| Claimed Gap | Actual Status |
|-------------|---------------|
| No guidance on context window cost | `MaxInvestigationsInContext=3`, token validation, Progressive Disclosure |
| No guidance on when to defer vs push back | 80+ lines in "Orchestrator Autonomy" section |
| No meta-principle about principles having scope | 4-criteria test prevents unbounded principles |

**Source:** Search results across `~/.kb/principles.md`, `~/.claude/skills/meta/orchestrator/SKILL.md`, `pkg/spawn/`

**Significance:** This is an instance of "Premise Before Solution" - the question assumed gaps exist without first verifying. The gaps are perceived, not real.

---

## Synthesis

**Key Insights:**

1. **Pragmatic limits belong in code, not principles** - The system correctly embeds cost guards in implementations (MaxInvestigationsInContext, token validation) rather than adding principles about "don't over-contextualize." Principles are for judgment calls; fixed limits aren't judgment calls.

2. **Role-specific guidance belongs in skills, not principles** - Trust dynamics (when to act autonomously vs defer) are properly documented in the orchestrator skill, not principles. Principles should be universal; trust calibration is role-dependent.

3. **Principle hygiene is built into the promotion criteria** - The 4-criteria test (tested, generative, non-derivable, has teeth) already prevents principle accumulation. You can't add "don't over-apply Gate Over Remind" as a principle because that violation hasn't caused real problems.

**Answer to Investigation Question:**

**No new principles are recommended.** The three "gaps" are either:
1. Already handled by implementation (cost/efficiency)
2. Documented in role-specific skills (trust dynamics)
3. Prevented by existing criteria (principle failure modes)

Adding principles for these would violate the "must have teeth" criterion - there's no evidence these gaps have caused real failures.

---

## Structured Uncertainty

**What's tested:**

- ✅ MaxInvestigationsInContext exists and is set to 3 (verified: read kbcontext.go:57-59)
- ✅ Orchestrator Autonomy section exists with three-tier model (verified: read SKILL.md:614-695)
- ✅ 4-criteria test for new principles exists (verified: read principles.md:416-421)

**What's untested:**

- ⚠️ Whether the current limits (3 investigations, token thresholds) are optimal
- ⚠️ Whether the orchestrator autonomy guidance is actually followed in practice
- ⚠️ Whether the 4-criteria test is consistently applied to principle candidates

**What would change this:**

- Finding would be wrong if evidence emerged of repeated failures due to over-contextualization, mismatched autonomy expectations, or principle bloat
- Finding would be wrong if current safeguards proved insufficient under load

---

## Implementation Recommendations

**Recommended Approach: Document Implicit Guards**

Instead of adding new principles, add a brief "Practical Limits" section to the "Applying Principles" section of principles.md that acknowledges:

> **Built-in guards:**
> These concerns are handled through implementation and role-specific guidance rather than principles:
> - **Cost/efficiency:** Context limits in spawn code, Progressive Disclosure
> - **Trust dynamics:** Orchestrator Autonomy section in orchestrator skill
> - **Principle scope:** 4-criteria test prevents accumulation

**Why this approach:**
- Makes implicit guards visible without elevating them to principle status
- Prevents future investigations from "discovering" these gaps
- Maintains principle hygiene (only tested failures become principles)

**Trade-offs accepted:**
- Slightly longer principles.md
- Risk of documentation drift if implementations change

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Existing principles and criteria
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator autonomy guidance
- `pkg/spawn/kbcontext.go` - Context limit implementation
- `pkg/spawn/tokens.go` - Token validation

**Commands Run:**
```bash
# Search for cost-related patterns
grep -r "MaxInvestigations|tiered|context.*limit" pkg/spawn/

# Search for autonomy patterns
grep -r "authority|autonomy|decide|escalate" ~/.claude/skills/meta/orchestrator/SKILL.md
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-spawn-context-include-related-prior.md` - Documented MaxInvestigationsInContext rationale
- **Investigation:** `.kb/investigations/2025-12-28-inv-post-mortem-orchestrator-session-inefficiency.md` - Example of real failure that could lead to new principle

---

## Self-Review

- [x] Real test performed (searched code and documentation)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered (no new principles needed)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-30:** Investigation started
- Initial question: Are there three gaps in meta-orchestration principles?
- Context: Spawned to investigate perceived gaps

**2025-12-30:** Evidence gathered
- Found cost/efficiency is handled pragmatically via MaxInvestigationsInContext and token validation
- Found trust dynamics are documented in orchestrator skill's Autonomy section
- Found principle failure modes are prevented by 4-criteria test

**2025-12-30:** Investigation completed
- Status: Complete
- Key outcome: All three "gaps" are already addressed; no new principles recommended

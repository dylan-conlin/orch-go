---
linked_issues:
  - orch-go-jb0w
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The orch system produces symptom fixes because completion gates reward correctness (tests pass, artifacts exist) but not depth (design questioned), and design questioning only triggers on repeated failure, not first-time success.

**Evidence:** Analyzed 3 skill files (systematic-debugging, feature-impl, investigation) - all gate on completion criteria without design review; examined 2 SYNTHESIS.md files from today showing "root cause addressed" without "design questioned"; verified `orch complete` checks artifacts not design implications.

**Knowledge:** Proximate root cause (immediate technical fix) ≠ distal root cause (why design allows this failure); skills conflate these; design questioning exists but is conditional (3+ fails or git history patterns), not mandatory.

**Next:** Add "## Design Implications" phase gate to systematic-debugging skill - require non-empty section asking "Why does the system architecture allow this failure mode?" before completion.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Why Does Orch System Consistently Fix Symptoms Rather Than Finding Root Causes?

**Question:** Why does the orch system consistently produce symptom fixes rather than root cause analysis? Is this due to skill framing, issue descriptions, or fundamental prompting patterns?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent (og-inv-orch-system-consistently-30dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting approach

**Evidence:** Task requests analysis of skill framing, issue descriptions, and prompting patterns.

**Source:** SPAWN_CONTEXT.md - mentions 4 symptom fixes from today's session (untracked visibility, agent error reporting, stale architect, headless spawn race)

**Significance:** Will examine each of these examples plus skill files to understand the pattern.

---

### Finding 2: Skills SAY root cause but GATE on completion

**Evidence:** 
- systematic-debugging has "The Iron Law: NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST" (line 30-31)
- BUT Completion Criteria (line 347) gates on "Root cause identified", "Fix implemented", "Tests passing", "Smoke-test passed"
- No gate on "Is this the DEEPEST root cause?" or "Why does this failure mode exist?"
- Success criteria: "You understand root cause, not just symptoms" (line 116) - but this is a Phase 1 checkpoint, NOT a completion gate

**Source:** 
- `/Users/dylanconlin/.claude/skills/systematic-debugging/SKILL.md:30-34, 116, 347-358`

**Significance:** The skill encourages depth but gates on "did you fix it?" not "did you question the design?". Agent incentive is to reach completion criteria, not to maximize depth.

---

### Finding 3: Issue descriptions are already symptom-framed when agents receive them

**Evidence:** 
- `orch-go-rhcs`: "Untracked agents have no phase visibility - CLI shows stalled, dashboard shows active"
- `orch-go-6vr6`: "Fix headless spawn race condition - add message verification after SendPrompt"
- Both frame as "X isn't working, fix X" rather than "Why does this architecture produce X?"
- Issue titles describe WHAT to fix, not WHY it's happening

**Source:**
- `bd show orch-go-rhcs` - issue description in beads
- `bd show orch-go-6vr6` - issue description in beads

**Significance:** By the time agents see issues, the framing has already narrowed to symptom-fixing. The "5 whys" would need to happen BEFORE issue creation, at symptom-reporting time.

---

### Finding 4: issue-creation skill explicitly asks WHY but limits to "15-30 minutes"

**Evidence:**
- issue-creation skill Phase 2: "Goal: Understand WHY, not just WHERE" (line 68)
- Time budget: "10-20 min" for root cause investigation (line 66)
- "If you'd need to investigate before fixing, use this skill" (line 39)
- Result: Creates an issue with P-S-E structure, not a design question

**Source:**
- `/Users/dylanconlin/.claude/skills/issue-creation/SKILL.md:66-87`

**Significance:** The skill investigates the symptom deeply enough to create a good issue, but doesn't ask "should this class of problem exist at all?" That's a different question.

---

### Finding 5: Today's fixes show the pattern - symptom addressed, design unquestioned

**Evidence from SYNTHESIS.md files:**

1. **Untracked agents visibility** (og-debug-untracked-agents-no-30dec):
   - Fix: Add `.phase` file mechanism, add stalled detection to dashboard
   - Root cause addressed: "No phase reporting mechanism without beads"
   - Design question NOT asked: "Why do we support spawning without tracking? Why is tracking opt-out?"
   
2. **Headless spawn race** (og-debug-fix-headless-spawn-30dec):
   - Fix: Add `WaitForMessage` + `SendPromptWithVerification` with retry
   - Root cause addressed: "No delay between CreateSession and SendPrompt"
   - Design question NOT asked: "Why does CreateSession return before the session is ready to receive messages? Should the API change?"

**Source:**
- `.orch/workspace/og-debug-untracked-agents-no-30dec/SYNTHESIS.md`
- `.orch/workspace/og-debug-fix-headless-spawn-30dec/SYNTHESIS.md`

**Significance:** Both agents correctly identified the "proximate root cause" (the immediate technical reason). Neither questioned the "distal root cause" (why the system was designed to allow this failure mode).

---

### Finding 6: No "5 whys" or design-questioning forcing function exists

**Evidence:**
- `rg "5.?why" ~/.claude/skills/` - 0 matches
- Orchestrator skill doesn't have a "before closing, ask if the design is questioned" step
- `orch complete` verification checks for tests, commits, artifacts - NOT for design review
- No skill or phase asks "Why did the system allow this failure mode?"

**Source:**
- Searched skills directory for "5 whys" pattern
- Reviewed orchestrator skill completion workflow
- Reviewed `pkg/verify/` for completion checks

**Significance:** There's no forcing function that interrupts the "fix → complete" flow to ask about design. Agents have no structural reason to question beyond the immediate fix.

---

### Finding 7: investigation skill asks for testing, not design questioning

**Evidence:**
- Core rule: "You cannot conclude without testing" (line 29)
- Self-review checklist: "Real test performed", "Conclusion from evidence", "Question answered" (lines 211-218)
- No checklist item: "Did you ask why this failure mode exists in the design?"
- D.E.K.N. asks for "Next: Recommended action" but not "Design implications"

**Source:**
- `/Users/dylanconlin/.claude/skills/investigation/SKILL.md:29, 211-218, 89-99`

**Significance:** Investigation skill optimizes for empirical rigor (good!) but not for design depth. An agent can complete a valid investigation without ever questioning whether the underlying system should change.

---

### Finding 8: systematic-debugging HAS the "question architecture" guidance but it's buried and conditional

**Evidence:**
- Phase 4 Implementation (line 34): "If 3+ Fixes Failed OR Whack-a-Mole Pattern Detected: Question Architecture"
- Phase 1 Root Cause (line 40-49): "If whack-a-mole pattern detected: STOP fixing symptoms, Investigate systemic cause"
- BUT these are conditional triggers (3+ fails, 2+ similar fixes in history)
- The guidance says "STOP and question fundamentals" but doesn't make it a mandatory step

**Source:**
- `/Users/dylanconlin/.claude/skills/worker/systematic-debugging/phases/phase1-root-cause.md:40-49`
- `/Users/dylanconlin/.claude/skills/worker/systematic-debugging/phases/phase4-implementation.md:34-64`

**Significance:** The architectural questioning exists but is triggered only by repeated failure or git history patterns. A FIRST-TIME fix doesn't trigger design questioning. The agent can complete with a perfect root-cause fix that doesn't question why this class of bug can exist.

---

## Synthesis

**Key Insights:**

1. **Skills gate on "correct fix" not "design depth"** - All skills require agents to find and fix root causes, but the completion criteria measure correctness (tests pass, phase complete, artifacts exist) not depth (did you question the design?). An agent can perfectly follow systematic-debugging and still produce a symptom fix.

2. **Issue descriptions pre-frame as symptoms** - By the time agents receive issues, the framing is already "fix X" not "why does X happen at all?" The 5-whys would need to happen at issue-creation time, not fix time. But issue-creation skill also doesn't ask this.

3. **Design questioning is conditional, not mandatory** - The systematic-debugging skill has excellent "whack-a-mole detection" and "question architecture" guidance, but it triggers only on: (a) 3+ failed fixes, or (b) 2+ similar fixes in git history. A novel, first-time bug that gets fixed correctly on the first attempt never triggers design review.

4. **Proximate vs distal root cause distinction is missing** - Skills ask for "root cause" but mean the immediate technical cause. They don't distinguish between "why did this code fail?" (proximate) and "why does the system architecture allow this failure mode?" (distal).

**Answer to Investigation Question:**

The orch system fixes symptoms instead of finding root causes because:

1. **Completion gates reward correctness, not depth.** Skills measure "did you fix it correctly?" not "did you question the design?"

2. **Design questioning is triggered only by failure.** Success on the first attempt doesn't trigger architectural review.

3. **Issue framing is already symptom-level.** By the time agents see issues, the question is "fix X" not "why does X exist?"

4. **No forcing function for design questions.** There's no checklist item, completion gate, or skill phase that asks "Why does this failure mode exist in the design?"

This is **solvable with process changes**, not a fundamental limitation of prompting. The skill infrastructure already supports conditional design questioning (whack-a-mole detection). The fix is to make design questioning unconditional for certain issue types, or add a completion gate that asks for design implications.

---

## Structured Uncertainty

**What's tested:**

- ✅ Skill files contain "root cause" language but gate on completion, not depth (verified: read all 3 skill files)
- ✅ Today's SYNTHESIS.md files show "root cause addressed" but not "design questioned" (verified: read 2 synthesis files)
- ✅ Issue descriptions frame as "fix X" not "why X" (verified: read 2 issue descriptions)
- ✅ Design questioning triggers are conditional in systematic-debugging (verified: read phase1-root-cause.md, phase4-implementation.md)
- ✅ `orch complete` verification checks artifacts exist, not design review (verified: read pkg/verify/check.go)

**What's untested:**

- ⚠️ Adding a "design question" gate would actually change behavior (not tested - would require experiment)
- ⚠️ Agents would follow a mandatory design questioning step (not tested - agent compliance varies)
- ⚠️ Issue-creation skill changes would improve downstream agent depth (not tested)

**What would change this:**

- Finding would be wrong if: agents with design-questioning gates still produce symptom fixes
- Finding would be wrong if: issue framing doesn't matter (same issue framed differently produces same fix)
- Finding would be wrong if: conditional design questioning (current) produces same depth as unconditional

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add "Design Implications" phase gate to systematic-debugging** - Before completion, agents must document "Why does this failure mode exist in the design?" as a separate artifact section.

**Why this approach:**
- Builds on existing infrastructure (completion gates already exist)
- Doesn't block work on first-time fixes (still allows completion)
- Creates a record of design questions for future architectural review
- Minimal friction - just one additional section, not a full investigation

**Trade-offs accepted:**
- Doesn't guarantee agents actually think deeply (can fill with shallow answers)
- Creates more artifacts to review (orchestrator burden)
- May slow completion of simple bugs

**Implementation sequence:**
1. **Add SKILL-CONSTRAINTS to systematic-debugging**: `design-implication: |Investigation file must have "## Design Implications" section with non-empty content`
2. **Update investigation template**: Add "## Design Implications" section with prompt: "Why does the system architecture allow this failure mode? What would prevent this class of bug?"
3. **Update `orch complete` verification**: Check for non-empty design implications section when skill is systematic-debugging

### Alternative Approaches Considered

**Option B: Make whack-a-mole detection unconditional**
- **Pros:** Uses existing pattern, encourages historical research
- **Cons:** Requires git history search, may miss novel bugs, adds friction
- **When to use instead:** For projects with deep git history and recurring patterns

**Option C: Add "5 whys" to issue-creation skill**
- **Pros:** Catches depth earlier in the pipeline, better issue descriptions
- **Cons:** Increases issue-creation time (15-30 min → 30-45 min), may over-engineer simple issues
- **When to use instead:** When issue quality is the bottleneck, not fix quality

**Option D: Orchestrator post-completion review with design question**
- **Pros:** Human in the loop, no agent behavior change needed
- **Cons:** Moves burden to orchestrator, doesn't improve agent thinking
- **When to use instead:** When skeptical of agent design reasoning

**Rationale for recommendation:** Option A (design implications gate) provides forcing function without excessive friction. It builds on existing verification infrastructure and creates a record for future review. Unlike options B/C/D, it directly targets the gap identified: agents complete without questioning design.

---

### Implementation Details

**What to implement first:**
- Update investigation template with "## Design Implications" section
- Update systematic-debugging SKILL.md self-review checklist to require this section
- Test with one spawn before rolling out verification gate

**Things to watch out for:**
- ⚠️ Agents may fill section with shallow content ("This is a one-off bug")
- ⚠️ May need examples in skill guidance of what good design implications look like
- ⚠️ Should not apply to all skills (feature-impl doesn't need design questioning for new features)

**Areas needing further investigation:**
- What do "good" design implications look like? Need examples.
- Should there be a follow-up mechanism (create issue for design work)?
- How does this interact with architect skill? (should design implications trigger architect spawn?)

**Success criteria:**
- ✅ systematic-debugging agents produce investigation files with "## Design Implications" section
- ✅ At least 50% of design implications sections ask "why does this failure mode exist?" type questions
- ✅ Some design implications lead to follow-up architectural work (issues created)

---

## References

**Files Examined:**
- `/Users/dylanconlin/.claude/skills/systematic-debugging/SKILL.md` - Main debugging skill, looking for root cause language and completion gates
- `/Users/dylanconlin/.claude/skills/feature-impl/SKILL.md` - Feature implementation skill for comparison
- `/Users/dylanconlin/.claude/skills/investigation/SKILL.md` - Investigation skill for comparison
- `/Users/dylanconlin/.claude/skills/issue-creation/SKILL.md` - Issue creation to see how issues are framed
- `/Users/dylanconlin/.claude/skills/worker/systematic-debugging/phases/phase1-root-cause.md` - Phase 1 guidance for root cause
- `/Users/dylanconlin/.claude/skills/worker/systematic-debugging/phases/phase4-implementation.md` - Phase 4 guidance for design questioning
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go` - Completion verification logic
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` - Spawn context template
- `.orch/workspace/og-debug-untracked-agents-no-30dec/SYNTHESIS.md` - Today's example of symptom fix
- `.orch/workspace/og-debug-fix-headless-spawn-30dec/SYNTHESIS.md` - Today's example of symptom fix

**Commands Run:**
```bash
# Check today's workspaces
ls -la .orch/workspace/ | grep "30dec"

# Show beads issues for context
bd show orch-go-rhcs
bd show orch-go-6vr6

# Search for design questioning patterns
grep "recur|systemic|deeper" /Users/dylanconlin/.claude/skills/
```

**Related Artifacts:**
- **Epic:** `bd show orch-go-jb0w` - Parent epic for this investigation
- **Workspace:** `.orch/workspace/og-debug-untracked-agents-no-30dec/SYNTHESIS.md` - Example symptom fix
- **Workspace:** `.orch/workspace/og-debug-fix-headless-spawn-30dec/SYNTHESIS.md` - Example symptom fix

---

## Investigation History

**2025-12-30 11:12:** Investigation started
- Initial question: Why does orch system consistently fix symptoms rather than root causes?
- Context: Epic orch-go-jb0w identified 4 symptom fixes from today's session

**2025-12-30 11:25:** Key finding: Skills gate on completion, not depth
- Discovered that systematic-debugging has root cause language but completion criteria don't require design questioning

**2025-12-30 11:35:** Key finding: Design questioning is conditional
- Discovered whack-a-mole detection exists but only triggers on failure, not success

**2025-12-30 11:45:** Investigation complete
- Status: Complete
- Key outcome: Symptom-fixing is caused by completion gates that reward correctness over depth; solvable with process changes

---

## Self-Review

- [x] Real test performed (read actual files, checked today's SYNTHESIS.md examples)
- [x] Conclusion from evidence (based on skill file analysis and concrete examples)
- [x] Question answered (why symptom fixes happen)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

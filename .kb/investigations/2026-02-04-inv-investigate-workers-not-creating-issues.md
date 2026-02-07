<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Workers skip discovered work issue creation because the instruction is buried deep in skills (lines 220-226 in a 267-line doc), not present in completion checklists, and not mentioned in spawn context templates.

**Evidence:** Analyzed 4 skills (investigation, systematic-debugging, feature-impl, architect) and 8+ worker workspaces. Only systematic-debugging has discovered work as a checklist item in completion criteria. Workers that DO create issues (1.75%) were explicitly tasked with auditing/creating issues.

**Knowledge:** Workers follow checklists literally - instructions buried as subsections under "Self-Review" are skipped. Successful issue creation correlates with: (1) explicit task to create issues, or (2) checklist-gated instruction.

**Next:** Recommend architectural fix: Add discovered work to worker-base skill completion criteria + add explicit section to SYNTHESIS.md template.

**Authority:** architectural - Crosses multiple skills and templates, requires coordinated changes across worker-base, spawn context, and SYNTHESIS.md

---

# Investigation: Why Workers Are Not Creating Issues for Discovered Work

**Question:** Why do 98.25% of workers never run `bd create` despite skills explicitly instructing them to track discovered work?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Investigation Worker
**Phase:** Complete
**Next Step:** None - recommendations ready for architect review
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Discovered Work Instruction Is Buried in Skill Documents

**Evidence:** Line number analysis of where "Discovered Work" instructions appear:

| Skill | Total Lines | Discovered Work Line | Position | In Checklist? |
|-------|-------------|---------------------|----------|---------------|
| investigation | 267 | 220 | 82% deep | NO |
| systematic-debugging | 758 | 629 | 83% deep | YES (line 709) |
| feature-impl | 555 | 431 | 78% deep | NO* |
| architect | 675 | 337 | 50% deep | YES (line 639) |

*feature-impl has it as a subsection under self-review (### Discovered Work) but NOT in the completion criteria checklist (lines 511-524).

**Source:**
- `/Users/dylanconlin/.claude/skills/src/worker/investigation/SKILL.md:220-228`
- `/Users/dylanconlin/.claude/skills/src/worker/systematic-debugging/SKILL.md:629-639, 700-713`
- `/Users/dylanconlin/.claude/skills/src/worker/feature-impl/SKILL.md:431-436, 511-524`
- `/Users/dylanconlin/.claude/skills/src/worker/architect/SKILL.md:337-369, 630-654`

**Significance:** Workers following the completion criteria checklist won't see discovered work unless it's in that checklist. The investigation skill's completion section (lines 249-257) doesn't mention discovered work at all.

---

### Finding 2: Worker-Base Skill Does Not Include Discovered Work

**Evidence:** Searched worker-base skill for discovered work or bd create instructions:

```bash
grep -i "discovered work\|bd create" /Users/dylanconlin/.claude/skills/src/shared/worker-base/
# No matches found
```

The worker-base skill defines:
- Authority delegation
- Hard limits (constitutional)
- Beads progress tracking
- Phase reporting
- Session complete protocol

But NOT discovered work tracking.

**Source:** `/Users/dylanconlin/.claude/skills/src/shared/worker-base/` (all files)

**Significance:** Since worker-base is inherited by ALL worker skills, this is the ideal place to add discovered work as a universal requirement. Currently, each skill has to implement it independently (and does so inconsistently).

---

### Finding 3: Workers That Create Issues Were Explicitly Tasked To Do So

**Evidence:** Sampled workspaces that successfully created issues:

1. **og-inv-audit-work-graph-02feb-5faa** - Task: "Audit Work Graph design docs"
   - Created 4 issues (orch-go-21121.2, .3, .4)
   - Closed 4 stale issues
   - The TASK was explicitly to audit and create issues for gaps

2. **og-feat-create-beads-issues-18jan-622c** - Task: "Create beads issues from recommend-yes investigations"
   - Created 10 issues
   - The TASK was explicitly to create issues

**Contrasted with workers that FOUND issues but didn't create them:**

3. **og-inv-verify-end-end-04feb-3cbd** - Found bug: `bd absorb` doesn't close absorbed issue
   - SYNTHESIS.md has "Bug to File" section documenting the bug
   - NO `bd create` was run
   - Bug was documented but not tracked

4. **og-arch-address-daemon-interaction-04feb-0d66** - Had unexplored questions
   - SYNTHESIS.md has 4+ potential issue candidates in "Unexplored Questions"
   - NO `bd create` was run for any

**Source:**
- `.orch/workspace/archived/og-inv-audit-work-graph-02feb-5faa/SYNTHESIS.md`
- `.orch/workspace/archived/og-feat-create-beads-issues-18jan-622c/SYNTHESIS.md`
- `.orch/workspace/og-inv-verify-end-end-04feb-3cbd/SYNTHESIS.md`
- `.orch/workspace/og-arch-address-daemon-interaction-04feb-0d66/SYNTHESIS.md`

**Significance:** Workers create issues when the TASK is to create issues. When discovered work is incidental to the task, workers document it but don't act on it. This suggests the instruction needs to be more prominent and gated.

---

### Finding 4: SYNTHESIS.md Template Lacks "Issues Created" Section

**Evidence:** The SYNTHESIS.md template (`pkg/spawn/context.go:724-845`) has these sections:
- TLDR
- Delta (What Changed) - Files Created, Files Modified, Commits
- Evidence (What Was Observed)
- Knowledge (What Was Learned)
- Next (What Should Happen)
- Unexplored Questions
- Session Metadata

Missing: **"Issues Created"** or **"Discovered Work"** section.

Workers see "Unexplored Questions" and document discoveries there, but there's no explicit section that says "list the beads issues you created for discovered work."

**Source:** `pkg/spawn/context.go:724-845` (DefaultSynthesisTemplate)

**Significance:** The SYNTHESIS.md template shapes what workers document. Without an "Issues Created" section, workers don't think to create issues. Adding this section would make issue creation a natural part of the workflow.

---

### Finding 5: Spawn Context Template Does Not Mention Discovered Work

**Evidence:** The SPAWN_CONTEXT.md template (`pkg/spawn/context.go:54-404`) covers:
- Task, Tier, KB Context
- Session complete protocol
- Authority delegation
- Deliverables
- Beads progress tracking

But does NOT mention:
- Creating issues for discovered work
- Tracking bugs/tech debt found during the session

**Source:** `pkg/spawn/context.go:54-404` (SpawnContextTemplate)

**Significance:** The spawn context is the primary instruction document workers receive. If discovered work isn't mentioned here, workers won't prioritize it.

---

## Synthesis

**Key Insights:**

1. **Instruction Placement Matters** - Workers follow checklists literally. Discovered work instructions buried as subsections (not checklist items) are skipped. The systematic-debugging skill has discovered work in its completion criteria checklist and is the model to follow.

2. **Worker-Base Is The Leverage Point** - Adding discovered work to worker-base would propagate to ALL worker skills automatically. Currently each skill implements it independently (and inconsistently).

3. **Template Sections Shape Behavior** - Workers document what the template asks for. SYNTHESIS.md's "Unexplored Questions" section captures discoveries but doesn't prompt action. An "Issues Created" section would make issue creation explicit.

4. **Task Framing Drives Compliance** - Workers who were explicitly tasked with creating issues did so. Workers who found issues incidentally documented them but didn't create beads issues. The instruction needs to feel like a requirement, not a suggestion.

**Answer to Investigation Question:**

Workers don't create issues for discovered work because:

1. **The instruction is buried** - 78-83% into skill documents, as a subsection rather than a checklist item
2. **No universal requirement** - Worker-base doesn't include it, so implementation varies by skill
3. **Templates don't prompt it** - Neither SYNTHESIS.md nor SPAWN_CONTEXT.md has explicit "discovered work" sections
4. **Framing is passive** - "If you found bugs, create issues" vs. mandatory checklist item

The 1.75% of workers who DO create issues were either explicitly tasked with issue creation or following skills (systematic-debugging, architect) that have discovered work as a checklist-gated requirement.

---

## Structured Uncertainty

**What's tested:**

- ✅ Investigation skill lines 220-226 contain discovered work instruction but NOT in completion criteria (verified: read skill file)
- ✅ Systematic-debugging skill has discovered work in completion criteria checklist at line 709 (verified: read skill file)
- ✅ Worker-base skill does NOT contain discovered work instructions (verified: grep search returned no matches)
- ✅ SYNTHESIS.md template does NOT have "Issues Created" section (verified: read DefaultSynthesisTemplate in context.go)
- ✅ Workers that created issues were explicitly tasked to do so (verified: read 4 workspace SYNTHESIS.md files)

**What's untested:**

- ⚠️ Whether adding to worker-base would actually improve compliance (needs A/B test)
- ⚠️ Whether workers run out of context before reaching discovered work section (hypothesis not tested)
- ⚠️ Whether `bd create` verbosity/slowness is a friction factor (not measured)

**What would change this:**

- Finding would be wrong if workers with discovered work in checklists also fail to create issues
- Finding would be wrong if workers explicitly skip discovered work section despite checklist presence
- Finding would be wrong if most discovered work is actually tracked (2,110 workers creating 111 issues might be appropriate)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add discovered work to worker-base skill | architectural | Crosses all worker skills, changes universal behavior |
| Add "Issues Created" section to SYNTHESIS.md | architectural | Affects all spawn templates and verification |
| Add discovered work reminder to spawn context | architectural | Affects spawn template generation in pkg/spawn |

### Recommended Approach ⭐

**Multi-Layer Reinforcement** - Add discovered work to worker-base completion section, SYNTHESIS.md template, and make it a checklist-gated requirement.

**Why this approach:**
- Addresses root cause: instruction not prominent enough
- Single change to worker-base propagates to all skills
- SYNTHESIS.md section makes documentation natural
- Checklist gating ensures compliance

**Trade-offs accepted:**
- Slightly longer SYNTHESIS.md files
- More beads issues created (might need better triage)
- Adds friction to completion flow

**Implementation sequence:**

1. **Add to worker-base completion section:**
   ```markdown
   ### Discovered Work (Mandatory)

   Before marking complete:
   - [ ] Reviewed for discovered work (bugs, tech debt, enhancements, questions)
   - [ ] Created issues via `bd create` OR noted "No discovered work" in completion comment

   ```bash
   bd create "description" --type bug|task|feature|question -l triage:review
   ```
   ```

2. **Add "Issues Created" section to SYNTHESIS.md template:**
   ```markdown
   ### Issues Created
   - `orch-go-XXXXX` - Description of discovered work
   - (None - no discovered work during this session)
   ```

3. **Add reminder to spawn context template (optional):**
   ```markdown
   🔍 DISCOVERED WORK: If you find bugs, tech debt, or enhancement ideas during this session,
   create issues via `bd create "description" --type bug|task -l triage:review`.
   Report created issues in your SYNTHESIS.md "Issues Created" section.
   ```

### Alternative Approaches Considered

**Option B: Skill-by-skill fixes**
- **Pros:** Targeted, lower risk
- **Cons:** Inconsistent coverage, maintenance burden
- **When to use instead:** If worker-base changes have unintended side effects

**Option C: Verification gate**
- **Pros:** Enforces compliance via orch complete
- **Cons:** High friction, may block legitimate completions, complex to implement
- **When to use instead:** If instruction-based approaches fail to improve compliance

**Option D: Coaching plugin detection**
- **Pros:** Real-time feedback during session
- **Cons:** Complex, requires plugin infrastructure
- **When to use instead:** If post-hoc fixes don't work

**Rationale for recommendation:** Multi-layer reinforcement addresses the root cause (instruction not prominent) without adding verification gates that could block completions. It's the lowest-friction path to improved compliance.

---

### Implementation Details

**What to implement first:**
- Add to worker-base skill completion section (single file change with maximum impact)
- Add "Issues Created" section to SYNTHESIS.md template

**Things to watch out for:**
- ⚠️ Workers might create low-quality issues to satisfy the checklist
- ⚠️ May increase triage burden on orchestrators
- ⚠️ Need to update orch complete verification to check for "Issues Created" section

**Areas needing further investigation:**
- What makes a "quality" discovered work issue?
- Should discovered work issues require triage:review by default?
- Should there be a mechanism to link discovered work to the parent issue?

**Success criteria:**
- ✅ >50% of workers either create issues or explicitly note "No discovered work"
- ✅ SYNTHESIS.md files consistently have "Issues Created" sections
- ✅ No increase in verification failures from completion gate friction

---

## References

**Files Examined:**
- `/Users/dylanconlin/.claude/skills/src/worker/investigation/SKILL.md` - Investigation skill discovered work placement
- `/Users/dylanconlin/.claude/skills/src/worker/systematic-debugging/SKILL.md` - Model skill with checklist-gated discovered work
- `/Users/dylanconlin/.claude/skills/src/worker/feature-impl/SKILL.md` - Feature impl discovered work placement
- `/Users/dylanconlin/.claude/skills/src/worker/architect/SKILL.md` - Architect skill discovered work tracking
- `/Users/dylanconlin/.claude/skills/src/shared/worker-base/` - Worker base skill (no discovered work)
- `pkg/spawn/context.go` - Spawn context and SYNTHESIS.md templates
- Multiple workspace SYNTHESIS.md files for success/failure pattern analysis

**Commands Run:**
```bash
# Find discovered work instructions
grep -n "## Self-Review\|### Discovered Work\|## Completion" /Users/dylanconlin/.claude/skills/src/worker/investigation/SKILL.md

# Check worker-base for discovered work
grep -i "discovered work\|bd create" /Users/dylanconlin/.claude/skills/src/shared/worker-base/

# List workspace SYNTHESIS.md files mentioning bd create
grep -l "bd create" .orch/workspace/*/SYNTHESIS.md .orch/workspace/archived/*/SYNTHESIS.md
```

**Related Artifacts:**
- **Issue:** `orch-go-21258` - Parent issue for this investigation
- **Blocked by:** `orch-go-21259` - Architect session to implement recommendations

---

## Investigation History

**2026-02-04 14:22:** Investigation started
- Initial question: Why do 98.25% of workers never run `bd create`?
- Context: Action log analysis showed only 37/2110 workers created issues

**2026-02-04 14:30:** Found instruction placement pattern
- Discovered work instructions are 78-83% into skill documents
- Only systematic-debugging has it in completion checklist

**2026-02-04 14:45:** Identified worker-base as leverage point
- Worker-base doesn't include discovered work
- Adding here would propagate to all skills

**2026-02-04 15:00:** Analyzed successful vs failed cases
- Success: Workers explicitly tasked with issue creation
- Failure: Workers found issues but documented without creating

**2026-02-04 15:15:** Investigation completed
- Status: Complete
- Key outcome: Root cause is instruction placement (buried, not checklist-gated). Recommend multi-layer reinforcement: worker-base + SYNTHESIS.md template + spawn context reminder.

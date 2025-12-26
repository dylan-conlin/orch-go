<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Made visual verification skill-aware - only feature-impl (UI-focused) requires visual verification for web/ changes.

**Evidence:** Tests pass, build succeeds. 213 false positives were from architects/investigations modifying web/ files incidentally.

**Knowledge:** Visual verification should only apply to UI work (feature-impl), not all web/ file touches. Used permissive default for unknown skills.

**Next:** Close - fix implemented and tested.

**Confidence:** High (90%) - straightforward fix, comprehensive tests, clear root cause.

---

# Investigation: Orch Review Shows Needs Review False Positives

**Question:** Why does `orch review` show NEEDS_REVIEW for 213 agents with 'web/ files modified but no visual verification evidence found' when most are architects/investigations that don't do UI work?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** og-debug-orch-review-shows-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Visual verification checks all web/ changes without skill awareness

**Evidence:** In `pkg/verify/visual.go:174-218`, `VerifyVisualVerification` only checks:
1. Whether web/ files were modified in recent commits
2. Whether visual verification evidence exists in beads comments or SYNTHESIS.md

There was no check for what skill the agent was using.

**Source:** `pkg/verify/visual.go:174-218`

**Significance:** This causes false positives for non-UI skills (architect, investigation, systematic-debugging) that may incidentally modify web/ files as part of broader work without doing actual UI implementation.

---

### Finding 2: Skill name extraction already exists

**Evidence:** `ExtractSkillNameFromSpawnContext` function in `pkg/verify/skill_outputs.go:50-85` already extracts the skill name from SPAWN_CONTEXT.md using the pattern `## SKILL GUIDANCE (skill-name)`.

**Source:** `pkg/verify/skill_outputs.go:50-85`

**Significance:** We can reuse this function to determine if the skill requires visual verification, avoiding duplicate code.

---

### Finding 3: Only feature-impl is UI-focused

**Evidence:** Reviewing the skill system, `feature-impl` is the primary skill for implementing UI features. Other skills like architect, investigation, systematic-debugging, research, etc. may touch web/ files but don't do UI implementation work.

**Source:** Spawn context skill list, orchestrator skill documentation

**Significance:** Only feature-impl needs visual verification enforcement. Other skills should be excluded to prevent false positives.

---

## Synthesis

**Key Insights:**

1. **Skill awareness is essential for visual verification** - The check should only apply to skills that actually do UI work, not all skills that happen to touch web/ files.

2. **Permissive default prevents false positives** - By defaulting to "no verification required" for unknown skills, we avoid blocking work when skill detection fails or a new skill is added.

3. **Reusing existing code** - The `ExtractSkillNameFromSpawnContext` function already exists and works reliably.

**Answer to Investigation Question:**

The false positives occur because `VerifyVisualVerification` was checking for visual verification evidence on ANY web/ file modification, regardless of whether the skill was doing UI work. Architects, investigations, and debugging agents often modify web/ files incidentally (e.g., updating tests, fixing bugs, creating artifacts) but don't do actual UI implementation that would benefit from visual verification.

The fix adds skill awareness: only `feature-impl` requires visual verification for web/ changes. All other known skills (architect, investigation, systematic-debugging, research, etc.) are explicitly excluded, and unknown skills default to not requiring verification.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The root cause was clearly identified in the code, the fix is straightforward, and comprehensive tests verify the behavior.

**What's certain:**

- ✅ Root cause identified - no skill awareness in visual verification
- ✅ Fix implemented and all tests pass
- ✅ Permissive default prevents future false positives

**What's uncertain:**

- ⚠️ Whether there are other UI-focused skills that should require verification
- ⚠️ Whether edge cases exist for feature-impl modifying web/ files incidentally

**What would increase confidence to Very High (95%+):**

- Verify the fix resolves the 213 false positives in production
- Get orchestrator feedback after running `orch review` post-fix

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Skill-aware visual verification** - Only require visual verification for feature-impl skill.

**Why this approach:**
- Directly addresses root cause
- Reuses existing skill extraction code
- Permissive default prevents future false positives

**Trade-offs accepted:**
- May miss cases where feature-impl modifies web/ files incidentally (accepted)
- New UI-focused skills would need to be added to the list (rare, easy to update)

**Implementation sequence:**
1. Add skill classification maps (included/excluded)
2. Add `IsSkillRequiringVisualVerification` function
3. Modify `VerifyVisualVerification` to check skill before requiring evidence
4. Add tests

### Alternative Approaches Considered

**Option B: Detect UI vs non-UI work from commit messages**
- **Pros:** More context-specific
- **Cons:** Unreliable, depends on commit message quality
- **When to use instead:** Never - too fragile

**Option C: Always require visual verification for web/ changes**
- **Pros:** Strictest enforcement
- **Cons:** Creates false positives (the problem we're solving)
- **When to use instead:** If all agents doing web/ work should have visual verification

---

## References

**Files Examined:**
- `pkg/verify/visual.go` - Main visual verification logic
- `pkg/verify/visual_test.go` - Existing tests
- `pkg/verify/skill_outputs.go` - Skill name extraction function
- `pkg/verify/check.go` - VerifyCompletionFull that calls visual verification

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification
go test -v ./pkg/verify/... -run Visual
go test ./pkg/verify/...
```

---

## Investigation History

**2025-12-25 12:00:** Investigation started
- Initial question: Why 213 false positives for non-UI skills?
- Context: `orch review` flagging architects/investigations incorrectly

**2025-12-25 12:30:** Root cause identified
- Found `VerifyVisualVerification` lacks skill awareness
- Found existing `ExtractSkillNameFromSpawnContext` to reuse

**2025-12-25 13:00:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Skill-aware visual verification implemented and tested

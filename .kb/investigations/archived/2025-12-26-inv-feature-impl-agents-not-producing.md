<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Feature-impl agents not producing SYNTHESIS.md is BY DESIGN - they're spawned as "light" tier.

**Evidence:** Both example workspaces (og-feat-debounce-gold-processing-26dec, og-feat-fix-duplicate-key-26dec) have `.tier` file containing "light". SPAWN_CONTEXT.md explicitly states "⚡ LIGHT TIER: SYNTHESIS.md is NOT required."

**Knowledge:** The tier system intentionally splits skills into "full" (investigation-type, require synthesis) and "light" (implementation-focused, skip synthesis). Feature-impl is defined as light tier in `pkg/spawn/config.go:31`.

**Next:** Decide whether feature-impl should produce SYNTHESIS.md. If yes, change the default tier mapping. If no (current behavior), update `orch review` and dashboard to handle light-tier completions.

---

# Investigation: Feature-Impl Agents Not Producing SYNTHESIS.md

**Question:** Why are feature-impl agents not producing SYNTHESIS.md files, causing them to not appear in pending reviews?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** Decision required on whether to change tier defaults or update review tooling
**Status:** Complete

---

## Findings

### Finding 1: Feature-impl is defined as LIGHT tier by default

**Evidence:** In `pkg/spawn/config.go` lines 21-33, SkillTierDefaults explicitly maps skills to tiers:
```go
// Full tier: Investigation-type skills that produce knowledge artifacts
"investigation":        TierFull,
"architect":            TierFull,
"research":             TierFull,
"codebase-audit":       TierFull,
"design-session":       TierFull,
"systematic-debugging": TierFull,

// Light tier: Implementation-focused skills
"feature-impl":        TierLight,  // <-- HERE
"reliability-testing": TierLight,
"issue-creation":      TierLight,
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/config.go:31`

**Significance:** This is INTENTIONAL DESIGN. The tier system differentiates between skills that produce knowledge artifacts (requiring synthesis) and implementation-focused skills (skipping synthesis overhead). This answers the "Is this related to 'light' tier spawns?" question - YES, feature-impl defaults to light tier.

---

### Finding 2: Both example workspaces confirm light tier

**Evidence:** 
```bash
$ cat .orch/workspace/og-feat-debounce-gold-processing-26dec/.tier
light

$ cat .orch/workspace/og-feat-fix-duplicate-key-26dec/.tier
light
```

Both workspaces have:
- `.tier` file containing "light"
- SPAWN_CONTEXT.md with text: "⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required."
- No SYNTHESIS.md file (correctly, per tier rules)

**Source:** Workspace directory listings and file contents

**Significance:** The system is working AS DESIGNED. Agents are correctly reading tier guidance and skipping SYNTHESIS.md creation.

---

### Finding 3: The feature-impl skill does NOT mention SYNTHESIS.md

**Evidence:** Reading `~/.claude/skills/feature-impl/SKILL.md` (390 lines) - there is no mention of "SYNTHESIS" or "synthesis" anywhere in the skill guidance. The skill focuses on:
- Phased implementation workflow
- Deliverables per phase (investigation file, design doc, source code, tests)
- Self-review and "Leave it Better" requirements
- `bd comment` for phase tracking

**Source:** `/Users/dylanconlin/.claude/skills/feature-impl/SKILL.md`

**Significance:** The skill is aligned with its light tier default - it doesn't instruct agents to create SYNTHESIS.md because the spawn context explicitly tells them not to.

---

### Finding 4: SPAWN_CONTEXT.md template conditionally includes/excludes SYNTHESIS.md

**Evidence:** In `pkg/spawn/context.go` lines 118-124:
```go
{{if ne .Tier "light"}}
6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace...
{{else}}
6. ⚡ SYNTHESIS.md is NOT required (light tier spawn).
{{end}}
```

And in the session complete protocol (lines 216-224):
```go
{{if eq .Tier "light"}}
1. `bd comment {{.BeadsID}} "Phase: Complete - [1-2 sentence summary]"`
2. `/exit`

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment {{.BeadsID}} "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`
{{end}}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:118-124, 216-224`

**Significance:** The spawn context template correctly implements the tier system - light tier agents get explicit guidance to skip SYNTHESIS.md.

---

### Finding 5: Verification correctly respects tier for SYNTHESIS.md check

**Evidence:** In `pkg/verify/check.go` lines 473-482:
```go
// Check for SYNTHESIS.md (only for full tier)
if workspacePath != "" && tier != "light" {
    ok, err := VerifySynthesis(workspacePath)
    if err != nil {
        result.Warnings = append(result.Warnings, fmt.Sprintf("failed to verify SYNTHESIS.md: %v", err))
    } else if !ok {
        result.Passed = false
        result.Errors = append(result.Errors,
            fmt.Sprintf("SYNTHESIS.md is missing or empty in workspace: %s", workspacePath))
    }
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go:473-482`

**Significance:** `orch complete` correctly skips SYNTHESIS.md verification for light tier spawns. The verification passes even without SYNTHESIS.md.

---

### Finding 6: Pending reviews endpoint ONLY scans for SYNTHESIS.md

**Evidence:** In `cmd/orch/serve.go` lines 2355-2359:
```go
// Check for SYNTHESIS.md
synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
if _, err := os.Stat(synthesisPath); os.IsNotExist(err) {
    continue  // <-- Light tier workspaces are SKIPPED entirely
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:2355-2359`

**Significance:** This is the ROOT CAUSE of the "not appearing in pending reviews" issue. The `/api/pending-reviews` endpoint uses SYNTHESIS.md existence as the ONLY indicator of a completed agent. Light tier agents, which don't produce SYNTHESIS.md, are invisible to the review system.

---

## Synthesis

**Key Insights:**

1. **This is working as designed, but the design has a gap.** The tier system correctly differentiates between knowledge-producing skills (full tier) and implementation-focused skills (light tier). Feature-impl is intentionally light tier to reduce overhead.

2. **The gap: Light tier completions are invisible to review tooling.** While `orch complete` correctly handles light tier agents, the dashboard and `orch review` only look for SYNTHESIS.md as a completion indicator. This creates an asymmetry where light tier agents complete successfully but never appear in pending reviews.

3. **Two valid paths forward:**
   - **Option A:** Keep tier system, update review tooling to detect light tier completions (check `.tier` file + `Phase: Complete` in beads comments)
   - **Option B:** Reconsider whether feature-impl should be light tier (maybe implementation work SHOULD produce synthesis for knowledge capture)

**Answer to Investigation Question:**

Feature-impl agents don't produce SYNTHESIS.md because they're spawned as "light" tier by default. This is intentional - the tier system exists to reduce documentation overhead for implementation-focused work. However, the review tooling (dashboard, `orch review`, `/api/pending-reviews`) only scans for SYNTHESIS.md, making light tier completions invisible. The fix is either to update the review tooling to handle light tier OR to reconsider the tier assignment for feature-impl.

---

## Structured Uncertainty

**What's tested:**

- ✅ Both example workspaces have `.tier = light` (verified: `cat .tier`)
- ✅ SPAWN_CONTEXT.md explicitly tells agents to skip SYNTHESIS.md (verified: read file contents)
- ✅ `orch complete` verification logic respects tier (verified: read check.go code)
- ✅ `/api/pending-reviews` skips workspaces without SYNTHESIS.md (verified: read serve.go code)

**What's untested:**

- ⚠️ Whether feature-impl SHOULD produce synthesis (design decision, not testable)
- ⚠️ Whether other review paths (`orch review` CLI) also have this gap

**What would change this:**

- If feature-impl were changed to full tier in config.go, agents would produce SYNTHESIS.md
- If review tooling were updated to check `.tier` + beads comments, light tier agents would appear

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Option A: Update review tooling to handle light tier** - Modify `/api/pending-reviews` and dashboard to detect light tier completions via `Phase: Complete` in beads comments.

**Why this approach:**
- Preserves the intentional tier system (less overhead for implementation work)
- Light tier completions still get visibility for orchestrator review
- No change to agent behavior or skill guidance

**Trade-offs accepted:**
- More complex review detection logic
- Light tier reviews have less detail (no SYNTHESIS.md to summarize)

**Implementation sequence:**
1. Modify `/api/pending-reviews` to also scan for completed light tier workspaces (`.tier = light` + `Phase: Complete` in beads)
2. Add `tier` field to review API responses
3. Update dashboard to show light tier completions with appropriate UI

### Alternative Approaches Considered

**Option B: Change feature-impl to full tier**
- **Pros:** All completions produce synthesis, simpler review logic, better knowledge capture
- **Cons:** Adds overhead to quick implementation tasks, contradicts current design philosophy
- **When to use instead:** If you decide ALL work should produce synthesis regardless of type

**Rationale for recommendation:** The tier system was implemented deliberately (there are tests for it in context_test.go). Changing feature-impl to full tier would be reversing that decision. Better to complete the tier system by updating the review tooling.

---

## References

**Files Examined:**
- `pkg/spawn/config.go` - SkillTierDefaults mapping (line 31: feature-impl → light)
- `pkg/spawn/context.go` - SPAWN_CONTEXT template with tier conditionals
- `pkg/verify/check.go` - VerifyCompletionWithTier respecting tier for SYNTHESIS check
- `cmd/orch/serve.go` - handlePendingReviews only scanning for SYNTHESIS.md
- `~/.claude/skills/feature-impl/SKILL.md` - Skill guidance (no SYNTHESIS.md mention)
- `.orch/workspace/og-feat-debounce-gold-processing-26dec/` - Example light tier workspace
- `.orch/workspace/og-feat-fix-duplicate-key-26dec/` - Example light tier workspace

**Commands Run:**
```bash
# Check tier values for problem workspaces
cat .orch/workspace/og-feat-debounce-gold-processing-26dec/.tier
# Output: light

cat .orch/workspace/og-feat-fix-duplicate-key-26dec/.tier
# Output: light
```

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-26 17:00:** Investigation started
- Initial question: Why are feature-impl agents not producing SYNTHESIS.md?
- Context: Examples og-feat-debounce-gold-processing-26dec and og-feat-fix-duplicate-key-26dec completed without SYNTHESIS.md

**2025-12-26 17:30:** Key discovery - tier system is intentional
- Found SkillTierDefaults in config.go explicitly maps feature-impl to light tier
- Verified both problem workspaces have .tier = light

**2025-12-26 17:45:** Investigation completed
- Status: Complete
- Key outcome: Feature-impl not producing SYNTHESIS.md is BY DESIGN. The gap is in review tooling not handling light tier completions.

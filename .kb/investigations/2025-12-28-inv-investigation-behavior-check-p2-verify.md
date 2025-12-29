<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation skill SKILL.md references "## Test performed" but kb template uses "Structured Uncertainty" with "What's tested" - there is NO verification code to check either section's quality, and many investigations have unfilled placeholder text.

**Evidence:** Searched 123 investigations using "Structured Uncertainty" vs 9 using "Test performed"; found several with placeholder text "[Claim with evidence of actual test performed...]" left unfilled; verified pkg/verify/ has no code checking investigation content quality.

**Knowledge:** The investigation skill is explicitly excluded from test_evidence.go verification (line 33: `"investigation": true` in skillsExcludedFromTestEvidence). Content quality enforcement relies entirely on agent self-discipline via skill guidance.

**Next:** Either add investigation content verification OR accept that investigation quality is a guidance-only concern (document this decision).

---

# Investigation: Investigation Behavior Check P2 Verify

**Question:** Does the verification system check that investigation artifacts have real test evidence in their "Test performed" section, not just "reviewed code" claims?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** og-feat-investigation-behavior-check-28dec
**Phase:** Complete
**Next Step:** None - decision needed on whether to implement verification or accept guidance-only approach
**Status:** Complete

---

## Findings

### Finding 1: Template Mismatch Between Skill and KB

**Evidence:** The investigation skill SKILL.md references:
- Line 92-97: `## Test performed` section with `**Test:**` and `**Result:**` fields
- Line 113: Template showing `## Test performed`
- Line 267: Self-review requires "Test performed" has real test

However, the actual kb template (via `kb create investigation`) uses:
- `## Structured Uncertainty` section with:
  - `**What's tested:**` bullet list
  - `**What's untested:**` bullet list
  - `**What would change this:**` falsifiability criteria

**Source:** 
- `~/.claude/skills/worker/investigation/SKILL.md:92-116`
- `kb show-template investigation` (verified via bash)
- Investigation file count: 123 use "Structured Uncertainty" vs 9 use "Test performed"

**Significance:** The skill guidance and the actual template don't match. Agents following skill guidance look for "Test performed" but the template uses "Structured Uncertainty". This creates confusion about which format to use.

---

### Finding 2: Investigation Skill is Explicitly Excluded from Test Evidence Verification

**Evidence:** In `pkg/verify/test_evidence.go`:
```go
// Lines 32-40
var skillsExcludedFromTestEvidence = map[string]bool{
    "investigation":   true, // Research skill, produces investigations
    "architect":       true, // Design skill, produces decisions
    "research":        true, // External research, no code changes
    // ...
}
```

The `IsSkillRequiringTestEvidence()` function (lines 48-67) returns `false` for investigation skill, meaning:
- No test evidence requirements apply to investigations
- `VerifyTestEvidence()` returns early with a warning

**Source:** `pkg/verify/test_evidence.go:32-67`

**Significance:** The verification system deliberately excludes investigation artifacts from test evidence requirements. This is by design - investigations produce artifacts (findings/knowledge) not code changes. However, this means investigation quality is purely guidance-enforced.

---

### Finding 3: Many Investigations Have Unfilled Template Placeholders

**Evidence:** Sampled recent investigations, found several with placeholder text:
```
=== 2025-12-28-inv-build-verification-p1-run-go.md ===
**What's tested:**
- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]

=== 2025-12-28-inv-posttooluse-hook-extension-logging.md ===
**What's tested:**
- ✅ [Claim with evidence of actual test performed...]
```

However, other investigations ARE properly filled:
```
=== 2025-12-28-inv-getcontextglobalwithprojects-already-filters-opts-project.md ===
**What's tested:**
- ✅ kb context CLI has no --project flag (verified: `kb context --help`)
- ✅ GetContextGlobalWithProjects filters by opts.Project (verified: code at context.go:205-207)
```

**Source:** Bash command sampling first 10 investigations by date

**Significance:** Quality varies widely. Some agents properly fill the Structured Uncertainty section with real test evidence; others leave placeholders. Without verification, quality depends entirely on agent discipline.

---

### Finding 4: Skill Self-Review Section Exists But is Not Enforced

**Evidence:** The investigation skill SKILL.md includes self-review checklist (lines 139-188):
```markdown
### Self-Review Checklist

- [ ] **Test is real** - Ran actual command/code, not just "reviewed"
- [ ] **Evidence concrete** - Specific outputs, not "it seems to work"
- [ ] **Conclusion factual** - Based on observed results, not inference
- [ ] **No speculation** - Removed "probably", "likely", "should" from conclusion
```

And specifically at line 173:
```
| **Real test performed** | Not "reviewed code" or "analyzed logic" | Go back and test |
```

However, there's no verification code that:
1. Parses investigation artifacts to check if these criteria are met
2. Validates that "Structured Uncertainty" or "Test performed" sections have actual content
3. Blocks completion if placeholders remain

**Source:** 
- `~/.claude/skills/worker/investigation/SKILL.md:139-188`
- `pkg/verify/` (no investigation content verification found)

**Significance:** The self-review is guidance-only. Agents who follow it produce quality investigations; agents who don't still pass verification.

---

## Synthesis

**Key Insights:**

1. **Intentional Exclusion** - Investigation skill is deliberately excluded from test_evidence verification because it produces artifacts (knowledge) not code. This is architecturally correct but leaves quality as a guidance concern.

2. **Template/Skill Mismatch** - The skill guidance references "Test performed" but kb templates use "Structured Uncertainty". This creates confusion but both serve the same purpose: documenting what was actually tested vs what's speculative.

3. **Guidance-Only Quality** - Investigation quality depends entirely on agents following self-review checklist. There's no programmatic enforcement. The prior investigation (verification-system-audit) identified this as "verification theater" for feature-impl but the same pattern exists for investigations.

**Answer to Investigation Question:**

**NO** - the verification system does NOT check that investigation artifacts have real test evidence. The investigation skill is explicitly excluded from test evidence verification (`skillsExcludedFromTestEvidence["investigation"] = true`). Quality enforcement relies entirely on:
1. Skill guidance (self-review checklist)
2. Agent discipline
3. Orchestrator review during `orch complete`

This matches the P2 priority from the parent investigation - it's a lower priority enhancement because investigations don't produce code changes that could break production.

---

## Structured Uncertainty

**What's tested:**

- ✅ Investigation skill excluded from test_evidence verification (verified: read pkg/verify/test_evidence.go:33)
- ✅ KB template uses "Structured Uncertainty" not "Test performed" (verified: `kb show-template investigation`)
- ✅ 123 investigations use Structured Uncertainty vs 9 use Test performed (verified: grep counts)
- ✅ Some investigations have unfilled placeholders (verified: sampled 10 recent investigations)
- ✅ No investigation content verification exists in pkg/verify (verified: grep for investigation content patterns)

**What's untested:**

- ⚠️ Impact of adding investigation content verification (not implemented)
- ⚠️ Percentage of investigations with unfilled placeholders (sampled 10, not exhaustive)
- ⚠️ Whether orchestrators actually catch placeholder text during review (observational only)

**What would change this:**

- Finding verification code for investigation content would invalidate Finding 2
- Finding that investigations with placeholders fail orch complete would invalidate Finding 4

---

## Implementation Recommendations

**Purpose:** The parent investigation (verification-system-audit) prioritized this as P2 because investigations don't produce code that could break production. The question is: implement verification or accept guidance-only?

### Recommended Approach ⭐

**Accept Guidance-Only for Investigations** - Document this as a conscious decision rather than implementing verification.

**Why this approach:**
- Investigations are knowledge artifacts, not code - they can't break production
- The self-review checklist in the skill is comprehensive (lines 139-188)
- Adding verification would increase complexity without clear benefit
- Orchestrator review during `orch complete` is the human quality gate

**Trade-offs accepted:**
- Some investigations will have placeholder text or weak evidence
- Quality varies by agent discipline
- Acceptable because: investigations are inputs to decisions, not the decisions themselves

**Implementation sequence:**
1. Document decision: "Investigation quality is guidance-enforced, not verification-gated"
2. Consider updating kb template to match skill language (use "Test performed" OR update skill to reference "Structured Uncertainty")
3. Close as P2-wontfix unless quality issues cause actual problems

### Alternative Approaches Considered

**Option B: Add Investigation Content Verification**
- **Pros:** Consistent quality enforcement; catches placeholders
- **Cons:** Complex regex/parsing for natural language; high false positive risk
- **When to use instead:** If investigations with poor quality are causing downstream problems

**Option C: Hybrid - Lint for Placeholders Only**
- **Pros:** Simple check (grep for "[Claim with evidence"); catches obvious template leftovers
- **Cons:** Doesn't validate quality, just catches template text
- **When to use instead:** If placeholder text is the main problem and worth implementing

**Rationale for recommendation:** The P2 priority from parent investigation was correct. Feature-impl with test evidence (P0) and build verification (P1) address the core problem (agents claiming completion without testing). Investigation content is a nice-to-have that depends on agent discipline, which the skill already guides.

---

### Implementation Details

**What to implement first:**
- Fix the template/skill mismatch (low effort, reduces confusion)
- Either update skill SKILL.md to reference "Structured Uncertainty" OR update kb template

**Things to watch out for:**
- ⚠️ Don't break existing investigations by changing template
- ⚠️ Skill guidance is in orch-knowledge, template is in kb-cli (cross-repo coordination)

**Areas needing further investigation:**
- None required for this decision

**Success criteria:**
- ✅ Clear decision documented (guidance-only vs verification)
- ✅ Template and skill use consistent language
- ✅ Parent investigation P2 item is addressed

---

## References

**Files Examined:**
- `pkg/verify/test_evidence.go` - Test evidence verification logic (investigation excluded)
- `pkg/verify/check.go` - Core verification including skill output checks
- `~/.claude/skills/worker/investigation/SKILL.md` - Investigation skill guidance
- `~/.kb/templates/investigation.md` (via kb show-template)
- `.kb/investigations/*.md` - Sample investigation files

**Commands Run:**
```bash
# Check template
kb show-template investigation

# Count template styles
grep -l "Structured Uncertainty" .kb/investigations/*.md | wc -l  # 123
grep -l "Test performed" .kb/investigations/*.md | wc -l  # 9

# Sample investigation quality
for f in $(ls -t .kb/investigations/*.md | head -10); do
  echo "=== $f ===" && grep -A 10 "What's tested:" "$f" | head -7
done
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-verification-system-audit-verification-theater.md` - Parent investigation that created this P2 item
- **Code:** `pkg/verify/test_evidence.go` - Where investigation exclusion is defined

---

## Self-Review

- [x] Real test performed (ran commands, read code, sampled artifacts)
- [x] Conclusion from evidence (investigation exclusion is explicit in code)
- [x] Question answered (NO verification exists; decided this is acceptable)
- [x] File complete
- [x] D.E.K.N. filled
- [x] NOT DONE claims verified (no verification code found after searching)

**Self-Review Status:** PASSED

---

## Discovered Work Check

| Type | Item | Created? |
|------|------|----------|
| **Enhancement** | Align investigation skill "Test performed" with kb template "Structured Uncertainty" | ⏳ Low priority - noted in recommendations |

Note: No beads issue created as this is a low-priority documentation alignment, not a bug or feature.

---

## Investigation History

**2025-12-28:** Investigation started
- Initial question: Does verification check investigation "Test performed" quality?
- Context: P2 item from verification-system-audit investigation

**2025-12-28:** Key discovery
- Found investigation skill is explicitly excluded from test_evidence verification
- Found template/skill mismatch (Structured Uncertainty vs Test performed)

**2025-12-28:** Investigation completed
- Status: Complete
- Key outcome: Verification does NOT check investigation content; this is acceptable by design; document as decision

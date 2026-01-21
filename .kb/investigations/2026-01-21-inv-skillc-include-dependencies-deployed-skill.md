<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** This is a duplicate - prior investigation (2026-01-18) already verified skillc DOES include dependencies in deployed SKILL.md frontmatter.

**Evidence:** (1) Prior investigation found no code bug; (2) Deployed skills currently have dependencies in frontmatter; (3) Root cause was deploy directory mismatch, not code.

**Knowledge:** Issue orch-go-4rboe was created Jan 14, resolved via investigation Jan 18, but never closed in beads - creating orphan issue that was re-spawned.

**Next:** Close as duplicate - reference prior investigation 2026-01-18-inv-skillc-include-dependencies-deployed-skill.md

**Promote to Decision:** recommend-no (duplicate issue, already resolved)

---

# Investigation: Skillc Include Dependencies Deployed Skill (Duplicate Verification)

**Question:** Is the bug "skillc doesn't include dependencies in deployed SKILL.md" still present?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage -->
**Superseded-By:** /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-18-inv-skillc-include-dependencies-deployed-skill.md

---

## Findings

### Finding 1: Prior Investigation Already Resolved This Issue

**Evidence:**
- Investigation `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-18-inv-skillc-include-dependencies-deployed-skill.md` exists
- Dated 2026-01-18, marked Status: Complete
- Conclusion: "skillc DOES include dependencies in deployed SKILL.md frontmatter"
- Root cause: User workflow issue (deploying from wrong directory), not code bug

**Source:** `.kb/investigations/2026-01-18-inv-skillc-include-dependencies-deployed-skill.md`

**Significance:** This issue was already investigated and resolved 3 days before this spawn. The beads issue was created Jan 14 but never closed after the Jan 18 investigation.

---

### Finding 2: Deployed Skills Currently Have Dependencies

**Evidence:**
```yaml
# ~/.claude/skills/worker/investigation/SKILL.md (lines 1-6)
---
name: investigation
skill-type: procedure
description: Record what you tried, what you observed, and whether you tested. Key discipline - you cannot conclude without testing.
dependencies:
  - worker-base
---
```
- Verified via `head -15 ~/.claude/skills/worker/investigation/SKILL.md`
- Dependencies field is present in deployed frontmatter
- Last compiled: 2026-01-19 10:40:12

**Source:** `~/.claude/skills/worker/investigation/SKILL.md`

**Significance:** The bug does not exist - deployed skills have dependencies in their frontmatter.

---

### Finding 3: Compiler Code Correctly Writes Dependencies

**Evidence:**
- Code at `skillc/pkg/compiler/compiler.go:214-219` writes dependencies to frontmatter:
```go
if len(inlineFrontmatter.Dependencies) > 0 {
    output.WriteString("dependencies:\n")
    for _, dep := range inlineFrontmatter.Dependencies {
        output.WriteString(fmt.Sprintf("  - %s\n", dep))
    }
}
```
- All current skills use `skill-type:` field (inline frontmatter approach), which includes dependencies
- No skills currently use deprecated `frontmatter:` field (external file approach)

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go:214-219`

**Significance:** Code is correct and functional. The bug report is invalid.

---

## Synthesis

**Key Insights:**

1. **Duplicate Issue** - This is a duplicate of an already-resolved investigation from Jan 18. The beads issue wasn't closed after investigation, causing it to be re-spawned.

2. **No Code Bug** - The skillc compiler correctly includes dependencies in deployed SKILL.md frontmatter when using the `skill-type:` approach (which all current skills use).

3. **Root Cause Was Workflow** - The original issue was caused by deploying from the wrong directory, which preserved a nested structure. This was a user error, not a code bug.

**Answer to Investigation Question:**

The bug does NOT exist. This is a duplicate issue that was already investigated and resolved on Jan 18, 2026. The prior investigation found that skillc correctly includes dependencies in deployed SKILL.md frontmatter. The original perceived bug was caused by deploying from the wrong directory (`~/orch-knowledge/skills` instead of `~/orch-knowledge/skills/src`), which caused files to deploy to a nested location while old files without dependencies remained at the expected location. When deployed correctly, dependencies are included.

---

## Structured Uncertainty

**What's tested:**

- ✅ Prior investigation exists and is marked Complete (verified: read file)
- ✅ Deployed investigation skill has dependencies in frontmatter (verified: head command output)
- ✅ Compiler code writes dependencies (verified: read source code)

**What's untested:**

- ⚠️ Why the beads issue wasn't closed after Jan 18 investigation (process gap)

**What would change this:**

- Finding would be wrong if deployed SKILL.md files were missing dependencies (tested - they're not)
- Finding would be wrong if prior investigation didn't exist (tested - it does)

---

## Implementation Recommendations

**Purpose:** Close duplicate issue, no code changes needed.

### Recommended Approach ⭐

**Close as Duplicate** - Close orch-go-4rboe with reference to prior investigation

**Why this approach:**
- Issue was already resolved Jan 18
- No code bug exists
- Deployed skills currently have dependencies

**Trade-offs accepted:**
- None - this is just closing a stale issue

**Implementation sequence:**
1. Close orch-go-4rboe with reason: "Duplicate - already resolved in investigation 2026-01-18-inv-skillc-include-dependencies-deployed-skill.md"
2. Done

### Alternative Approaches Considered

**None** - Clear duplicate, no alternatives needed.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-18-inv-skillc-include-dependencies-deployed-skill.md` - Prior investigation
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go` - Frontmatter generation
- `~/.claude/skills/worker/investigation/SKILL.md` - Deployed skill with dependencies

**Commands Run:**
```bash
# Check prior investigation
cat .kb/investigations/2026-01-18-inv-skillc-include-dependencies-deployed-skill.md

# Verify deployed skill has dependencies
head -15 ~/.claude/skills/worker/investigation/SKILL.md

# Check beads issue
bd show orch-go-4rboe
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-18-inv-skillc-include-dependencies-deployed-skill.md` - Original investigation that resolved this issue

---

## Investigation History

**2026-01-21 20:45:** Investigation started
- Initial question: Is the skillc dependencies bug still present?
- Context: Bug report filed Jan 14, but prior investigation exists from Jan 18

**2026-01-21 20:50:** Found prior investigation
- Discovered investigation from Jan 18 already resolved this issue
- Root cause was user workflow, not code bug

**2026-01-21 20:55:** Verified current state
- Confirmed deployed skills have dependencies in frontmatter
- Confirmed compiler code is correct

**2026-01-21 21:00:** Investigation completed
- Status: Complete
- Key outcome: Duplicate issue - skillc correctly includes dependencies, issue was already resolved Jan 18

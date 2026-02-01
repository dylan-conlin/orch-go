# Session Synthesis

**Agent:** og-arch-download-analyze-https-28jan-5b34
**Issue:** orch-go-20968
**Duration:** 2026-01-28 ~10:00 → ~10:50
**Outcome:** success

---

## TLDR

Analyzed she-llac.com Claude limits article via WebFetch; found it reveals credit-based usage economics (value multipliers, free cache reads for subscribers) that validates our Max subscription choice but is orthogonal to stealth mode decision (about access, not usage).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-28-inv-download-analyze-https-she-llac.md` - Complete investigation with 6 findings analyzing article and comparing to kb

### Files Modified
- None (investigation-only task)

### Commits
- Pending commit of investigation file

---

## Evidence (What Was Observed)

- **Article extracted via WebFetch** - Contains credit formulas, per-model rates, actual limit values vs marketing claims
- **Credit formula exposed**: `ceil(input_tokens × input_rate + output_tokens × output_rate)` with model-specific rates
- **Cache reads FREE for subscribers** (vs 10% API cost) - major advantage for agentic work
- **Max 5× overdelivers** (6× session limit vs marketed 5×) while **Max 20× underdelivers weekly** (16.67× vs 20×)
- **Value multipliers**: Pro 8.1×, Max 13.5× - validates our $200/mo vs $70-80/day API burn rate
- **Data source**: Reverse engineered via SSE float precision + Stern-Brocot tree algorithm
- **No overlap with stealth mode** - article addresses usage economics, not OAuth/fingerprinting/access

### Tests Run
```bash
# WebFetch extraction
WebFetch https://she-llac.com/claude-limits
# Result: Successfully extracted credit formulas and limit values

# KB comparison
Grep for credit, cache, weekly quota patterns in .kb/
# Result: Confirmed no prior credit formula knowledge, partial cache pricing knowledge
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-28-inv-download-analyze-https-she-llac.md` - Complete analysis of she-llac article and comparison to stealth mode decision

### Decisions Made
- No changes to stealth mode decision (validated, not changed)
- Recommend updating orchestration-cost-economics model with credit formula details

### Constraints Discovered
- Article data is reverse-engineered, not official - could become stale
- Credit formula and limits are internal implementation details

### Key New Insights
1. **Free cache reads for subscribers** - Massive advantage for tool-heavy agentic work (36× value in warm cache scenarios)
2. **Max 5× overdelivers** - May be sufficient for some workloads at half the cost
3. **13.5× value multiplier** - External validation of our Max subscription economics

### Externalized via `kn`
- None (insights captured in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation artifact)
- [x] Tests passing (N/A - research task)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-20968`

### Follow-up Consideration
**Potential issue:** Update orchestration-cost-economics model with credit formula and free cache read insight
**Skill:** investigation
**Context:**
```
The she-llac article revealed internal credit formulas and the critical insight that
cache reads are FREE for Max subscribers. This should be added to the
orchestration-cost-economics model for future reference.
```
*Note: Low priority - existing model is accurate, this would add implementation detail.*

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does our stealth mode implementation receive the same free cache read benefits as native Claude Code?
- Should we consider downgrading from Max 20× to Max 5× given the overdelivery pattern?

**Areas worth exploring further:**
- Testing cache behavior with stealth mode vs native Claude Code
- Cost tracking to determine if Max 5× would be sufficient

**What remains unclear:**
- Whether Anthropic will patch the float precision leak (article methodology)
- Current validity of credit formulas (may have changed since article)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-download-analyze-https-28jan-5b34/`
**Investigation:** `.kb/investigations/2026-01-28-inv-download-analyze-https-she-llac.md`
**Beads:** `bd show orch-go-20968`

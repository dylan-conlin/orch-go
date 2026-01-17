# Session Synthesis

**Agent:** og-work-design-stuck-agent-15jan-faa3
**Issue:** orch-go-uq6se
**Duration:** 2026-01-15 10:00 → 2026-01-15 11:00
**Outcome:** success

---

## TLDR

Designed stuck agent recovery mechanism using tiered approach: auto-resume (non-destructive) for idle agents, then surface in dashboard for human decision. Established "advisory recovery over automatic recovery" as the design principle, following patterns from stalled detection and ghost visibility decisions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md` - Complete design for stuck agent recovery with tiered approach

### Files Modified
- None (design investigation, no implementation)

### Commits
- To be committed as part of session completion

---

## Evidence (What Was Observed)

- **Existing resume mechanism**: `orch resume` command exists and works (`cmd/orch/resume.go:90-100`)
- **Stalled detection pattern**: Jan 8 investigation established "advisory only" approach - surface in dashboard, don't auto-act
- **Ghost visibility decision**: Jan 15 decision chose "filter over cleanup" - reversibility matters
- **Daemon architecture**: Poll-spawn-complete cycle with parallel loops supports adding recovery loop
- **Four failure modes identified**: Rate limit (resume works), server restart (resume may work), context exhaustion (resume fails), infinite loop (resume perpetuates)

### Tests Run
```bash
# No code implementation - design investigation only
# Verified existing command exists:
glob "**/resume*.go"
# Found: cmd/orch/resume.go, cmd/orch/resume_test.go
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md` - Complete design with findings, synthesis, and implementation recommendations

### Decisions Made
- **Tiered recovery approach**: Resume first (non-destructive), then surface for human decision (preserves reversibility)
- **Advisory-first principle**: Follows established patterns from stalled detection and ghost visibility
- **Daemon is the right location**: Recovery fits into existing poll-based architecture as third loop

### Constraints Discovered
- **Different failure modes need different recovery**: One-size-fits-all is wrong
- **Destructive actions need human decision**: Auto-respawn and auto-abandon violate reversibility principle
- **Rate limiting prevents loops**: Resume should be limited to 1/hour per agent

### Externalized via `kb`
- Investigation file captures all decisions and rationale
- Recommended for promotion to decision: "advisory recovery over automatic recovery" principle

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with full D.E.K.N.)
- [x] Tests passing (N/A - design investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-uq6se`

### Implementation Path (for future)
1. **Create issue**: `bd create "Implement tiered stuck agent recovery" --type feature`
2. **Skill**: feature-impl
3. **Implementation sequence**:
   - Stuck detection in daemon (identify idle >10min without Phase: Complete)
   - Auto-resume with rate limiting (1/hour per agent)
   - Needs Attention integration (surface if still stuck after 15min)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What's the actual success rate of resume for different failure modes? (needs production data)
- Should resume include diagnostic message asking agent to self-assess if stuck in loop?
- Should recovery respect business hours? (don't wake agents at 3am)

**Areas worth exploring further:**
- Integration with stalled agent detection (Jan 8 design) - could share same thresholds
- Whether to add resume button to dashboard agent detail panel

**What remains unclear:**
- Optimal threshold for stuck detection (10 min is educated guess)
- Whether 1 resume/hour rate limit is too aggressive or too conservative

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-design-stuck-agent-15jan-faa3/`
**Investigation:** `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md`
**Beads:** `bd show orch-go-uq6se`

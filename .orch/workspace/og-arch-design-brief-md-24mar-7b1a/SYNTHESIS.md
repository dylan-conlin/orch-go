# Session Synthesis

**Agent:** og-arch-design-brief-md-24mar-7b1a
**Issue:** orch-go-o7c0u
**Outcome:** success

---

## TLDR

Designed BRIEF.md as a 3-section comprehension artifact (Frame/Resolution/Tension) produced by full-tier agents alongside SYNTHESIS.md. Written for Dylan using the 4 writing primers, delivered to `.kb/briefs/` on completion, rendered in the dashboard review queue. The Tension section is the structural guard against false comprehension — briefs end with open questions requiring Dylan's judgment, not summaries that feel complete.

---

## Plain-Language Summary

Dylan currently can't process agent completions unless we're in a conversation — comprehension is gated on orchestrator narration. BRIEF.md fixes this by having agents write a half-page reading artifact that Dylan can read async (over coffee, between meetings). It's three sections: Frame (what was the question), Resolution (what was found), Tension (what still needs Dylan's judgment). The critical design choice is that Tension section — without it, briefs become summaries Dylan passively consumes without the reactive thinking that produces strategic reframes. SYNTHESIS.md continues to exist for the orchestrator pipeline; BRIEF.md is additive, not a replacement.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- Investigation complete with 5 findings, substrate consultation, and implementation recommendations
- BRIEF.md template created at `.orch/templates/BRIEF.md`
- 3 implementation issues + 1 integration issue created
- 2 blocking questions surfaced for orchestrator/Dylan decision

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-24-inv-design-brief-md-comprehension-artifact.md` — Full architect investigation
- `.orch/templates/BRIEF.md` — BRIEF.md template with style guidance

### Files Modified
- None

---

## Architectural Choices

### Three sections (Frame/Resolution/Tension) vs more detailed template
- **What I chose:** Minimal 3-section template with stance primers
- **What I rejected:** Detailed template with 6+ sections mirroring SYNTHESIS.md structure
- **Why:** Behavioral grammars model says constraint dilution starts at 5+ items. Writing-style model says 4 primers > 20 rules. The goal is a half-page reading artifact, not a comprehensive report.
- **Risk accepted:** Agents may produce superficial briefs with too little structure. Mitigation: template comments provide detailed guidance; Tension section is required.

### orch complete remains sole comprehension gate
- **What I chose:** Mark-as-read in dashboard is reading confirmation, not comprehension confirmation
- **What I rejected:** Mark-as-read removing comprehension:pending
- **Why:** Thread identifies false comprehension risk. Reading != comprehending. Gate 13 (explain-back) exists precisely because passive consumption doesn't produce understanding.
- **Risk accepted:** This is a recommendation, not a decision — surfaced as blocking question `orch-go-c29fl` for strategic resolution.

---

## Knowledge (What Was Learned)

### Decisions Made
- BRIEF.md is additive to SYNTHESIS.md, not a replacement (different audiences)
- Full-tier only (light-tier work is trivial, no comprehension needed)
- Agent generates brief (only actor with full context, zero infrastructure cost)
- Template ends with Tension (structural false-comprehension guard)

### Constraints Discovered
- worker-base is governance-protected — skill protocol changes must be orchestrator direct session
- Adding BRIEF.md to completion protocol must stay under ~15 lines to avoid constraint dilution

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation, template, issues)
- [x] Design questions answered with substrate traces
- [x] Implementation decomposed into 3 component issues + 1 integration issue
- [x] Blocking questions surfaced as beads issues
- [x] Ready for `orch complete orch-go-o7c0u`

---

## Unexplored Questions

- Whether agents can actually write good briefs with stance primers alone (needs first real application)
- Brief quality measurement heuristic (analogous to `orch debrief --quality`)
- Cleanup lifecycle for `.kb/briefs/` to prevent stale artifact accumulation
- Whether the Tension section becomes formulaic over time

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** architect
**Workspace:** `.orch/workspace/og-arch-design-brief-md-24mar-7b1a/`
**Investigation:** `.kb/investigations/2026-03-24-inv-design-brief-md-comprehension-artifact.md`
**Beads:** `bd show orch-go-o7c0u`

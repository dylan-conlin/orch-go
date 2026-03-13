# Session Synthesis

**Agent:** og-research-claude-code-now-13mar-4d17
**Issue:** orch-go-8mdvr
**Duration:** 2026-03-13
**Outcome:** success

---

## Plain-Language Summary

Claude Code showing "Opus 4.6 (1M context)" is normal and does NOT cost extra money beyond the Max subscription. Opus 4.6 and Sonnet 4.6 ship with 1M tokens as their native context window at standard pricing — this is different from older models where 1M was a premium add-on. Anthropic's pricing page explicitly confirms: "A 900k-token request is billed at the same per-token rate as a 9k-token request." Claude Code v2.1.75 (today, Mar 13) enabled 1M by default for Max subscribers. You can disable it with `CLAUDE_CODE_DISABLE_1M_CONTEXT=1` if desired, but there's no financial reason to do so.

---

## TLDR

Investigated whether "1M context" shown in Claude Code status bar costs extra on Max subscription. It does not — 1M is the native context window for Opus 4.6/Sonnet 4.6 at standard pricing, fully covered by Max flat rate. Updated cost economics model with 1M context section.

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestration-cost-economics/probes/2026-03-13-probe-1m-context-max-plan-pricing.md` - Probe documenting 1M context pricing research
- `.orch/workspace/og-research-claude-code-now-13mar-4d17/SYNTHESIS.md` - This file
- `.orch/workspace/og-research-claude-code-now-13mar-4d17/VERIFICATION_SPEC.yaml` - Verification spec

### Files Modified
- `.kb/models/orchestration-cost-economics/model.md` - Added "1M Context Window" section with pricing details, updated date and probe references

### Commits
- (pending) Research findings commit

---

## Evidence (What Was Observed)

### Source 1: Anthropic Models Page (platform.claude.com)
- Opus 4.6 context window: **1M tokens** (native, not beta)
- Opus 4.5/4.1/4 context window: 200K tokens
- Sonnet 4.6: 1M tokens (native)
- Sonnet 4.5/4: 200K default, 1M via beta header with premium pricing

### Source 2: Anthropic Pricing Page — Long Context Section
Verbatim quote:
> "Claude Opus 4.6 and Sonnet 4.6 include the full 1M token context window at standard pricing. (A 900k-token request is billed at the same per-token rate as a 9k-token request.)"

Premium pricing (2x input, 1.5x output) applies ONLY to legacy Sonnet 4.5/4 with beta header.

### Source 3: Claude Code Changelog
- v2.1.45 (Feb 17): Initial Opus 4.6 + 1M support
- v2.1.50 (Feb 20): `CLAUDE_CODE_DISABLE_1M_CONTEXT` env var added
- v2.1.75 (Mar 13): 1M enabled by default for Max/Team/Enterprise

### Source 4: Local Version Check
```bash
claude --version
# 2.1.75 (Claude Code)
```

---

## Architectural Choices

No architectural choices — research task within existing patterns.

---

## Knowledge (What Was Learned)

### Key Findings

1. **1M is NOT a premium feature on 4.6 models.** It's the standard context window. The premium long-context pricing that existed for Sonnet 4.5/4 does NOT apply.

2. **Max subscription fully covers 1M.** Same credit formula, same flat rate. No extra billing.

3. **Economic trade-off exists but is favorable.** Larger context = more credits per request, but fewer total requests needed (less compaction, fewer respawns). Net effect likely neutral or positive.

4. **Control mechanism available.** `CLAUDE_CODE_DISABLE_1M_CONTEXT=1` can disable if desired. No reason to use it from a cost perspective, but useful if you want to limit credit consumption per request.

### Constraints Discovered
- None new — existing credit system and Max flat-rate model confirmed to cover 1M context

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes:
- Anthropic pricing page confirms standard pricing for 1M on 4.6 models
- No premium/surcharge documentation found for Max + 1M
- Model updated with new 1M context section

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (probe, model update, synthesis)
- [x] Research sources verified (4 authoritative Anthropic sources)
- [x] Probe merged into parent model
- [x] Ready for `orch complete orch-go-8mdvr`

---

## Unexplored Questions

- **Credit consumption rate change:** How much faster do sessions consume weekly quota with 1M context vs 200K? Would need empirical measurement over several days.
- **Compaction interaction:** Does Claude Code's compaction trigger differently with 1M context? The compaction threshold may have changed.

---

## Friction

Friction: none — web sources were comprehensive and authoritative.

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-research-claude-code-now-13mar-4d17/`
**Beads:** `bd show orch-go-8mdvr`

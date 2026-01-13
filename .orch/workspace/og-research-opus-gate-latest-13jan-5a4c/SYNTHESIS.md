# Session Synthesis

**Agent:** og-research-opus-gate-latest-13jan-5a4c
**Issue:** Ad-hoc spawn (--no-track)
**Duration:** 2026-01-13 ~15:30 → 2026-01-13 ~16:00
**Outcome:** success

---

## TLDR

Researched current status of Opus 4.5 auth gate; confirmed it is still active and was recently reinforced (Jan 9, 2026) with community workaround (opencode-anthropic-auth) no longer functional. Two official paths remain: direct API or Claude Code CLI with Max subscription.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-13-inv-opus-gate-latest-status-still.md` - Research investigation documenting current status of Opus gate

### Files Modified
None - This was a research-only investigation with no code changes.

### Commits
Not yet committed - investigation file ready for commit.

---

## Evidence (What Was Observed)

### Official API Access
- Opus 4.5 announced Nov 24, 2025
- Available via API: `claude-opus-4-5-20251101` at $5 input/$25 output per MTok
- No OAuth restrictions mentioned for direct API key access
- Source: https://www.anthropic.com/news/claude-opus-4-5, https://platform.claude.com/docs/en/api/overview

### Community Workaround Failure
- GitHub repository `anomalyco/opencode-anthropic-auth` (151 stars)
- Issue #12 opened Jan 9, 2026: "The auth no longer works"
- Error: "This credential is only authorized for use with Claude Code and cannot be used for other API requests"
- 30+ reactions indicating widespread community impact
- Source: https://github.com/anomalyco/opencode-anthropic-auth/issues/12

### Official Access Methods
- Claude Code requires Pro ($17/month), Max ($100/month), Teams, or Enterprise subscription
- Can also use Claude Console (API) account with pre-paid credits
- Pricing page shows no changes to OAuth restrictions
- Source: https://code.claude.com/docs/en/quickstart, https://claude.com/pricing

### No Announcements About Lifting
- Checked Anthropic news page (through Jan 13, 2026) - no mention of OAuth changes
- API documentation unchanged - standard API key auth only
- No signals of future policy changes

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-13-inv-opus-gate-latest-status-still.md` - Comprehensive research on Opus gate status with evidence from official docs and community discussions

### Decisions Made
- Decision: No new decisions required - this research reinforces existing dual spawn mode architecture decision (2026-01-09-dual-spawn-mode-architecture.md)

### Constraints Discovered
- **Constraint: Anthropic actively enforces OAuth restrictions** - Jan 9, 2026 update shows this is not a one-time block but ongoing enforcement
- **Constraint: Community workarounds are not sustainable** - Pattern from prior knowledge confirmed: "cat and mouse game not worth chasing"
- **Constraint: Two official paths only** - Direct API ($5/$25/MTok) or Claude Code CLI with subscription; no hybrid/workaround path exists

### Key Insights
1. **Auth gate recently reinforced** - The Jan 9, 2026 enforcement update shows Anthropic is actively maintaining this restriction
2. **Error message is explicit** - "This credential is only authorized for use with Claude Code" leaves no ambiguity about intent
3. **No timeline for changes** - Official docs and announcements contain no indication restrictions will be lifted

### Externalized via `kb`
This investigation updates existing constraint knowledge rather than creating new constraints. The findings should be referenced in:
- Existing constraint: "Never spawn agents with Opus model until auth gate bypassed" → Update to "Auth gate still active as of Jan 13, 2026"
- Existing constraint: "Opus 4.5 blocked via OAuth for opencode" → Update to add Jan 9, 2026 enforcement detail

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with full research)
- [x] Tests passing (N/A - research only, no code changes)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete {issue-id}` (or `/exit` for ad-hoc spawn)

### Follow-up Actions Recommended
1. **Update constraint documentation** - Add note to relevant `.kb/` constraints about Jan 9, 2026 status
2. **Remove stale references** - Audit orch-go config for any opencode-anthropic-auth plugin references
3. **Validate dual spawn mode** - Confirm opencode+sonnet default and claude+opus escape hatch working correctly

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why did Anthropic choose Jan 9, 2026 specifically to tighten enforcement? (No public announcement, just silently updated)
- What is the technical mechanism of the new enforcement? (Server-side credential validation? Request fingerprinting?)
- Are there any legitimate use cases for OAuth API access that Anthropic might support in the future? (Enterprise integrations?)

**Areas worth exploring further:**
- Claude Max subscription value proposition given restrictions (is $100-200/month worth it now that workaround is gone?)
- Pay-per-token API cost analysis at scale (when does subscription become more economical than API?)

**What remains unclear:**
- Whether Anthropic has internal plans to offer OAuth API access in the future (no public signals, but possible)
- Whether the community will continue attempting workarounds (likely yes, but prior knowledge says "not worth chasing")

---

## Session Metadata

**Skill:** research
**Model:** claude-sonnet-4.5 (via opencode)
**Workspace:** `.orch/workspace/og-research-opus-gate-latest-13jan-5a4c/`
**Investigation:** `.kb/investigations/2026-01-13-inv-opus-gate-latest-status-still.md`
**Beads:** Ad-hoc spawn (--no-track)

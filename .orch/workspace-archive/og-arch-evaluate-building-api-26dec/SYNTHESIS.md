# Session Synthesis

**Agent:** og-arch-evaluate-building-api-26dec
**Issue:** orch-go-untracked-1766770779
**Duration:** 2025-12-26
**Outcome:** success

---

## TLDR

Evaluated whether to build an API proxy layer to share Claude Max subscriptions with colleagues. **Conclusion: Do not build** - account sharing explicitly violates Anthropic's Consumer Terms of Service. Recommended legitimate alternatives: API credits for occasional users, or individual Max subscriptions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-evaluate-building-api-proxy-layer.md` - Full investigation with ToS analysis and recommendations

### Files Modified
- None

### Commits
- (pending) - architect: evaluate API proxy layer for account sharing - not recommended due to ToS

---

## Evidence (What Was Observed)

- **Consumer Terms Section 2:** "You may not share your Account login information, Anthropic API key, or Account credentials with anyone else. You also may not make your Account available to anyone else." (Source: https://www.anthropic.com/legal/consumer-terms)
- **Claude for Teams:** $25-150/person/month with 5-seat minimum (Source: https://www.anthropic.com/pricing)
- **Claude Max:** $100+/person/month, explicitly "per person" individual plans (Source: https://www.anthropic.com/pricing)
- **API Pricing:** Pay-per-token with no account sharing restrictions (Opus: $5 input / $25 output per MTok)

### Tests Run
```bash
# Fetched and analyzed Anthropic Terms of Service
WebFetch: https://www.anthropic.com/legal/consumer-terms
# Result: Clear prohibition on account sharing

# Fetched pricing information
WebFetch: https://www.anthropic.com/pricing
# Result: Team plans have 5-seat minimum, API is pay-per-token
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-evaluate-building-api-proxy-layer.md` - "Do not build" recommendation with legitimate alternatives

### Decisions Made
- **Do not build API proxy:** Violates ToS, risk of account termination
- **Recommended alternative for occasional users:** API credits (ToS-compliant, flexible)
- **Recommended alternative for heavy users:** Individual Max subscriptions ($100/mo each)
- **Recommended alternative for business use:** Claude for Teams (5-seat minimum)

### Constraints Discovered
- **Hard constraint:** Consumer Terms explicitly prohibit account sharing ("make your Account available to anyone else")
- **Soft constraint:** Claude for Teams has 5-seat minimum, doesn't fit 3-person use case well

### Externalized via `kn`
- N/A - Key constraint documented in investigation file. Not a project-specific decision worth a kn entry.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete {issue-id}`

### Follow-up (for Dylan to decide)
**Question:** Does Dylan want to facilitate colleagues' Claude access, and if so, which approach?

**Options:**
1. **API Credits** - Purchase $100-200 of API credits, share for occasional use. Cheapest for low usage.
2. **Individual Max Subscriptions** - Each person pays $100/mo for their own account. Best for heavy users.
3. **Do Nothing** - Dylan's accounts remain for personal orchestration use only.

**Recommendation:** If Lea and Kenneth are occasional users, API credits. If heavy users, individual subscriptions.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is Lea and Kenneth's expected usage pattern? (affects API vs subscription economics)
- Would Anthropic consider custom arrangements for teams below 5-seat minimum?
- Does SendCutSend have specific compliance or enterprise needs that would benefit from Team plan features?

**Areas worth exploring further:**
- Usage-based cost modeling: At what usage level do API credits become more expensive than subscriptions?
- Enterprise alternatives: Are there reseller or partner programs that might fit better?

**What remains unclear:**
- Dylan's actual goals: Is this about value sharing, cost optimization, or enabling specific colleagues?

---

## Session Metadata

**Skill:** architect
**Model:** Opus
**Workspace:** `.orch/workspace/og-arch-evaluate-building-api-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-evaluate-building-api-proxy-layer.md`
**Beads:** (untracked issue - orch-go-untracked-1766770779)

# Session Synthesis

**Agent:** og-inv-benchmark-worker-reliability-26mar-ff7d
**Issue:** orch-go-1dhv8
**Duration:** 2026-03-26
**Outcome:** success

---

## Plain-Language Summary

We tried to benchmark orch-go's worker reliability across Claude Code, GPT-5.4, and a cheaper fallback — and discovered there's nothing to benchmark against. Every single post-protocol agent in the system (130 out of 130) runs on Opus via Claude Code. GPT-5.4's infrastructure is ready (model aliases, routing, API key) but has never completed a single session because the OpenCode server needed to serve it hasn't been running when tests were attempted. The Claude Code baseline is solid: 93-100% Phase:Complete on labeled skills in the last 7 days. The deliverable is a recommendation matrix with go/no-go thresholds and a concrete 15-minute benchmark protocol that produces the missing data.

---

## TLDR

Asked "which worker backends are operationally viable" and found the system has only ever run one (Opus/Claude Code, which works great at 97% completion). GPT-5.4 and Sonnet are infrastructure-ready but empirically untested. Provided a 15-minute benchmark protocol and routing decision matrix.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Full investigation with recommendation matrix, go/no-go thresholds, and benchmark protocol

### Files Modified
- None (investigation-only session)

---

## Evidence (What Was Observed)

- 130/130 post-protocol AGENT_MANIFEST.json files show `model: anthropic/claude-opus-4-5-20251101` — zero model diversity
- Last 7 days Phase:Complete rates: investigation 97% (N=37), architect 93% (N=15), feature-impl 100% (N=4), debugging 100% (N=2)
- SYNTHESIS.md compliance: debugging 100%, investigation 72%, feature-impl 4% (known protocol weight issue, not a stall)
- GPT-5.4 dry-run routes correctly: `Backend: opencode, Model: openai/gpt-5.4`
- OPENAI_API_KEY is set (length 164)
- OpenCode server is down (curl localhost:4096 fails)
- Prior GPT-5.4 test (orch-go-rj8hi, 2026-03-24) blocked by AI_LoadAPIKeyError — Codex OAuth not configured
- Zero Sonnet agents in post-protocol archive
- Prior GPT-5.2-codex stall rate: 67.5% (Feb 2026 audit, N=123) — stale, can't extrapolate to GPT-5.4

### Tests Run
```bash
# Infrastructure verification
orch spawn --dry-run --model gpt-5.4 feature-impl "test" → routes to opencode backend ✅
orch spawn --dry-run --model sonnet feature-impl "test" → routes to claude backend ✅
curl localhost:4096/health → NOT RUNNING ❌

# Data mining (Python analysis of beads + archive)
# 1,978 beads issues analyzed
# 136 post-protocol manifests analyzed
# 1,206 March 2026 issues analyzed
```

---

## Architectural Choices

No architectural choices — investigation task within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Recommendation matrix and benchmark protocol

### Constraints Discovered
- OpenCode server must be running for any non-Anthropic model testing
- Codex OAuth (flat-rate GPT-5.4) requires interactive Dylan login to ChatGPT Pro
- OPENAI_API_KEY path works but at pay-per-token rates ($2.50/$15 per MTok)

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for:
- Data mining verification (beads, archive counts)
- Infrastructure verification (dry-run results)
- Recommendation matrix structure

Key outcome: Claude Code / Opus baseline established, GPT-5.4 and Sonnet require empirical validation via the 15-minute benchmark protocol.

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, action items are in recommendation matrix)

### If Close
- [x] All deliverables complete (investigation + recommendation matrix + benchmark protocol)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-1dhv8`

### Follow-up Actions (for Dylan)
1. Start OpenCode: `orch-dashboard start`
2. Spawn 5 GPT-5.4 tasks per the benchmark protocol in the investigation
3. Spawn 3 Sonnet tasks to validate fallback path
4. Apply go/no-go thresholds from recommendation matrix

---

## Unexplored Questions

- **Does GPT-5.4's 1.05M context window actually help with SPAWN_CONTEXT compliance?** Prior GPT-5.2 stalls were partly attributed to 128K context exhaustion. GPT-5.4's larger window could eliminate this failure mode entirely.
- **Would Sonnet-on-Claude-backend be cheaper in practice?** Same Claude Max subscription covers both, so the "cost savings" from Sonnet are really about rate limit headroom, not dollars.
- **Is the SYNTHESIS.md compliance gap on feature-impl worth fixing before benchmarking alternatives?** 4% compliance makes cross-model comparison noisy for that skill.

---

## Friction

- `gap`: OpenCode server not running prevented any empirical GPT-5.4 testing — the entire benchmark dimension was blocked
- `ceremony`: Investigation file template is 249 lines; filling it for a data-mining investigation feels disproportionate to the actual content

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-benchmark-worker-reliability-26mar-ff7d/`
**Investigation:** `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md`
**Beads:** `bd show orch-go-1dhv8`

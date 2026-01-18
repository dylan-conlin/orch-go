# Session Synthesis

**Agent:** og-research-compare-deepseek-models-18jan-9eee
**Issue:** orch-go-hr51c
**Duration:** 2026-01-18 ~12:00 → 2026-01-18 ~12:30
**Outcome:** success

---

## TLDR

Researched DeepSeek (V3, R1) vs Claude (Sonnet 4.5, Opus 4.5) for agent orchestration. DeepSeek offers 10-30x lower pricing but lacks stable function calling, making Claude essential for agentic workflows; DeepSeek suitable for cost-sensitive non-agentic bulk work.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-research-compare-deepseek-models-anthropic-models.md` - Comprehensive research comparing DeepSeek vs Anthropic models

### Files Modified
- None

### Commits
- (pending) Research investigation file

---

## Evidence (What Was Observed)

- DeepSeek V3.2 pricing: $0.25/$0.38 per MTok vs Claude Opus 4.5 at $5/$25 (10-65x cheaper)
- DeepSeek R1-0528 pricing: $0.45/$2.15 per MTok (~12x cheaper than Opus)
- Claude Opus 4.5: 80.9% SWE-bench Verified vs DeepSeek R1: 49.2% (massive gap)
- Claude: 62.3% scaled tool use vs DeepSeek R1: 43.8%
- DeepSeek API docs explicitly state: "Function Calling capability is unstable"
- DeepSeek has no hard rate limits (dynamic throttling instead)
- DeepSeek available via OpenRouter and Amazon Bedrock (data stays in AWS)
- DeepSeek V4 expected mid-February 2026 with improved coding

### Tests Run
```bash
# Research-only session - no code tests applicable
# Verified pricing and benchmarks via official documentation and third-party analysis sites
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-research-compare-deepseek-models-anthropic-models.md` - Full comparison with 7 findings

### Decisions Made
- Claude Opus 4.5 remains correct default for orch-go because function calling stability is critical for agent orchestration
- DeepSeek viable for cost-sensitive non-agentic work (large context analysis, document processing)
- R1 could serve as planning layer in hybrid architecture but not as executor

### Constraints Discovered
- DeepSeek function calling is "unstable" per their own docs - disqualifies it for primary orchestration use
- Context window: DeepSeek 128K vs Claude 200K+ (Claude has edge for large context + agentic)
- Claude's "Tool Search Tool" feature enables scaling to many tools without context bloat - DeepSeek lacks equivalent

### Externalized via `kn`
- Not applicable (findings captured in investigation file, no persistent kb changes needed)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (research investigation file)
- [x] Tests passing (N/A - research task)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-hr51c`

**Optional follow-up work (not blocking):**
- Add `deepseek` and `deepseek-r1` aliases to pkg/model/model.go
- Update model selection guide with DeepSeek guidance
- Re-evaluate after DeepSeek V4 release (Feb 2026)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How would a hybrid DeepSeek R1 (planning) + Claude (execution) architecture perform?
- What's the latency difference between OpenRouter/Bedrock vs direct DeepSeek API?
- Could DeepSeek's MIT-licensed models be self-hosted for sensitive workloads?

**Areas worth exploring further:**
- DeepSeek V4 capabilities when released (~Feb 17, 2026)
- Testing DeepSeek function calling stability in practice
- Multi-agent architectures that leverage R1 for reasoning

**What remains unclear:**
- Whether DeepSeek V4 will have stable function calling
- Performance characteristics of DeepSeek via third-party providers vs direct API

---

## Session Metadata

**Skill:** research
**Model:** opus
**Workspace:** `.orch/workspace/og-research-compare-deepseek-models-18jan-9eee/`
**Investigation:** `.kb/investigations/2026-01-18-research-compare-deepseek-models-anthropic-models.md`
**Beads:** `bd show orch-go-hr51c`

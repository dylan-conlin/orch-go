# Benchmark: Logout Fix - 6 Model Comparison

**Status:** Complete
**Created:** 2026-01-28
**Updated:** 2026-01-29
**Type:** Benchmark / Model Performance Comparison
**Source:** specs-platform (SCS)

## Summary

Benchmarked 6 AI models on the same debugging task: fixing admin logout not working.

**Results: 2 succeeded, 5 failed (including Opus retest on Jan 29).**

| Rank | Model | Time | Result | Approach |
|------|-------|------|--------|----------|
| 1 | Codex (gpt-5.2-codex medium) | ~3m 41s | PASS | Backend cookie `path="/"` |
| 2 | DeepSeek Chat | ~13m 28s | PASS | Backend cookie + `prompt=login` |
| 3 | Gemini Pro 2.5 | ~1m 12s | FAIL | Frontend-only (LoginPage) |
| 4 | Opus 4.5 (Jan 29 retest) | ~3m | FAIL | Frontend redirect workaround |
| 5 | GPT 5.2 | ~4m 18s | FAIL | Frontend-only (AdminLogin) |
| 6 | Sonnet 4.5 | ~4m 41s | FAIL | Frontend-only (Login.tsx) |
| 7 | Opus 4.5 (Jan 28) | ~7 min | FAIL | URL routing confusion |

## The Problem

Admin logout button doesn't actually log user out - they can immediately click "Admin" and be auto-logged back in via OAuth SSO.

**Root cause:** Cookie `path` not set when creating/clearing JWT cookie. Without explicit `path="/"`, cookie operations don't match.

**Correct fix:**
```python
# api/app/auth.py
response.set_cookie(
    ...
    path="/",  # <-- This was missing
)
response.delete_cookie(
    key="access_token",
    path="/",  # <-- This was missing
)
```

## Key Findings

### Finding 1: Most Models Made Frontend-Only Fixes

4 out of 6 models created a login page UI component instead of fixing the backend cookie issue:
- Gemini Pro -> `LoginPage.tsx`
- GPT 5.2 -> `AdminLogin.tsx`
- Sonnet -> `Login.tsx`
- Opus -> Changed URL routing (different failure mode)

This suggests the investigation file's mention of frontend files may have biased models toward frontend solutions.

### Finding 2: Codex vs GPT 5.2 Difference

Same model family, different results:
- **Codex (gpt-5.2-codex medium):** PASS - Found backend fix
- **GPT 5.2:** FAIL - Frontend-only fix

The "codex medium" variant may be specialized for debugging/code analysis tasks, or this could be run-to-run variance.

### Finding 3: Speed Doesn't Correlate with Correctness

| Model | Speed Rank | Correctness |
|-------|------------|-------------|
| Gemini Pro | 1st (1m 12s) | FAIL |
| Codex | 2nd (3m 41s) | PASS |
| GPT 5.2 | 3rd (4m 18s) | FAIL |
| Sonnet | 4th (4m 41s) | FAIL |
| Opus | 5th (7m) | FAIL |
| DeepSeek | 6th (13m 28s) | PASS |

Fastest model (Gemini Pro) was wrong. Slowest model (DeepSeek) was correct.

### Finding 4: Token Usage Varies Wildly

- DeepSeek: 83.6K tokens (found fix)
- Opus: 9.4K tokens (wrong)

DeepSeek used 9x more tokens but found the correct fix. Suggests more thorough exploration.

## Hypothesis Evaluation

### Original Hypothesis: Anthropic Model Degradation Day

**Status:** Partially supported but incomplete explanation

**Evidence for:**
- Both Anthropic models (Opus, Sonnet) failed
- Both non-Anthropic models that succeeded are different providers

**Evidence against:**
- GPT 5.2 (OpenAI) also failed
- Gemini Pro (Google) also failed
- Only 2/6 models succeeded regardless of provider

**Revised hypothesis:** This task may be genuinely difficult for most models, with Codex and DeepSeek having specific strengths in backend debugging.

## Preserved States

| Model | Branch |
|-------|--------|
| Codex | `codex-benchmark.txt` (transcript) |
| Opus | `benchmark/logout-fix-opus-attempt` |
| Sonnet | `benchmark/logout-fix-sonnet-attempt` |
| GPT 5.2 | `benchmark/logout-fix-gpt52-attempt` |
| DeepSeek | (committed then reset) |
| Gemini Pro | (not preserved) |

## Model Details

### Codex (gpt-5.2-codex medium) PASS

**Time:** Investigation 3m 25s + Implementation 16s = ~3m 41s total

**Key insight:** "Cookie deletion may fail if the original cookie uses a specific path like '/api' while deletion defaults to '/'"

**Fix:** Added `path="/"` to both set_cookie and delete_cookie in `api/app/auth.py`

---

### DeepSeek Chat PASS

**Time:** ~13m 28s | **Tokens:** 83.6K

**Approach:** Thorough exploration, made 2 commits

**Fix:** Added `path="/"` to delete_cookie + `prompt=login` to OAuth URL (belt-and-suspenders)

---

### Gemini Pro 2.5 FAIL

**Time:** ~1m 12s (fastest)

**Failure mode:** Created `LoginPage.tsx`, frontend-only changes

---

### GPT 5.2 FAIL

**Time:** ~4m 18s

**Failure mode:** Created `AdminLogin.tsx`, frontend-only changes

---

### Claude Sonnet 4.5 FAIL

**Time:** ~4m 41s

**Failure mode:** Created `Login.tsx`, frontend-only changes

---

### Claude Opus 4.5 (Jan 28) FAIL

**Time:** ~7 min | **Tokens:** 9.4K

**Failure mode:** Changed URL routing (`root_path`, endpoint paths) - confused URL paths with cookie paths

---

### Claude Opus 4.5 (Jan 29 Retest) FAIL

**Time:** ~3m

**Prompt:** Fresh start with bug description only (no investigation file to avoid biasing toward frontend)

**Failure mode:** Changed logout redirect from `/api/auth/login` to `/` in AuthContext.tsx. This is a workaround (don't trigger OAuth after logout) rather than fixing the root cause (cookie not being deleted).

**Note:** Faster than Jan 28 attempt but still wrong. Different failure mode suggests variance, but consistent pattern of missing backend root cause.

## Opus Consistency Analysis

Opus failed both attempts but with different wrong approaches:
- **Jan 28:** URL routing confusion (changed endpoint paths)
- **Jan 29:** Frontend redirect workaround (changed where logout redirects)

Neither attempt investigated why `delete_cookie()` wasn't working. Both Anthropic models (Opus, Sonnet) show a pattern of frontend-focused fixes when the root cause is backend.

## Actionable Recommendations

1. **For debugging tasks:** Consider Codex or DeepSeek over general-purpose models
2. **For benchmarking:** Run multiple times to account for variance
3. **For prompts:** Be explicit about checking backend when frontend symptoms appear
4. **For investigation files:** Avoid listing frontend files first if backend is likely root cause

## Related: Industry Degradation Tracking

On Jan 29, 2026, a daily Claude Code benchmark tracker hit #1 on Hacker News:
- **Link:** https://marginlab.ai/trackers/claude-code/
- **HN Discussion:** https://news.ycombinator.com/item?id=46810282

**Key points from discussion:**
- MarginLab runs daily SWE-bench subset (50 tasks) showing ~4% drop over past month
- SWE-bench co-author notes high variance from small sample size and single daily run
- Speculation about load-based degradation, quantization, or A/B testing
- antirez notes Claude Code harness changes (system prompt, tools) may explain variance independent of model changes

**Relevance to this benchmark:**
- Our benchmark is more controlled: same task, same codebase, multiple models compared
- Opus failing same task two days in a row (different wrong approaches) suggests model-level pattern, not just variance
- We're running via OpenCode/orch, not Claude Code CLI, so different harness than MarginLab benchmark

## Orchestration Implications

| Insight | Implication for orch |
|---------|---------------------|
| DeepSeek thorough but slow (83K tokens, 13m) | Good for cost-sensitive debugging where correctness > speed |
| Codex fast AND correct | Consider for debugging tasks if available |
| Frontend bias in prompts | `orch spawn` prompts should emphasize backend-first investigation |
| Speed != quality | Don't optimize for fast completion - optimize for correct completion |
| Anthropic models missed backend | May need explicit "check backend first" in debugging skill prompts |

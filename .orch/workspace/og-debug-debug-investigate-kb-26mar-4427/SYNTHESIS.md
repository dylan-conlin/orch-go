# Session Synthesis

**Agent:** og-debug-debug-investigate-kb-26mar-4427
**Issue:** orch-go-304ta
**Duration:** 2026-03-26 → 2026-03-26
**Outcome:** success

---

## Plain-Language Summary

The 0/100 context score on GPT-5.4 spawns is not coming from genuinely empty knowledge. The spawn pipeline asks `kb context` for prior knowledge, but the helper in `pkg/spawn/kbcontext.go` kills that command after 5 seconds and treats the timeout exactly like "no matches," so `AnalyzeGaps` records a false `no_context` result. Real sample queries from those GPT-5.4 events returned useful context when run manually, but they took about 5.8-8.8 seconds, which is just longer than the hardcoded timeout.

## Verification Contract

See `./VERIFICATION_SPEC.yaml` for the exact commands, reproduced symptom, and evidence trail. Key outcome: dry-run still shows the false 0/100 score, manual `kb context` returns real matches for the same derived queries, and the code path from timeout to `AnalyzeGaps(nil, ...)` is confirmed.

## TLDR

I traced the reported GPT-5.4 0/100 context scores to a timeout bug in pre-spawn KB lookup. `RunKBContextCheckForDir()` uses a hardcoded 5 second timeout, and slow-but-successful `kb context` queries get collapsed into nil results, which makes the gap scorer report `no_context` instead of "context exists but lookup was too slow."

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-debug-debug-investigate-kb-26mar-4427/VERIFICATION_SPEC.yaml` - Verification evidence for the investigation.
- `.orch/workspace/og-debug-debug-investigate-kb-26mar-4427/SYNTHESIS.md` - Findings, evidence, and recommended follow-up.
- `.orch/workspace/og-debug-debug-investigate-kb-26mar-4427/BRIEF.md` - Dylan-facing comprehension artifact.

### Files Modified
- None - investigation only.

### Commits
- Pending local commit for workspace artifacts.

---

## Evidence (What Was Observed)

- `pkg/spawn/kbcontext.go:137` creates `context.WithTimeout(..., 5*time.Second)` for every `kb context` query.
- `pkg/spawn/kbcontext.go:154` to `pkg/spawn/kbcontext.go:158` swallows command errors and returns `nil`, so timeouts are indistinguishable from true no-result cases.
- `pkg/orch/spawn_kb_context.go:83` to `pkg/orch/spawn_kb_context.go:106` uses that nil result to print "No prior knowledge found" and sets `GapAnalysis` from `AnalyzeGaps(result, keywords, projectDir)`.
- `orch spawn --dry-run --headless --model gpt-5.4 feature-impl "add IsReasoningModel test for gpt-5.4 in model_test.go"` reproduced the symptom exactly: query `"isreasoningmodel test gpt"`, then `Context quality: 0/100`, then `No prior knowledge found.`
- Manual `kb context` for sample event-derived queries returned real context but took longer than the timeout: `isreasoningmodel test gpt` in 6.82s, `comment explaining skillmodelmapping` in 6.756s, `verify dao claim` in 6.611s, `gpt entry guides` in 6.581s, `how does account` in 5.834s.
- `~/.orch/events-2026-03.jsonl` contains GPT-5.4 headless spawn events with `gap_context_quality: 0`, `gap_match_total: 0`, and `gap_types: ["no_context"]` for tasks whose equivalent manual `kb context` queries now return matches.

### Tests Run
```bash
go test ./pkg/spawn -run 'Test.*Gap|Test.*KBContext|Test.*gpt|Test.*GPT'
# PASS: ok github.com/dylan-conlin/orch-go/pkg/spawn

orch spawn --dry-run --headless --model gpt-5.4 feature-impl "add IsReasoningModel test for gpt-5.4 in model_test.go"
# PASS: reproduced Context quality 0/100 and No prior knowledge found
```

---

## Architectural Choices

No architectural choices - investigation only. Because the confirmed root cause sits in hotspot spawn code, I routed the fix to architect review instead of implementing directly.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.orch/workspace/og-debug-debug-investigate-kb-26mar-4427/SYNTHESIS.md` - Investigation synthesis.
- `.orch/workspace/og-debug-debug-investigate-kb-26mar-4427/BRIEF.md` - Reader-facing brief.
- `.orch/workspace/og-debug-debug-investigate-kb-26mar-4427/VERIFICATION_SPEC.yaml` - Verification evidence.

### Decisions Made
- Route the code change through architect follow-up first because the fault lives in the hotspot spawn context path and needs a design choice, not a blind timeout bump.

### Constraints Discovered
- The current implementation treats any `kb context` command error, including timeout, as equivalent to no context.
- The 5 second budget is below observed real-world query latency for several GPT-5.4 spawn tasks in this repo.

### Externalized via `kb quick`
- No new kb quick entry - investigation finding is captured in follow-up issue `orch-go-k6c0v` and this workspace synthesis.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** `orch-go-k6c0v`
**Skill:** `architect`
**Context:**
```text
Pre-spawn KB scoring falsely records no_context when kb context takes longer than the hardcoded 5 second timeout. Sample GPT-5.4/headless queries return real matches in 5.8-8.8 seconds, but pkg/spawn/kbcontext.go collapses timeout to nil and pkg/orch/spawn_kb_context.go turns that into a 0/100 score. Design the right timeout/fallback/observability behavior for hotspot spawn code before implementation.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should slow `kb context` be surfaced as a distinct degraded state instead of being folded into `no_context`?
- Should local and global KB queries have separate timeout budgets or progressive partial-result behavior?

**Areas worth exploring further:**
- Whether the same timeout is skewing non-GPT spawn quality metrics beyond the sampled GPT-5.4 cases.
- Whether `kb context` latency has recently regressed enough to justify instrumentation in the spawn pipeline.

**What remains unclear:**
- The best timeout budget versus responsiveness trade-off for daemon spawns.

---

## Friction

`tooling`: `git status --short` in this repo is extremely noisy because the worktree already contains many unrelated changes, which makes worker verification harder to scan.

---

## Session Metadata

**Skill:** `systematic-debugging`
**Model:** `openai/gpt-5.4`
**Workspace:** `.orch/workspace/og-debug-debug-investigate-kb-26mar-4427/`
**Investigation:** Investigation captured in workspace synthesis rather than a `.kb/investigations/` artifact.
**Beads:** `bd show orch-go-304ta`

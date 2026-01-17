# Session Synthesis

**Agent:** og-arch-deep-analysis-opencode-17jan-5651
**Issue:** orch-go-fkw2l
**Duration:** 2026-01-17 21:23 → 2026-01-17 22:00
**Outcome:** success

---

## TLDR

Deep analysis of OpenCode coaching plugin architecture reveals a comprehensive behavioral detection system implementing "Pain as Signal" pattern; primary reliability issue is the detectWorkerSession caching bug (already fixed, needs deployment verification) that misclassifies worker sessions as orchestrators, preventing worker-specific health metrics from being emitted.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Comprehensive analysis of coaching plugin architecture, issues, and nervous system concept

### Files Modified
- None (research-only investigation)

### Commits
- Pending - investigation artifact ready for commit

---

## Evidence (What Was Observed)

- Coaching metrics file has recent data (tail -30 shows action_ratio, analysis_paralysis, compensation_pattern entries)
- Zero worker-specific metrics exist (grep for tool_failure_rate/context_usage returns empty despite workers being active)
- Detection code implements 8 behavioral patterns: action_ratio, analysis_paralysis, behavioral_variation, frame_collapse, circular_pattern, dylan_signal_prefix, compensation_pattern, premise_skipping
- Worker health tracking code exists at lines 1157-1289 in coaching.ts but never fires
- detectWorkerSession bug documented in prior investigations:
  - 2026-01-17-inv-design-review-coaching-plugin-failures.md - root cause analysis
  - 2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md - fix implementation
- noReply injection pattern used in 5+ plugin files for context injection
- Two injection mechanisms: config.instructions (file refs, config-time) vs client.session.prompt (content, runtime)

### Commands Run
```bash
# Check recent coaching metrics
tail -30 ~/.orch/coaching-metrics.jsonl
# OUTPUT: Shows action_ratio, analysis_paralysis, compensation_pattern entries

# Check for worker-specific metrics
grep -E "tool_failure_rate|context_usage|time_in_phase|commit_gap" ~/.orch/coaching-metrics.jsonl
# OUTPUT: Empty - zero worker metrics

# List global plugins
ls -la ~/.config/opencode/plugin/
# OUTPUT: friction-capture.ts, session-compaction.ts, guarded-files.ts, session-resume.js
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Comprehensive analysis including 8 findings on architecture, detection, injection, and issues

### Decisions Made
- "Pain as Signal" architectural pattern is valid: behavioral friction should be surfaced to agents in real-time via tool-layer injection
- Worker detection bug is root cause of missing worker metrics (not a detection logic issue)
- Two injection mechanisms serve distinct purposes (don't consolidate them)

### Constraints Discovered
- **Plugins cannot see LLM response text** - All behavioral detection must use tool calls as proxies. This is fundamental to OpenCode plugin architecture.
- **In-memory session state lost on restart** - Injection coupled to observation means server restarts break coaching for long-running sessions.
- **Tool call order affects worker detection** - First signals matter; if non-worker tool call comes first, caching bug (before fix) permanently misclassified session.

### Patterns Identified
1. **Three plugin execution patterns:** Gates (tool.execute.before + throw), Context Injection (tool.execute.before/event), Observation (tool.execute.after/event)
2. **Two injection mechanisms:** config.instructions (static, config-time), client.session.prompt(noReply:true) (dynamic, runtime)
3. **Pain as Signal architecture:** detection → threshold transformation → tool-layer injection

### Externalized via `kn`
- Recommend: `kn constraint "Plugins cannot see LLM response text - all behavioral detection must use tool calls as proxies" --reason "OpenCode plugin API limitation"`
- Recommend: `kn decide "Pain as Signal is the architectural pattern for agent self-awareness - friction surfaces in real-time, not post-hoc" --reason "Enables autonomous error correction; documented in CLAUDE.md and coaching.ts"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation artifact with 8 findings)
- [x] Tests passing (N/A - research investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-fkw2l`

### Follow-up Work (Optional)
**Issue:** Verify detectWorkerSession fix deployment
**Skill:** investigation
**Context:**
```
Restart OpenCode server with orch-dashboard restart, spawn a worker agent, verify
worker-specific metrics (tool_failure_rate, context_usage) appear in coaching-metrics.jsonl.
This confirms the fix from 2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md is active.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How effective are coaching messages at changing agent behavior? No measurement exists.
- Could OpenCode provide token counting API for accurate context_usage? Currently just estimated.
- Should injection be decoupled from observation via daemon architecture? Jan 11 investigation recommended this but not yet implemented.

**Areas worth exploring further:**
- Effectiveness measurement: do agents actually change behavior after coaching injection?
- Daemon-based injection to survive server restarts
- Session correlation: mapping OpenCode session IDs to orch agent IDs

**What remains unclear:**
- Whether the detectWorkerSession fix is currently deployed (needs server restart to verify)
- Performance impact of not caching false results in detection

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-deep-analysis-opencode-17jan-5651/`
**Investigation:** `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md`
**Beads:** `bd show orch-go-fkw2l`

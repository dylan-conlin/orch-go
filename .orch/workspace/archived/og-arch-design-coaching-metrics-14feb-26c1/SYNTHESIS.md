# Session Synthesis

**Agent:** og-arch-design-coaching-metrics-14feb-26c1
**Issue:** orch-go-0pc
**Duration:** 2026-02-14 ~23:00 → 2026-02-14 ~23:45
**Outcome:** success

---

## TLDR

Produced a comprehensive design for the coaching metrics redesign across 3 subsystems: plugin metric pruning (8→5 orchestrator metrics), orch stats behavioral health section, and session-start auto-surfacing via `session.created` event hook. Identified that `direct_implementation` is functionally identical to existing `frame_collapse` and should be merged rather than duplicated. Recommended 6-phase implementation plan spanning 2-4 agents.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-14-design-coaching-metrics-redesign.md` — Full architecture design with 5 decision forks navigated
- `.kb/models/coaching-plugin/probes/2026-02-14-metrics-redesign-architecture-validation.md` — Model probe confirming invariants hold under redesign
- `.orch/workspace/og-arch-design-coaching-metrics-14feb-26c1/SYNTHESIS.md` — This file

### Files Modified
- None (design-only session)

### Commits
- (pending — will commit all 3 files)

---

## Evidence (What Was Observed)

- 1022 coaching metrics in JSONL: action_ratio (381), analysis_paralysis (355), behavioral_variation (145), compensation_pattern (79), context_ratio (41), frame_collapse (13), circular_pattern (7), context_usage (1)
- action_ratio + analysis_paralysis = 72% of all metrics — high volume indicates noise, not signal
- frame_collapse (13 events) and circular_pattern (7 events) are genuinely low-noise, high-signal
- coaching.ts is 1831 lines; removing 6 metrics drops ~500 lines to ~1380 (below accretion boundary)
- plugins/coaching.ts and .opencode/plugin/coaching.ts are identical (no drift)
- OpenCode `event` hook supports `session.created` events with full session metadata
- `session.metadata.role` IS available at session creation time (set before event fires)
- serve_coaching.go hardcodes action_ratio and analysis_paralysis in health calculation — must be updated

### Tests Run
```bash
# Checked coaching metrics distribution
cat ~/.orch/coaching-metrics.jsonl | python3 -c "..."
# Result: 1022 lines, 8 metric types as documented above

# Checked plugin file sync
diff plugins/coaching.ts .opencode/plugin/coaching.ts
# Result: no diff (files identical)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-14-design-coaching-metrics-redesign.md` — Architecture for all 3 subsystems

### Decisions Made
1. `direct_implementation` should merge into `frame_collapse` (not create duplicate metric) — they detect the same thing
2. Shared metrics reader in `pkg/coaching/metrics.go` (not inline duplication between stats and serve)
3. Session-start surfacing via `event` hook on `session.created` — not the "coupled observation" anti-pattern
4. Go-side metric writing for `completion_backlog` is safe (JSONL supports concurrent appenders)
5. Kill metrics via full removal, not disabling — reduces coaching.ts below accretion boundary

### Constraints Discovered
- Plugin `session.created` fires for ALL sessions including workers — must filter by metadata.role
- Completion backlog detection requires Go backend (not plugin) because it needs agent phase state
- Shell-out from plugin to `orch stats` adds ~500ms latency — must be async for session-start

---

## Verification Contract

See design investigation for implementation plan and acceptance criteria. Key verification:
- After Phase 3: coaching.ts < 1500 lines
- After Phase 2: `orch stats` shows behavioral health section
- After Phase 5: new orchestrator sessions receive health summary injection
- Throughout: worker health metrics (4 types) unchanged

---

## Next (What Should Happen)

**Recommendation:** close + spawn follow-ups

### Follow-up Spawns

**Issue 1 (already exists): orch-go-0pc implementation**
Spawn 2-4 agents per the phased plan:
- Agent 1 (feature-impl): Phase 1 + 2 — shared metrics reader + stats integration
- Agent 2 (feature-impl): Phase 3 + 4 — plugin metric redesign + serve_coaching update
- Agent 3 (feature-impl, follow-up): Phase 5 — session-start auto-surfacing
- Agent 4 (feature-impl, follow-up): Phase 6 — completion backlog detection

### Escalation Point
**`direct_implementation` vs `frame_collapse`:** The design recommends merging these. If the orchestrator intended them as separate metrics, the implementation plan needs adjustment.

---

## Unexplored Questions

- Should `behavioral_variation` threshold be raised from 3 to 5? (Data suggests yes — 145 events is high)
- Should completion_backlog write to coaching-metrics.jsonl or a separate file? (Recommend same file)
- Should the dashboard add a dedicated coaching metrics panel showing historical trends? (Currently just shows aggregate status)
- Will pruning (1000-line cap) conflict if both Go and plugin write to the file? (Only plugin prunes at startup — should be fine, but worth monitoring)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-coaching-metrics-14feb-26c1/`
**Investigation:** `.kb/investigations/2026-02-14-design-coaching-metrics-redesign.md`
**Beads:** `bd show orch-go-0pc`

# Session Synthesis

**Agent:** og-arch-design-implementation-architecture-27feb-4348
**Issue:** orch-go-pgjz
**Outcome:** success

---

## Plain-Language Summary

Dylan realized his knowledge system serves agents well (via kb context at spawn time) but doesn't serve him directly — his mental model is maintained through the flow of work, not by reading knowledge files. This session designed how knowledge surfaces at the moments he's already engaged, so the flow itself becomes the medium for staying immersed.

The core recommendation: build an `orch orient` command for session start that composites model summaries, throughput baseline, and freshness warnings into a single orientation output the orchestrator presents conversationally. Then add a model-impact touchpoint to the completion review pipeline that tells the orchestrator "this agent's work confirms/extends/contradicts your model of [domain]." These two integration points — session start and completion review — are where knowledge surfaces through the flow rather than requiring separate study.

Open question 5 (where does meta-monitoring terminate?) is resolved: it terminates at the orchestrator conversation itself. The orchestrator IS the comprehension layer. Dylan's explain-back responses ARE the verification signal. The anti-sycophancy constraint prevents the loop from closing silently. No additional monitoring layer is needed.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root for acceptance criteria.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-27-design-implementation-architecture-flow-integrated-knowledge-surfacing.md` — Full design investigation with 5 navigated forks, substrate traces, and implementation-ready output
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-27-probe-flow-integrated-knowledge-surfacing-architecture.md` — Probe confirming and extending the orchestrator session lifecycle model

### Beads Issues Created
- `orch-go-j8xf` — Implement orch orient command (P2, feature)
- `orch-go-oxrz` — Add model-impact touchpoint to orch complete (P2, feature, depends on j8xf)
- `orch-go-6vux` — Add trust calibration tier to completion review (P3, feature, depends on j8xf + oxrz)
- `orch-go-o8k1` — Update orchestrator skill session start protocol (P2, task, depends on j8xf)

### Dependency Graph
```
orch-go-j8xf (orch orient)
  ├── blocks orch-go-oxrz (model-impact touchpoint)
  ├── blocks orch-go-6vux (trust calibration)
  └── blocks orch-go-o8k1 (skill update)

orch-go-oxrz (model-impact touchpoint)
  └── blocks orch-go-6vux (trust calibration)
```

---

## Evidence (What Was Observed)

- Session start protocol (orchestrator skill lines 329-335) has NO programmatic command — purely skill guidance with manual `bd ready` + `orch status`
- Completion pipeline has 4 knowledge touchpoints but none answer "how does this change your model of X?" across full model corpus
- `kb context` is designed for agent injection (80k char budget, full model summaries) — needs human-facing adaptation (500-800 char budget, 2-3 sentence summaries)
- `orch stats` already parses events.jsonl with throughput metrics — format adaptation needed, not new data collection
- 22 models exist in `.kb/models/` — each has `Last Updated` date but no programmatic freshness query
- Trust calibration signals already exist in the system (model freshness, verification level, per-skill completion rate, probe verdicts) but aren't aggregated

---

## Architectural Choices

### Thread 1 first over Thread 3

- **What I chose:** Model Surfacing at Engagement Moments (Thread 1) as first prototype
- **What I rejected:** Calibrated Trust (Thread 3) first
- **Why:** Thread 1 has the highest-leverage existing integration points and provides the model freshness data that Thread 3 needs. Thread 3 is a natural follow-on, not a prerequisite.
- **Risk accepted:** Completion review pacing remains manual (skill guidance) until Thread 3 ships. Acceptable because current Light/Medium/Heavy pacing already works.

### New `orch orient` command over skill-only guidance

- **What I chose:** New `orch orient` command that composites data sources
- **What I rejected:** Updating orchestrator skill to instruct running kb context, orch stats, bd ready separately
- **Why:** Principle (Gate Over Remind) — a command produces structured output or it doesn't. Skill guidance says "remember to check models" and is easily skipped. Session Amnesia also favors infrastructure over instructions.
- **Risk accepted:** New command means new maintenance. Mitigated by keeping it thin — composites existing data sources.

### New model-impact touchpoint over enhancing existing touchpoints

- **What I chose:** New touchpoint between probe verdicts and architectural choices
- **What I rejected:** Enhancing existing knowledge maintenance or probe verdicts touchpoints
- **Why:** Principle (Evolve by Distinction) — model-impact analysis and quick entry lifecycle are different concerns. Mixing them would conflate two feedback loops.
- **Risk accepted:** Another step in completion pipeline increases review time. Mitigated by keeping it 1-3 informational lines.

### Meta-monitoring terminates at orchestrator conversation

- **What I chose:** No additional monitoring layer — orchestrator conversation IS the termination point
- **What I rejected:** Building drift detection metrics, comprehension scoring, surfacing quality monitoring
- **Why:** Principle (Verification Bottleneck) — the human IS the termination. Principle (Pressure Over Compensation) — don't compensate for missed surfacing, let stale models create visible pressure at next orientation.
- **Risk accepted:** No automated detection of surfacing quality. Acceptable because the explain-back gate already tests comprehension quality, and the anti-sycophancy constraint prevents silent closure.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-27-design-implementation-architecture-flow-integrated-knowledge-surfacing.md` — Implementation architecture for all 4 threads

### Constraints Discovered
- `kb context` budget (80k chars) is agent-sized — human-facing surfacing needs 500-800 char budget
- Model freshness has no programmatic API — need to parse `Last Updated` from model.md frontmatter

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (architectural recommendation, concrete implementation shape, OQ5 resolution, sequenced issues)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-pgjz`

---

## Unexplored Questions

- How should `orch orient` handle cross-project models? The `--global` flag on `kb context` surfaces cross-project knowledge, but at session start the focus might be narrower.
- Should model summaries be cached between orient calls, or regenerated each time? For 22 models with freshness checks, the cost is low enough to always regenerate.
- Could the trust tier (Thread 3) eventually become a gate instead of advisory? If so, what would the override mechanism look like? Probably `--force-trust` similar to `--force-hotspot`.

---

## Session Metadata

**Skill:** architect
**Workspace:** `.orch/workspace/og-arch-design-implementation-architecture-27feb-4348/`
**Investigation:** `.kb/investigations/2026-02-27-design-implementation-architecture-flow-integrated-knowledge-surfacing.md`
**Beads:** `bd show orch-go-pgjz`

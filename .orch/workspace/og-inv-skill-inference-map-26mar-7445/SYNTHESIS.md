# Session Synthesis

**Agent:** og-inv-skill-inference-map-26mar-7445
**Issue:** orch-go-hv9lc
**Duration:** 2026-03-26 09:18 -> 2026-03-26 09:34
**Outcome:** success

---

## Plain-Language Summary

I traced how the daemon decides which skill to spawn from an issue and found that descriptions are only the third chance to influence routing. The code first trusts an explicit `skill:*` label, then checks the title for words like `Design`, `Investigate`, or `Fix`, and only then scans the description for a small set of hard-coded phrases. That matters because many generic `task` issues fall straight to `feature-impl` unless they carry one of those stronger signals.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcomes: repo location verified, focused daemon inference tests passed, and the description heuristic cases were validated with verbose subtest output.

## TLDR

The daemon's skill inference is a four-stage deterministic chain: label, title, description, then type. Description-based routing is not semantic NLP; it is simple substring matching that only maps to `investigation`, `research`, or detailed `systematic-debugging` before falling back to coarse type defaults.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-inv-skill-inference-map-26mar-7445/VERIFICATION_SPEC.yaml` - Verification contract for the code trace and tests.
- `.orch/workspace/og-inv-skill-inference-map-26mar-7445/SYNTHESIS.md` - Session synthesis for orchestrator review.
- `.orch/workspace/og-inv-skill-inference-map-26mar-7445/BRIEF.md` - Dylan-facing comprehension brief.

### Files Modified
- `.kb/investigations/2026-03-26-inv-skill-inference-map-task-descriptions.md` - Filled findings, synthesis, uncertainty, recommendations, references, and completion metadata.

### Commits
- Pending commit at session end.

---

## Evidence (What Was Observed)

- `InferSkillFromIssue()` enforces a fixed precedence chain: `skill:*` label -> title pattern -> description heuristic -> type fallback (`pkg/daemon/skill_inference.go:214`).
- `InferSkillFromDescription()` uses lowercase substring matching, not semantic parsing, to map text into investigation, research, or detailed debugging buckets (`pkg/daemon/skill_inference.go:146`).
- Vague bug descriptions intentionally return no match and let type fallback decide, which is why generic `task` issues often still route to `feature-impl` (`pkg/daemon/skill_inference.go:178`, `pkg/daemon/skill_inference_test.go:86`).
- Spawn-time consumers (`pkg/daemon/daemon.go:376`, `pkg/daemon/preview.go:152`, `pkg/daemon/ooda.go:181`) all call the same inference function, while `pkg/daemon/allocation.go:177` mirrors the chain for scoring without logging.
- The daemon logs which inference tier won using `spawn.skill_inferred` event flags, and stats later reconstruct the method from those booleans (`pkg/events/logger.go:711`, `cmd/orch/stats_inference.go:14`).

### Tests Run
```bash
# Verify repo location
pwd
# /Users/dylanconlin/Documents/personal/orch-go

# Focused daemon inference tests
go test ./pkg/daemon -run 'TestInferSkill|TestInferModelFromSkill|TestInferBrowserToolFromLabels|TestQuestionTypeRoutesToArchitect|TestOriginalBugReproduction'
# PASS

# Description heuristic subtests
go test ./pkg/daemon -run TestInferSkillFromDescription -v
# PASS with named subtests covering audit/analyze/investigate/how does/compare/evaluate/error with stack trace/vague fix/etc.
```

---

## Architectural Choices

No architectural choices - task was within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-skill-inference-map-task-descriptions.md` - Full trace of the daemon's description-based skill routing.

### Decisions Made
- Use explicit labels or title prefixes when routing must be reliable; the description parser is later in precedence and intentionally narrow.

### Constraints Discovered
- Description heuristics only help when labels and titles are silent.
- The parser depends on exact substring lists, so synonyms outside those lists do not influence routing.

### Externalized via `kb quick`
- `kb quick decide "Daemon skill inference resolves skill in a fixed order: skill label, title pattern, description heuristic, then issue type fallback." --reason "Verified in pkg/daemon/skill_inference.go and pkg/daemon tests during orch-go-hv9lc on 2026-03-26."` - Created `kb-18fbc0`.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-hv9lc`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How often real daemon-routed issues are corrected by title or description rather than by type fallback.
- Whether question-type behavior in the live daemon fully matches all model documentation after recent fixes.

**Areas worth exploring further:**
- Precision and recall of the description keyword lists against the real issue corpus.

**What remains unclear:**
- Whether expanding the description heuristics would improve routing more than simply enforcing better issue enrichment.

---

## Friction

No friction - smooth session.

---

## Session Metadata

**Skill:** investigation
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-inv-skill-inference-map-26mar-7445/`
**Investigation:** `.kb/investigations/2026-03-26-inv-skill-inference-map-task-descriptions.md`
**Beads:** `bd show orch-go-hv9lc`

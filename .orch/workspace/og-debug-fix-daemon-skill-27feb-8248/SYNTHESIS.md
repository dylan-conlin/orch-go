# Session Synthesis

**Agent:** og-debug-fix-daemon-skill-27feb-8248
**Issue:** orch-go-ideb
**Outcome:** success

---

## Plain-Language Summary

The daemon's skill inference was only detecting skill keywords when they appeared in a `"Keyword: description"` colon-prefix format. Titles using natural language like "Investigate Claude Code --worktree flag" or "Design orchestrator diagnostic mode" had no colon-prefix match, so they fell through to type-based inference where `task` → `feature-impl`. This wasted 6 agent slots across 2 sessions by spawning investigation/architect work with the wrong skill and model.

The fix adds first-word keyword detection as a second pass in `InferSkillFromTitle`. After checking for colon-prefix patterns (existing), it now checks if the title's first word is a known skill keyword (investigate, design, explore, fix, broken, debug, architect). This catches natural language titles while preserving the existing priority order: labels > title > description > type.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification commands.

Key outcomes:
- `go test ./pkg/daemon/ -run TestInferSkill -v` → 62 tests pass, 0 fail
- Specific reproductions verified: orch-go-3wga (investigate→investigation), orch-go-cp52 (design→architect)
- Model inference confirmed: investigation and architect both → opus

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/skill_inference.go` - Added first-word keyword detection fallback in `InferSkillFromTitle`; added `design`, `explore`, `broken` to colon-prefix map
- `pkg/daemon/skill_inference_test.go` - Added `TestInferSkillFromIssue_TitleKeywordDetection` with 7 cases reproducing the 3 failure scenarios; updated existing test expectations for first-word detection

---

## Evidence (What Was Observed)

- `InferSkillFromTitle` only checked colon-prefix patterns (`SplitN(":", 2)` then matched `parts[0]`)
- "Investigate Claude Code --worktree flag" has no colon → `len(parts) < 2` → immediate return ""
- "Design orchestrator diagnostic mode: time-limited..." splits to prefix "design orchestrator diagnostic mode" which doesn't match single-word skillMap keys
- Both fall through to `InferSkill("task")` → "feature-impl"
- `skillModelMapping` already maps investigation/architect → opus (no model inference fix needed)

### Tests Run
```bash
go test ./pkg/daemon/ -run TestInferSkill -v
# PASS: 62 tests, 0 failures (0.008s)
go vet ./pkg/daemon/ && go build ./cmd/orch/
# Build OK
```

---

## Architectural Choices

### First-word detection vs contains-keyword detection
- **What I chose:** First-word only matching (check `words[0]` against keyword map)
- **What I rejected:** Substring/contains matching on full title
- **Why:** Contains matching is too aggressive — "Add feature for investigating users" would false-positive on "investigating". First-word matching is precise: investigation/design/explore titles naturally start with the action verb.
- **Risk accepted:** Titles like "The investigation into X" won't match. Acceptable because these are rare and the colon-prefix + description heuristic layers still provide fallbacks.

---

## Knowledge (What Was Learned)

### Decisions Made
- First-word keyword detection added as second pass (after colon-prefix, before description heuristic) — maintains cascade priority order

### Constraints Discovered
- None — fix is within existing patterns

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (62/62)
- [x] Ready for `orch complete orch-go-ideb`

---

## Unexplored Questions

Straightforward session, no unexplored territory.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-fix-daemon-skill-27feb-8248/`
**Beads:** `bd show orch-go-ideb`

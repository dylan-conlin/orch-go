# SYNTHESIS: Fix daemon bug→architect skill inference (orch-go-1211)

## Plain-Language Summary

The daemon's skill inference was routing bug-type issues to the `architect` skill instead of `systematic-debugging`. This meant bugs filed as `--type=bug` (like issues 1208 and 1209) would spawn an architect agent instead of a debugger. The root cause was two functions — `InferSkill` in `pkg/daemon/skill_inference.go` and `InferSkillFromIssueType` in `cmd/orch/spawn_cmd.go` — both hardcoded `bug → "architect"` with a comment about "Premise Before Solution". The fix changes both to return `"systematic-debugging"`, which already includes Phase 1 root cause investigation (the "understand before fixing" guarantee the architect routing was trying to provide). Help text in `spawn_cmd.go` and `gendoc/main.go` was also updated.

## Delta

| File | Change |
|------|--------|
| `pkg/daemon/skill_inference.go` | `InferSkill("bug")` returns `"systematic-debugging"` instead of `"architect"` |
| `pkg/daemon/skill_inference_test.go` | Test expectations updated for bug type |
| `cmd/orch/spawn_cmd.go` | `InferSkillFromIssueType("bug")` returns `"systematic-debugging"` + help text updated |
| `cmd/orch/work_test.go` | Test expectation updated for bug type |
| `cmd/gendoc/main.go` | Help text updated |

## Verification Contract

See `VERIFICATION_SPEC.yaml` for exact commands and results.

Key outcome: `go test ./pkg/daemon/ ./cmd/orch/ -run TestInferSkill` — all pass with bug→systematic-debugging.

## Model Impact

The daemon model at `.kb/models/daemon-autonomous-operation.md` documents `bug→systematic-debugging` as expected behavior. The code was contradicting the model. This fix aligns code with the documented model — no model update needed.

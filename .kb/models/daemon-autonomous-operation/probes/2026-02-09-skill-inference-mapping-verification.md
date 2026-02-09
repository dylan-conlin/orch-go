# Probe: Skill Inference Mapping Verification

**Model:** .kb/models/daemon-autonomous-operation.md
**Date:** 2026-02-09
**Status:** Complete

---

## Question

The model's Skill Inference table claims four mappings and a fallback:

| Issue Type | Inferred Skill       |
| ---------- | -------------------- |
| `task`     | feature-impl         |
| `bug`      | systematic-debugging |
| `feature`  | feature-impl         |
| `epic`     | architect            |
| (fallback) | investigation        |

Additionally, the Constraints section claims: "Daemon uses issue type to infer skill, ignoring labels" and "skill:X label has no effect."

Are these claims accurate against the actual implementation?

---

## What I Tested

**Command/Code:**

```bash
# 1. Ran existing unit tests for both inference paths
go test ./pkg/daemon/ -run "TestInferSkill" -v
go test ./cmd/orch/ -run "TestInferSkillFromIssueType" -v

# 2. Wrote standalone Go program to verify each model-claimed mapping
# against InferSkill() and IsSpawnableType()
go run /tmp/skill_inference_check.go
```

**Environment:**

- Branch: dev (orch-go)
- Two inference implementations: `pkg/daemon/skill_inference.go` (daemon path) and `cmd/orch/spawn_skill_inference.go` (CLI path)
- Daemon calls `InferSkillFromIssue()` at `daemon_spawn.go:240` and `:384`

---

## What I Observed

**Test output (existing tests):**

```
=== RUN   TestInferSkill
--- PASS: TestInferSkill/bug       → systematic-debugging
--- PASS: TestInferSkill/feature   → feature-impl
--- PASS: TestInferSkill/task      → feature-impl
--- PASS: TestInferSkill/investigation → investigation
--- PASS: TestInferSkill/question  → investigation
--- PASS: TestInferSkill/epic      → error (not spawnable)
--- PASS: TestInferSkill/unknown   → error
PASS
```

**Standalone verification output:**

```
=== Skill Inference Mapping Verification ===
PASS: task → feature-impl
PASS: bug → systematic-debugging
PASS: feature → feature-impl
FAIL: epic → error: cannot infer skill for issue type: epic (expected architect, spawnable=false)

=== Types in Code But NOT in Model ===
  investigation → investigation (spawnable=true)
  question → investigation (spawnable=true)

=== Fallback Test ===
  unknown_type → error (model claims fallback to 'investigation')
  epic spawnable: false
```

**Key observations:**

1. **`epic → architect` is WRONG.** Code returns error: "cannot infer skill for issue type: epic". `IsSpawnableType("epic")` returns `false`. Epics are explicitly non-spawnable, not mapped to architect.

2. **Fallback to `investigation` is WRONG.** `InferSkill()` returns an error for unknown types, not a default skill. The daemon skips issues with inference errors (`daemon_spawn.go:241-247`). The CLI path (`cmd/orch/spawn_skill_inference.go:60`) falls back to `feature-impl` on error, not `investigation`.

3. **Two types missing from model table:** `investigation → investigation` and `question → investigation` are implemented and tested but absent from the model's table.

4. **Label override claim is WRONG.** The model's constraint section says "skill:X label has no effect." In reality, `InferSkillFromIssue()` (the function the daemon actually calls) checks `skill:*` labels FIRST, before type inference. Priority order: `skill:* label > title pattern > issue type`. This is tested: `TestInferSkillFromIssue/skill_label_takes_priority`.

5. **Title-based inference undocumented.** `InferSkillFromTitle()` maps "Synthesize ... investigations" titles to `kb-reflect`. Not mentioned in the model.

---

## Model Impact

**Verdict:** contradicts — Skill Inference table and "labels ignored" constraint

**Details:**

The model has four inaccuracies in the Skill Inference section:

1. **`epic → architect` is fictional.** Epics return an error and are not spawnable. The model should remove this row or note that epics are non-spawnable.

2. **Fallback behavior is wrong.** Unknown types produce errors (skipped by daemon), not a default `investigation` skill. No silent fallback exists.

3. **Table is incomplete.** `investigation → investigation` and `question → investigation` are real, tested mappings omitted from the table.

4. **"Labels ignored" constraint is stale.** `InferSkillFromIssue()` respects `skill:*` labels with highest priority, directly contradicting the constraint section. This was likely accurate at one point but the code evolved without the model being updated.

The complete, accurate mapping table should be:

| Issue Type      | Inferred Skill        | Notes |
| --------------- | --------------------- | ----- |
| `bug`           | systematic-debugging  |       |
| `feature`       | feature-impl          |       |
| `task`          | feature-impl          |       |
| `investigation` | investigation         |       |
| `question`      | investigation         |       |
| `epic`          | error (non-spawnable) |       |
| (unknown)       | error (skipped)       |       |

With override priority: `skill:*` label > title pattern ("Synthesize...") > issue type.

**Confidence:** High — Verified against source code, unit tests (all passing), and standalone test program. Both `pkg/daemon` and `cmd/orch` paths examined.

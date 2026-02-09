# Probe: Is label-based model routing wired through daemon spawn paths?

**Model:** multi-model-evaluation-feb2026
**Date:** 2026-02-08
**Status:** Complete

---

## Question

Does the current codebase implement the model claim that per-issue label routing (`model:*`) is still not wired into daemon-driven `orch work` spawns?

---

## What I Tested

**Command/Code:**
```bash
python3 - <<'PY'
from pathlib import Path
checks = {}
checks['daemon_has_infer_model_fn'] = 'InferModelFromLabels(' in Path('pkg/daemon/skill_inference.go').read_text()
work = Path('cmd/orch/work_cmd.go').read_text()
checks['work_cmd_reads_model_label'] = 'inferModelFromBeadsIssue' in work or 'InferModelFromLabels' in work
checks['work_cmd_sets_spawn_model'] = 'spawnModel =' in work
focus = Path('cmd/orch/focus.go').read_text()
checks['focus_has_model_flag'] = 'focusModel' in focus or '--model' in focus
for k,v in checks.items():
    print(f"{k}: {'YES' if v else 'NO'}")
PY

go test ./cmd/orch ./pkg/daemon -run 'TestInferSkillFromIssueType|TestInferSkillFromLabels' -count=1
```

**Environment:**
- Branch: current workspace branch for `orch-go-21350`
- Project: `/Users/dylanconlin/Documents/personal/orch-go`

---

## What I Observed

**Output:**
```text
daemon_has_infer_model_fn: NO
work_cmd_reads_model_label: NO
work_cmd_sets_spawn_model: NO
focus_has_model_flag: NO
ok   github.com/dylan-conlin/orch-go/cmd/orch 0.017s
ok   github.com/dylan-conlin/orch-go/pkg/daemon 0.010s
```

**Key observations:**
- No model-label inference function is present in daemon inference code, while skill-label inference exists and is tested.
- `orch work` currently infers skill and MCP only; it does not extract model labels or set `spawnModel` before calling the shared spawn pipeline.
- `orch focus` has no model preference flag today, so no existing focus-level signal can influence daemon routing.

---

## Model Impact

**Verdict:** confirms - Label-based per-issue routing remains unimplemented in daemon spawn path.

**Details:**
This probe confirms the open claim in `multi-model-evaluation-feb2026.md` that default model routing works via config but label-based per-issue routing is not yet wired. The current path supports `skill:*` and `needs:*` labels only; model routing requires new inference and plumbing from issue labels into spawn config.

**Confidence:** High - Direct code path checks and targeted test execution both align on the same absence.

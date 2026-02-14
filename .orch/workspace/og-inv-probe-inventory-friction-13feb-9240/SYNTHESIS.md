# Session Synthesis

**Agent:** og-inv-probe-inventory-friction-13feb-9240
**Issue:** orch-go-xy9
**Duration:** 2026-02-13 → 2026-02-13
**Outcome:** success

---

## TLDR

Inventoried 48 friction gates across spawn (10), completion (12), and daemon (20+) subsystems. Only 3 of 12 completion gates have healthy bypass:fail ratios (build 0.7:1, git_diff 1.1:1, verification_spec 1.1:1). 73.4% of 1,008 bypass events stem from three systemic patterns: skill-class blindness (31.7%), model incompatibility (25.6%), and blanket override (16.7%). Recommends skill-class-aware gate selection as highest-impact fix.

---

## Delta (What Changed)

### Files Created
- `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` — Full probe with inventory table, bypass:fail ratios, and KEEP/SOFTEN/REMOVE classifications for all 48 gates
- `.kb/investigations/2026-02-13-inv-probe-inventory-friction-gates-across.md` — Investigation coordination artifact with D.E.K.N. summary and implementation recommendations

### Files Modified
- None (read-only investigation)

### Commits
- (pending — investigation and probe artifacts)

---

## Evidence (What Was Observed)

- Build gate is the only completion gate with failures > bypasses (171 failed vs 114 bypassed = 0.7:1 ratio)
- agent_running gate has ∞:1 bypass:fail ratio (183 bypassed, 0 failures) — 94% due to GPT model incompatibility
- model_connection gate has 71:1 ratio — almost never catches anything
- 320 bypass events (31.7%) cite "docs-only change" as reason — skill-class blindness
- Daemon dedup fires 3,866 times vs 41 successful spawns — working as designed (polling model)
- --force usage is 1.8% (4/219 completions), down from 72.8% pre-targeted-skip rollout
- Spawn and daemon gates are well-calibrated with no evidence of systematic noise

### Tests Run
```bash
# Events analysis via python3
python3 -c "parse ~/.orch/events.jsonl for bypass rates per gate"
# Result: 7,029 events, 1,008 bypass events across 10 distinct gates

# Code search for gate implementations
rg --type go "Gate" pkg/verify/check.go
# Result: 12 gate constants identified

# Daemon gate analysis
rg --type go "IsSpawnableType|triage|MaxSpawns" pkg/daemon/
# Result: 20+ daemon gates identified
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` — Complete cross-subsystem gate inventory with quantified value assessment

### Decisions Made
- None (this is analysis, not implementation)

### Constraints Discovered
- Completion gates are skill-class-blind — same gates fire for investigation/docs work as for feature-impl code work
- Daemon gates are fundamentally different from completion gates — they're filters (skip silently), not blockers (fail loudly)
- Some gates in events data (agent_running, model_connection, commit_evidence, dashboard_health, verification_spec) don't appear in current pkg/verify/check.go constants — may be from earlier code or different subsystem

### Externalized via `kn`
- `kb quick constrain "Completion gates must be skill-class-aware" --reason "31.7% of bypass events are docs-only changes where test_evidence, build, and synthesis gates are structurally inapplicable"` — (to be run)

---

## Next (What Should Happen)

**Recommendation:** close — probe deliverable is complete

### If Close
- [x] All deliverables complete (probe + investigation + SYNTHESIS.md)
- [x] Tests passing (read-only investigation — no code changes)
- [x] Investigation file has **Phase:** Complete
- [x] Ready for `orch complete orch-go-xy9`

### Spawn Follow-up (Recommended)
**Issue:** Implement skill-class-aware gate selection for orch complete
**Skill:** feature-impl
**Context:**
```
Prior probe (orch-go-xy9) inventoried all 48 friction gates and found 31.7% of completion bypass events
are docs-only changes. Implement auto-skip for test_evidence, build, and synthesis gates when completing
knowledge-producing skills (investigation, architect, research, capture-knowledge). See probe at
.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
```

---

## Unexplored Questions

- Events data only covers 5 days (Feb 9-13). A broader window might reveal different patterns.
- What happens when agent_running, model_connection, and commit_evidence gates are actually removed? Are there edge cases?
- How should the "attempting to skip everything" pattern (16.7%) be addressed? Deprecate --force? Require justification?
- Some event gate names don't match code constants — what subsystem emits them?

---

## Verification Contract

**VERIFICATION_SPEC.yaml** is present in workspace root with:
- Probe artifact exists and is complete
- Investigation artifact exists with D.E.K.N. filled
- SYNTHESIS.md exists with all sections
- No code changes (read-only probe)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-probe-inventory-friction-13feb-9240/`
**Investigation:** `.kb/investigations/2026-02-13-inv-probe-inventory-friction-gates-across.md`
**Probe:** `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md`
**Beads:** `bd show orch-go-xy9`

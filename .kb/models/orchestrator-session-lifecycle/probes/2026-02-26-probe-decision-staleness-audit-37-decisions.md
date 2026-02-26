# Probe: Decision Staleness Audit — 37 Uncited Decisions Against Current Architecture

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-26
**Status:** Complete

---

## Question

The orchestrator-session-lifecycle model claims (1) a three-level hierarchy (meta → orchestrator → worker), (2) session registry at `~/.orch/sessions.json`, and (3) orchestrators produce SESSION_HANDOFF.md. With 37 decisions (ages 11-67 days) having 0 citations, do these decisions still reflect the actual architecture, or has the model drifted from implementation?

---

## What I Tested

Read all 37 decision files and validated their core claims against the codebase using targeted searches:

```bash
# Session registry existence (model claims it exists; decisions #9, #27 say it shouldn't)
grep -r "sessions.json" cmd/ pkg/
# Found: architecture_lint_test.go FORBIDS sessions.json (line 36)

# Registry package (model mentions session registry)
ls pkg/session/registry.go 2>/dev/null  # Does not exist (deleted)

# Strategic orchestrator framing (decisions #7, #10)
grep -r "strategic comprehender" ~/.claude/skills/meta/orchestrator/SKILL.md
# Found: current framing is "strategic comprehender", not "meta-orchestrator"

# Five-tier escalation (decision #30, archived)
grep -r "EscalationLevel" pkg/verify/
# Found: pkg/verify/escalation.go with 5 levels fully implemented

# Daemon config (decision #37)
grep -r "daemonConfigFromFlags" cmd/orch/daemon.go
# Found: function exists with 4 call sites

# Default model (decision #15 says Gemini; reality is Claude)
grep "DefaultModel" pkg/model/model.go
# Found: anthropic/claude-sonnet-4-5 is default (not Gemini)
```

---

## What I Observed

### Model Staleness Confirmed: Session Registry Claims Are Wrong

The orchestrator-session-lifecycle model states:
> "Orchestrators [...] track via session registry (not beads) because orchestrators manage conversations, not work items."

**Reality:** The session registry (`~/.orch/sessions.json`, `pkg/session/registry.go`) has been **deleted** and is **actively forbidden** by architecture lint tests. Decision #9 (orchestrator lifecycle without beads) proposed the registry, but Decision #27 (registry-contract-spawn-cache-only) redefined it as spawn-cache, and it was subsequently removed entirely. The lint test at `architecture_lint_test.go:36` explicitly prevents re-introduction.

### Model Staleness Confirmed: "Meta-Orchestrator" Framing Outdated

The model describes a "meta-orchestrator → orchestrator → worker" hierarchy. Decisions #7 and #10 collapsed this into a single "strategic orchestrator" whose current implementation uses the framing "strategic comprehender" in the orchestrator skill. The three-level hierarchy is no longer the primary model.

### Decision #15 Fully Superseded

Decision #15 (abandon Claude Max OAuth, use Gemini primary) is archived and **completely reversed** — Claude Sonnet via Claude CLI is now the default backend, not Gemini. The model-access-spawn-paths model and orchestration-cost-economics model correctly reflect this, but Decision #15 still exists in the archived directory and could mislead agents.

### 6 Decisions Already Archived Correctly

Decisions #15, #19, #24, #25, #26, #30 are in `.kb/decisions/archived/` — correctly identified as superseded. However, they remain in the kb reflect audit because archiving doesn't remove them from the citation-tracking system.

---

## Model Impact

- [x] **Contradicts** invariant: "track via session registry (not beads)" — Session registry is deleted and architecturally forbidden. State is derived from OpenCode + Beads + Tmux at query time.
- [x] **Contradicts** invariant: "three-level hierarchy (meta-orchestrator → orchestrator → worker)" — Collapsed to single strategic orchestrator/comprehender role with daemon handling coordination.
- [x] **Extends** model with: 37-decision audit reveals the model's summary and "Why This Fails" sections reference deleted infrastructure (registry, sessions.json). The model needs updating to reflect current state-derivation pattern (no local state, query authoritative sources directly).

---

## Notes

### Decisions Categorized by Disposition

**Category A — Valid and Implemented (19 decisions):**
#1, #3, #4, #6, #7, #8, #10, #11, #13, #14, #20, #22, #27, #28, #29, #32, #33, #36, #37

**Category B — Already Archived (6 decisions):**
#15, #19, #24, #25, #26, #30

**Category C — Superseded / Needs Updating (5 decisions):**
#2 (--preview never implemented), #9 (registry removed), #12 (load_bearing not in code), #21 (observation/intervention not operationalized), #35 (investigations still active alongside probes)

**Category D — Process/Principle Decisions Still Valid (7 decisions):**
#5, #16, #17, #18, #23, #31, #34

The full categorized breakdown is in SYNTHESIS.md.

### Model Update Needed

The orchestrator-session-lifecycle model was last updated 2026-01-12. Key claims needing correction:
1. Remove all session registry references
2. Update hierarchy from three-level to strategic comprehender pattern
3. Update "Session Registry Drift" failure mode (registry no longer exists)

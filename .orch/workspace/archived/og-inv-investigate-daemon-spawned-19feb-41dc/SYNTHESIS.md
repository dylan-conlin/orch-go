# Session Synthesis

**Agent:** og-inv-investigate-daemon-spawned-19feb-41dc
**Issue:** orch-go-1104
**Duration:** 2026-02-19
**Outcome:** success

---

## Plain-Language Summary

Three daemon-spawned agents all stalled because the daemon was inadvertently using GPT-5.2-codex (an OpenAI model) instead of a Claude model. This happened due to a config precedence bug: the user config `default_model: codex` gets injected as CLI-level priority in `runWork()`, silently overriding the project config's `opencode.model: flash`. GPT-5.2-codex doesn't reliably follow the worker agent protocol — one agent hallucinated a non-existent "orchestrator policy" and self-blocked, another consumed 145K tokens exploring without completing, and the third made changes but never finished. The fix is two-fold: (1) change the user config default model, and (2) fix the config precedence so `runWork()` doesn't elevate `default_model` to CLI priority.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification criteria. Key evidence is in the probe file at `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md`.

---

## Delta (What Changed)

### Files Created
- `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md` - Root cause analysis probe documenting the GPT-5.2-codex stall pattern

### Files Modified
- None (investigation only, no code changes)

---

## Evidence (What Was Observed)

1. **User config** (`~/.orch/config.yaml`): `default_model: codex` → resolves to `openai/gpt-5.2-codex`
2. **Project config** (`.orch/config.yaml`): `opencode.model: flash` — intended for daemon use but silently overridden
3. **Config precedence in `cmd/orch/spawn_cmd.go:429-436`**: `runWork()` sets package-level `spawnModel` from user config, which enters resolve pipeline as `CLI.Model` (highest priority)
4. **Session 1098**: GPT-5.2 hallucinated "orchestrator policy forbids reading code files" after 30 seconds — self-blocked on a non-existent constraint
5. **Session 1099**: GPT-5.2 made actual code changes (4 patches) but stopped without completing the session protocol
6. **Session 1092**: GPT-5.2 consumed 145K tokens on extensive file reading, likely hit context window limits
7. **SPAWN_CONTEXT sizes**: 63-76KB — large enough to consume significant context budget with GPT tokenizers

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md` - New failure mode: Model Incompatibility Stall

### Constraints Discovered
- GPT-5.2-codex cannot reliably follow the orch worker agent protocol (hallucinated constraints, failed session close)
- `runWork()` elevates user config `default_model` to CLI priority, bypassing project config backend-specific model overrides
- 63-76KB SPAWN_CONTEXT files consume ~40-50K tokens with GPT tokenizers vs ~20-25K with Claude

---

## Next (What Should Happen)

**Recommendation:** close + spawn follow-up for config fix

### If Close
- [x] Root cause analysis complete with evidence from all 3 session logs
- [x] Probe file documents the failure mode and extends the daemon model
- [x] Config precedence bug identified with exact code location

### Follow-up Work

**Issue 1: Fix config precedence in runWork()**
- In `cmd/orch/spawn_cmd.go:429-436`, `runWork()` should NOT set `spawnModel` from user config
- Instead, let `default_model` flow through the normal resolve pipeline (where project config `opencode.model` can override it)
- This is a code change, not just a config fix

**Issue 2: User config fix (immediate)**
- Change `~/.orch/config.yaml` `default_model` to a Claude model, or remove it
- This unblocks the daemon immediately

**Issue 3: Model suitability gate**
- Add validation that blocks daemon spawning with models that don't support the worker protocol
- Or at minimum, add early stall detection (agent didn't report Phase: Planning within 2 minutes)

---

## Unexplored Questions

- Does GPT-5.2-codex work better with smaller spawn contexts? The 63-76KB contexts may be above a practical threshold.
- Would the flash model (project config's intended opencode model) have worked? Flash was explicitly set as the opencode model but was never tested because of the config override.
- Should the daemon have a separate model config independent of both user config `default_model` and project config?

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-daemon-spawned-19feb-41dc/`
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md`
**Beads:** `bd show orch-go-1104`

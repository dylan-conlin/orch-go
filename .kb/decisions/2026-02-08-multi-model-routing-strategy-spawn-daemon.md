---
status: decided
---

# Decision: Multi-Model Routing Strategy for Spawn and Daemon

**Date:** 2026-02-08
**Status:** Decided
**Context:** orch-go-21350
**Authority:** Architectural - affects spawn defaults, daemon automation, and model-selection UX

---

## Problem

GPT-5.3-codex is now production-viable for worker spawns, but routing policy is still fragmented:

- Spawn supports `--model`, `default_model`, and `skill_models`
- Daemon `orch work` infers skill/MCP from labels but does not infer model
- Existing decision (`2026-02-06-daemon-resilience-retry-staging-model-routing.md`) defines `model:*` routing but implementation has not landed

We need one routing policy that answers: task-type routing, quota influence, per-project overrides, daemon decision order, and focus-level model preference.

---

## Should We Route Multi-Model?

**Yes, selectively.**

Evidence from Feb 6-8 work shows:

- GPT-5.3-codex is good enough as default worker model across many real spawns (model: `.kb/models/multi-model-evaluation-feb2026.md`)
- The dominant GPT failure mode is completion-protocol compliance, not core implementation quality
- Session-death concerns in the non-mechanical test were control-plane restart events, not model-specific instability
- Quota/cost behavior differs materially by provider, so a single-model policy leaves useful capacity unmanaged

This decision updates the Jan 24 Claude-only posture from a GPT-5.2 era assumption to a GPT-5.3 era routing policy.

---

## Decision

Adopt a **deterministic, intent-first routing stack**:

1. Explicit operator intent (`--model`, `--backend`, `--opus`)
2. Issue-level routing (`model:*` labels)
3. Focus-level preference (`orch focus --model`, when set)
4. Skill-level defaults (`skill_models`)
5. Global/project defaults (`default_model`, backend defaults)

Quota is **advisory by default**, not primary routing logic.

---

## Routing Rules

### Rule 1: Routing is primarily by intent + skill, not quota

- Skill and label signals are stable and available at spawn time.
- Quota telemetry is incomplete/noisy for automatic hard switching (especially cross-provider comparability and current token accounting limitations).
- Therefore, quota-based switching is an opt-in pressure valve, not baseline policy.

### Rule 2: `model:*` labels are the daemon's per-issue override

Supported labels (v1):

| Label | Resolved model | Backend rule |
| --- | --- | --- |
| `model:opus` | `opus` | Force Claude backend (`--opus` semantics) |
| `model:sonnet` | `sonnet` | Use resolved backend/default |
| `model:haiku` | `haiku` | Use resolved backend/default |
| `model:gpt-5.3-codex` | `openai/gpt-5.3-codex` | Use OpenCode backend |
| `model:deepseek` | `deepseek` | Use OpenCode backend |

If multiple `model:*` labels exist, first match in label order wins (same behavior as existing `skill:*` parsing).

### Rule 3: Skill defaults remain the baseline

- Keep `default_model` as the global fallback.
- Use `skill_models` for predictable exceptions by skill class.
- Recommended baseline now: GPT-5.3-codex as worker default, with explicit label/skill overrides when Claude depth is required.

### Rule 4: Per-project configuration is supported

- Global default remains in `~/.orch/config.yaml`.
- Project-specific default remains in `.orch/config.yaml` (backend/model fields).
- Routing precedence is unchanged: explicit > project > global > built-in.

### Rule 5: Add focus-level model preference (advisory default)

- `orch focus` should accept an optional `--model` preference.
- This preference applies only when no explicit CLI model and no issue `model:*` label is present.
- Purpose: reduce repetitive model flagging during focused work, without overriding per-issue intent.

---

## Answers to Design Questions

1. **Route by skill type?** Yes, as the baseline default layer (`skill_models`), but always under explicit label/CLI overrides.
2. **Route by quota?** Not as primary logic. Use quota as advisory/opt-in fallback policy only.
3. **Configurable per-project?** Yes. Keep project-level backend/model defaults; layer beneath explicit intent.
4. **How daemon decides model?** Use deterministic precedence: CLI/flags > `model:*` label > focus preference > skill model > default model.
5. **`orch focus --model` preference?** Yes. Add as advisory default signal below labels and explicit flags.

---

## Daemon Integration Plan

### Phase 1 - Land label-based daemon routing (required)

1. Add `InferModelFromLabels(labels []string) string` in `pkg/daemon/skill_inference.go`.
2. Add `inferModelFromBeadsIssue(issue *beads.Issue) string` in `cmd/orch/spawn_skill_inference.go`.
3. In `cmd/orch/work_cmd.go`:
   - read inferred model alongside skill/MCP
   - set `spawnModel` before calling `runSpawnWithSkillInternal`
   - if model resolves to Opus, set `spawnOpus = true` (or equivalent backend-forcing path) to preserve Opus access invariant
4. Add unit tests:
   - label parsing for model inference
   - `model:opus` backend forcing behavior
   - precedence checks vs fallback skill/default model behavior

### Phase 2 - Add focus preference signal (optional but recommended)

1. Extend focus state schema with optional model preference.
2. Add `orch focus --model <alias|provider/model>`.
3. Feed focus preference into `orch work` model resolution when no explicit issue model label exists.

### Phase 3 - Quota-aware advisory mode (deferred)

1. Add threshold-based warnings (not silent rerouting) when preferred model is near quota exhaustion.
2. Allow explicit opt-in auto-fallback policy once telemetry reliability is proven.

---

## Trade-offs Accepted

- Adds one more label convention (`model:*`) but keeps routing visible and auditable in beads.
- Keeps quota logic conservative for now, trading optimal automation for predictable behavior.
- Introduces another signal layer (focus preference), but only in advisory precedence to avoid hidden overrides.

---

## Evidence

- `.kb/investigations/archived/2026-02-06-inv-side-by-side-code-quality-gpt53-vs-claude-opus.md`
- `.kb/investigations/archived/2026-02-06-inv-test-gpt-codex-non-mechanical.md`
- `.kb/investigations/archived/2026-02-06-inv-research-chatgpt-pro-gpt-5.3-codex-quota.md`
- `.kb/investigations/archived/2026-02-06-inv-token-counting-discrepancy-gpt-vs-claude.md`
- `.kb/models/multi-model-evaluation-feb2026.md`
- `.kb/models/multi-model-evaluation-feb2026/probes/2026-02-08-daemon-model-routing-not-yet-wired.md`
- `pkg/daemon/skill_inference.go`
- `cmd/orch/work_cmd.go`
- `cmd/orch/focus.go`

---

## Related

- Decision: `.kb/decisions/2026-02-06-daemon-resilience-retry-staging-model-routing.md`
- Decision: `.kb/decisions/2026-01-24-claude-specific-orchestration-accepted.md`
- Model: `.kb/models/multi-model-evaluation-feb2026.md`

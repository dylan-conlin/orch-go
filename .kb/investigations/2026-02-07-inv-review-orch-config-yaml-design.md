## Summary (D.E.K.N.)

**Delta:** Orch currently mixes user-level policy in `~/.orch/config.yaml` with project-level runtime config in `.orch/config.yaml`, but many operational defaults still live only in flags/hardcoded constants and should move to declarative config.

**Evidence:** Verified config structs, backend/model resolution, daemon build path, gap thresholding, dashboard status thresholds, and completion timeouts directly in code.

**Knowledge:** The right model is layered config with scoped ownership: built-in defaults, user global defaults, project overrides for repo-local behavior, and CLI flags for one-off execution.

**Next:** Adopt the proposed schema and layering decision, then implement in phases starting with typed schema + merge logic + compatibility shims.

**Authority:** architectural - This changes cross-command behavior (spawn/daemon/serve/complete) and config precedence across global+project scope.

---

# Investigation: Review Orch Config Yaml Design

**Question:** What should be configurable in `~/.orch/config.yaml`, should `.orch/config.yaml` provide project overrides, and what should the precedence layering be?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** Architect spawn (orch-go-21426)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-01-26-inv-test-skill-models-config.md` | extends | yes | none |
| `.kb/investigations/2025-12-21-inv-scope-orch-init-project-standardization.md` | deepens | yes | none |

---

## Findings

### Finding 1: Config exists in two files, but ownership boundaries are implicit rather than explicit

**Evidence:** Global config defines user/session/daemon/model defaults, while project config defines spawn mode, backend-local model defaults, and server ports.

**Source:** `pkg/userconfig/userconfig.go:103`, `pkg/config/config.go:20`, `cmd/orch/init.go:280`, `/Users/dylanconlin/.orch/config.yaml`, `.orch/config.yaml`.

**Significance:** The system already has a global-vs-project split, so the question is refinement and formal layering, not introducing a new pattern.

---

### Finding 2: Key operational defaults are still hardcoded or flag-only

**Evidence:** Defaults for daemon cleanup/dead-session/orphan/watchdog are flag-defined; dashboard dead/idle/stale thresholds are hardcoded; context gate threshold defaults to 20; completion has hardcoded export/rebuild timeouts.

**Source:** `cmd/orch/daemon.go:149`, `cmd/orch/daemon_loop.go:47`, `cmd/orch/serve_agents_collect.go:45`, `pkg/spawn/gap.go:336`, `cmd/orch/complete_helpers.go:20`, `cmd/orch/complete_actions.go:185`.

**Significance:** These are exactly the policy knobs users need to tune across environments; keeping them hardcoded causes drift between intent and runtime behavior.

---

### Finding 3: Precedence already exists per feature, but not as a unified contract

**Evidence:** Backend resolution is explicit flag -> project config -> global config -> default; model selection is flag -> global skill/default -> project backend model -> backend default; tier selection is flags -> global default_tier -> skill defaults.

**Source:** `cmd/orch/backend.go:19`, `cmd/orch/spawn_cmd.go:257`, `cmd/orch/spawn_usage.go:17`.

**Significance:** Behavior is coherent in each subsystem, but fragmented overall. Formalizing one global layering contract will reduce surprises and implementation divergence.

---

## Synthesis

**Key Insights:**

1. **Codify ownership by scope, not by command** - Global config should hold machine/user policy, project config should hold repo-local policy, and CLI should stay as ephemeral overrides.
2. **Move policy constants to config, keep mechanics in code** - thresholds/timeouts/defaults are config-worthy; algorithmic logic (e.g., dead-session detection method) should remain code.
3. **Use one precedence contract everywhere** - defaults -> global -> project -> CLI, with safety floors/ceilings that CLI cannot silently violate.

**Answer to Investigation Question:**

`~/.orch/config.yaml` should expand to include spawn defaults, model routing policy, context quality policy, daemon/cleanup policy, dashboard threshold preferences, and completion timeout policy. `.orch/config.yaml` should continue to exist as a project override layer, but only for project-scoped keys (backend/model defaults, spawn behavior, local context thresholds, server metadata). Final precedence should be: built-in defaults -> global user config -> project config -> CLI flags, with validation and explicit erroring when a later layer violates safety constraints.

---

## Structured Uncertainty

**What's tested:**

- ✅ Global and project config schemas verified in code (`pkg/userconfig/userconfig.go`, `pkg/config/config.go`).
- ✅ Existing precedence chains verified in code paths (`cmd/orch/backend.go`, `cmd/orch/spawn_cmd.go`, `cmd/orch/spawn_usage.go`).
- ✅ Hardcoded/default-only policy values verified in code (`cmd/orch/daemon.go`, `cmd/orch/serve_agents_collect.go`, `pkg/spawn/gap.go`, `cmd/orch/complete_*`).

**What's untested:**

- ⚠️ Full migration impact from current flat keys to proposed nested schema (not implemented).
- ⚠️ Backward compatibility behavior for every historical config variant (no compatibility tests run yet).

**What would change this:**

- If migration tests show existing configs cannot be losslessly mapped, schema must be revised for compatibility-first rollout.
- If project overrides create unsafe divergence in multi-project daemon mode, project-scope key allowlist must be narrowed.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Adopt layered config schema with global+project+CLI precedence contract | architectural | Cross-cutting behavior change across spawn/daemon/serve/complete flows |

### Recommended Approach ⭐

**Layered Policy Schema** - Define an explicit, typed config contract with scoped ownership and strict precedence.

**Why this approach:**
- Aligns with existing two-file architecture instead of replacing it.
- Converts hidden constants into explicit policy.
- Makes behavior explainable and testable across commands.

**Trade-offs accepted:**
- Slightly larger config surface area.
- Requires migration and compatibility shims.

**Implementation sequence:**
1. Add typed schema + validation + merge engine (no behavioral changes).
2. Migrate current consumers to read effective config from merge engine.
3. Deprecate legacy keys with warnings, then remove after migration window.

### Alternative Approaches Considered

**Option B: Keep current schema and add ad-hoc keys only as needed**
- **Pros:** Minimal immediate work.
- **Cons:** Continues precedence drift and hidden constants.
- **When to use instead:** Short-lived prototype only.

**Option C: Global config only, remove project overrides**
- **Pros:** Simpler mental model.
- **Cons:** Breaks repo-local autonomy and existing `.orch/config.yaml` patterns.
- **When to use instead:** Single-project personal workflow with no cross-project daemon usage.

**Rationale for recommendation:** Option A preserves existing architectural intent while fixing policy fragmentation.

---

## References

**Files Examined:**
- `pkg/userconfig/userconfig.go` - user-level schema/default getters.
- `pkg/config/config.go` - project-level schema/defaults.
- `cmd/orch/backend.go` - backend precedence.
- `cmd/orch/spawn_cmd.go` - model resolution precedence.
- `cmd/orch/spawn_usage.go` - tier precedence.
- `cmd/orch/spawn_pipeline.go` - spawn config assembly.
- `cmd/orch/daemon.go` - daemon flags and default policy.
- `cmd/orch/daemon_loop.go` - daemon config build path.
- `cmd/orch/serve_agents_collect.go` - dashboard state thresholds.
- `pkg/spawn/gap.go` - context gate threshold.
- `cmd/orch/complete_helpers.go` - rebuild timeout default.
- `cmd/orch/complete_actions.go` - transcript export timeout loop.

**Commands Run:**
```bash
orch phase orch-go-21426 Planning "Reviewing current orch config and designing layered schema"
kb create investigation review-orch-config-yaml-design
bd comment orch-go-21426 "investigation_path: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-07-inv-review-orch-config-yaml-design.md"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-07-orch-config-layered-schema.md` - recommended target schema and precedence contract.
- **Investigation:** `.kb/investigations/2026-01-26-inv-test-skill-models-config.md` - prior model selection behavior validation.
- **Workspace:** `.orch/workspace/og-arch-review-orch-config-07feb-f7f9/` - this spawn session workspace.

---

## Investigation History

**2026-02-07 16:10:** Investigation started
- Initial question: Review what should be configurable in `~/.orch/config.yaml` and define layering.
- Context: Spawn task `orch-go-21426` requested schema recommendation and override model.

**2026-02-07 16:25:** Codebase audit completed
- Confirmed current schema surface, hardcoded defaults, and precedence chains.

**2026-02-07 16:40:** Investigation completed
- Status: Complete.
- Key outcome: Proposed layered schema and precedence decision documented.

# Session Synthesis

**Agent:** og-arch-architect-design-enforcement-24feb-33f9
**Issue:** orch-go-1187
**Outcome:** success

---

## Plain-Language Summary

After two implementations (orch-go-1182/1183) were reverted because workers jumped from investigation findings to code without architect review, this design addresses the enforcement gap. The core problem: `--force-hotspot` is an unconditional bypass — anyone can override the hotspot gate with no proof that an architect reviewed the area. The design recommends three layers of enforcement: (1) make `--force-hotspot` require `--architect-ref <issue-id>` that verifies a closed architect issue exists, (2) make the daemon's skill inference escalate feature-impl to architect when issues target hotspot files, and (3) inject hotspot status into SPAWN_CONTEXT.md so investigation agents can recommend architect follow-up. Layer 1 alone prevents the exact failure mode that cost 3 spawn cycles and a regression.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 4 decision forks navigated with substrate traces, acceptance criteria defined, file targets listed for implementation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md` — Full architect design with 4 forks, recommendations, implementation plan
- `.kb/models/spawn-architecture/probes/2026-02-24-probe-architect-gate-hotspot-enforcement.md` — Probe confirming 3 enforcement gaps in spawn gate infrastructure
- `.orch/workspace/og-arch-architect-design-enforcement-24feb-33f9/VERIFICATION_SPEC.yaml` — Verification specification
- `.orch/workspace/og-arch-architect-design-enforcement-24feb-33f9/SYNTHESIS.md` — This file

---

## Evidence (What Was Observed)

- `pkg/spawn/gates/hotspot.go:59-63` — --force-hotspot is unconditional bypass, prints warning and returns nil error
- `pkg/spawn/gates/hotspot.go:50-53` — daemon-driven spawns return silently, never check HasCriticalHotspot
- `pkg/daemon/skill_inference.go:35-38` — feature/task always infer to feature-impl, no hotspot awareness
- `pkg/orch/extraction.go:375-379` — hotspot gate is last in pre-flight chain (correct ordering)
- orch-go-1184 probe confirmed: orch-go-1182 violated two-lane decision Invariants 6 and 7

---

## Knowledge (What Was Learned)

### Decisions Made
- Three-layer enforcement over single-layer: spawn gate (hard) + daemon routing (gap closure) + context injection (advisory)
- --architect-ref over removing --force-hotspot: preserves escape hatch per "Escape Hatches" principle
- Daemon hotspot awareness over trusting triage: triage checks issue validity, not architectural appropriateness

### Constraints Discovered
- Daemon-driven spawns silently skip the hotspot gate — a gap not covered by existing enforcement
- Investigation skill has no mechanism to detect hotspot context — spawn context injection needed

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation (3 phases)

**Phase 1 (highest priority):** Add --architect-ref requirement to --force-hotspot
- Files: pkg/spawn/gates/hotspot.go, cmd/orch/spawn_cmd.go, pkg/orch/extraction.go
- ~250 lines (gate logic + flag plumbing + tests)

**Phase 2:** Add hotspot check to daemon skill inference
- Files: pkg/daemon/skill_inference.go, cmd/orch/daemon.go
- ~160 lines (inference logic + integration + tests)

**Phase 3:** Inject hotspot status into SPAWN_CONTEXT.md
- Files: pkg/spawn/context.go, cmd/orch/spawn_cmd.go
- ~30 lines (context generation + template)

---

## Unexplored Questions

- Should --architect-ref verification be synchronous (bd show) or cached? bd show is a CLI call — adds latency to spawn.
- What about cross-project hotspot enforcement? Current hotspot detection is project-scoped.
- Should the orchestrator skill template be updated to reference the --architect-ref mechanism? (Advisory, not enforcement)

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-architect-design-enforcement-24feb-33f9/`
**Investigation:** `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md`
**Beads:** `bd show orch-go-1187`

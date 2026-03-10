# Model: Code Extraction Patterns

**Domain:** Architecture / Refactoring / Context Management
**Last Updated:** 2026-03-06
**Synthesized From:** 13 investigations (Jan 3-8, 2026) into Go (main.go, serve.go) and Svelte component extraction

---

## Summary (30 seconds)

Code extraction is the primary mechanism for **Context Management** in AI-orchestrated environments. Large files (>800 lines) create "Context Noise" that degrades agent performance and increases implementation risk. The system uses a **Phase-based Extraction strategy** (Shared Utilities → Domain Handlers → Sub-domain Infrastructure) to maintain "Cohesive Extraction Units" that fit within a single agent's cognitive window.

---

## Core Mechanism

### Extraction as Context Filtering

In an AI system, file size is a proxy for **Context Noise**. Extraction isn't just about "clean code"; it's about "Agent-Ready Code."

1. **Cohesive Extraction Units:** Identifying groups of functions, types, and helpers that share a single infrastructure substrate (e.g., `workspaceCache`).
2. **Package main Convenience:** Using `package main` for Go CLI commands allows file splitting without import cycles or visibility overhead, keeping implementation simple for agents.
3. **Barreled Component Isolation:** In Svelte, extraction follows a "Tabbed Pattern" where large panels are split into self-contained tab components with a single-prop interface (`agent: Agent`).

### The Extraction Hierarchy

| Phase | Target | Purpose | Constraint |
| :--- | :--- | :--- | :--- |
| **1. Shared** | `shared.go` | Break cross-dependencies | Must be extracted FIRST |
| **2. Domain** | `{name}_cmd.go` | Isolate CLI commands | One command per file |
| **3. Handler** | `serve_{name}.go` | Isolate HTTP logic | Domain-specific handlers only |
| **4. Sub-Domain** | `serve_{name}_cache.go` | Isolate infrastructure | Infrastructure vs Logic |

**Note — `pkg/` package extraction** (simpler than `cmd/orch/`): Within a flat package like `pkg/orch`, file splitting requires no import changes and has no circular dependency risk. Extraction domains can be parallelized after types are extracted first. However, when moving functions from `cmd/orch` to `pkg/orch`, leftover copies create silent divergent duplicates — always grep the source package for copies after extraction.

**Pipeline phase extraction pattern** (distinct from monolithic file extraction): When a file contains sequentially-composed pipeline phases (e.g., resolve → verify → advise → transition), extract by phase into separate files. Shared utilities aren't the first priority here — the phases themselves are the cohesive units. Advisory dispatcher functions in this pattern will have inherently high coupling cluster scores; this is structural, not pathological.

**Extraction emergency threshold:** High churn rate + high line count = extraction emergency, not just either alone. A 2.5x over-limit file (2011 lines) with 22 commits/28 days showing fix-on-fix patterns (fix→revert→fix→fix in git log) signals severe degradation. Partial extraction of 1-2 domains from a 9-domain monolith is insufficient — most domains must be extracted to get below the gate.

---

## Why This Matters

### The Verification Bottleneck

Large files are harder to verify. When an agent modifies a 5,000-line file, the "Verification Bottleneck" is hit:
- `git diff` becomes noisy
- `go build` takes longer
- Human review (Dylan) becomes exhausting

Extraction reduces the **surface area of change**, allowing for faster verification cycles.

### Session Amnesia Resilience

Smaller, cohesive files are more resilient to **Session Amnesia**. A new agent can quickly "comprehend" a 400-line command file, whereas a 4,000-line monolithic file requires extensive (and expensive) exploration that often leads to "Understanding Lag."

---

## Constraints

### Why Not Go Packages?

**Constraint:** We prefer splitting files within `package main` over creating new sub-packages for CLI commands.

**Reasoning:**
- Prevents circular import errors during rapid refactoring.
- Simplifies "Strategic Dogfooding" (agents can move code between files without updating 50 call sites).
- Minimizes "Boilerplate Noise" (no `Exported` vs `private` visibility hurdles).

### The Three-Number Framework (200 / 400 / 800)

**Constraint:** Three thresholds govern file size:
- **200 lines:** Ideal satellite size. Extraction should produce satellites of 100-300 lines.
- **400 lines:** Target maximum for residual files post-extraction. This is the POST-EXTRACTION goal.
- **800 lines:** Extraction trigger. Files crossing this threshold must be extracted to reach the 400-line target.

**Reasoning:** 800 lines is the heuristic limit where "Context Noise" begins to degrade agent reasoning. But empirical data (Mar 2026 probe, n=12 files) shows that extracting to "just under 800" fails — residuals left at 600-700 re-cross 800 within weeks. Residuals extracted to <400 lines resist re-accretion:
- Residuals at <400: doctor.go (269), extraction.go (280), session.go (121) — stable
- Residuals at 600-700: daemon.go (715→896), context.go (~600→895) — re-accreted past 800

**Key evidence:** Extracted satellite files (100-300 lines) show zero post-extraction commits across 9 files sampled. All new feature work lands in the residual parent, never satellites. This means: the more code moved to satellites, the more code resists re-accretion.

**Cross-cutting concern correlation:** Files <300 lines average 2.8 concerns; files >800 average 5.9 concerns. The concern accumulation threshold is ~300 lines — files below this maintain single responsibility.

**Previous formulation (superseded):** "Files should not exceed 800 lines." This remains true as a trigger but is insufficient as a target. The 800 line gate now triggers extraction TO 200-400, not just extraction BELOW 800.

---

## Evolution

### Jan 3-4, 2026: The "Great Split"
- Refactored `main.go` (4,900 lines) and `serve.go` (2,900 lines).
- Established the `shared.go` and `serve_agents_cache.go` patterns.
- Achieved ~40% line reduction in monolithic files.

### Jan 6-8, 2026: Frontend Tab Pattern
- Applied extraction to `agent-detail-panel.svelte`.
- Proved that tab-based component splitting reduces Svelte file size while maintaining reactivity.

### Feb 19, 2026: `pkg/orch/extraction.go` hotspot analysis
- At 2011 lines (2.5x gate), 9 cohesive extraction domains identified
- 22 commits/28 days with fix-on-fix pattern confirmed degradation signal
- Established: `pkg/` package extraction is simpler (no circular import risk) than `cmd/orch/` extraction

### Mar 1, 2026: Partial extraction insufficient; pipeline phase pattern established
- After extracting `spawn_modes.go` (530 lines) and `spawn_helpers.go` (148 lines), `extraction.go` remained at 1632 lines (2x gate)
- 7 remaining extraction domains mapped; complete extraction plan documented (spawn_types, spawn_inference, spawn_preflight, spawn_kb_context, spawn_backend, spawn_beads, spawn_design)
- `complete_pipeline.go` (970 lines) probe established "pipeline phase extraction" as a distinct pattern
- Advisory dispatcher fan-out (10+ callsites across 6+ files) is inherently high-coupling — structural, not pathological

### Mar 10, 2026: Three-Number Framework established (200/400/800)
- Empirical analysis of 13 extraction commits: residuals under 400 lines stay stable, residuals over 600 re-accrete
- Satellite files (100-300 lines) have zero post-extraction commits — all accretion hits the residual parent
- Cross-cutting concerns jump from 2.8 (files <300 lines) to 5.9 (files >800 lines)
- 76% of source files naturally cluster at 100-400 lines
- Phase 2 extraction target reframed: not "under 800" but "land at 200-400"
- See: `.kb/investigations/2026-03-10-design-determine-optimal-file-size-targets.md`

---

## Integration Points

- **Principles:** Supports **Session Amnesia** and **Verification Bottleneck**.
- **Guides:** Complements `.kb/guides/code-extraction-patterns.md` (procedural guidance).
- **Automation:** The 800-line gate informs "Hotspot Detection" in `orch learn`.

---

## References

**Key Investigations:**
- `2026-01-03-inv-extract-serve-agents-go-serve.md` (Domain extraction)
- `2026-01-03-inv-extract-shared-go-utility-functions.md` (Shared utilities first)
- `2026-01-04-inv-phase-extract-serve-agents-cache.md` (Infrastructure separation)
- `2026-01-06-inv-extract-synthesistab-component-part-orch.md` (Svelte tab pattern)

### Merged Probes

| Probe | Date | Key Finding |
|-------|------|-------------|
| `2026-02-19-probe-extraction-go-hotspot-analysis.md` | 2026-02-19 | extraction.go at 2.5x gate (2011 lines), 9 domains, 22 commits/28 days; pkg/ extraction simpler than cmd/orch/ (no circular import risk); high churn + high line count = extraction emergency |
| `2026-03-01-probe-extraction-go-self-hotspot.md` | 2026-03-01 | After partial extraction still at 1632 lines; 7 domains remain; complete 4-phase plan with target files; duplicate `isInfrastructureWork` in cmd/orch/spawn_cmd.go and pkg/orch (tech debt) |
| `2026-03-01-probe-complete-pipeline-extraction-boundaries.md` | 2026-03-01 | Pipeline phase extraction pattern established; `complete_pipeline.go` (970 lines) extracts to complete_verification.go + complete_lifecycle.go; advisory dispatcher inherently high-coupling |

**Primary Evidence (Verify These):**
- `cmd/orch/shared.go` - Shared utilities extraction (extracted first to break cross-dependencies)
- `cmd/orch/serve_agents.go` - Domain handler showing extraction from monolithic serve.go
- `cmd/orch/serve_agents_cache.go` - Sub-domain infrastructure extraction
- `cmd/orch/main.go` - Package main showing multiple file split within same package
- `web/src/lib/components/agent-detail/` - Svelte tab component extraction pattern

## Summary (D.E.K.N.)

**Delta:** All 4 orch-go coordination primitives (Route, Sequence, Throttle, Align) can be implemented as OpenClaw plugins without forking — the plugin SDK exposes the exact hooks needed.

**Evidence:** Direct examination of OpenClaw plugin SDK types (`src/plugins/types.ts` lines 1402-1978) confirms: `before_tool_call` receives file paths with `block` return for Route enforcement; `before_prompt_build` injects system prompt for Align; `subagent_spawning` rejects spawns for Throttle; `registerService()` runs background daemon for Sequence.

**Knowledge:** The mapping works because both systems follow the same pattern: observe lifecycle events, make coordination decisions, inject context or block actions. Skill protocols (phase reporting, SYNTHESIS.md, probes) are convention-dependent, not model-dependent — GPT 5.4 should handle them.

**Next:** Strategic decision for Dylan — if pursuing OpenClaw as distribution channel, build proof-of-concept `orch-coordination` plugin starting with Route + Align hooks.

**Authority:** strategic - Whether to pursue OpenClaw plugin distribution is a career/positioning decision, not an implementation choice.

---

# Investigation: Can orch-go Coordination Primitives Port to OpenClaw via Plugin SDK?

**Question:** Does OpenClaw's plugin SDK expose sufficient seams for orch-go's 4 coordination primitives (Route, Sequence, Throttle, Align), or does porting require a fork?

**Started:** 2026-03-23
**Updated:** 2026-03-23
**Owner:** orch-go-2oiy3
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md | extends | yes | none — confirms OpenClaw lacks coordination primitives; this investigation maps how to add them via plugin |

---

## Findings

### Finding 1: Plugin SDK exposes 27 lifecycle hooks — 5 directly map to coordination seams

**Evidence:** From `~/Documents/personal/clawdbot/src/plugins/types.ts` (lines 1402-1978):

| Hook | Orch-go Primitive | Coordination Capability |
|------|-------------------|------------------------|
| `before_tool_call` | **Route** (file-level) | Inspect tool params (file paths), return `block: true` to prevent writes to wrong regions |
| `before_prompt_build` | **Align** (skills + knowledge) | Inject `prependSystemContext` / `appendSystemContext` for SKILL.md and .kb/ context |
| `subagent_spawning` | **Throttle** (gates) | Return `{ status: "error" }` to reject spawns when thresholds exceeded |
| `subagent_ended` | **Sequence** (ordering) | Observe completion, trigger next agent |
| `registerService()` | **Sequence + Throttle** (daemon) | Long-running background service for priority queue and health tracking |

Critical detail: `PluginHookBeforeToolCallEvent.params` is `Record<string, unknown>` — file paths from write/edit tools are accessible. `PluginHookBeforeToolCallResult` has `block?: boolean` and `blockReason?: string`. A plugin can inspect every file write and enforce routing constraints at runtime.

**Source:** `src/plugins/types.ts:1721-1734` (before_tool_call types), `src/plugins/types.ts:1505-1532` (before_prompt_build types)

**Significance:** The SDK was not designed for coordination, but its hooks accidentally provide every seam needed. This is because coordination is fundamentally about: intercepting decisions (hooks), injecting context (prompt building), and controlling flow (spawn gating).

---

### Finding 2: Complete primitive-to-hook mapping — all 4 primitives are plugin-viable

**Evidence:** Systematic mapping of orch-go coordination code against OpenClaw plugin API:

**Route** (agents don't collide) — orch-go: `RouteIssueForSpawn()` in `pkg/daemon/coordination.go:40-101`
- `before_tool_call` → Block file writes outside assigned region (runtime enforcement)
- `before_prompt_build` → Inject structural placement constraints (attractor-based, proven in 329 trials)
- `subagent_spawning` → Route tasks to correct agent config based on metadata

**Sequence** (right order) — orch-go: `PrioritizeIssues()` in `pkg/daemon/coordination.go:107-137`
- `registerService()` → Background service manages priority queue (epic expansion, focus boost, scoring)
- `runtime.subagent.run()` → Programmatically spawn next agent when previous completes
- `subagent_ended` → React to completion events, advance queue

**Throttle** (velocity bounded) — orch-go: `CheckPreSpawnGates()` in `pkg/daemon/compliance.go:28-85`
- `subagent_spawning` → Reject spawns when: verification pause hit, completion health failing, comprehension queue full, rate cap exceeded
- `registerService()` → Track metrics (spawn count, completion failures, unreviewed work)

**Align** (shared understanding) — orch-go: `pkg/skills/loader.go` + `skills/src/worker/*/SKILL.md`
- `before_prompt_build` → Inject SKILL.md via `prependSystemContext` (cacheable) and .kb/ context via `prependContext` (per-turn)
- `registerTool()` → Provide governance tools (beads wrappers, kb queries)
- `before_tool_call` → Enforce governance (block commits violating criteria)

**Source:** Side-by-side comparison of `pkg/daemon/coordination.go`, `pkg/daemon/compliance.go`, `pkg/skills/loader.go` against `src/plugins/types.ts`

**Significance:** Zero primitives require core changes. The mapping is clean because orch-go's coordination primitives are already structured as observe→decide→inject/block, matching OpenClaw's hook pattern.

---

### Finding 3: Skill format is convention-portable, packaging needs thin adapter

**Evidence:** Compared orch-go SKILL.md format with OpenClaw skill format:

orch-go frontmatter: `name`, `skill-type`, `description`, `dependencies`
OpenClaw frontmatter: `name`, `description`, `metadata.openclaw.requires`, `metadata.openclaw.install`

What transfers directly (100% portable):
- Markdown body (stance, phases, deliverables) — just system prompt text
- Phase reporting (`bd comments add`) — convention, any model executes bash
- SYNTHESIS.md template — markdown convention
- Probe methodology — structured investigation workflow

What needs adaptation:
- Frontmatter schema differs (thin mapping layer)
- Dependency resolution (orch-go skillc concatenates; OpenClaw has no dependency chain — plugin handles)
- Delivery mechanism (orch-go: `~/.claude/skills/`; OpenClaw: `before_prompt_build` injection)

**Source:** `skills/src/worker/feature-impl/SKILL.md`, `~/Documents/personal/clawdbot/skills/canvas/SKILL.md`, `~/Documents/personal/clawdbot/skills/clawhub/SKILL.md`

**Significance:** The content is portable. Only packaging needs a thin adapter in the plugin.

---

### Finding 4: Fork is impractical — 267 commits/day, 42MB codebase

**Evidence:**
- 12,598 commits between Feb 6 - Mar 23 (~267 commits/day average)
- 42MB TypeScript in `/src/`
- 82 extension packages, each evolving independently
- Core agent runtime tightly coupled to pi-agent-core library

**Source:** `git log --since="2026-02-06" --oneline | wc -l` in `~/Documents/personal/clawdbot`

**Significance:** Fork maintenance would consume all available bandwidth. Plugin SDK is comprehensive enough — provides hooks for all coordination needs without touching core.

---

### Finding 5: GPT 5.4 skill protocol compatibility — convention-dependent, not model-dependent

**Evidence:** Skill protocols examined:
1. Phase reporting — bash command (`bd comments add`), any model can execute
2. SYNTHESIS.md — markdown template, any model can fill structured sections
3. Probe methodology — "What I Tested / What I Observed / Model Impact" — convention
4. D.E.K.N. format — structured summary, convention

Prior data: GPT-4o/GPT-5.2-codex had 67-87% stall rates on protocol-heavy skills (from CLAUDE.md gotchas). GPT 5.4 represents a major capability jump. The critical test: does the model reliably follow multi-step structured instructions? GPT 5.4 should pass this bar.

**Source:** CLAUDE.md gotchas section (non-Anthropic model stall rates), skill protocol analysis

**Significance:** Coordination methodology is model-agnostic. OpenClaw's multi-model support (Anthropic, OpenAI, Ollama, etc.) means the coordination plugin could work with any supported model, though Opus-class models likely outperform on hardest coordination tasks.

---

### Finding 6: Honest gaps — solvable within plugin boundary

| Gap | Why It's a Gap | Plugin Mitigation |
|-----|----------------|-------------------|
| Issue tracker | OpenClaw has no built-in tracker | `runtime.system.runCommandWithTimeout()` shells out to `bd` CLI |
| Cross-agent shared state | No inter-plugin message queues | Write to disk or external store; hooks are the coordination channel |
| Accretion measurement | No file complexity analysis | Plugin reads files via runtime APIs, runs analysis locally |
| Workspace isolation | Different pattern than `.orch/workspace/` | `before_prompt_build` injects workspace-specific context |

None require core changes.

---

## Synthesis

**Key Insights:**

1. **Hook-based coordination is sufficient because coordination IS observation + injection** — orch-go's OODA loop (Sense→Orient→Decide→Act) maps directly to OpenClaw's hook lifecycle. The primitives don't need to "control" agents — they need to observe events, make decisions, and inject constraints.

2. **Attractors are prompt injection, which the SDK was designed for** — The key coordination finding (329 trials) is that structural placement constraints work better than runtime gates. `before_prompt_build` with `prependSystemContext` is literally designed for injecting persistent agent guidance — exactly what attractors are.

3. **The coordination model is platform-independent** — This investigation confirms the prior investigation's conclusion: orch-go's coordination findings are methodology, not infrastructure. They can ride any platform that provides: (a) pre-spawn interception, (b) prompt injection, (c) tool call interception, (d) background services. OpenClaw provides all four.

**Answer to Investigation Question:**

Plugin approach is viable for 100% of the coordination primitives. No fork required. The mapping is clean, the SDK hooks are sufficient, and the skill format transfers with a thin adapter layer.

---

## Structured Uncertainty

**What's tested:**

- ✅ Hook signatures confirmed by reading actual TypeScript types in `src/plugins/types.ts`
- ✅ `before_tool_call` receives `params` with file paths and supports `block: true` return (verified: read type definitions)
- ✅ `before_prompt_build` supports `prependSystemContext` for cacheable injection (verified: read type definitions)
- ✅ `subagent_spawning` supports `{ status: "error" }` rejection (verified: read type definitions)
- ✅ Prior investigation confirmed OpenClaw lacks coordination primitives (verified: extends that investigation)

**What's untested:**

- ⚠️ Actual plugin implementation — no code written, only type-level mapping
- ⚠️ `before_tool_call` file path availability depends on tool naming conventions (e.g., `write_file` vs `edit`)
- ⚠️ Hook execution order with multiple plugins not verified
- ⚠️ `registerService()` background service reliability and lifecycle not tested
- ⚠️ GPT 5.4 protocol compliance is inferred from capability claims, not measured

**What would change this:**

- If `before_tool_call` params don't include file paths for OpenClaw's native edit tools → file-level Route enforcement would need alternative approach (prompt-only, no runtime enforcement)
- If hook execution latency is high → Throttle decisions could cause spawn timeouts
- If `registerService()` doesn't survive gateway restarts → Sequence daemon would lose queue state

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Build proof-of-concept `orch-coordination` plugin | strategic | Whether to invest in OpenClaw ecosystem is a positioning/career decision |
| Start with Route + Align hooks | implementation | These have the clearest mapping and can validate the approach |
| Defer issue tracker integration | implementation | External dependency (beads), can shell out initially |

### Recommended Approach: Proof-of-Concept Plugin

**Build a minimal `orch-coordination` plugin** that implements Route (file-level via `before_tool_call`) and Align (skill injection via `before_prompt_build`) to validate the mapping with real agents.

**Why this approach:**
- Validates type-level mapping with actual behavior
- Route + Align are the most impactful primitives (329-trial evidence)
- Minimal scope — doesn't require issue tracker or daemon

**Implementation sequence:**
1. Scaffold plugin with `definePluginEntry()` — register `before_tool_call` and `before_prompt_build` hooks
2. Implement Align — load SKILL.md files, inject via `prependSystemContext`
3. Implement Route — define file routing config, block writes outside assigned regions
4. Test with 2 concurrent subagents editing same file — verify conflict prevention

### Alternative Approaches Considered

**Option B: Publish coordination model as paper/blog only**
- **Pros:** No code maintenance, pure knowledge export
- **Cons:** Loses actionable distribution; coordination without tooling is just advice
- **When to use instead:** If OpenClaw plugin ecosystem doesn't gain traction

**Option C: Fork OpenClaw and embed coordination in core**
- **Pros:** Deepest integration, no hook limitations
- **Cons:** 267 commits/day merge burden, 42MB codebase — impractical
- **When to use instead:** Never, given current velocity

---

## References

**Files Examined:**
- `~/Documents/personal/clawdbot/src/plugins/types.ts` — Plugin API types, hook definitions (2000+ lines)
- `~/Documents/personal/clawdbot/src/plugins/runtime/types.ts` — PluginRuntime API
- `~/Documents/personal/clawdbot/src/plugin-sdk/plugin-entry.ts` — Plugin definition helper
- `~/Documents/personal/clawdbot/src/agents/subagent-spawn.ts` — Subagent spawn mechanism
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/coordination.go` — Route/Sequence primitives
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/compliance.go` — Throttle gates
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/ooda.go` — OODA loop
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go` — Skill loading/dependency resolution
- `/Users/dylanconlin/Documents/personal/orch-go/skills/src/worker/feature-impl/SKILL.md` — Example skill
- `~/Documents/personal/clawdbot/skills/canvas/SKILL.md` — OpenClaw skill example
- `~/Documents/personal/clawdbot/skills/clawhub/SKILL.md` — OpenClaw skill with requirements

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-openclaw-current-state-platform.md` — Prior OpenClaw platform analysis
- **Model:** `.kb/models/coordination/model.md` — Coordination model (329 trials)
- **Thread:** `.kb/threads/2026-03-23-constrain-shape-probe-three-layer.md` — Gates vs attractors framework

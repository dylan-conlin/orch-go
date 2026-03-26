<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch spawn` builds kb context before workspace creation, threads the formatted markdown through `orch.SpawnContext` and `spawn.Config`, then `GenerateContext()` injects it into the `{{.KBContext}}` slot of the worker template before `WriteContext()` writes `SPAWN_CONTEXT.md`.

**Evidence:** Code trace across `cmd/orch/spawn_cmd.go`, `pkg/orch/spawn_kb_context.go`, `pkg/spawn/kbcontext*.go`, `pkg/orch/spawn_pipeline.go`, `pkg/spawn/context.go`, and `pkg/spawn/atomic.go`, plus `go test ./pkg/spawn -run 'Test(GenerateContext_InvestigationDeliverableGating|GenerateContext_ProcessesSkillContentTemplates|WriteContext_FullTierCreatesSynthesisTemplate|FilterForScopedTask)$'` and `go test ./pkg/orch -run 'Test(.*Spawn.*|.*KB.*)'` both passed.

**Knowledge:** The query is derived from task keywords, optionally widened with orientation-frame keywords, local kb search runs first in the target project directory, formatting decides whether model content is injected, and only then does atomic spawn Phase 1 persist the composed context file.

**Next:** Close this investigation and use it as a reference trace when debugging spawn-context quality, model injection, or workspace-write behavior.

**Authority:** implementation - This is a descriptive trace of existing behavior and does not propose a cross-boundary architectural change.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Kb Context System Work Trace

TLDR: I traced the kb context path from `orch spawn` into the final workspace file. The system gathers and formats kb context before workspace creation, stores it on spawn structs, and injects it into `SPAWN_CONTEXT.md` via the worker template's `{{.KBContext}}` section.

**Question:** How does kb context move from `orch spawn` through context gathering and formatting into the final `SPAWN_CONTEXT.md` file that a worker reads?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** spawn-architecture

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - no related investigations were injected into SPAWN_CONTEXT for this run | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: `orch spawn` gathers kb context before it builds the worker config

**Evidence:** `cmd/orch/spawn_cmd.go` resolves settings, filters skill sections, then calls `orch.GatherSpawnContext(...)` and stores the returned `kbContext`, gap analysis, model-injection flags, and model paths on `orch.SpawnContext` before calling `BuildSpawnConfig(...)`.

**Source:** `cmd/orch/spawn_cmd.go:453`; `cmd/orch/spawn_cmd.go:480`; `pkg/orch/spawn_types.go:21`

**Significance:** The kb context system is part of spawn assembly, not a later post-processing step; once `BuildSpawnConfig` runs, the context payload is already computed and attached to the spawn state.

---

### Finding 2: Query derivation and kb formatting happen inside the pre-spawn context path

**Evidence:** `GatherSpawnContext()` either honors explicit skill requirements or runs `runPreSpawnKBCheckFull()`. That function derives keywords from `task` and optional `orientationFrame`, runs `RunKBContextCheckForDir()` with local-first then global fallback behavior, applies scoped-task filtering when needed, and formats the result through `FormatContextForSpawnWithLimitAndMeta()`. The formatter emits markdown headed by `## PRIOR KNOWLEDGE (from kb context)` and records whether model content was injected.

**Source:** `pkg/orch/spawn_kb_context.go:12`; `pkg/orch/spawn_kb_context.go:68`; `pkg/spawn/kbcontext_keywords.go:61`; `pkg/spawn/kbcontext.go:73`; `pkg/spawn/kbcontext_format.go:23`; `pkg/spawn/kbcontext_format.go:182`

**Significance:** This is the point where raw `kb context` CLI output becomes the structured markdown block that later appears verbatim in `SPAWN_CONTEXT.md`, including constraints, decisions, models, guides, and investigations.

---

### Finding 3: Template injection happens in `GenerateContext()` and file persistence happens in atomic spawn Phase 1

**Evidence:** `BuildSpawnConfig()` copies `KBContext`, `HasInjectedModels`, and related fields into `spawn.Config`. `GenerateContext()` places `cfg.KBContext` on template data, and `SpawnContextTemplate` renders it through `{{if .KBContext}}{{.KBContext}}{{end}}`. `AtomicSpawnPhase1()` then calls `WriteContext()`, which writes the rendered content to `<workspace>/SPAWN_CONTEXT.md`; `MinimalPrompt()` points the spawned worker at that exact file.

**Source:** `pkg/orch/spawn_pipeline.go:255`; `pkg/spawn/context.go:84`; `pkg/spawn/worker_template.go:80`; `pkg/spawn/atomic.go:18`; `pkg/spawn/context.go:233`; `pkg/spawn/context.go:332`

**Significance:** The final injection mechanism is plain template rendering plus file write; there is no extra transformation after `GenerateContext()`, so debugging final spawn context means checking the formatter output and the template slot.

---

## Synthesis

**Key Insights:**

1. **Context assembly is front-loaded** - By the time the spawn config is built, kb context has already been queried, analyzed, formatted, and classified for model injection.

2. **Formatting is the semantic boundary** - `FormatContextForSpawnWithLimitAndMeta()` is where raw kb results become the exact markdown section later embedded in `SPAWN_CONTEXT.md`.

3. **Persistence is atomic but simple** - The data path ends with ordinary template execution in `GenerateContext()` and a workspace write in `AtomicSpawnPhase1()`, which makes the trace straightforward once the pre-spawn pipeline is understood.

**Answer to Investigation Question:**

The kb context system works in three stages. First, `cmd/orch/spawn_cmd.go` calls `orch.GatherSpawnContext(...)`, which derives keywords from the task and optional orientation frame and runs the kb lookup pipeline. Second, `pkg/spawn/kbcontext*.go` parses and formats the results into markdown, optionally marking injected model content and scoped-task reductions. Third, `pkg/orch/spawn_pipeline.go` threads that formatted string into `spawn.Config`, `pkg/spawn/context.go` binds it to the template data as `KBContext`, and `WriteContext()` writes the rendered result into the workspace's `SPAWN_CONTEXT.md` during atomic spawn Phase 1.

---

## Structured Uncertainty

**What's tested:**

- ✅ Focused spawn tests passed for context generation, skill-content processing, full-tier workspace scaffolding, and scoped-task filtering via `go test ./pkg/spawn -run 'Test(GenerateContext_InvestigationDeliverableGating|GenerateContext_ProcessesSkillContentTemplates|WriteContext_FullTierCreatesSynthesisTemplate|FilterForScopedTask)$'`.
- ✅ Focused orch tests covering spawn- and kb-related behavior passed via `go test ./pkg/orch -run 'Test(.*Spawn.*|.*KB.*)'`.
- ✅ The live workspace used for this session contains a real `SPAWN_CONTEXT.md`, demonstrating the final artifact shape and confirming the minimal prompt targets that file.

**What's untested:**

- ⚠️ I did not run a full interactive `orch spawn` end-to-end session under a debugger; this trace relies on code reading plus targeted tests.
- ⚠️ I did not inspect runtime `kb` CLI stdout for this specific task outside the existing spawned workspace.
- ⚠️ I did not test cross-repo model injection behavior, only traced where the code handles it.

**What would change this:**

- A failing end-to-end spawn that bypasses `GatherSpawnContext()` or writes a different template slot would contradict this trace.
- A formatter path outside `FormatContextForSpawnWithLimitAndMeta()` producing the embedded markdown would revise the semantic-boundary finding.
- A future refactor that moves file persistence out of `AtomicSpawnPhase1()` or `WriteContext()` would change the final stage of the flow.

---

## Implementation Recommendations

No code change is recommended from this investigation. The useful output is a verified reference trace for future debugging.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Keep this investigation as reference material for spawn-context debugging and onboarding | implementation | It documents current behavior without changing architecture or product direction |

### Recommended Approach ⭐

**Reference trace only** - Use this investigation when debugging missing kb context, unexpected model injection, or malformed `SPAWN_CONTEXT.md` output.

**Why this approach:**
- It answers the original question without introducing unnecessary changes.
- It gives future debugging work a concrete handoff across the exact files involved.
- It preserves the distinction between investigation and implementation.

**Trade-offs accepted:**
- No product or infrastructure behavior changes are made.
- Any bug uncovered later will still require a follow-up issue.

**Implementation sequence:**
1. Read this trace to locate the failing stage.
2. Confirm whether the failure is in query derivation, formatting, or file write.
3. Open a focused follow-up issue if behavior diverges from this documented flow.

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - traced where spawn gathers kb context and constructs `orch.SpawnContext`
- `pkg/orch/spawn_kb_context.go` - traced the pre-spawn kb lookup and formatting path
- `pkg/spawn/kbcontext.go` - traced local/global kb query behavior and project-dir scoping
- `pkg/spawn/kbcontext_keywords.go` - traced keyword derivation from task title and orientation frame
- `pkg/spawn/kbcontext_format.go` - traced markdown formatting for injected prior knowledge
- `pkg/orch/spawn_pipeline.go` - traced handoff from `orch.SpawnContext` into `spawn.Config`
- `pkg/spawn/context.go` - traced template rendering, workspace write, and minimal prompt generation
- `pkg/spawn/worker_template.go` - confirmed the `{{.KBContext}}` injection point in `SPAWN_CONTEXT.md`
- `pkg/spawn/atomic.go` - confirmed atomic spawn Phase 1 writes the workspace before session start

**Commands Run:**
- `pwd`
- `kb create investigation kb-context-system-work-trace --model spawn-architecture`
- `go test ./pkg/spawn -run 'Test(GenerateContext_InvestigationDeliverableGating|GenerateContext_ProcessesSkillContentTemplates|WriteContext_FullTierCreatesSynthesisTemplate|FilterForScopedTask)$'`
- `go test ./pkg/orch -run 'Test(.*Spawn.*|.*KB.*)'`

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
<!-- All URLs must use markdown hyperlinks: [Display Name](https://url) — never bare URLs or plain text -->
- [Display Name](https://url) - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

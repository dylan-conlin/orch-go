<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Thread-to-model promotion requires a new `promoted` status, a `promoted_to` frontmatter field, a `orch thread promote` command that scaffolds model directories with provenance, and an orient integration that surfaces converged threads as "ready to promote."

**Evidence:** Analyzed 2 live test cases (generative-systems thread = model candidate, product-surface thread = decision candidate), thread lifecycle code (5 statuses, no promotion path), orient filtering (excludes all resolved/converged threads), and model template structure. The gap is structural — converged is terminal with no outward path.

**Knowledge:** Promotion is not always model-creation. Threads can mature into models, decisions, or principles. The command must support multiple target types. Provenance must flow bidirectionally (thread→artifact, artifact→thread lineage). Orient should surface converged threads without `promoted_to` as actionable, paralleling the comprehension:pending pattern.

**Next:** Implement in 3 phases: (1) thread package: add `StatusPromoted`, `PromotedTo` field, `Promote()` function; (2) command: `orch thread promote <slug> --as model|decision`; (3) orient: add `PromotionReady` section.

**Authority:** architectural - Cross-component design (thread pkg, orient pkg, command layer, model/decision conventions) requiring orchestrator synthesis

---

# Investigation: Design Thread-to-Model Promotion Lifecycle

**Question:** How should converged threads transition into durable artifacts (models, decisions, principles), and what command/data/display changes enable this?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** architect (orch-go-t37xi)
**Phase:** Complete
**Next Step:** Implementation issues created
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Decision: Models as Understanding Artifacts (2026-01-12) | extends | Yes — model template and provenance chain confirmed | None |
| Decision: Thread/Comprehension Layer Is Primary Product (2026-03-26) | extends | Yes — threads confirmed as primary organizing artifact | None |
| Guide: Understanding Artifact Lifecycle | extends | Confirms 3-phase model (session→epic→domain); promotion is a new pathway that bypasses the investigation-cluster route | Lifecycle guide says models need 15+ investigations; threads offer a parallel maturation path |
| Investigation: Refine Thread Home Surface Ordering (2026-03-26) | deepens | Confirmed orient thread rendering at `orient.go:368` | None |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Converged is a dead-end status

**Evidence:** `pkg/thread/lifecycle.go:14-19` — `IsResolved()` returns true for `converged`, `subsumed`, and `resolved`. Orient's `collectActiveThreads()` at `orient_cmd.go:476-478` calls `thread.ActiveThreads()` which filters out all `IsResolved()` threads. Once a thread reaches `converged`, it disappears from the thinking surface with no outward path.

**Source:** `pkg/thread/lifecycle.go:14-19`, `pkg/thread/thread.go:447-462`, `cmd/orch/orient_cmd.go:475-496`

**Significance:** The system recognizes that a thread has reached conclusion (`converged`) but provides no mechanism to extract value from that conclusion. This is the core gap — converged threads have done the hard thinking work but their output sits inert in a markdown file. The provenance chain (code → investigation → model → decision) has no thread entry point.

---

### Finding 2: Thread frontmatter has no promotion fields

**Evidence:** The `Thread` struct (`pkg/thread/thread.go:18-32`) defines: Title, Status, Created, Updated, ResolvedTo, SpawnedFrom, Spawned, ActiveWork, ResolvedBy. No `PromotedTo` field exists. The frontmatter parser handles these fields exclusively. `ResolvedTo` is the closest analog but semantically means "this thread was resolved by X," not "this thread became X."

**Source:** `pkg/thread/thread.go:18-32`

**Significance:** `ResolvedTo` could be overloaded for promotion, but this conflates two different relationships: (a) "resolved by an external artifact" (passive — something else answered this thread's question) vs. (b) "promoted into a new artifact" (active — this thread's thinking became the foundation of something new). These need distinct fields because the provenance semantics differ.

---

### Finding 3: Model creation has no thread-aware pathway

**Evidence:** The model template (`TEMPLATE.md`) has `Synthesized From:` listing investigations. There's no `Promoted From:` field. Models are created manually or via architect spawns that synthesize investigation clusters. The provenance chain (code → investigation → model → decision) documented in the "Models as Understanding Artifacts" decision has no thread entry point.

**Source:** `.kb/models/TEMPLATE.md`, `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md`

**Significance:** Threads represent a parallel maturation pathway to models. The lifecycle guide says models need 15+ investigations, but threads can reach model-ready maturity through direct thinking convergence. The generative-systems thread is a concrete example — it absorbed 3 threads and produced a clear meta-model candidate without any investigation intermediary.

---

### Finding 4: Two live test cases show different promotion targets

**Evidence:**
- **Generative-systems thread** (`status: forming`, but self-identifies as "candidate for meta-model"): Would promote to `.kb/models/named-incompleteness/model.md`. This is a model — it describes a mechanism that explains system behavior.
- **Product-surface thread** (`status: converged`): Would promote to either a decision (defining the 5-element product surface) or inform a model update (updating the orient-related model). This is a design claim with a concrete recommendation, not a mechanism description.

**Source:** `.kb/threads/2026-03-27-generative-systems-are-organized-around.md`, `.kb/threads/2026-03-27-product-surface-five-elements-not.md`

**Significance:** Promotion target is not always "model." The command must support at least: model (mechanism/understanding), decision (choice with rationale). Principle promotion (to `~/.kb/principles.md`) is possible but rare and can be deferred — principles are hand-curated by Dylan.

---

### Finding 5: Absorbed threads lose contribution credit

**Evidence:** When threads set `resolved_to` pointing at an absorbing thread, the absorbing thread accumulates their thinking. But if that absorbing thread later gets manually turned into a model, the absorbed threads' `resolved_to` still points at the absorbing thread, not the model. The contribution chain breaks: absorbed-thread → absorbing-thread → (manual creation) → model.

**Source:** `resolved_to` field in thread frontmatter; no back-reference system exists (write-only from thread side per exploration findings).

**Significance:** Promotion must update not just the promoted thread but all threads whose `resolved_to` points at it. Their `resolved_to` should be updated to point at the new artifact, preserving the full contribution lineage in the model's provenance section.

---

## Synthesis

**Key Insights:**

1. **Promotion is a new lifecycle transition, not a variant of resolution** — `converged` means "thinking is done"; `promoted` means "thinking became a durable artifact." These are semantically distinct. A thread can be converged indefinitely without promotion (if the thinking hasn't found its artifact form yet). Promotion is the deliberate act of structuring convergent thinking into the provenance chain.

2. **Thread maturation is a parallel path to investigation clusters** — The lifecycle guide's "15+ investigations → model" pathway is the bottom-up route (evidence accumulates, synthesis emerges). Thread promotion is the top-down route (thinking converges, evidence is pre-embedded in the thread entries). Both are valid; the system should support both without conflating them.

3. **Orient integration follows the comprehension:pending pattern** — Converged threads without `promoted_to` are analogous to unread briefs. They represent completed thinking that hasn't been externalized into the artifact system. Surfacing them in orient creates the feedback loop that prevents stale convergence.

**Answer to Investigation Question:**

The promotion lifecycle requires four changes:

1. **Thread package**: Add `StatusPromoted = "promoted"` to lifecycle constants. Add `PromotedTo` field to Thread struct. Add `Promote(threadsDir, slug, artifactType, targetPath string) error` function that: sets status to promoted, sets promoted_to, updates resolved_to on ancestor threads.

2. **Command**: `orch thread promote <slug> --as model|decision` that scaffolds the target artifact with provenance from the thread, then calls the package-level Promote function.

3. **Orient**: Add `PromotionReady []PromotionCandidate` to OrientationData, populated by scanning for converged threads without promoted_to. Render as "Ready to promote:" in FormatOrientation.

4. **Model/Decision templates**: Add `Promoted From:` provenance field listing the thread and its absorbed ancestors.

---

## Structured Uncertainty

**What's tested:**

- ✅ Thread lifecycle has 5 statuses with converged as terminal (verified: `lifecycle.go:4-10`)
- ✅ Orient excludes all resolved/converged threads (verified: `thread.go:447-462`, `orient_cmd.go:475-496`)
- ✅ Thread frontmatter has no promotion fields (verified: `thread.go:18-32`)
- ✅ Model template has no thread provenance path (verified: `TEMPLATE.md`)
- ✅ Two live test cases need different promotion targets (verified: read both thread files)
- ✅ `updateFrontmatter` and `updateFrontmatterQuoted` exist for field updates (verified: `thread.go:253,379`)

**What's untested:**

- ⚠️ Whether `updateFrontmatter` can add new fields (it only updates existing ones — promotion may need field insertion)
- ⚠️ Whether `orch thread promote` should be interactive in tmux or always explicit (user interaction model says orchestrator uses flags, but promotion might benefit from interactive artifact-type selection)
- ⚠️ Performance impact of scanning all threads for promotion-ready candidates in orient (thread count unknown — could be 10 or 500)

**What would change this:**

- If Dylan decides promotion should only target models (not decisions), the command simplifies
- If the lifecycle guide's 15+ investigation requirement is considered mandatory for models, thread-promoted models would need a different artifact type (e.g., "thesis" or "claim")
- If threads accumulate at a rate that makes orient scanning expensive, promotion-ready detection needs caching

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `StatusPromoted` to lifecycle constants | implementation | Extends existing pattern, stays within thread package |
| Add `PromotedTo` field to Thread struct | implementation | Single-scope field addition |
| Create `orch thread promote` command | architectural | Cross-component (thread pkg → command → model/decision template) |
| Add promotion-ready to orient | architectural | Cross-component (thread pkg → orient pkg → orient command) |
| Thread can promote to model OR decision | strategic | Defines what kinds of artifacts threads can become — shapes the knowledge system |

### Recommended Approach ⭐

**Promotion with multi-target scaffolding** — `orch thread promote <slug> --as model|decision` creates the target artifact with thread provenance, updates the thread and its ancestors, and surfaces unpromoted converged threads in orient.

**Why this approach:**
- Mirrors the principle "evolve by distinction" — promotion is distinct from resolution
- Supports both live test cases (generative-systems → model, product-surface → decision)
- Bidirectional provenance prevents contribution loss from absorbed threads
- Orient integration closes the feedback loop (converged threads don't silently accumulate)
- Probe-first claims bootstrap: promotion creates scaffold with the thread's core claim as initial thesis, probes fill in the claims table organically — aligned with the named-incompleteness principle itself

**Trade-offs accepted:**
- Adding a 6th thread status increases lifecycle complexity slightly
- `--as` flag requires the promoter to know the target type (acceptable: orchestrator makes this judgment)
- Model template gets a new optional field (`Promoted From:`) that most models won't use

**Implementation sequence:**

1. **Thread package changes** (foundational — everything else depends on this)
   - Add `StatusPromoted` constant to `lifecycle.go`
   - Update `IsResolved()` to include `promoted`
   - Add `PromotedTo` field to `Thread` struct
   - Add `PromotedTo` to frontmatter parser
   - Add `Promote()` function: updates status, sets promoted_to, propagates to ancestors
   - Add `PromotionReady()` function: returns converged threads without promoted_to

2. **Command: `orch thread promote`** (user-facing — uses package functions)
   - Add `promote` subcommand to thread command group
   - `--as model` (default): creates `.kb/models/{slug}/model.md` with provenance from thread, empty `probes/` dir
   - `--as decision`: creates `.kb/decisions/YYYY-MM-DD-{slug}.md` with thread content as context
   - Both paths call `thread.Promote()` after artifact creation
   - `--dry-run` flag for preview

3. **Orient integration** (visibility — surfaces promotion-ready threads)
   - Add `PromotionReady []PromotionCandidate` to `OrientationData`
   - Add `collectPromotionReady()` in orient_cmd.go calling `thread.PromotionReady()`
   - Add `formatPromotionReady()` renderer in orient.go
   - Place between active threads and briefs in FormatOrientation

### Alternative Approaches Considered

**Option B: Overload `resolved_to` for promotion**
- **Pros:** No new fields; simpler data model
- **Cons:** Conflates "resolved by" (passive) and "became" (active); loses ability to distinguish promoted threads from resolved ones; orient can't specifically surface promotion-ready threads
- **When to use instead:** If the thread count stays very small and the distinction between resolution and promotion proves unnecessary in practice

**Option C: Promotion creates a new thread type ("thesis") instead of a model**
- **Pros:** Avoids tension with the 15+ investigation model creation threshold
- **Cons:** Introduces a new artifact type that overlaps heavily with models; fragments the knowledge system; contradicts "Models as Understanding Artifacts" decision which is the authoritative pattern
- **When to use instead:** If thread-promoted models consistently lack the depth of investigation-cluster models and this causes problems

**Rationale for recommendation:** Option A keeps the existing artifact types (models, decisions) while adding a new pathway into them. It respects both the "Models as Understanding Artifacts" decision and the "Thread/Comprehension Layer Is Primary Product" decision by connecting threads into the provenance chain rather than creating parallel structures.

---

### Implementation Details

**What to implement first:**
- Thread package: `StatusPromoted`, `PromotedTo` field, `Promote()` function, `PromotionReady()` function
- These are the foundation; command and orient changes depend on them

**Things to watch out for:**
- ⚠️ Defect Class 3 (Stale Artifact Accumulation): Converged threads that never get promoted will accumulate. Orient integration mitigates this by making them visible. Consider a "stale converged" warning at 14+ days without promotion (parallel to stale model detection).
- ⚠️ Defect Class 5 (Contradictory Authority Signals): Thread `promoted_to` and model `Promoted From:` must stay consistent. The promote command should write both atomically (thread update + artifact creation in same operation).
- ⚠️ Defect Class 1 (Filter Amnesia): The new `StatusPromoted` must be added to `IsResolved()` so promoted threads don't appear as active. Also verify the HTTP API thread endpoint at `serve_threads.go` handles the new status.
- ⚠️ Frontmatter insertion: `updateFrontmatter()` only updates existing fields. If `promoted_to` doesn't exist in the thread file, a new `insertFrontmatter()` function or field pre-population may be needed.

**Areas needing further investigation:**
- Whether `orch compose` (digest creation) should include promotion-ready threads as a signal
- Whether the daemon should auto-spawn promotion work when converged threads accumulate
- How `kb context` searches should handle promoted threads (currently only searches models/decisions/investigations)

**Success criteria:**
- ✅ `orch thread promote generative-systems-organized-around --as model` creates `.kb/models/named-incompleteness/model.md` with provenance
- ✅ Thread status updates to `promoted` with `promoted_to` pointing at model path
- ✅ `orch orient` shows "Ready to promote: 1 converged thread" when unpromoted converged threads exist
- ✅ Absorbed threads' `resolved_to` updates to point at the new model
- ✅ `orch thread list` displays promoted threads with a distinct icon

---

## Composition Claims

| ID | Claim | Components Involved | How to Verify |
|----|-------|--------------------|----|
| CC-1 | "Promoted threads disappear from active threads but appear nowhere else confusingly" | lifecycle.go IsResolved + orient filtering + thread list display | `orch thread list` shows promoted threads with icon; `orch orient` does not show them as active or promotion-ready |
| CC-2 | "Provenance flows bidirectionally after promotion" | thread.Promote() + model scaffold template | After promoting, read model's `Promoted From:` section AND thread's `promoted_to` field — both exist and point at each other |
| CC-3 | "Absorbed thread ancestry chains update on promotion" | thread.Promote() ancestor propagation | Create 3 threads: A (subsumed→B), B (subsumed→C), C (converged). Promote C. Verify A and B `resolved_to` updated to model path |

---

## References

**Files Examined:**
- `pkg/thread/lifecycle.go` — Thread status constants and lifecycle predicates
- `pkg/thread/thread.go:18-32` — Thread struct definition
- `pkg/thread/thread.go:350-376` — UpdateStatus/Resolve functions
- `pkg/thread/thread.go:378-394` — Frontmatter update helpers
- `pkg/thread/thread.go:446-462` — ActiveThreads filtering function
- `cmd/orch/orient_cmd.go:56-96` — Orient data collection
- `cmd/orch/orient_cmd.go:475-496` — collectActiveThreads
- `pkg/orient/orient.go:39-45` — ActiveThread struct
- `pkg/orient/orient.go:118-146` — OrientationData struct
- `pkg/orient/orient.go:229-251` — FormatOrientation renderer
- `pkg/orient/orient.go:368-381` — formatActiveThreads renderer
- `.kb/models/TEMPLATE.md` — Model template structure
- `.kb/threads/2026-03-27-generative-systems-are-organized-around.md` — Live test case (model candidate)
- `.kb/threads/2026-03-27-product-surface-five-elements-not.md` — Live test case (decision candidate)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` — Established models as understanding artifacts with provenance chains
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Threads as primary organizing artifact
- **Guide:** `.kb/guides/understanding-artifact-lifecycle.md` — Three-phase lifecycle that promotion extends

---

## Investigation History

**2026-03-27:** Investigation started
- Initial question: How should converged threads transition into durable artifacts?
- Context: Spawned by orchestrator after observing two threads ready for promotion with no mechanism

**2026-03-27:** Codebase exploration complete
- Found 5 thread statuses, no promotion path, orient filters out converged
- Two live test cases confirm need for multi-target promotion (model vs decision)
- Thread frontmatter and model template both missing provenance fields

**2026-03-27:** Design complete with 7 forks navigated
- Recommended: multi-target promotion with bidirectional provenance
- 3-phase implementation: thread package → command → orient integration
- 3 composition claims for integration verification

**2026-03-27:** Investigation completed
- Status: Complete
- Key outcome: Promotion lifecycle designed with `StatusPromoted`, `orch thread promote --as model|decision`, and orient promotion-ready surface

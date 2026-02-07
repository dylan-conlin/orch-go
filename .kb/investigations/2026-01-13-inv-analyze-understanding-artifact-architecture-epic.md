<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The three artifacts (Epic Model template, Understanding sections, Models) are coherent lifecycle progression, not redundancy - they represent different temporal scopes of understanding (session → epic → domain).

**Evidence:** Epic Model 1-Page Brief (lines 97-115) has identical structure to Understanding section format; N=11 models created in 1 day validates Models serve distinct purpose; live epics (orch-go-4tven, orch-go-95vz4, orch-go-mg301) all have Understanding sections matching Epic Model Ready Gate questions.

**Knowledge:** Perceived redundancy stems from Epic Model bundling process + artifact + coordination (three concerns in one template), and lifecycle progression being implicit rather than explicit in documentation.

**Next:** Document the lifecycle explicitly (`.kb/guides/understanding-artifact-lifecycle.md`), update orchestrator skill with Epic readiness workflow, close Models decision open question (Epic Model unbundling deferred indefinitely).

**Promote to Decision:** recommend-no - This confirms existing architecture is coherent; no new architectural choice needed, just documentation of lifecycle.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Analyze Understanding Artifact Architecture Epic

**Question:** Are Epic Model template, Understanding sections (beads epics), and Models (.kb/models/) redundant manifestations of same forcing function pattern, or coherent distinct artifacts?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** architect agent (orch-go-r6mp5)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Three Distinct Temporal Phases in Understanding Progression

**Evidence:**

1. **Epic Model template** (`.orch/templates/epic-model.md`): Session-scoped working document
   - Phase awareness: Probing → Forming → Ready
   - "Where Am I?" table guides progression
   - "Session Log" for multi-session tracking
   - Contains the 1-Page Brief at Ready phase

2. **Understanding sections** (beads epic descriptions): Readiness gate at creation
   - Format: Problem / Why previous failed / Constraints / Risks / Done
   - Matches Epic Model's "Ready Gate" questions (lines 77-89 of epic-model.md)
   - Created when epic goes from Probing/Forming → Ready
   - Required via `bd create --type epic --understanding` (as of 2026-01-07)

3. **Models** (`.kb/models/`): Long-term queryable understanding
   - Structure: What This Is / How This Works / Why This Fails / Constraints / Evolution
   - Created from 15+ investigation clusters (validated N=11, Jan 2026)
   - Answers "enable/constrain" strategic questions
   - Example: `spawn-architecture.md` (synthesized 36 investigations)

**Source:**
- `.orch/templates/epic-model.md` (lines 1-151)
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` (lines 45-54, Epic readiness = model completeness)
- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` (full decision)
- `.kb/investigations/2026-01-07-inv-epic-readiness-gate-understanding-section.md` (implementation)
- Live epics: `orch-go-4tven`, `orch-go-95vz4`, `orch-go-mg301` (all have ## Understanding sections)

**Significance:** These three artifacts represent different **temporal scopes** of understanding, not redundant manifestations. Epic Model is session-scoped (working), Understanding is epic-scoped (gate), Models are domain-scoped (persistent).

---

### Finding 2: Epic Model Template Bundles Three Concerns

**Evidence:**

Epic Model template contains:
1. **Process scaffold** - How to approach complex problems (lines 1-151, esp. "Where Am I?", "Probes Sent")
2. **Understanding artifact** - The 1-Page Brief (lines 97-115)
3. **Work coordination** - Session logs, probe tracking (lines 118-140)

The Models decision (2026-01-12) noted this bundling explicitly:
> "The Epic Model template tried to bundle: Process scaffold (how to approach), Understanding artifact (the model), Work tracking (sessions/probes/tasks)" (lines 111-115)

And deferred action:
> "**Decision deferred:** How to handle Epic Model template. Will revisit after models prove themselves in practice." (line 121)

**Source:**
- `.orch/templates/epic-model.md` (structure analysis)
- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md:111-121`

**Significance:** Dylan's concern about "artifact proliferation" may stem from Epic Model bundling. The template tries to be process + artifact + coordination simultaneously. Models decision recognized this but deferred unbundling.

---

### Finding 3: Understanding Section Auto-Population is Already Happening

**Evidence:**

The "1-Page Brief" in Epic Model template (lines 97-115) has identical structure to Understanding sections in beads epics:

**Epic Model Ready Gate (lines 77-89):**
- What problem are we actually solving?
- Why did previous approaches fail?
- What are the key constraints?
- Where do the risks live?
- What does "done" look like?

**Understanding Section in Beads** (format from 2026-01-07 implementation):
- Problem: [what's broken]
- Previous approach: [what failed and why]
- Constraints: [what must we work within]
- Risks: [what could go wrong]
- Done: [what success looks like]

The Epic Model's "1-Page Brief" (lines 100-115) is literally the template for Understanding sections.

**Source:**
- `.orch/templates/epic-model.md:77-115`
- `.kb/investigations/2026-01-07-inv-epic-readiness-gate-understanding-section.md:47` (Understanding section format)
- Live epic examples (orch-go-4tven, orch-go-95vz4, orch-go-mg301)

**Significance:** Dylan's question "should Epic Model 1-page brief auto-populate Understanding section at Ready phase?" is already the design. The 1-Page Brief IS the Understanding section - they're the same artifact at different lifecycle stages (working doc → committed gate).

---

### Finding 4: Models Decision Left Epic Model Status Open

**Evidence:**

From Models decision (2026-01-12):
> "Now that models are distinct, we should re-evaluate this template:
> - Extract process guide → `.kb/guides/complex-problem-solving.md`
> - Extract understanding → `.kb/models/{domain}.md`
> - Simplify epics → beads issues that reference models
>
> **Decision deferred:** How to handle Epic Model template. Will revisit after models prove themselves in practice." (lines 117-121)

Open questions in Models decision (lines 181-186):
- "Should Epic Model template be split?" (line 182)

**Source:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md:117-121, 181-186`

**Significance:** The Epic Model template was recognized as potentially conflating concerns, but action was deferred pending evidence. We're now 1 day past that decision with N=11 models created - this investigation is the "revisit after models prove themselves" checkpoint.

---

## Synthesis

**Key Insights:**

1. **Coherent, Not Redundant** - The three artifacts are distinct temporal phases in understanding progression, not redundant manifestations:
   - Epic Model = **working document** (session-scoped, ephemeral)
   - Understanding section = **readiness gate** (epic-scoped, committed when Ready)
   - Models = **queryable knowledge** (domain-scoped, synthesized from investigations)

2. **Epic Model Bundles Process + Artifact** - The confusion stems from Epic Model template bundling three concerns:
   - Process guidance (How to approach complex problems - the "Probing → Forming → Ready" framework)
   - Understanding artifact (The 1-Page Brief that becomes the Understanding section)
   - Work coordination (Session logs, probe tracking)

   This bundling creates apparent redundancy when in fact it's a **progression**.

3. **Auto-Population Already Exists** - Dylan's question about "should Epic Model 1-page brief auto-populate Understanding section?" reveals the design is already correct but not explicit. The 1-Page Brief IS the Understanding section at the Ready phase - it's the same content at different lifecycle stages (working → committed).

4. **Models Validate the Architecture** - N=11 models created in 1 day (Jan 12-13) validates that Models serve a distinct purpose. They answer strategic "enable/constrain" questions that Understanding sections (point-in-time epic gates) and Epic Model working docs (session-scoped) cannot serve.

**Answer to Investigation Question:**

**These are coherent distinct artifacts, not redundancy.** The progression is:

```
Epic Model (working doc)
    ↓ (at Ready phase)
Understanding section (epic gate)
    ↓ (when epic completes + 15+ investigations exist)
Model (long-term queryable knowledge)
```

**The apparent redundancy comes from:**
1. Epic Model bundling process + artifact + coordination (three concerns in one template)
2. The lifecycle relationship not being explicit (1-Page Brief → Understanding section transition is implicit)

**The real question is:** Should Epic Model template be unbundled to make the progression clearer?

---

## Structured Uncertainty

**What's tested:**

- ✅ Epic Model template structure analyzed (verified: read `.orch/templates/epic-model.md`)
- ✅ Understanding section format matches Epic Model Ready Gate (verified: compared live epics orch-go-4tven, orch-go-95vz4, orch-go-mg301 against epic-model.md:77-115)
- ✅ Models exist and have distinct structure (verified: read `spawn-architecture.md`, checked 11 models in `.kb/models/`)
- ✅ Models decision deferred Epic Model decision (verified: `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md:117-121`)

**What's untested:**

- ⚠️ Whether unbundling Epic Model improves adoption (no usage data)
- ⚠️ Whether orchestrators actually use Epic Model template in practice (no observability of template usage)
- ⚠️ Whether explicit lifecycle documentation would reduce confusion (hypothesis, not validated)
- ⚠️ Impact of unbundling on epic workflow friction (would need A/B comparison)

**What would change this:**

- If Epic Model template is rarely used, unbundling would be premature optimization
- If orchestrators skip Understanding sections despite gates, problem is enforcement not architecture
- If Models cluster count stays <15 investigations, four-factor test needs revision

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Document the Lifecycle, Keep Current Architecture** - Explicitly document the Epic Model → Understanding section → Model progression without unbundling.

**Why this approach:**
- Architecture is coherent (Finding 1) - no redundancy exists, just implicit progression
- Models validated at N=11 in 1 day - the system works
- Auto-population concern is misframed (Finding 3) - 1-Page Brief IS the Understanding section, not a separate artifact needing population
- Unbundling would separate process from artifact, making the connection less obvious

**Trade-offs accepted:**
- Epic Model template remains "heavy" (bundles process + artifact + coordination)
- Progression stays implicit in documentation rather than explicit in tooling
- Understanding sections may feel redundant to those who filled out Epic Model working doc

**Implementation sequence:**
1. **Document lifecycle** - Create `.kb/guides/understanding-artifact-lifecycle.md` showing progression
2. **Update orchestrator skill** - Add explicit guidance: "When epic reaches Ready phase, copy 1-Page Brief to Understanding section"
3. **Update Models decision** - Close open question: Epic Model unbundling deferred indefinitely, architecture is coherent as-is

---

### Alternative Approaches Considered

**Option B: Unbundle Epic Model Template**
- **Pros:**
  - Separates process guide (how to probe) from artifact template (1-Page Brief format)
  - Each component simpler, single-purpose
  - Could reduce perceived redundancy
- **Cons:**
  - Breaks connection between process and artifact (Finding 2 - bundling is deliberate)
  - No evidence template is causing adoption problems (untested)
  - Would require three separate artifacts: process guide + brief template + epic coordination
  - Risk: orchestrators use process guide but skip artifact creation
- **When to use instead:** If evidence shows Epic Model template is rarely used BECAUSE it's too complex (not observed)

**Option C: Auto-Populate Understanding Section via Tooling**
- **Pros:**
  - Removes manual copy-paste step
  - Enforces lifecycle explicitly
- **Cons:**
  - Misframes the problem (Finding 3 - auto-population already exists conceptually)
  - Adds tooling complexity for marginal benefit
  - Understanding section requires JUDGMENT, not just copy-paste (orchestrator must confirm readiness)
  - Creates coupling between Epic Model working doc format and beads epic format
- **When to use instead:** If manual copy-paste is observed friction point (not tested)

**Rationale for recommendation:**

The investigation revealed no actual redundancy - just an implicit lifecycle progression. The "problem" is documentation, not architecture.

**Principle applied:** Coherence Over Patches (`.kb/principles.md`) - don't patch perceived problems before understanding root cause. Root cause here is implicit progression, not architectural redundancy.

---

### Implementation Details

**What to implement first:**
1. **Create lifecycle guide** (`.kb/guides/understanding-artifact-lifecycle.md`):
   - Document progression: Epic Model working doc → Understanding section → Model
   - Explain temporal scopes: session (working) → epic (gate) → domain (queryable)
   - Show example: Epic Model 1-Page Brief content becomes Understanding section verbatim

2. **Update orchestrator skill** - Add to strategic orchestrator section:
   ```markdown
   **Epic readiness workflow:**
   1. Use Epic Model template for probing → forming → ready progression
   2. When Ready phase reached, copy 1-Page Brief to epic Understanding section
   3. Use `bd create --type epic --understanding "..."` to create epic with gate
   ```

**Things to watch out for:**
- ⚠️ Epic Model template usage not observable - we assume it's used but don't track
- ⚠️ If Understanding sections are frequently skipped despite gates, problem is enforcement not architecture
- ⚠️ Models may drift from Understanding sections over time (Understanding is point-in-time, Models evolve)

**Areas needing further investigation:**
- Epic Model template adoption: Do orchestrators actually use it or skip it?
- Understanding section quality: Are they substantive or checkbox compliance?
- Model evolution frequency: How often do Evolution sections get updated after creation?

**Success criteria:**
- ✅ Fresh orchestrator can read lifecycle guide and understand progression
- ✅ No more "these feel redundant" questions (explicit documentation reduces confusion)
- ✅ Epic Model template usage either increases OR we get evidence it's not valuable (observability needed)

---

## References

**Files Examined:**
- `.orch/templates/epic-model.md` - Epic Model template structure and components
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Epic readiness = model completeness
- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Models as distinct artifact type
- `.kb/investigations/2026-01-07-inv-epic-readiness-gate-understanding-section.md` - Understanding section implementation
- `.kb/models/spawn-architecture.md` - Example model (synthesized 36 investigations)
- Beads epics: `orch-go-4tven`, `orch-go-95vz4`, `orch-go-mg301` - Live Understanding sections

**Commands Run:**
```bash
# List existing models
ls .kb/models/

# Find epics with Understanding sections
grep -r "## Understanding" .beads/

# Show live epic examples
bd show orch-go-4tven
bd show orch-go-95vz4
bd show orch-go-mg301
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Defines epic readiness as understanding completeness
- **Decision:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Models as understanding artifacts (deferred Epic Model decision)
- **Investigation:** `.kb/investigations/2026-01-07-inv-epic-readiness-gate-understanding-section.md` - Implementation of Understanding section gate
- **Template:** `.orch/templates/epic-model.md` - Working document for probing → forming → ready

---

## Investigation History

**2026-01-13 10:45:** Investigation started
- Initial question: Are Epic Model, Understanding sections, and Models redundant or coherent?
- Context: Dylan suspects artifact proliferation, wants analysis of redundancy vs coherent architecture

**2026-01-13 11:30:** Key finding - temporal progression
- Discovered three artifacts represent different temporal scopes (session → epic → domain)
- Not redundancy, but lifecycle progression

**2026-01-13 11:45:** Investigation completed
- Status: Complete
- Key outcome: Architecture is coherent; perceived redundancy stems from implicit lifecycle progression (documentation issue, not architectural redundancy)

# Understanding Artifact Lifecycle

**Purpose:** Document the lifecycle progression of understanding artifacts to make implicit temporal progression explicit and prevent perceived redundancy.

**Scope:** Applies to Epic Model templates (`.orch/templates/epic-model.md`), Understanding sections (beads epic descriptions), and Models (`.kb/models/`).

**Synthesized from:** Architect analysis (2026-01-13, orch-go-r6mp5)

---

## Quick Reference

| Artifact | Temporal Scope | When to Use | Lifespan |
|----------|---------------|-------------|----------|
| **Epic Model** | Session | Complex problem needs probing | Ephemeral (working doc) |
| **Understanding Section** | Epic | Creating epic, readiness gate | Epic lifetime |
| **Model** | Domain | 15+ investigations on topic | Long-term (evolving) |

**Promotion path:**
```
Epic Model 1-Page Brief → Understanding Section → Model
   (session working)    →    (epic gate)      → (domain knowledge)
```

---

## The Problem

**Before explicit lifecycle documentation:**
- Three artifacts (Epic Model, Understanding sections, Models) appeared redundant
- The progression between them was implicit, not documented
- Orchestrators questioned: "Why do I need all three?"
- Epic Model template bundles process + artifact + coordination, creating confusion

**Dylan's symptom:**
> "These feel like redundant manifestations of the same forcing function pattern. Why do we need Epic Model template AND Understanding sections AND Models?"

**Root cause:** Not redundancy, but **temporal progression** that was never made explicit in documentation.

**Principle:** Session Amnesia - artifacts must be self-documenting. Implicit progressions are invisible to fresh Claude instances.

---

## How It Works

### Three Temporal Phases of Understanding

Understanding artifacts progress through three distinct temporal scopes:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    UNDERSTANDING LIFECYCLE                          │
└─────────────────────────────────────────────────────────────────────┘
         SESSION SCOPE           EPIC SCOPE          DOMAIN SCOPE
              │                       │                    │
              ▼                       ▼                    ▼
    ┌─────────────────┐      ┌─────────────────┐   ┌──────────────┐
    │  Epic Model     │      │ Understanding   │   │   Model      │
    │  Template       │──────│    Section      │───│  (.kb/)      │
    │                 │      │                 │   │              │
    │ • Working doc   │      │ • Readiness     │   │ • Queryable  │
    │ • Probing→Ready │      │   gate          │   │ • Synthesized│
    │ • Session-bound │      │ • Epic-bound    │   │ • Long-term  │
    │ • Ephemeral     │      │ • Committed     │   │ • Evolving   │
    └─────────────────┘      └─────────────────┘   └──────────────┘
         (hours)                  (days-weeks)          (months+)

    When Ready?              When epic created      15+ investigations
         │                           │                    cluster?
         └───────> Copy 1-Page       │                    │
                   Brief ─────────────┘                    │
                                      └────────────────────┘
                                      Synthesize into Model
```

---

### Phase 1: Epic Model (Session-Scoped Working Document)

**Purpose:** Working document for probing complex problems when cause is unclear.

**Lifespan:** Single session or multi-session epic work (ephemeral).

**Structure:**
- **Process scaffold:** "Where Am I?" table (Probing → Forming → Ready)
- **Probe tracking:** What we tried, what we learned
- **1-Page Brief:** Understanding summary at Ready phase
- **Session logs:** Multi-session coordination

**When to use:**
- Problem exists but cause unclear (3+ spawns without resolution)
- Recurring frustration ("we keep trying things but nothing sticks")
- Dylan can't explain the problem in 1 paragraph
- Need structured approach to build understanding

**Completion criteria:** Reach Ready phase (can explain problem in 1 page).

**What happens at Ready:**
- The **1-Page Brief** becomes the **Understanding section** for the epic
- This is NOT a separate artifact - it's the same content transitioning from working doc to committed gate

**Example:** Epic Model for "spawn failures" epic guides probing across multiple investigations, tracking patterns, until Ready phase reveals root cause.

---

### Phase 2: Understanding Section (Epic-Scoped Gate)

**Purpose:** Readiness gate when creating epic - proves orchestrator understands the problem before spawning implementation work.

**Lifespan:** Epic lifetime (committed when epic created).

**Structure (matches Epic Model 1-Page Brief):**
```markdown
## Understanding

**Problem:** [what's broken]

**Previous approach:** [what failed and why]

**Constraints:** [what must we work within]

**Risks:** [what could go wrong]

**Done:** [what success looks like]
```

**When to use:**
- Creating epic via `bd create --type epic --understanding "..."`
- Epic has reached Ready phase (you can explain the problem)
- Before spawning implementation agents (prevents premature coding)

**Creation method:**
```bash
bd create --type epic \
  --title "Fix spawn reliability" \
  --understanding "Problem: spawn fails 30% of time when...
Previous: tried adding retries, didn't address root cause...
Constraints: must work with existing registry.json format...
Risks: race conditions in multi-agent spawns...
Done: 0 spawn failures across 100 test runs"
```

**Relationship to Epic Model:** The Understanding section IS the Epic Model's 1-Page Brief, copied verbatim at Ready phase.

**Why it's required:** Forces orchestrator to build mental model before creating epic. Prevents "fire and forget" epic creation where problem isn't understood.

---

### Phase 3: Model (Domain-Scoped Queryable Knowledge)

**Purpose:** Long-term queryable understanding synthesized from investigation clusters, enabling strategic questions to be answered quickly.

**Lifespan:** Long-term (months+), evolves via Evolution section.

**Structure:**
```markdown
# {Topic} Model

## What This Is
[Purpose, scope, problem this solves]

## How This Works
[Mechanisms, what this enables, state diagrams]

## Why This Fails
[Failure modes, what breaks and why]

## Constraints
[What this constrains/disallows, boundaries]

## Integration Points
[How this connects to other systems]

## Evolution
[Timestamped history of changes]
```

**When to create:**
- **Trigger:** 15+ investigations on single topic (investigation cluster detected)
- **Four-factor test (all required):**
  1. **HOT** - Cluster exists (15+ investigations)
  2. **COMPLEX** - Has failure modes, constraints, state transitions
  3. **OWNED** - Our system internals (not external tools)
  4. **STRATEGIC_VALUE** - "Enable/constrain" answers save hours vs minutes

**Creation method:**
```bash
# Spawn architect to synthesize cluster
orch spawn architect "Create model for spawn-architecture from investigation cluster"
```

**Example:** `spawn-architecture.md` model (synthesized 36 investigations) enables answering "What does spawn enable/constrain?" in <60s vs hours of reading investigations.

**Relationship to Understanding sections:** Models can reference Understanding sections from related epics, but serve different purpose:
- Understanding section = point-in-time readiness gate for ONE epic
- Model = long-term queryable knowledge synthesizing MANY investigations across multiple epics

---

## When to Use Each Artifact

### Decision Tree

```
Do I have a complex problem that needs structured probing?
│
├─ YES: Use Epic Model template
│   │
│   └─ Reached Ready phase (can explain in 1 page)?
│       │
│       ├─ YES: Copy 1-Page Brief → Understanding section, create epic
│       │
│       └─ NO: Continue probing
│
└─ NO: Do I have 15+ investigations on a topic?
    │
    ├─ YES: Check four-factor test → Create Model if all pass
    │
    └─ NO: Use standard investigations

Creating an epic right now?
│
└─ YES: Need Understanding section (via --understanding flag)
    Even if you didn't use Epic Model template,
    you must answer the 5 readiness questions
```

---

## Why All Three Are Needed

| Need | Which Artifact | Why Others Can't Serve |
|------|---------------|------------------------|
| **Work through complex problem** | Epic Model | Understanding/Models are outputs, not process guides |
| **Prove readiness before epic creation** | Understanding section | Epic Model is working doc (not committed), Models too broad |
| **Answer strategic "enable/constrain" questions** | Model | Understanding sections point-in-time, Epic Models ephemeral |
| **Session-level coordination** | Epic Model | Understanding/Models don't track probes or sessions |
| **Long-term queryable knowledge** | Model | Epic Models deleted after epic, Understanding sections buried in beads |

**Key insight:** These aren't redundant - they're the **same understanding at different lifecycle stages**.

---

## Promotion Path

### Epic Model → Understanding Section

**Trigger:** Epic Model reaches Ready phase.

**Process:**
1. Orchestrator completes Epic Model template, filling 1-Page Brief
2. Validates Ready gate questions (5/5 answered)
3. Copies 1-Page Brief content verbatim
4. Creates epic: `bd create --type epic --understanding "[paste 1-Page Brief]"`

**Not auto-populated because:** Understanding section requires JUDGMENT. Orchestrator must confirm they truly understand before creating epic. Auto-population would bypass the forcing function.

**Example:**
```markdown
# In Epic Model (working doc)
## 1-Page Brief (Ready Phase)

**Problem:** Spawn fails 30% of time when registry lock held by stale process

**Previous approach:** Added retries, but didn't address root cause (stale locks)

**Constraints:** Must preserve registry.json format for backward compat

**Risks:** Race condition if multiple spawns cleanup locks simultaneously

**Done:** 0 spawn failures across 100 concurrent spawn tests
```

→ Becomes Understanding section (committed):

```markdown
## Understanding

**Problem:** Spawn fails 30% of time when registry lock held by stale process

**Previous approach:** Added retries, but didn't address root cause (stale locks)

**Constraints:** Must preserve registry.json format for backward compat

**Risks:** Race condition if multiple spawns cleanup locks simultaneously

**Done:** 0 spawn failures across 100 concurrent spawn tests
```

---

### Understanding Section → Model

**Trigger:** Investigation cluster emerges (15+ investigations) on topic related to completed epic.

**Process:**
1. Orchestrator detects cluster via `kb reflect` (when available) or manual observation
2. Runs four-factor test (HOT, COMPLEX, OWNED, STRATEGIC_VALUE)
3. Spawns architect to synthesize: `orch spawn architect "Create model for {topic}"`
4. Architect reads Understanding sections from related epics + investigations
5. Synthesizes into Model with 6-section structure

**Relationship:** Understanding sections provide point-in-time context. Model synthesizes across multiple Understanding sections + investigations into queryable form.

**Example:** Three epics on spawn reliability (orch-go-abc, orch-go-def, orch-go-ghi), each with Understanding sections, plus 36 investigations → synthesized into `spawn-architecture.md` model.

---

### Model Evolution

**Trigger:** System behavior changes (new feature, constraint discovered, failure mode found).

**Process:**
1. Add timestamped entry to Evolution section
2. Update affected sections (How This Works, Why This Fails, Constraints)
3. Commit update

**Example:**
```markdown
## Evolution

### 2026-01-15: Added cleanup-on-exit behavior
- **What changed:** spawn now registers cleanup handler for SIGTERM
- **Why:** Stale locks were not cleaned up on graceful shutdown
- **Impact:** Constraints section updated - lock timeout no longer needed
```

**Why Evolution section matters:** Models are living documents. Evolution section shows what changed and why, preventing "why was this designed this way?" questions.

---

## Common Patterns

### Pattern: Epic Without Epic Model

**Scenario:** Orchestrator understands the problem immediately (cause is clear).

**What to do:** Skip Epic Model template, but still provide Understanding section when creating epic.

**Example:**
```bash
# No Epic Model needed - problem is obvious
bd create --type epic \
  --title "Add --json flag to orch status" \
  --understanding "Problem: orch status only outputs text, can't parse...
Previous: N/A (new feature)
Constraints: Must maintain backward compat with text output...
Risks: JSON schema changes break downstream tools...
Done: orch status --json outputs valid JSON schema v1"
```

**When valid:** Simple epics where orchestrator can answer 5 readiness questions without probing.

---

### Pattern: Model Without Epic

**Scenario:** Investigation cluster emerges, but never went through epic (many small bugs, not epic-level work).

**What to do:** Create Model directly from investigations via architect spawn.

**Example:** `tmux-architecture.md` model created from 20+ investigations on tmux integration, but no single epic existed.

**When valid:** Topic has enough investigations to warrant model (15+), but work was distributed across many small issues.

---

### Pattern: Epic Model Abandoned

**Scenario:** Started Epic Model, probing revealed problem is trivial or already solved.

**What to do:** Delete Epic Model working doc, close without creating epic.

**Why valid:** Epic Model is a working document, not a commitment. If probing reveals no epic needed, that's success (saved wasted effort).

---

## Anti-Patterns

### ❌ Creating Epic Without Understanding Section

**Symptom:** `bd create --type epic --title "Fix spawn"` (no --understanding flag)

**Why wrong:** Violates readiness gate - creating epic before understanding problem.

**Fix:** Either fill out Epic Model until Ready, or answer 5 readiness questions directly.

**Exception:** None. Understanding section is MANDATORY for epics (enforced by beads as of 2026-01-07).

---

### ❌ Creating Model Too Early

**Symptom:** 3 investigations on topic → "let's create a model"

**Why wrong:** Models are synthesis artifacts. 3 investigations don't provide enough perspective.

**Fix:** Wait for cluster threshold (15+) or use standard decision/guide instead.

**When to override:** Never below 10 investigations. Between 10-15, must pass four-factor test with clear justification.

---

### ❌ Treating Epic Model as Permanent Artifact

**Symptom:** Committing Epic Model working doc to git, referencing it in other docs.

**Why wrong:** Epic Model is ephemeral. Once epic created, the Understanding section is the permanent record.

**Fix:** Delete Epic Model after copying 1-Page Brief to Understanding section.

**Exception:** Multi-session epics may keep Epic Model in workspace until complete, but still delete after.

---

### ❌ Auto-Populating Understanding Section

**Symptom:** Tooling that copies Epic Model 1-Page Brief to epic without orchestrator review.

**Why wrong:** Bypasses the forcing function. Understanding section requires JUDGMENT - orchestrator must confirm readiness.

**Fix:** Manual copy-paste is deliberate friction. Keep it.

**Principle:** Gates > Reminders (`.kb/principles.md`) - auto-population turns a gate into a reminder.

---

## Troubleshooting

### "These still feel redundant"

**Check:** Are you confusing temporal phases with redundant artifacts?

- Epic Model is WORKING document (session-scoped)
- Understanding section is COMMITTED gate (epic-scoped)
- Model is QUERYABLE knowledge (domain-scoped)

**They're the same understanding at different lifecycle stages.**

---

### "When should I create a Model vs just use decisions?"

**Decision tree:**

| Criterion | Model | Decision |
|-----------|-------|----------|
| Investigation count | 15+ | 1-5 |
| Temporal scope | Long-term (months+) | Point-in-time |
| Query type | "Enable/constrain?" | "Why chose X?" |
| Complexity | Failure modes, state, constraints | Trade-off comparison |
| Evolution | Frequent (Evolution section) | Rare (create new decision) |

**Rule of thumb:** If you're asking "how does X work?" → Model. If asking "why did we choose X?" → Decision.

---

### "Should I unbundle Epic Model template?"

**Answer:** No (deferred indefinitely as of 2026-01-13).

**Why:** Epic Model deliberately bundles process + artifact + coordination. Separating them breaks the connection between "how to probe" and "what to produce."

**Evidence:** N=11 models created in 1 day validates architecture is coherent. Problem was implicit lifecycle, not architectural redundancy.

**When to revisit:** If Epic Model template is rarely used BECAUSE it's too complex (no evidence of this yet).

---

## References

**Source investigation:** `.kb/investigations/2026-01-13-inv-analyze-understanding-artifact-architecture-epic.md`

**Related decisions:**
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Epic readiness = model completeness
- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Models as distinct artifact type

**Related guides:**
- `.kb/guides/spawned-orchestrator-pattern.md` - When to use Epic Model template

**Templates:**
- `.orch/templates/epic-model.md` - Epic Model template
- `~/.kb/templates/model.md` - Model template (via `kb create model`)

---

## Changelog

**2026-01-13:** Guide created
- Synthesized from architect analysis (orch-go-r6mp5)
- Documents lifecycle progression to prevent perceived redundancy
- Establishes Epic Model → Understanding → Model promotion path

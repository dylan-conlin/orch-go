# Lens Sessions

**Purpose:** Generate strategic questions and actionable work from existing knowledge artifacts by applying structured lenses. Each lens is a repeatable session-starter that works without context from previous sessions.

**Last verified:** 2026-02-09

**Primary audience:** Orchestrator sessions, meta-orchestration, strategic reviews.

---

## Quick Start

Pick a lens from the table. Read the artifacts it targets. Ask the question. The combination of documented understanding + specific lens produces questions you couldn't ask without both.

```
Documented understanding + Specific lens = Novel questions you couldn't ask without both
```

**Why this works:** Artifacts are fuel, the lens is ignition. A fresh Claude with zero session history can read models, apply a lens, and generate immediately actionable strategic questions. Amnesia-proof by design.

---

## The Lenses

| When you want to... | Read... | Ask... | Typical yield |
|---------------------|---------|--------|---------------|
| Find strategic gaps | Models (`.kb/models/`) | "What questions are begging to be asked?" | 5-10 strategic questions, probes, design sessions |
| Check system integrity | Principles (`.kb/principles.md`) | "Where is the system violating its own principles?" | Gaps between intent and reality |
| Detect stale assumptions | Decisions (`.kb/decisions/`) | "Has the context that made this decision correct changed?" | Stale decisions to revisit or retire |
| Find model blind spots | Probes grouped by model | "What keeps extending the same model?" | Blind spots where models are incomplete |
| Map solution boundaries | Failed attempts (`kb quick list --type tried`) | "What's actually viable given these walls?" | Bounded solution space, viable approaches |
| Find knowledge debt | Orphaned investigations | "Where do they cluster?" | Knowledge debt clusters needing synthesis |
| Find contradictions | Overlapping models | "Where do they disagree?" | Diverged understanding needing reconciliation |
| Find design opportunities | Constraints (`kb quick list --type constrain`) | "Which pairs are in tension?" | Design opportunities from conflicting requirements |

---

## Lens Details

### 1. Surface Area for Questions (Models)

**Read:** All models in `.kb/models/` (focus on Summary, Constraints, Failure sections)

**Ask:** "What questions are begging to be asked? What's implicit that should be explicit? What does the system NOT model?"

**How it works:** Models make constraints explicit. Explicit constraints reveal gaps between what's modeled and what's real. Each gap is a question.

**Proven yield:** Feb 9 2026 session generated 10 strategic questions from 20 models, spawned 4 probes, created 3 design questions, led to a 4-agent implementation batch that closes the knowledge loop. Total session value: identified that 77% of investigations are orphaned and designed the fix.

**Session protocol:**
```bash
# Read the model index
cat .kb/models/README.md

# Read each model (focus on Constraints and Why This Fails)
# For each: what's implied but not stated?
# For each: what would break if this model is wrong?
```

---

### 2. Principle Violations (Principles)

**Read:** `.kb/principles.md`

**Ask:** "For each principle, find one place the system violates it."

**How it works:** Principles are documented values with teeth. Reality drifts from values. The drift IS the finding.

**Example:** "Track Actions Not Just State" predicted the episodic memory gap before any probe confirmed it. The principle was already right — nobody had tested it against the system.

**Session protocol:**
```bash
cat .kb/principles.md
# For each principle with "Teeth" section:
# Does the system actually enforce this?
# Where is the gap between principle and practice?
```

---

### 3. Stale Context Detection (Decisions)

**Read:** Recent decisions in `.kb/decisions/` (focus on Context sections)

**Ask:** "Is the context section still true? What changed since this was decided?"

**How it works:** Every decision documents WHY it was made. When the why changes, the decision may be wrong. Stale decisions cause downstream drift.

**Example:** Cost economics model was based on Anthropic credits. System switched to GPT-5.3. The decision's context is stale — the entire token-economics framework may be irrelevant for workers.

**Session protocol:**
```bash
# Read 10 most recent decisions
ls -t .kb/decisions/ | head -10
# For each: read Context section
# Ask: has this context changed?
# If yes: is the decision still correct given new context?
```

---

### 4. Blind Spot Detection (Probes)

**Read:** Probes grouped by model (`.kb/models/*/probes/`)

**Ask:** "Which models have 3+ 'extends' probes? What are they all pointing at?"

**How it works:** When multiple probes extend a model in the same direction, the model has a blind spot. The probes are the system trying to tell you the model is incomplete.

**Example:** Completion-verification model received 3 extending probes about bypass behavior — the model wasn't capturing the friction pattern that the probes kept finding.

**Session protocol:**
```bash
# Count probes per model
for dir in .kb/models/*/probes; do
  count=$(ls "$dir" 2>/dev/null | wc -l)
  [ "$count" -gt 0 ] && echo "$count $(dirname $dir | xargs basename)"
done | sort -rn

# For top models: read probes, look for directional pattern
# Multiple "extends" in same area = blind spot
```

---

### 5. Solution Space Mapping (Failed Attempts)

**Read:** All `kn tried` entries for a domain

**Ask:** "What approaches are actually viable given these walls?"

**How it works:** Failed attempts document what DOESN'T work and why. Read them together and they map the boundaries of the solution space. The viable path is what remains.

**Session protocol:**
```bash
kb quick list --type tried
# Group by domain
# For each cluster: what do the failures have in common?
# What approaches remain that avoid ALL the failure reasons?
```

---

### 6. Knowledge Debt Mapping (Orphaned Investigations)

**Read:** Investigations not referenced by any model, decision, or guide

**Ask:** "Do they cluster by topic? Which cluster is largest?"

**How it works:** Orphaned investigations are understanding that was produced but never integrated. Clusters of orphans reveal domains where the system kept trying to understand something but never synthesized.

**Example:** 77% of investigations (627/816) were unreferenced. If 200 of those are about completion lifecycle, that's not random waste — it's a knowledge debt cluster.

**Session protocol:**
```bash
# This needs tooling (or an investigation agent) to:
# 1. List all investigations
# 2. Check which are referenced by models/decisions/guides
# 3. Group unreferenced ones by topic
# 4. Identify largest clusters
```

---

### 7. Contradiction Detection (Cross-Model)

**Read:** Two models that describe overlapping territory

**Ask:** "Where do they disagree?"

**How it works:** Models are written independently at different times. Independent descriptions of the same thing naturally diverge. Contradictions reveal where understanding split.

**Example:** The daemon model's skill inference claim contradicted what the skill-inference-mapping probe found in actual code. Labels weren't ignored — they were partially wired.

**Session protocol:**
```bash
# Identify model pairs with overlap
# Read both models' Claims/Constraints sections side by side
# Where do they make different claims about the same thing?
# Contradiction = one model is stale, or the system is inconsistent
```

---

### 8. Tension Detection (Constraints)

**Read:** All constraints

**Ask:** "Which pairs pull in opposite directions?"

**How it works:** Constraints accumulate over time from different contexts. Some inevitably conflict. Tensions aren't bugs — they're design opportunities where the system is trying to serve two masters.

**Example:** "All spawns must be tracked" vs "investigation artifacts are self-tracking." Both are reasonable. The tension reveals a question: should tracking be about work-graph visibility (beads) or artifact discoverability (kb)?

**Session protocol:**
```bash
kb quick list --type constrain
# Read all constraints
# Look for pairs where satisfying one makes the other harder
# Each tension is a design question worth surfacing
```

---

## Combining Lenses

Lenses compound. This session started with Lens 1 (surface area for questions), which revealed the 77% orphan stat, which is a Lens 6 finding (knowledge debt), which led to checking Lens 2 (principle violations — "Track Actions Not Just State"), which confirmed the gap.

**Natural progressions:**
- Surface Area → Principle Violations (questions reveal where principles are violated)
- Stale Decisions → Contradiction Detection (stale decision + current model = contradiction)
- Blind Spots → Knowledge Debt (model blind spots often correlate with orphaned investigation clusters)
- Tension Detection → Design Sessions (tensions become architect or design-session spawns)

---

## When to Run a Lens Session

- **Session start with no clear task** — Pick the lens that matches your energy
- **After major completion batch** — Surface Area or Blind Spot to find what emerged
- **Monthly maintenance** — Stale Decisions + Principle Violations
- **Before strategic planning** — Surface Area + Solution Space Mapping
- **After investigation spikes** — Knowledge Debt Mapping

---

## Origin

Discovered Feb 9 2026 during an orchestrator session that read all 20+ models and asked "what questions are begging to be asked?" The session generated 10 strategic questions, identified the broken knowledge loop (77% orphaned investigations), and produced a 4-agent implementation batch to fix it. The meta-insight — that this process itself is a repeatable pattern — led to this guide.

The core idea comes from `.kb/models/README.md`: *"Models create surface area for questions by making implicit constraints explicit."*

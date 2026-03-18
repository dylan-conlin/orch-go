# Probe: Scoped Annotation Loop Pattern — Context Hub vs orch-go kb quick

**Model:** context-injection
**Date:** 2026-03-18
**Status:** Complete

---

## Question

Does the Context Injection Architecture model account for **scoped, point-of-use knowledge surfacing** — the pattern where annotations tied to specific resources (files, packages, skills) auto-surface when those resources are accessed? Context Hub (andrewyng/context-hub) implements this; does orch-go's kb quick pattern have an auto-surfacing gap that this pattern could fill?

---

## What I Tested

### 1. Context Hub Source Code Analysis (via GitHub API)

Examined Context Hub's annotation system implementation across:
- `cli/src/lib/annotations.js` — storage layer
- `cli/src/commands/get.js` — auto-surfacing mechanism
- `cli/src/mcp/tools.js` — MCP tool exposure
- `cli/src/lib/telemetry.js` — feedback loop

### 2. orch-go Knowledge Surfacing Mechanisms

Read and analyzed:
- `pkg/spawn/gap.go` — gap detection at spawn time (quality scoring, gap type classification)
- `pkg/spawn/learning.go` — recurring gap tracking and learning suggestions
- `cmd/orch/knowledge_maintenance.go` — completion-time knowledge review
- `pkg/spawn/kbcontext.go` — KB context query and injection into SPAWN_CONTEXT.md
- `.kb/decisions/2026-02-25-auto-memory-kb-cli-reconciliation.md` — lane definitions for auto-memory vs kb-cli

### 3. Architectural Comparison

Mapped both systems' data models, scoping mechanisms, surfacing triggers, and lifecycle management.

---

## What I Observed

### Context Hub Annotation Architecture

**Storage:** One JSON file per annotated entry in `~/.chub/annotations/`. Schema is minimal: `{ id, note, updatedAt }`. Entry ID (e.g., `stripe/api`) is filename-encoded (`stripe--api.json`). No database, no index.

**Auto-Surfacing Mechanism (the key insight):** When `chub get <id>` fetches a doc, the code does:
```javascript
const annotation = readAnnotation(r.id);
if (annotation) {
  process.stdout.write(`\n\n---\n[Agent note — ${annotation.updatedAt}]\n${annotation.note}\n`);
}
```
Annotations are injected at **read-time**, not stored with doc content. The surfacing is **automatic and unconditional** — if an annotation exists for the entry being fetched, it always appears. No relevance scoring, no search, no keyword matching.

**Feedback Loop:** Separate from annotations. `chub feedback <id> up|down` sends data to a remote API (`api.aichub.org`). Fire-and-forget. Does NOT affect local retrieval or future `chub get` output. It's a signal to doc maintainers, not a local knowledge system.

**Key Properties:**
| Property | Value |
|----------|-------|
| Scoping | Per entry ID (one note per doc/skill) |
| Surfacing trigger | `chub get <id>` (read-time injection) |
| Retrieval effort | Zero — annotation appears automatically |
| Lifecycle | Permanent until manually cleared |
| Search/indexing | None — annotations are not searchable |
| Granularity | Single free-text note per entry (overwrites) |

### orch-go kb quick Architecture

**Storage:** External `kb` CLI stores entries with richer schema: `{ id, type, content, status, created_at, reason, ref_count }`. Four types: decide, constraint, attempt/tried, question.

**Surfacing Mechanisms (three touchpoints):**

1. **Completion Review** (`knowledge_maintenance.go`): Extracts keywords from skill name, issue description, Phase: Complete summary. Scores entries by keyword matches in content (weight 2) and reason (weight 1). Surfaces top 5 entries for interactive review. **This is keyword-based, not scope-based.**

2. **Spawn Context** (`kbcontext.go`): Runs `kb context "<query>"` which does full-text search across all kb artifacts. Results are categorized (constraints, decisions, investigations, guides) and injected into SPAWN_CONTEXT.md. Quick entries surface if their content matches the query. **This is query-based, not scope-based.**

3. **Daemon Health** (`knowledge_health.go`): Counts active entries by type. Creates triage issue when threshold exceeded. **This is accumulation monitoring, not surfacing.**

**Key Properties:**
| Property | Value |
|----------|-------|
| Scoping | Global (not tied to specific resources) |
| Surfacing trigger | Keyword matching (completion) or full-text search (spawn) |
| Retrieval effort | Non-zero — requires keyword or query match |
| Lifecycle | Active → promoted (formal decision) or obsolete |
| Search/indexing | Full-text search via `kb context` |
| Granularity | Multiple entries per topic, each with metadata |

### The Architectural Gap

**orch-go has three knowledge surfacing mechanisms, none of which implement scoped auto-surfacing:**

| Mechanism | Trigger | Scope | Precision |
|-----------|---------|-------|-----------|
| `kb context` at spawn | Query keywords derived from task | Global pool | Medium (false positives from generic terms) |
| `knowledge_maintenance` at completion | Keyword extraction from skill/issue/summary | Global pool | Medium (top-5 keyword match) |
| `gap.go` at spawn | Match count thresholds | N/A (detects absence) | N/A |

Context Hub's pattern fills a gap none of these cover: **knowledge that is tied to a specific resource and surfaces automatically when that resource is accessed**.

### Concrete Example of the Gap

When spawning an agent to work on `pkg/spawn/`, orch-go runs `kb context "spawn"` and gets keyword matches from the global pool. This may return:
- Relevant: spawn architecture decisions, spawn guide
- Irrelevant: entries mentioning "spawn" in unrelated contexts

What it **cannot** do: surface an annotation like "pkg/spawn/gap.go: The gap gate threshold was lowered from 30 to 20 after false positive analysis in Feb 2026" that was discovered during a previous session working on that exact file. This annotation would need to be a global kb quick entry and match by keyword — or be lost.

### The Generalizable Pattern

**"Scoped annotations at point-of-use" vs "global knowledge via search" represents a fundamental retrieval tradeoff:**

| Dimension | Scoped (Context Hub) | Global (kb quick) |
|-----------|---------------------|-------------------|
| Precision | High (exact scope match) | Medium (keyword/query match) |
| Recall | Low (only surfaces at matching scope) | Higher (surfaces across contexts if keywords match) |
| Retrieval cost | Zero (auto-injected) | Non-zero (requires search hit) |
| Best for | Point-specific learnings | Cross-cutting concerns |
| Failure mode | Knowledge siloed to scope | Knowledge drowns in global pool |
| Accumulation behavior | Self-limiting (one per scope) | Unbounded (50+ entries needing triage) |

**Both patterns are necessary. They serve different knowledge types:**

| Knowledge Type | Best Pattern | Example |
|----------------|-------------|---------|
| File-specific gotcha | Scoped | "This file uses unusual error handling pattern because of X" |
| Package-level constraint | Scoped | "pkg/spawn race condition under concurrent calls" |
| Skill-level learning | Scoped | "investigation skill agents often skip prior work check" |
| Architectural decision | Global | "Claude CLI chosen over OpenCode for crash resistance" |
| Cross-cutting principle | Global | "Pressure over compensation" |
| Operational quirk | Could be either | Depends on whether it's resource-specific or general |

---

## Model Impact

- [x] **Extends** model with: The Context Injection Architecture model describes two injection paths (SessionStart hooks for orchestrator, SPAWN_CONTEXT.md for workers) but both use **global retrieval** (query-based kb context, skill loading). Neither path supports scoped annotations tied to specific resources. This is an architectural gap, not a bug — the system was designed around cross-cutting knowledge (constraints, decisions, guides) rather than point-specific learnings. The Context Hub annotation pattern demonstrates a third retrieval primitive (scoped auto-surfacing) that could complement existing global retrieval without replacing it.

- [x] **Confirms** invariant: "SPAWN_CONTEXT.md as authority" — Context Hub's approach validates this principle. SPAWN_CONTEXT.md is the right injection point for scoped annotations, just as it is for global kb context. The annotations would be merged at generation time, exactly as kb context results are today.

- [x] **Extends** model with: The model's "Pressure Over Compensation" constraint (§4.3) says "If an agent lacks context, do not manually paste it. Let the failure surface the gap, then update the template or hook." Context Hub's annotation loop is the automated version of this — the failure surfaces the gap, the annotation captures it, and the next session benefits automatically. orch-go's gap detection system (`gap.go` + `learning.go`) detects gaps but resolves them into a global pool rather than scoping resolutions to the resource where the gap occurred.

---

## Notes

### Could orch-go implement scoped annotations?

Yes, and the infrastructure exists:

1. **Scope identifiers already flow through spawn**: Skill name, target files (from issue/task description), package paths are all available in `SpawnConfig`.

2. **SPAWN_CONTEXT.md generation already merges data from multiple sources** (`context.go`): Adding a "scoped annotations" data source would follow the existing pattern.

3. **Storage could be simple**: Like Context Hub, annotations could be flat files in `.orch/annotations/` or `.kb/annotations/` scoped by resource identifier. No database needed.

4. **The completion pipeline is the natural annotation point**: `knowledge_maintenance.go` already runs at completion. Instead of only offering promote/obsolete/skip for global entries, it could also offer "annotate <scope>" to tie the learning to a specific resource.

### Design sketch (not a recommendation — just mapping the possibility)

```
# Write (at completion or during work)
orch annotate pkg/spawn/gap.go "Gap gate threshold was lowered from 30→20 after false positive analysis"
orch annotate skill:investigation "Agents frequently skip prior work check — consider making it a gate"

# Read (at spawn time — auto-injected into SPAWN_CONTEXT.md)
# When spawning work that touches pkg/spawn/:
# → SPAWN_CONTEXT.md includes: "## Scoped Annotations\n- pkg/spawn/gap.go: Gap gate threshold was..."
# When spawning with skill=investigation:
# → SPAWN_CONTEXT.md includes: "## Scoped Annotations\n- skill:investigation: Agents frequently skip..."
```

### What this probe does NOT recommend

- Replacing kb quick with scoped annotations (they serve different purposes)
- Building this system now (it's an architectural possibility, not an urgent need)
- Adopting Context Hub's feedback API (the up/down mechanism is for doc publishers, not relevant to orch-go's single-user setup)

### Risks of scoped annotations

1. **Scope explosion**: If annotations are too fine-grained (per-line, per-function), they become noise
2. **Staleness**: Unlike kb quick entries with promote/obsolete lifecycle, scoped annotations could rot
3. **Token budget**: More injected content means less room for other spawn context

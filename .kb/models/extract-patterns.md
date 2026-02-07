# Model: Code Extraction Patterns

**Domain:** Architecture / Refactoring / Context Management
**Last Updated:** 2026-01-17
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

### The 800-Line Gate

**Constraint:** Files should not exceed 800 lines.

**Reasoning:** 800 lines is the heuristic limit where "Context Noise" begins to degrade agent reasoning. When a file hits this limit, it triggers a **Sub-domain Extraction** (e.g., moving cache logic to `_cache.go`).

---

## Evolution

### Jan 3-4, 2026: The "Great Split"
- Refactored `main.go` (4,900 lines) and `serve.go` (2,900 lines).
- Established the `shared.go` and `serve_agents_cache.go` patterns.
- Achieved ~40% line reduction in monolithic files.

### Jan 6-8, 2026: Frontend Tab Pattern
- Applied extraction to `agent-detail-panel.svelte`.
- Proved that tab-based component splitting reduces Svelte file size while maintaining reactivity.

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

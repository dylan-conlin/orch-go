# Code Extraction Patterns

**Purpose:** Single authoritative reference for extracting code into separate files during refactoring. Read this before starting extraction work.

**Last verified:** 2026-01-08

---

## Overview

This guide synthesizes learnings from 13 extraction investigations covering Go packages (serve.go, main.go), Svelte components (+page.svelte, agent-detail panel tabs), and TypeScript services (SSE connection management). It provides proven patterns and gotchas for safely splitting large files.

---

## Architecture

```
Before Extraction:           After Extraction:
┌─────────────────────┐      ┌─────────────────────┐
│  monolithic.go      │      │  main.go (setup)    │
│  - shared utilities │      ├─────────────────────┤
│  - domain A code    │  →   │  shared.go          │
│  - domain B code    │      ├─────────────────────┤
│  - domain C code    │      │  domain_a_cmd.go    │
│  (~3000+ lines)     │      │  domain_b_cmd.go    │
└─────────────────────┘      │  domain_c_cmd.go    │
                             │  (~300-800 lines each)│
                             └─────────────────────┘

Extraction Order:
1. Shared utilities FIRST (breaks cross-dependencies)
2. Domain files in parallel (independent units)
```

---

## How It Works

### Phase 1: Extract Shared Utilities First

**What:** Move cross-command/cross-handler functions to a dedicated shared.go file.

**Key insight:** Shared utilities MUST be extracted first. If you extract domain files first, you'll duplicate shared code or create complex import chains.

| Aspect | Details |
|--------|---------|
| Target file | `shared.go` for Go, `lib/utils/` for Svelte |
| Content | Functions used by 2+ domains (e.g., truncate, findWorkspaceByID) |
| Line reduction | Usually 200-300 lines from source |

### Phase 2: Extract Domain-Specific Code

**What:** Move command handlers and their helpers to dedicated files.

**Key insight:** Extract the ENTIRE domain unit together: command definition + flags + init() + run function + types + helpers. Partial extraction creates confusion.

| Naming Convention | Example |
|-------------------|---------|
| Go commands | `status_cmd.go`, `clean_cmd.go`, `send_cmd.go` |
| Go handlers | `serve_agents.go`, `serve_beads.go`, `serve_system.go` |
| Svelte components | `lib/components/stats-bar/stats-bar.svelte` |

### Phase 3: Extract Sub-Domains (If Needed)

**What:** If a domain file grows too large (>800 lines), split further by responsibility.

**Key insight:** Use descriptive suffixes: `serve_agents_cache.go` (cache infrastructure), `serve_agents_events.go` (SSE event handlers).

### Phase 4: Extract Feature Tabs (Svelte)

**What:** Extract tab/section content from large panel components into dedicated sub-components within the same directory.

**Key insight:** Unlike new component extraction (which creates new directories with barrel exports), feature tab extraction adds files to an existing component directory. Props-based design with `$props()` rune keeps interface clean.

| Aspect | Details |
|--------|---------|
| Pattern | Large panel → multiple tab components |
| Example | `agent-detail-panel.svelte` → `ActivityTab.svelte`, `SynthesisTab.svelte` |
| Interface | Single `agent: Agent` prop, self-contained state with `$state()` |
| Exports | Add to existing `index.ts` barrel export |
| Line reduction | ~200-230 lines per tab |

**When to use:** Panel component exceeds 400 lines with multiple logical sections (tabs, accordions, distinct UI regions).

### Phase 5: Extract Shared Services (TypeScript)

**What:** Extract duplicate infrastructure patterns (SSE connections, API clients, state management) into shared services.

**Key insight:** Look for the same connection/lifecycle patterns repeated across multiple stores. Extract the infrastructure, keep domain logic in stores.

| Aspect | Details |
|--------|---------|
| Pattern | Duplicate infrastructure → shared service factory |
| Example | SSE connection logic in `agents.ts` + `agentlog.ts` → `lib/services/sse-connection.ts` |
| Interface | Factory function with callbacks for domain-specific handling |
| Benefits | Centralized lifecycle management, single fix point for bugs |
| Line reduction | ~70+ lines per consumer |

**When to use:** Same infrastructure pattern (EventSource lifecycle, reconnection, timers) appears in 2+ stores.

---

## Key Concepts

| Concept | Definition | Why It Matters |
|---------|------------|----------------|
| Package-level visibility | All Go files in same package share function visibility | No imports needed between `main.go` and `status_cmd.go` |
| Cohesive extraction unit | All related code for one concern | Keep command + helpers + types together |
| Test file co-location | Tests move with their source | `status_cmd.go` → `status_cmd_test.go` |
| Svelte bindable props | `$bindable` for two-way parent-child binding | Enables clean component extraction with state sharing |

---

## Common Problems

### "Redeclaration error after extraction"

**Cause:** Code exists in both old and new files.

**Fix:** 
1. Remove extracted code from the source file after creating the new file
2. Run `go build ./cmd/orch/` to verify no duplicates

**NOT the fix:** Creating the new file without removing from old file

### "Parallel agent already completed the extraction"

**Cause:** Multiple agents working on related tasks can race.

**This is expected behavior.** When extraction work overlaps:

**Fix options:**
1. Check `git log --oneline -5` at task start to detect prior completion
2. If already done, verify correctness and close as "already completed"
3. If partial, fix remaining conflicts (usually duplicate test functions)

### "Test function redeclared in this block"

**Cause:** Tests exist in both old test file and new test file after extraction.

**Fix:**
1. Identify which test file should own each test
2. Remove duplicates from the wrong file
3. Follow pattern: tests stay with their handlers

### "Import not used after extraction"

**Cause:** Imports remain in source file after code is moved out.

**Fix:** Remove unused imports from source file. Go compiler will tell you which ones.

---

## Key Decisions (from investigations)

These are settled. Don't re-investigate:

- **Extract shared utilities first** - Prevents duplication and complex imports
- **Keep all related code together** - Command + flags + init + run + types in one file
- **Tests follow handlers** - serve_agents_test.go moves with serve_agents.go
- **Target ~300-800 lines per file** - Larger files should be split further

---

## What Lives Where

| File Type | Location | Purpose |
|-----------|----------|---------|
| Shared Go utilities | `cmd/orch/shared.go` | Cross-command helpers (truncate, findWorkspace, etc.) |
| Command files | `cmd/orch/{name}_cmd.go` | Single CLI command + all related code |
| Handler files | `cmd/orch/serve_{domain}.go` | HTTP handlers for one domain |
| Sub-domain files | `cmd/orch/serve_{domain}_{aspect}.go` | Split large handlers (cache, events) |
| Svelte components | `web/src/lib/components/{name}/` | Reusable UI components |
| Svelte feature tabs | `web/src/lib/components/{parent}/{name}-tab.svelte` | Sub-components within panel directories |
| TypeScript services | `web/src/lib/services/{name}.ts` | Shared infrastructure (SSE, API clients) |

---

## Extraction Workflow

### For Go Files

```bash
# 1. Identify extraction targets (shared utilities, then domains)
wc -l cmd/orch/main.go  # Check current size

# 2. Create new file with extracted code
# - Add package header
# - Copy types, functions, helpers
# - Add necessary imports

# 3. Remove extracted code from source file
# - Delete copied code
# - Clean up unused imports

# 4. Verify
go build ./cmd/orch/
go test ./cmd/orch/...

# 5. Move tests if applicable
# - Create {name}_test.go
# - Move related tests from old test file
# - Remove duplicates
```

### For Svelte Components (New Directory)

```bash
# 1. Create component directory
mkdir -p web/src/lib/components/{name}

# 2. Create component file
# - Move template, script, style
# - Import stores directly
# - Use $bindable for two-way props

# 3. Create index.ts barrel export
echo "export { default as ComponentName } from './component-name.svelte';" > index.ts

# 4. Update parent
# - Import component
# - Replace inline code with <ComponentName />

# 5. Verify
# - Build: npm run build (or similar)
# - Visual: Check in browser
```

### For Svelte Feature Tabs (Within Existing Directory)

```bash
# 1. Identify tab boundaries in parent component
# - Look for sections with clear responsibilities (Activity, Synthesis, etc.)
# - Each tab should be ~150-250 lines

# 2. Create tab component in same directory
# - File: web/src/lib/components/{parent}/{name}-tab.svelte
# - Use $props() rune: let { agent }: { agent: Agent } = $props()
# - Use $state() for local state (filters, toggles)

# 3. Add to existing barrel export
# - Update index.ts: export { default as NameTab } from './name-tab.svelte';

# 4. Replace inline section in parent
# - Import NameTab
# - Replace markup with <NameTab {agent} />
# - Remove duplicated helper functions

# 5. Verify
# - Build: cd web && bun run build
# - TypeScript: cd web && bun run check
# - Visual: Check in browser
```

### For TypeScript Services (Shared Infrastructure)

```bash
# 1. Identify duplicate infrastructure patterns
# - Look for same EventSource/WebSocket lifecycle in multiple files
# - Look for same timer/reconnection logic in multiple stores

# 2. Create service in lib/services/
# - File: web/src/lib/services/{name}.ts
# - Export factory function: createXxxConnection(options)
# - Options include callbacks for domain-specific handling

# 3. Update consumers to use service
# - Import factory function
# - Replace inline infrastructure with service calls
# - Keep domain-specific event handlers in store

# 4. Verify
# - Build: cd web && bun run build
# - Check duplicate patterns removed: grep for old patterns
```

---

## Debugging Checklist

Before spawning an investigation about extraction issues:

1. **Check kb:** `kb context "extract"`
2. **Check this guide:** You're reading it
3. **Check git log:** `git log --oneline -10` for recent extractions
4. **Check build:** `go build ./cmd/orch/` for immediate errors
5. **Check tests:** `go test ./cmd/orch/...` for regressions

If those don't answer your question, then investigate. But update this guide with what you learn.

---

## Line Count Benchmarks

From successful extractions:

| Extraction | Before | After | Reduction |
|------------|--------|-------|-----------|
| serve_agents.go from serve.go | 2921 | 1815 | -1106 |
| status_cmd.go from main.go | 4964 | 3906 | -1058 |
| clean_cmd.go from main.go | ~1000 | ~330 | -670 |
| 5 small commands from main.go | 854 | 195 | -659 |
| serve_agents_cache.go | 1400 | 970 | -430 |
| serve_agents_events.go | 970 | 724 | -246 |
| StatsBar from +page.svelte | 920 | 678 | -242 |
| ActivityTab from agent-detail-panel | - | 229 | ~229 |
| SynthesisTab from agent-detail-panel | - | 195 | ~195 |
| SSE Connection Service | - | 171 | ~70 saved per consumer |

---

## References

- **Investigations consolidated (13 total):**
  - `.kb/investigations/2026-01-03-inv-extract-serve-agents-go-serve.md`
  - `.kb/investigations/2026-01-03-inv-extract-serve-learn-go-serve.md`
  - `.kb/investigations/2026-01-03-inv-extract-serve-system-go-serve.md`
  - `.kb/investigations/2026-01-03-inv-extract-shared-go-utility-functions.md`
  - `.kb/investigations/2026-01-03-inv-extract-status-cmd-go-main.md`
  - `.kb/investigations/2026-01-04-inv-extract-clean-cmd-go-main.md`
  - `.kb/investigations/2026-01-04-inv-extract-small-commands-send-tail.md`
  - `.kb/investigations/2026-01-04-inv-phase-extract-serve-agents-cache.md`
  - `.kb/investigations/2026-01-04-inv-phase-extract-serve-agents-events.md`
  - `.kb/investigations/2026-01-04-inv-phase-extract-statsbar-component-extract.md`
  - `.kb/investigations/2026-01-06-inv-extract-activitytab-component-part-orch.md` *(added 2026-01-08)*
  - `.kb/investigations/2026-01-06-inv-extract-synthesistab-component-part-orch.md` *(added 2026-01-08)*
  - `.kb/investigations/2026-01-04-inv-phase-extract-sse-connection-manager.md` *(added 2026-01-08)*

- **Note:** 5 other investigations with "extract" in the name are about different topics (knowledge extraction, constraint extraction, lineage headers) and are not covered by this guide.

- **Source code:** `cmd/orch/*.go`, `web/src/`

---

## History

- **2026-01-06:** Created from synthesis of 10 extraction investigations spanning serve.go refactoring, main.go command extraction, and Svelte component extraction.
- **2026-01-08:** Updated with 3 additional extraction patterns: Svelte feature tabs (ActivityTab, SynthesisTab) and TypeScript service extraction (SSE Connection Manager).

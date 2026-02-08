# Synthesis: Registry Evolution and Orch Identity

**Date:** 2025-12-21
**Purpose:** Weave together the full narrative of orch's evolution, the registry's role, and the path forward.

---

## The Core Identity (Nov 29, 2025)

From day one, orch was conceived as **"kubectl for AI agents"** - a command-line tool for managing AI coding agent lifecycle:

- **Spawn:** Launch agents with structured context
- **Monitor:** Track progress in real-time  
- **Coordinate:** Manage multiple agents working together
- **Complete:** Verify agent work and clean up

The analogy is precise: just as kubectl manages container lifecycle across Kubernetes clusters, orch manages agent lifecycle across AI sessions. This identity has remained stable through 22 days and 793 commits. Everything that followed was implementation and refinement, not redefinition.

---

## The Five Concerns Architecture (Dec 1)

Early on, the system was conflating multiple concerns. The Dec 1 decision established clean separation:

| Tool | Layer | Storage | Purpose |
|------|-------|---------|---------|
| `bd` | Memory | `.beads/` | Task state, dependencies, execution log |
| `kb` | Knowledge | `.kb/` | Investigations, decisions, patterns |
| `skills` | Guidance | `~/.claude/skills/` | Agent behavioral procedures |
| `orch` | Lifecycle | **(stateless)** | Spawn, monitor, complete, verify |
| `tmux` | Session | (runtime) | Persistence, attach, output |

**The key architectural principle:** Each tool owns one concern. **Lifecycle layer (orch) has no state of its own** - it orchestrates, but state lives in beads (tasks) and kb (knowledge).

This principle is why the registry is architecturally awkward. Orch was designed to be stateless, yet the registry is state.

---

## Why the Registry Exists

The registry (`~/.orch/agent-registry.json`) was present from the initial commit. It served three purposes:

1. **Agent lookup during session** - Fast O(1) lookup for commands like `orch check <id>`, `orch send <id>`. Without it, every command would need to query external systems.

2. **Tmux reconciliation** - Compare registry agents with tmux window state to detect crashed/completed agents. The registry was the "expected state" that tmux was reconciled against.

3. **History/analytics** - Track completed agents for `orch history --analytics`.

**The registry was necessary because:**
- tmux was the only way to manage agents (no programmatic API)
- Claude CLI was subprocess-based (no session persistence)
- Beads integration was immature (couldn't store agent metadata)
- There was no way to query "what's running" except by maintaining our own list

---

## The Pivot: Go + OpenCode (Dec 18)

The Dec 18 decision fundamentally changed the equation. OpenCode provided:

- **REST API** for session management (create, list, delete)
- **SSE** for real-time events (completion, errors, tool calls)
- **Native session persistence** - sessions survive process death
- **Q&A on completed sessions** - just send another message

This eliminated the need for:
- Subprocess management of Claude CLI
- tmux as the primary agent interface
- Complex stdout parsing for events

**Go was chosen because:**
- Single binary distribution (like bd, kn, kb)
- Goroutines for natural concurrency
- HTTP client is simpler than subprocess management
- Consistent toolchain with other tools

The rewrite reached near feature parity in 3 days (218 commits) by leveraging learned requirements from 27k lines of Python.

---

## The Registry Problem Today

The registry was the right choice when:
- tmux was the only way to manage agents
- OpenCode API didn't exist  
- Beads integration was immature

Now that:
- OpenCode provides REST API + SSE for real-time state
- Beads stores all agent metadata in comments
- Tmux window names contain `[beads-id]` for correlation

**The registry is solving yesterday's problem.**

The drift we're fighting today - stale sessions in `orch status`, ghost agents, four-layer reconciliation complexity - is a symptom of the registry being a cache that can't stay in sync with reality.

The Dec 6 investigation in orch-cli identified this and proposed a phased migration:

| Phase | Status | What |
|-------|--------|------|
| Phase 1 | ✅ Done | Store agent metadata in beads comments |
| Phase 2 | ✅ Done | Beads-first lookup with registry fallback |
| Phase 3 | ✅ Done | Registry stripped to minimal tmux mapping |
| Phase 4 | ❌ Not done | Remove registry file entirely |

The Python version got to Phase 3 before the Go rewrite started. The Go version inherited a fresh registry implementation without carrying forward the simplification work.

---

## What We'd Lose Without Registry

| Capability | What's Lost | Alternative |
|------------|-------------|-------------|
| **Fast agent lookup** | O(1) JSON lookup | Query beads + tmux + OpenCode directly (~100-300ms) |
| **Window ID tracking** | agent_id ↔ window_id mapping | Parse from tmux window names (already contain `[beads-id]`) |
| **Cross-session state** | Knowing what agents exist across restarts | Beads tracks all issues; tmux/OpenCode show live sessions |
| **Reconciliation baseline** | "Expected state" to reconcile against | No reconciliation needed - query live state directly |

---

## What We'd Gain Without Registry

| Benefit | Why It Matters |
|---------|----------------|
| **Single source of truth** | No more drift between registry, tmux, OpenCode, beads |
| **Architectural purity** | "orch is stateless" - fulfills five-concern design |
| **No reconciliation needed** | Query live state, not cached state |
| **Simpler code** | Remove 500+ lines of merge conflict, tombstone, locking logic |
| **No ghost agents** | Can't have stale entries if there's no cache |

---

## The Core Tradeoff

**Registry = fast but stale**
- O(1) lookup but drifts from reality
- Requires reconciliation to fix drift
- Reconciliation is itself complex and error-prone (four layers!)

**Direct query = slower but always correct**
- ~100-300ms latency per lookup
- Always reflects actual state
- No reconciliation needed because there's no cache to drift

---

## The Path Forward

The question isn't "should we remove the registry?" The question is: **"Is 100-300ms latency acceptable for `orch status`?"**

If yes → Registry becomes unnecessary
If no → Registry stays as read-through cache, we accept drift and reconciliation complexity

Given that:
- `orch status` is an interactive command (human waits for output)
- 100-300ms is imperceptible to humans
- The alternative is ongoing drift-fighting and reconciliation bugs

**Recommendation:** Complete Phase 4 of the Python migration in orch-go:

1. `orch status` queries OpenCode API + tmux directly
2. Beads is source of truth for "what work exists"
3. Registry becomes optional - remove it or keep as ephemeral cache
4. No reconciliation needed because we query live state

This fulfills the Dec 1 architectural vision: **orch orchestrates, but doesn't own state.**

---

## Summary

| When | What Happened | Why |
|------|---------------|-----|
| Nov 29 | Orch created with registry | tmux-only, no API, needed agent tracking |
| Dec 1 | Five concerns: "orch is stateless" | Architectural clarity |
| Dec 6 | Registry removal investigation | Beads can be source of truth |
| Dec 8 | Registry stripped to minimal | Phase 3 of migration |
| Dec 18 | Go + OpenCode decision | HTTP API changes everything |
| Dec 19-21 | Go rewrite | Fresh implementation, inherited registry pattern |
| Dec 21 | Registry drift causing pain | Four-layer reconciliation, ghost agents |

The registry was correct for its time. The architecture evolved. The registry didn't evolve with it.

**The answer to "what should orch be?"** remains what it was on Nov 29: kubectl for AI agents - stateless orchestration. The registry was a necessary compromise that's no longer necessary.

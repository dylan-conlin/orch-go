# Probe: OpenCode Fork Necessity Assessment — Is the Fork Still Required?

**Model:** opencode-fork
**Date:** 2026-03-24
**Status:** Complete
**claim:** n/a (cross-cutting assessment, not a single claim)
**verdict:** extends

---

## Question

The OpenCode fork was created when OpenCode was the primary execution backend. Since then, the default backend switched to `claude` (Claude Code CLI). Is the fork still necessary, or could upstream OpenCode (or dropping OpenCode entirely) replace it?

---

## What I Tested

### 1. Fork divergence metrics

```bash
cd ~/Documents/personal/opencode
git rev-list --count upstream/dev..dev    # 32 commits ahead
git rev-list --count dev..upstream/dev    # 975 commits behind
git log --format="%ai" b6c4605d5 -1       # Last sync: 2026-02-18
git log upstream/dev --format="%ai" -1    # Upstream head: 2026-03-24
```

### 2. Feature parity check: upstream vs fork

```bash
# Searched upstream for fork-specific features
git log upstream/dev --oneline --grep="metadata"    # No session.metadata
git log upstream/dev --oneline --grep="ttl\|TTL"    # No session time_ttl
git log upstream/dev --oneline --grep="ORCH_WORKER"  # No ORCH_WORKER header
```

### 3. Backend usage analysis

```bash
# In orch-go resolve.go
grep "Default backend" pkg/spawn/resolve.go
# → "Default backend is now claude since the default model is Anthropic (sonnet)."
```

### 4. Integration point mapping

Read all files in `pkg/opencode/` (12 files, ~3,600+ LoC) and `pkg/spawn/backends/` (headless, tmux, inline). Mapped every OpenCode API call.

### 5. orch-go dependency on fork-specific features

Searched orch-go for usage of fork-only APIs: `metadata`, `time_ttl`, `TimeTTL`, `SetSessionMetadata`, `GetSessionStatus`, `GetAllSessionStatus`.

---

## What I Observed

### Fork Origin and Motivation

The fork was created **late January 2026** (first custom commit 2026-01-28). Initial motivations:
1. **OAuth stealth mode** — Claude Max Opus access via OpenCode (commit 7a85d5754)
2. **Memory management** — 8.4GB unbounded growth in upstream (no LRU/TTL eviction)
3. **SSE cleanup** — leaked connections from missing teardown

### Fork Customizations (32 commits ahead)

| Category | Count | Still Needed? |
|----------|-------|---------------|
| Memory management (LRU/TTL instance eviction) | 3 | **YES** — upstream still has no eviction |
| SSE/server fixes | 3 | **MAYBE** — upstream may have fixed some |
| Session metadata + TTL | 4 | **YES for opencode backend** — orch-go headless uses metadata/TTL |
| ORCH_WORKER header chain | 4 | **YES for opencode backend** — worker detection pipeline |
| Auth/stealth (OAuth) | 2 | **DIMINISHED** — claude backend bypasses OpenCode OAuth |
| MCP race fixes | 2 | **MAYBE** — upstream may have fixed |
| Build/infra | 4 | **LOW** — maintenance artifacts |
| Investigations/probes | 4 | **NO** — documentation only |

### Maintenance Burden

- **975 commits behind** upstream (5 weeks of drift since Feb 18 sync)
- Each rebase requires cherry-picking 32 custom commits onto a moving target
- Upstream restructured `src/` into subdirectories — future rebases will be painful
- Previous rebase (268 upstream commits) was already a significant effort

### What orch-go Actually Uses from OpenCode

The OpenCode integration (`pkg/opencode/`, ~3,600 LoC) uses these APIs:

| API | Used By | Fork Feature Needed? |
|-----|---------|---------------------|
| `POST /session` (create) | Headless + Tmux backends | YES (metadata, time_ttl) |
| `POST /session/:id/prompt_async` | All backends | No |
| `GET /session` (list) | Status, clean, discovery | No |
| `GET /session/:id` | Status, completion | No |
| `GET /session/:id/message` | Transcript, completion | No |
| `GET /session/status` | Discovery (liveness) | No (endpoint exists upstream) |
| `DELETE /session/:id` | Clean | No |
| `PATCH /session/:id` | SetSessionMetadata | YES (metadata) |
| `GET /event` (SSE) | Monitor, streaming | No |

**Key finding:** Only 2 of 9 API integrations require fork-specific features (metadata + TTL on session creation, metadata PATCH).

### Backend Usage Split

The **default backend is now `claude`** (Claude Code CLI). The `opencode` backend is the secondary, multi-model path. The decision at `2026-01-09-dual-spawn-mode-architecture.md` documents this flip:

> "Anthropic OAuth ban (Feb 19, 2026) inverted the primary/secondary roles. Claude CLI is now the primary backend; OpenCode API is the multi-model path."

When using the `claude` backend:
- Agents spawn via `tmux` + `claude` CLI directly
- **No OpenCode session is created** — no session ID, no metadata, no TTL
- OpenCode is not involved at all in the agent lifecycle
- Liveness is tracked via tmux window existence, not OpenCode session status

When using the `opencode` backend:
- Agents spawn via OpenCode HTTP API
- Session metadata, TTL, and ORCH_WORKER detection are used
- This path requires the fork's custom features

### Could Upstream Replace the Fork?

**For the claude backend path: upstream is irrelevant.** The claude backend doesn't use OpenCode at all.

**For the opencode backend path: upstream cannot replace the fork today.** Upstream lacks:
- Session metadata (needed for beads_id/workspace binding)
- Session TTL (needed for auto-cleanup)
- ORCH_WORKER header pipeline (needed for worker detection in plugins)
- Instance LRU/TTL eviction (needed to prevent memory growth)

None of these have been upstreamed in the 975 commits since the last sync.

### Could We Drop OpenCode Entirely?

The `claude` backend path already works without OpenCode. To drop OpenCode completely:

1. **Remove opencode backend** — lose multi-model support (Gemini, GPT, etc.)
2. **Remove `orch sessions` lane** — lose visibility into non-tracked sessions
3. **Remove OpenCode plugins** — lose plugin-based gates/context injection
4. **Remove `pkg/opencode/`** — 3,600+ LoC removed
5. **Remove OpenCode from `orch-dashboard`** — simplify services

**What we'd lose:** Multi-model routing, OpenCode plugin system (gates, context injection, observation), SSE-based real-time monitoring for headless agents, session accumulation cleanup.

**What we'd keep:** Everything on the claude backend path works. Claude Code has its own hooks system that replaces some OpenCode plugin functionality.

---

## Model Impact

- [x] **Extends** model with: Fork necessity assessment framework — the fork's relevance has **bifurcated by backend**:
  - **Claude backend (primary):** Fork is irrelevant. OpenCode is not used at all.
  - **OpenCode backend (secondary):** Fork is still required. Upstream lacks 4 critical features.

Key extension to model:

### Fork Relevance Has Decayed

The fork was created when OpenCode was the sole execution backend. Since the claude backend became primary (Feb 19, 2026), the fork's importance has diminished proportionally to usage of the opencode backend. The fork is now **maintenance cost for a secondary path**.

### Strategic Options

| Option | Effort | What's Lost | Recommendation |
|--------|--------|-------------|----------------|
| **Keep fork, sync regularly** | HIGH (975+ commits behind, 32 cherry-picks) | Nothing | Only if multi-model is actively needed |
| **Keep fork, stop syncing** | LOW (no rebase work) | Upstream improvements, security patches | Acceptable short-term |
| **Drop opencode backend, keep claude-only** | MEDIUM (remove ~3,600 LoC, simplify architecture) | Multi-model, OpenCode plugins, headless monitoring | **Best if multi-model isn't used** |
| **Upstream custom changes** | HIGH (PR upstream + maintain until accepted) | Nothing long-term | Ideal but uncertain acceptance |

### Quantified Fork Dependency

- **32 custom commits** ahead of upstream
- **975 commits** behind upstream (5 weeks of drift)
- **2 of 9** API integrations require fork features
- **~3,600 LoC** in orch-go depends on OpenCode (any version)
- **~800 LoC** in orch-go depends on fork-specific features
- **Primary backend (claude)** has zero OpenCode dependency

---

## Notes

- The SPAWN_CONTEXT for this very investigation uses `Backend: claude` — confirming that claude is the primary backend for orchestrated work
- The `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md` explicitly documents the primary/secondary flip
- The opencode backend's main advantage (headless mode, HTTP API spawning) is less important now that most spawns route through tmux + claude CLI
- OpenCode plugins (gates, context injection) provided by the fork could potentially be replaced by Claude Code hooks, which already exist in `.claude/hooks/`

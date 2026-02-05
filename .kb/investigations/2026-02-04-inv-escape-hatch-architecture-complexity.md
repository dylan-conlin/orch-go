# Investigation: Escape Hatch Architecture Complexity

**Date:** 2026-02-04
**Status:** Complete
**Spawned from:** orch-go-21301

## Question

How complex is the escape hatch architecture (spawn modes + backends)? Is the complexity justified?

## Quantified Metrics

### Documentation Spread

| Document | Lines | Purpose |
|----------|-------|---------|
| `.kb/guides/spawn.md` | 607 | Single authoritative reference |
| `.kb/guides/dual-spawn-mode-implementation.md` | 450 | Implementation guide |
| `.kb/models/escape-hatch-visibility-architecture.md` | 386 | Dual-window visibility model |
| **Total core docs** | **1,443** | |
| Additional mentions | 725 | Across 219 files |

### Code Complexity

| Component | Lines | Purpose |
|-----------|-------|---------|
| `cmd/orch/spawn_*.go` | ~4,200 | Spawn command + helpers |
| `pkg/spawn/*.go` | ~15,400 | Context generation, backends |
| **Total** | **~19,625** | |

### Actual Usage (from 3,143 recorded spawns)

| Mode | Count | Percentage |
|------|-------|------------|
| **headless** | 2,082 | 66% |
| **tmux** | 460 | 15% |
| **claude** (escape hatch) | 463 | 15% |
| **docker** (escape hatch) | 133 | 4% |
| **inline** | 5 | <1% |

**Escape hatch total:** 596 spawns (19%)

## Complexity Analysis

### The Architecture (3 dimensions × 3 options)

**Spawn Modes (UI):**
1. Headless (default) - HTTP API, returns immediately
2. Tmux - Creates visible tmux window
3. Inline - Blocking, runs in current terminal

**Backends (execution):**
1. OpenCode (default) - HTTP API to OpenCode server
2. Claude - Claude CLI directly (Max subscription)
3. Docker - Claude CLI in container (fingerprint isolation)

**Flag Combinations:**
- 30+ flags on spawn command
- Priority cascade: `--backend` > `--opus` > `--infra` > config > default

### Why This Complexity Exists

**1. Cost optimization (3 different cost models):**
- OpenCode: Pay-per-token (dashboard visibility)
- Claude: Flat $200/mo Max subscription (Opus quality)
- Docker: Same as Claude but rate-limit escape

**2. Crash resilience (infrastructure work):**
- OpenCode agents die if OpenCode server crashes
- Claude agents survive server crashes (independent CLI)
- `--infra` flag exists solely for this scenario

**3. Rate limit escape (Anthropic fingerprinting):**
- Docker provides fresh Statsig fingerprint per spawn
- Only bypasses request-rate throttling (NOT weekly quota)
- Used 4% of time (133 spawns)

### Is Complexity Justified?

**YES for the 19% that use escape hatches:**
- Jan 10, 2026: OpenCode crashed 3x while agents fixed observability - claude escape hatch saved the work
- Rate limit scenarios: Docker fingerprint isolation prevents work stoppage

**QUESTIONABLE for the 81% on primary path:**
- Most spawns are headless via OpenCode API
- The cognitive load of understanding 3 backends × 3 modes is high
- 1,443 lines of documentation suggests the system requires significant explanation

### Minimal Viable Subset

**If starting fresh, you only need:**
1. **One backend:** OpenCode (dashboard, cost tracking) OR Claude (quality, crash resistance)
2. **One spawn mode:** Headless for automation, tmux for visibility
3. **Escape hatch:** Claude backend when OpenCode unavailable

**The Docker backend could be eliminated:**
- Only 4% usage (133/3143 spawns)
- Adds: 270 lines code, 100+ lines docs, Docker image dependency
- Rate limit bypass is rarely needed (weekly quota is account-level anyway)

## Cognitive Load Assessment

| Aspect | Burden | Evidence |
|--------|--------|----------|
| Conceptual | High | 3×3 matrix of modes×backends |
| Documentation | High | 1,443 lines core + 219 files referencing |
| Flag surface | High | 30+ flags, priority cascade |
| Usage clarity | Medium | `spawn.md` is good but 607 lines long |
| Error recovery | Medium | Escape hatches work but require understanding |

**Onboarding cost:** A new user would need to read ~600 lines of spawn.md to understand when to use which combination.

## Findings

1. **Documentation is comprehensive but sprawling** - 1,443 lines core + mentions in 219 files

2. **81% of spawns use the simple path** - headless/tmux via OpenCode

3. **19% use escape hatches** - this is high enough to justify their existence

4. **Docker backend is marginal** - 4% usage, could be cut

5. **Complexity is defensive, not offensive** - the escape hatches exist to handle failures, not enable new capabilities

## Recommendations

### If Simplifying (not recommended now)

1. **Deprecate Docker backend** - 4% usage doesn't justify complexity
2. **Make Claude default** - eliminates OpenCode server dependency
3. **Hide inline mode** - <1% usage

### If Documenting (recommended)

1. **Add "quick start" section** to spawn.md showing the 3 common patterns
2. **Create decision flowchart** - when to use which mode/backend
3. **Flag the 30+ flags** by importance tier

### Current State Assessment

The architecture is complex but **the complexity is load-bearing**:
- Escape hatches have prevented work loss (documented Jan 10 incident)
- 19% escape hatch usage validates their existence
- The dual-window visibility model enables intervention in critical work

**Verdict:** Keep but document better. The 81/19 split shows most users can ignore complexity, but the 19% who need escape hatches really need them.

## Delta (What's New)

- Quantified: 1,443 lines docs, 19,625 lines code, 596/3143 (19%) escape hatch usage
- Docker backend only 4% usage - marginal value
- Complexity is defensive (failure handling) not offensive (new capabilities)

## Next

- Consider adding quick-start section to spawn.md
- Docker backend deprecation could simplify system with minimal impact
- No immediate action required - system is stable and usage is understood

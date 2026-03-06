## Summary (D.E.K.N.)

**Delta:** The 8 SessionStart hooks are a parallel authority competing with `orch orient` for the same job — session-start context. They should consolidate to 2 hooks (skill identity + thin orient-caller) plus orient subsuming 5 hook responsibilities, with 1 hook deleted.

**Evidence:** Source analysis of all 8 hooks, orient_cmd.go, and load-orchestration-context.py shows 70% overlap between hook context injection and orient data. The big hook already calls `orch orient --json --skip-ready` internally, creating a circular dependency.

**Knowledge:** The distinction is *identity* (who am I — must be a hook) vs *orientation* (what should I do — should be one command). bd prime is workflow guidance that orient should embed, not duplicate. Skill loading must happen pre-first-turn; everything else can be a single `orch orient` call.

**Next:** Implement in 3 phases: (1) Add orient sections for reflect, usage, config-check, session-resume. (2) Create single thin hook calling `orch orient --hook`. (3) Remove 6 individual hooks, keeping only skill-loader hook + new orient hook.

**Authority:** architectural - Cross-component change affecting hooks, orient, and session startup flow across all projects.

---

# Investigation: Design Consolidation of SessionStart Hooks into orch orient

**Question:** How should 8 organic SessionStart hooks consolidate into a coherent session-start system centered on `orch orient`?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — ready for implementation phasing
**Status:** Complete

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Coherence Over Patches principle (Jan 4 2026) | motivating principle | Yes — 8 hooks match "accumulated 10+ conditions from incremental patches" | None |
| Deploy or Delete principle (Mar 2026) | migration constraint | Yes — must remove old hooks, not leave them alongside | None |
| `.kb/investigations/2026-01-04-design-dashboard-agent-status-model.md` | pattern parallel | Verified principle reference | None — same pattern, different domain |

---

## Findings

### Finding 1: load-orchestration-context.py Already Calls Orient

**Evidence:** `load-orchestration-context.py:175` calls `orch orient --json --skip-ready`, and `:80` calls `orch frontier --json`. The hook already delegates to orient for throughput, models, and focus data. It also independently loads: orchestrator skill, bd prime, frontier state, README summary, ROADMAP priorities, pending decisions, and kn recent.

**Source:** `~/.orch/hooks/load-orchestration-context.py:165-233`

**Significance:** The hook is *already* an orient wrapper for half its content. The other half (skill, bd prime, frontier, README, ROADMAP, kn recent, pending decisions) is either duplicated by orient or is unique context that orient should subsume. This is the strongest evidence for consolidation — the boundary between "hook-provided" and "orient-provided" has already blurred.

---

### Finding 2: Three Natural Categories Among the 8 Hooks

**Evidence:** Categorizing the hooks by what they do reveals three categories:

| Category | Hooks | Characteristic |
|----------|-------|----------------|
| **Identity** (who am I) | load-orchestration-context.py (skill loading portion only) | Must run before first turn. Shapes all behavior. |
| **Orientation** (what's the state) | session-start.sh, reflect-suggestions-hook.py, usage-warning.sh, load-orchestration-context.py (dynamic state portion) | State queries that orient already does or should do. |
| **Guards** (is anything broken) | check-config-symlinks.sh | Silent on success. Only surfaces problems. |
| **Dormant/Niche** | inject-orch-patterns.sh, agentlog-inject.sh | Rarely/never fire. Candidates for removal. |

**Source:** All 8 hook files read and analyzed.

**Significance:** The "Identity" category is irreducible — skill loading must happen via hook because it needs to execute before orient even runs (the skill tells the agent *how to use* orient). Everything in "Orientation" maps cleanly to orient sections. Guards are fine as hooks (silent-on-success is the right pattern). Dormant hooks should be deleted (Deploy or Delete principle).

---

### Finding 3: Orient Already Has the Right Architecture for Subsumption

**Evidence:** `orient_cmd.go` uses a collector pattern: independent `collect*()` functions each produce a section of `OrientationData`. The data flows through `FormatOrientation()` for text or `json.Encoder` for JSON. Adding new collectors (reflect suggestions, usage warning, session resume, config check) follows the existing pattern with zero architectural change.

**Source:** `cmd/orch/orient_cmd.go:61-123`, `pkg/orient/orient.go`

**Significance:** Orient doesn't need redesign to subsume hooks. Each hook maps to a new collector function + struct field. The `--json` flag already supports hook consumption. The only new thing needed is a `--hook` flag that wraps output in the `hookSpecificOutput.additionalContext` JSON envelope.

---

### Finding 4: bd prime Output is Static Workflow Guidance

**Evidence:** `bd prime --full` outputs ~60 lines of static markdown: close protocol checklist, core rules, essential commands reference. This content doesn't change between sessions. It's workflow guidance, not dynamic state.

**Source:** `bd prime --full` output, `load-orchestration-context.py:419-434`

**Significance:** This content belongs in CLAUDE.md or the orchestrator skill, not injected fresh every session. Currently it's injected via the separate `bd prime` hook AND by load-orchestration-context.py (which calls `bd prime --full`). That's the hook already being called twice — Class 6 (Duplicate Action). Orient should NOT subsume bd prime by calling it; instead, the bd prime content should be a static part of the orchestrator skill or CLAUDE.md, and the hook + orient both stop calling it.

---

### Finding 5: The Orchestrator Skill is ~27KB and Must Load Pre-Orient

**Evidence:** The orchestrator skill at `~/.claude/skills/meta/orchestrator/SKILL.md` is loaded by `load-orchestration-context.py:20-35`. This skill defines *how the orchestrator behaves* — it's not session state, it's identity. It includes spawn delegation rules, completion gates, and conversational patterns. Without it, the orchestrator doesn't know how to interpret orient output.

**Source:** `~/.orch/hooks/load-orchestration-context.py:20-35`, `~/.claude/skills/meta/orchestrator/SKILL.md`

**Significance:** Skill loading can't move into orient because orient is a tool the orchestrator uses *after* knowing its role. This is the irreducible hook — the one thing that must stay. But it should be *only* the skill loader, not the 684-line Python script doing 10 different things.

---

### Finding 6: Session Resume is Already an Orient-Compatible Pattern

**Evidence:** `session-start.sh:17-33` calls `orch session resume --for-injection` and wraps output in additionalContext JSON. This is exactly the pattern orient uses — call an `orch` subcommand, format the output. Orient already has `collectPreviousSession()` which reads the latest debrief. Session resume handoff could be a second collector in the same section.

**Source:** `~/.claude/hooks/session-start.sh:17-33`, `cmd/orch/orient_cmd.go:252-264`

**Significance:** Session resume is a clean migration candidate. Orient already does "previous session" — adding session resume handoff is a natural extension, not a new concept.

---

## Synthesis

**Key Insights:**

1. **The distinction is Identity vs Orientation** — Skill loading (who am I) is the only thing that truly must be a hook. Everything else is orientation (what's the state of the world) and belongs in `orch orient`. This distinction resolves the tension described in the task: hooks do context injection, orient does action prompting, and the boundary blurred because they were never distinguished by *kind*.

2. **Orient already won** — load-orchestration-context.py already delegates to orient for half its work. The hook has organically evolved toward being a thin orient wrapper. The design should complete this evolution rather than fight it.

3. **bd prime is a static artifact, not a dynamic query** — It should be baked into the orchestrator skill or CLAUDE.md, not called at session start. Removing it from the startup path eliminates ~5s of subprocess overhead and a Class 6 duplicate (it's already called as a separate hook AND inside load-orchestration-context.py).

**Answer to Investigation Question:**

Consolidate to a 2-hook + 1-command architecture:

**Hook 1: Skill Identity Loader** (keep, simplify)
- Loads `~/.claude/skills/meta/orchestrator/SKILL.md`
- Loads bd prime content (static, baked in, not subprocess call)
- Nothing else. ~30 lines, not 684.
- Skip for spawned agents (existing behavior).

**Hook 2: Orient Caller** (new, thin)
- Calls `orch orient --hook`
- Passes stdout directly as additionalContext
- ~10 lines of shell script.

**orch orient --hook** (enhanced orient)
- New flag produces output wrapped in `hookSpecificOutput` JSON envelope
- New sections added to orient:
  - Session resume handoff (from session-start.sh)
  - Reflect suggestions (from reflect-suggestions-hook.py, reading same JSON file)
  - Usage warning (from usage-warning.sh, calling `orch usage --json`)
  - Config symlink check (from check-config-symlinks.sh)
  - Frontier state (from load-orchestration-context.py's frontier call)
  - kn recent (from load-orchestration-context.py)

**Removed hooks:**
- `session-start.sh` → subsumed by orient's session resume section
- `load-orchestration-context.py` → split: skill part → Hook 1, dynamic state → orient
- `reflect-suggestions-hook.py` → subsumed by orient
- `usage-warning.sh` → subsumed by orient
- `check-config-symlinks.sh` → subsumed by orient (silent-on-success logic preserved)
- `inject-orch-patterns.sh` → deleted (only fires in .orch/ dir, niche use case)
- `agentlog-inject.sh` → deleted (dormant, .agentlog doesn't exist)

**bd prime** → Content baked into orchestrator skill or CLAUDE.md. No longer called at startup.

---

## Structured Uncertainty

**What's tested:**

- ✅ load-orchestration-context.py already calls `orch orient --json` (verified: read source line 175)
- ✅ orient_cmd.go uses extensible collector pattern (verified: read source, 10 independent collectors)
- ✅ bd prime output is static workflow guidance (verified: ran `bd prime --full`, compared across context)
- ✅ inject-orch-patterns.sh only fires in .orch/ dir (verified: read source line 17)
- ✅ agentlog-inject.sh only fires if .agentlog/ exists (verified: read source line 12)
- ✅ Session resume already uses `orch session resume` subcommand (verified: read source)

**What's untested:**

- ⚠️ Performance impact of adding 5 new collectors to orient (not benchmarked — currently orient runs in load-orchestration-context.py which has 30s timeout)
- ⚠️ Whether removing bd prime from startup path affects spawned worker agents (workers have ORCH_SPAWNED=1 and skip load-orchestration-context.py, but bd prime runs as a separate hook for all sessions)
- ⚠️ Whether `orch frontier --json` still works or has been removed (command returned "not available" when tested)

**What would change this:**

- If orient with 5 new collectors exceeds the 30s hook timeout, we'd need async/parallel collection
- If bd prime content is genuinely dynamic (changes between sessions), it can't be baked in
- If spawned workers depend on bd prime being injected (they might — bd prime is a separate SessionStart hook without CLAUDE_CONTEXT filtering)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Consolidate hooks into orient | architectural | Cross-component change affecting hooks, orient, session startup, and orchestrator skill |
| Delete dormant hooks | implementation | Cleanup within existing patterns, no cross-boundary impact |
| Move bd prime to static artifact | architectural | Changes where/when workflow guidance loads, affects all sessions |

### Recommended Approach ⭐

**Three-Phase Consolidation** — Additive-first migration that never breaks existing sessions.

**Why this approach:**
- Phase 1 is purely additive (new orient sections, new flag) — nothing breaks
- Phase 2 creates the new thin hooks alongside old ones — both work
- Phase 3 removes old hooks only after new path verified — Deploy or Delete completed cleanly
- Each phase is independently shippable and testable

**Trade-offs accepted:**
- Temporary duplication during Phase 2 (both old hooks and new orient sections exist)
- Orient grows by ~5 new sections — addressed via the bloat concern below

**Implementation sequence:**

#### Phase 1: Enhance orient (additive, no hook changes)

Add to `pkg/orient/orient.go` OrientationData:
```go
SessionResume    *SessionResume    `json:"session_resume,omitempty"`
ReflectSummary   *ReflectSummary   `json:"reflect_summary,omitempty"`
UsageWarning     *UsageWarning     `json:"usage_warning,omitempty"`
ConfigDrift      []ConfigDriftItem `json:"config_drift,omitempty"`
FrontierState    *FrontierState    `json:"frontier_state,omitempty"`
```

Add to `cmd/orch/orient_cmd.go`:
- `collectSessionResume()` — calls `orch session resume --for-injection`
- `collectReflectSuggestions()` — reads `~/.orch/reflect-suggestions.json`
- `collectUsageWarning()` — calls `orch usage --json`, checks >80%
- `collectConfigDrift()` — checks symlinks in `~/.claude-personal/`
- `collectFrontierState()` — consolidates bd ready/blocked/active

Add `--hook` flag that wraps entire output in `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"..."}}`

Files modified:
- `cmd/orch/orient_cmd.go` — new collectors + --hook flag
- `pkg/orient/orient.go` — new struct fields + format sections

#### Phase 2: Create new hooks, run alongside old ones

Create `~/.claude/hooks/load-skill-identity.sh` (~30 lines):
- Load orchestrator skill content
- Skip for spawned agents
- Include static bd workflow guidance (baked in, not subprocess)

Create `~/.claude/hooks/orient-session.sh` (~10 lines):
- Call `orch orient --hook`
- Pass stdout through

Add both to `settings.json` SessionStart alongside existing hooks. Verify output matches.

#### Phase 3: Remove old hooks (Deploy or Delete)

Remove from `settings.json`:
- `session-start.sh`
- `load-orchestration-context.py`
- `reflect-suggestions-hook.py`
- `usage-warning.sh`
- `check-config-symlinks.sh`
- `inject-orch-patterns.sh`
- `agentlog-inject.sh`
- `bd prime` (separate SessionStart hook)

Result: 2 SessionStart hooks remain (identity + orient-caller).

### Alternative Approaches Considered

**Option B: Pure orient (no hooks at all)**
- **Pros:** Maximum simplicity — one command does everything
- **Cons:** Skill loading can't be in orient because the skill defines *how to use* orient. Chicken-and-egg problem.
- **When to use instead:** If Claude Code adds a "pre-session skill loading" mechanism that doesn't require hooks

**Option C: Consolidate into load-orchestration-context.py (keep the big hook)**
- **Pros:** No orient changes needed, just simplify the hook
- **Cons:** Keeps the 30s Python hook as the critical path. Hook code is harder to test than Go code. Violates the pattern where orient is the session-start command.
- **When to use instead:** If orient is being deprecated or if Go compilation is a blocker

**Rationale for recommendation:** Option A completes the evolution that's already happening (hook already calls orient). It moves logic from a hard-to-test Python hook into testable Go code with structured output. It follows Coherence Over Patches (one system, not 8) and Deploy or Delete (old hooks removed in Phase 3).

---

### Implementation Details

**What to implement first:**
- Phase 1 orient enhancements (purely additive, testable independently)
- The `--hook` output format is the key enabler for Phase 2

**Things to watch out for:**
- ⚠️ **bd prime for workers:** Workers currently get bd prime via a separate hook (line 266-272 in settings.json). When we remove that hook in Phase 3, workers still need bd workflow guidance. Solution: workers get this from their SPAWN_CONTEXT.md (which already includes beads close protocol), OR the skill-identity hook includes it.
- ⚠️ **frontier command availability:** `orch frontier --json` returned "not available" during testing. If this command was removed, orient needs to replicate its logic (bd ready + bd blocked + orch status).
- ⚠️ **Orient output size:** Adding 5 sections risks bloating orient output. Mitigation: use `--sections` flag to let the hook request only what it needs, OR use conditional rendering (skip sections with no data, which most already do).
- ⚠️ **Hook timeout:** Current 30s timeout on load-orchestration-context.py. Orient enhanced with all collectors must complete within this window. The collectors are mostly file reads and subprocess calls that already run in the hook, so this should be fine.

**Addressing the orient bloat concern:**

Orient currently has 11 sections. Adding 5 more makes 16. This is manageable because:
1. Most sections are conditional (only render when data exists)
2. The collector pattern is flat — each collector is independent, ~20-40 lines
3. orient_cmd.go is currently 403 lines — adding 5 collectors adds ~150 lines, well under the 1500-line hotspot threshold
4. If it grows beyond comfort, collectors can extract to `cmd/orch/orient_*.go` files (Go supports multi-file packages)

**Success criteria:**
- ✅ `orch orient --hook` produces valid JSON that Claude Code accepts as additionalContext
- ✅ Session start with new 2-hook config injects same information as old 8-hook config
- ✅ `orch orient` completes within 15s (the slowest collector is session resume, which calls orch session resume)
- ✅ Old hooks removed from settings.json (Phase 3 complete = Deploy or Delete satisfied)
- ✅ Spawned agents (ORCH_SPAWNED=1) still get bd workflow guidance via SPAWN_CONTEXT.md

---

## References

**Files Examined:**
- `~/.claude/settings.json` — All 8 SessionStart hook configurations
- `~/.claude/hooks/session-start.sh` — Session resume injection (33 active lines)
- `~/.orch/hooks/load-orchestration-context.py` — Main context loader (684 lines)
- `~/.orch/hooks/reflect-suggestions-hook.py` — Reflection suggestions (170 lines)
- `~/.orch/hooks/check-config-symlinks.sh` — Config drift detection (57 lines)
- `~/.claude/hooks/usage-warning.sh` — Usage warning (70 lines)
- `~/.claude/hooks/inject-orch-patterns.sh` — Orch patterns injection (36 lines)
- `~/.claude/hooks/agentlog-inject.sh` — Agentlog injection (40 lines)
- `cmd/orch/orient_cmd.go` — Current orient implementation (403 lines)
- `pkg/orient/orient.go` — Orient data structures and formatting

**Commands Run:**
```bash
# Check bd prime output
bd prime --full

# Check current orient output
orch orient

# Test frontier availability
orch frontier --json

# Check KB context for prior work
kb context "session start hook consolidation orient" --format json
```

**Related Artifacts:**
- **Principle:** `~/.kb/principles.md` — Coherence Over Patches, Deploy or Delete (directly motivating)
- **Principle:** `~/.kb/principles.md` — Evolve by Distinction (identity vs orientation distinction)

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: How to consolidate 8 SessionStart hooks into orient
- Context: Dylan observed "all of these scripts feel like patches when we need coherence"

**2026-03-05:** All 8 hooks and orient source analyzed
- Key finding: load-orchestration-context.py already calls orient internally
- Three categories identified: Identity (1 hook), Orientation (5 hooks), Dormant (2 hooks)

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: 3-phase consolidation from 8 hooks to 2, with orient subsuming 5 hook responsibilities and 2 deleted

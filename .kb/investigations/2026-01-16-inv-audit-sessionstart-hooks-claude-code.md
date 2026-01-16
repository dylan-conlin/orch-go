<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** SessionStart hooks inject 25K+ tokens worst-case, with load-orchestration-context.py as main culprit (23K); only one hook detects spawned agents.

**Evidence:** Tested all 7 hooks by running them with simulated input; measured actual output sizes; verified spawn detection via CLAUDE_CONTEXT env var check in load-orchestration-context.py:436-447.

**Knowledge:** Manual sessions get orchestrator skill (86KB) whether needed or not; workers get 3KB beads guidance duplicated from bd prime; session resume context always injected regardless of session type.

**Next:** Implement role-aware injection: skip orchestrator skill for workers, skip session resume for spawned agents, deduplicate beads guidance.

**Promote to Decision:** recommend-yes - Establishes baseline for context injection architecture; findings support Option A in epic (hooks for manual, spawn context for spawned).

---

# Investigation: Audit SessionStart Hooks for Claude Code

**Question:** What does each SessionStart hook inject, how much context does it consume, and how do hooks detect spawned vs manual sessions?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-inv-audit-sessionstart-hooks-16jan-b4a3
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Epic:** `.orch/epics/context-injection-architecture.md` (Probe 1)

---

## Findings

### Finding 1: Hook Registration and Execution Order

**Evidence:** All 7 SessionStart hooks are registered in `~/.claude/settings.json`:

```json
"SessionStart": [
  { "command": "$HOME/.claude/hooks/session-start.sh" },
  { "command": "$HOME/.orch/hooks/load-orchestration-context.py" },
  { "matcher": "", "command": "bd prime" },
  { "command": "$HOME/.claude/hooks/inject-orch-patterns.sh" },
  { "command": "$HOME/.claude/hooks/agentlog-inject.sh" },
  { "command": "$HOME/.claude/hooks/usage-warning.sh" },
  { "command": "$HOME/.orch/hooks/reflect-suggestions-hook.py" }
]
```

**Source:** `~/.claude/settings.json` (lines via jq '.hooks.SessionStart')

**Significance:** Execution order is deterministic. All hooks run for every session unless they exit early based on conditions. No matcher pattern filtering - hooks must self-filter.

---

### Finding 2: Hook Output Size Matrix

**Evidence:** Tested each hook by piping simulated input and measuring output:

| # | Hook | File Size | Output (bytes) | Est. Tokens | Condition |
|---|------|-----------|----------------|-------------|-----------|
| 1 | session-start.sh | 5.3 KB | 4,246 | ~1,060 | `orch session resume --check` succeeds |
| 2 | load-orchestration-context.py | 16 KB | 93,631 | ~23,408 | Not spawned agent (CLAUDE_CONTEXT unset) |
| 3 | bd prime | CLI | 2,961 | ~740 | Always (beads installed) |
| 4 | inject-orch-patterns.sh | 742 B | 0 | 0 | CWD contains `/.orch/` AND patterns file exists |
| 5 | agentlog-inject.sh | 907 B | 0 | 0 | `.agentlog/` exists AND has errors |
| 6 | usage-warning.sh | 1.7 KB | 0 | 0 | Claude Max usage > 80% |
| 7 | reflect-suggestions-hook.py | 4.8 KB | 538 | ~135 | `~/.orch/reflect-suggestions.json` exists |

**Worst-case total: ~101,376 bytes (~25,344 tokens)**
**Typical manual session: ~100K bytes (~25K tokens)**
**Typical spawned worker: ~3K bytes (~750 tokens)** (only bd prime runs)

**Source:** Bash commands testing each hook with `echo '{"cwd":"...","source":"startup"}' | <hook> | wc -c`

**Significance:** The load-orchestration-context.py hook dominates context consumption at 93% of worst-case total. This single hook is responsible for the "context fills up fast" symptom.

---

### Finding 3: Spawn Detection Mechanisms

**Evidence:** Only ONE hook explicitly detects spawned agents:

**load-orchestration-context.py (lines 436-447, 450-456):**
```python
def is_spawned_agent():
    ctx = os.environ.get('CLAUDE_CONTEXT', '')
    return ctx in ('worker', 'orchestrator', 'meta-orchestrator')

def main():
    if is_spawned_agent():
        sys.exit(0)  # Skip for spawned agents
```

**Other hooks have NO spawn detection:**
- session-start.sh - Runs for ALL sessions, injects session resume
- bd prime - Runs for ALL sessions
- reflect-suggestions-hook.py - Runs if suggestions file exists (no session type check)

**Source:** Code review of all 7 hook files

**Significance:** Spawned agents (workers) correctly skip the orchestrator skill via CLAUDE_CONTEXT check, but still receive session resume context (wrong - they have SPAWN_CONTEXT.md) and bd prime guidance (duplicated in SPAWN_CONTEXT.md beads section).

---

### Finding 4: Content Overlap with SPAWN_CONTEXT.md

**Evidence:** Comparison of hook output vs SPAWN_CONTEXT.md template (`pkg/spawn/context.go`):

| Content | Hook Source | SPAWN_CONTEXT.md | Overlap? |
|---------|-------------|------------------|----------|
| Session resume/handoff | session-start.sh | Not included | Workers don't need this |
| Orchestrator skill | load-orchestration-context.py | Not included (separate) | N/A (role separation) |
| Beads workflow guidance | bd prime (~3KB) | `## BEADS PROGRESS TRACKING` section | **DUPLICATED** |
| Beads commands | bd prime | `bd comment`, `bd close` examples | **DUPLICATED** |
| Orch command patterns | inject-orch-patterns.sh | Not included | N/A (rarely fires) |
| Recent errors | agentlog-inject.sh | Not included | N/A (conditional) |
| Usage warnings | usage-warning.sh | Not included | N/A (conditional) |
| Reflection suggestions | reflect-suggestions-hook.py | Not included | N/A (conditional) |

**Key duplication:** Beads guidance appears in:
1. `bd prime` output (2,961 bytes)
2. SPAWN_CONTEXT.md template (embedded beads examples)
3. Orchestrator skill (embedded in Dynamic State section)

**Source:** Comparison of hook outputs vs `pkg/spawn/context.go:40-304` (SpawnContextTemplate)

**Significance:** Workers receive beads guidance from both bd prime hook AND SPAWN_CONTEXT.md, wasting ~740 tokens on duplication.

---

### Finding 5: load-orchestration-context.py Content Breakdown

**Evidence:** The 93KB output breaks down as:

| Component | Size (approx) | Purpose |
|-----------|---------------|---------|
| Orchestrator skill | 86,451 bytes | Full SKILL.md content |
| Beads workflow section | ~3,000 bytes | Calls `bd prime --full` |
| Active agents | ~500 bytes | Calls `orch status --json` |
| Recent knowledge (kn) | Variable | Calls `kn recent --n 10` |
| README summary | Variable | Reads `.orch/README.md` |
| ROADMAP priorities | Variable | Reads `.orch/ROADMAP.md` |
| Pending decisions | Variable | Scans decisions with Status: Proposed |
| Beads context | Variable | Calls `bd ready --json`, `bd blocked --json` |

**Source:** `~/.orch/hooks/load-orchestration-context.py` lines 20-541

**Significance:** The orchestrator skill (86KB / 86% of hook output) is loaded for ALL manual sessions. This is appropriate for orchestrators but would be devastating for workers without the spawn detection.

---

### Finding 6: Conditional Hook Analysis

**Evidence:** 4 of 7 hooks are conditional and rarely fire:

| Hook | Fires When | Current State |
|------|------------|---------------|
| inject-orch-patterns.sh | CWD is inside `.orch/` AND `~/.orch/docs/orch-command-patterns.md` exists | Patterns file is EMPTY (0 bytes) |
| agentlog-inject.sh | `.agentlog/` exists AND `agentlog prime` returns errors | No .agentlog in most projects |
| usage-warning.sh | Claude Max usage > 80% | Depends on account usage |
| reflect-suggestions-hook.py | `~/.orch/reflect-suggestions.json` exists | Daemon must have run recently |

**Source:** Code analysis and file system checks

**Significance:** These conditional hooks add negligible overhead in practice. Focus optimization efforts on the always-running hooks: session-start.sh, load-orchestration-context.py, bd prime.

---

## Synthesis

**Key Insights:**

1. **load-orchestration-context.py is the elephant** - 93% of worst-case context usage. It correctly skips for spawned agents via CLAUDE_CONTEXT, but this creates a binary choice: orchestrator gets everything, worker gets nothing from this hook.

2. **Session resume is role-blind** - session-start.sh injects session handoff for ALL sessions including spawned workers who have their own SPAWN_CONTEXT.md with task context. This is wrong content for workers.

3. **Beads guidance is triple-redundant** - Appears in bd prime (always), orchestrator skill (manual sessions), and SPAWN_CONTEXT.md (spawned sessions). At minimum ~740 tokens wasted per spawned worker.

4. **Spawn detection exists but is incomplete** - CLAUDE_CONTEXT env var is the mechanism, but only load-orchestration-context.py uses it. Other hooks need to adopt this pattern.

**Answer to Investigation Question:**

Each SessionStart hook injects:
- **session-start.sh**: ~4KB session resume/handoff context (conditional on previous session)
- **load-orchestration-context.py**: ~93KB orchestrator skill + dynamic state (only for non-spawned)
- **bd prime**: ~3KB beads workflow guidance (always)
- **inject-orch-patterns.sh**: 0KB (patterns file empty)
- **agentlog-inject.sh**: 0KB (no errors typically)
- **usage-warning.sh**: 0KB (usage usually <80%)
- **reflect-suggestions-hook.py**: ~0.5KB reflection suggestions (conditional)

Total worst-case is ~101KB (~25K tokens). The spawn detection mechanism is CLAUDE_CONTEXT env var, used only by load-orchestration-context.py.

---

## Structured Uncertainty

**What's tested:**

- ✅ Hook output sizes measured by piping simulated JSON input and counting bytes
- ✅ CLAUDE_CONTEXT spawn detection verified by reading load-orchestration-context.py source
- ✅ settings.json hook registration confirmed via jq
- ✅ bd prime output size verified: 2,961 bytes
- ✅ Orchestrator skill size verified: 86,451 bytes

**What's untested:**

- ⚠️ Actual token counts (estimated at ~4 chars/token)
- ⚠️ Hook execution timing/latency
- ⚠️ Whether CLAUDE_CONTEXT is set correctly by all spawn paths
- ⚠️ Impact of removing bd prime for spawned workers (already in SPAWN_CONTEXT)

**What would change this:**

- Finding would be wrong if CLAUDE_CONTEXT is not reliably set for spawned agents
- Finding would be wrong if hooks have async execution (measured serial execution assumed)
- Overlap analysis would be wrong if SPAWN_CONTEXT.md template changed since this audit

---

## Implementation Recommendations

**Purpose:** Address context injection bloat identified in this audit.

### Recommended Approach: Role-Aware Hook Filtering

**Hook modifications needed:**

1. **session-start.sh** - Add CLAUDE_CONTEXT check to skip for spawned agents
2. **bd prime** - Consider skipping when SPAWN_CONTEXT.md already includes beads guidance
3. **No changes needed** for load-orchestration-context.py (already has spawn detection)

**Implementation sequence:**
1. Add CLAUDE_CONTEXT check to session-start.sh (quick win, ~5 min)
2. Deduplicate beads guidance between bd prime and SPAWN_CONTEXT.md (moderate effort)
3. Consider lazy-loading orchestrator skill via on-demand skill invocation (larger architectural change)

### Alternative Approaches Considered

**Option B: Disable all hooks for spawned sessions**
- **Pros:** Clean separation, spawn context is authoritative
- **Cons:** Loses useful conditional hooks (agentlog errors, usage warnings)
- **When to use instead:** If context budgets are extremely tight

**Option C: Migrate hooks to spawn context machinery**
- **Pros:** Single injection mechanism, full visibility
- **Cons:** Major architectural change, manual sessions lose dynamic state
- **When to use instead:** Long-term if hooks prove unmanageable

**Rationale for recommendation:** Role-aware filtering preserves useful hooks while eliminating wrong/duplicate content. Minimal code changes with measurable impact.

---

## References

**Files Examined:**
- `~/.claude/settings.json` - Hook registration
- `~/.claude/hooks/session-start.sh` - Session resume injection
- `~/.orch/hooks/load-orchestration-context.py` - Main orchestrator context
- `~/.claude/hooks/inject-orch-patterns.sh` - Orch patterns (inactive)
- `~/.claude/hooks/agentlog-inject.sh` - Error injection
- `~/.claude/hooks/usage-warning.sh` - Usage warnings
- `~/.orch/hooks/reflect-suggestions-hook.py` - Reflection suggestions
- `~/.claude/skills/orchestrator/SKILL.md` - Orchestrator skill (86KB)
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template

**Commands Run:**
```bash
# Measure hook outputs
echo '{"cwd":"...","source":"startup"}' | ~/.orch/hooks/load-orchestration-context.py | wc -c

# Check orchestrator skill size
wc -c ~/.claude/skills/orchestrator/SKILL.md

# Check bd prime output
bd prime --full | wc -c

# Verify hook registration
jq '.hooks.SessionStart' ~/.claude/settings.json
```

**Related Artifacts:**
- **Epic:** `.orch/epics/context-injection-architecture.md` - Parent epic for this probe
- **Constraint:** ORCH_WORKER=1 for skipping orchestrator skill (from kb context)

---

## Investigation History

**2026-01-16 12:20:** Investigation started
- Initial question: What does each SessionStart hook inject and how big is it?
- Context: Probe 1 for context-injection-architecture epic

**2026-01-16 12:35:** Hook registration confirmed
- Found 7 hooks in settings.json SessionStart array
- Discovered cdd-hooks.json is separate/legacy mechanism

**2026-01-16 12:45:** Output sizes measured
- load-orchestration-context.py identified as main contributor (93KB)
- Spawn detection via CLAUDE_CONTEXT confirmed

**2026-01-16 13:00:** Investigation completed
- Status: Complete
- Key outcome: 25K token injection worst-case, single hook responsible for 93%

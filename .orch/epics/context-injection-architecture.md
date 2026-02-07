# Epic: Context Injection Architecture

**Phase:** 🟡 Forming
**Created:** 2026-01-16
**Owner:** Dylan + Orchestrator

---

## Problem Statement

Context injection for AI sessions is fragmented across multiple systems with no coherent architecture:

1. **Two session models** - spawned orchestrators vs manual `claude` invocation
2. **Two spawn backends** - Claude Code vs OpenCode with different mechanisms
3. **Two role types** - orchestrators vs workers needing different context
4. **Two injection mechanisms** - hooks (opaque) vs SPAWN_CONTEXT.md (visible)

Result: Sessions fill context with duplicated/wrong content, hooks are invisible, workers get orchestrator context, and there's no observability into what's being injected.

---

## Symptoms Observed

- [ ] Context fills up fast in spawned orchestrator sessions (12% remaining after minimal work)
- [ ] 7 SessionStart hooks inject ~25K+ tokens, partially duplicated
- [ ] Hooks inject content invisibly - no way to know what ran or what was added
- [ ] Duplicate content between hooks and SPAWN_CONTEXT.md
- [ ] Workers receiving session resume context (wrong role)
- [ ] Workers receiving orchestrator skill (wrong role)
- [ ] Beads guidance duplicated in `bd prime` hook AND orchestrator skill
- [ ] Two parallel systems (Claude Code vs OpenCode) behave entirely differently

---

## Questions to Answer

### Architecture Questions (must answer before implementing)

| # | Question | Status | Answer |
|---|----------|--------|--------|
| 1 | Should we migrate from hooks to spawn context machinery? | ❓ | |
| 2 | How do we detect spawned vs manual session? | ✅ | CLAUDE_CONTEXT env var (worker/orchestrator/meta-orchestrator) |
| 3 | What content should orchestrators get vs workers? | ❓ | |
| 4 | What content should manual sessions get vs spawned? | ❓ | |
| 5 | Who is authoritative for shared content (e.g., beads guidance)? | ❓ | |
| 6 | Should orchestrator skill be eager (hook) or lazy (on-demand)? | ❓ | |
| 7 | What's the ideal context budget for startup overhead? | ❓ | |

### Discovery Questions (need investigation)

| # | Question | Status | Answer |
|---|----------|--------|--------|
| A | What does each SessionStart hook inject? (full audit) | ✅ | See Probe 1 results below |
| B | What does OpenCode inject at session start? | ✅ | ~4KB direct, skill via file ref (See Probe 2) |
| C | What does SPAWN_CONTEXT.md contain for each role? | ✅ | Clean separation (See Probe 3) |
| D | Which injected content is actually used vs ignored? | ❓ | |
| E | What context do workers actually need? | ❓ | |

---

## Current Understanding

### Injection Mechanisms

**Claude Code (manual sessions via `claude` CLI):**
- SessionStart hooks fire automatically
- 7 hooks identified: session-start.sh, load-orchestration-context.py, bd prime, inject-orch-patterns.sh, agentlog-inject.sh, usage-warning.sh, reflect-suggestions-hook.py
- No visibility into what ran or what was injected
- ~25K+ tokens injected

**Claude Code (spawned via `orch spawn --backend claude`):**
- SPAWN_CONTEXT.md created with explicit content
- SessionStart hooks ALSO fire (duplication risk)
- Tmux window created

**OpenCode (spawned via `orch spawn` default):**
- SPAWN_CONTEXT.md created
- Different plugin system (session-resume.js, etc.)
- HTTP API session management
- Unknown overlap with Claude Code hooks

### Role Boundaries (Current State - Unclear)

| Content | Orchestrator | Worker | Manual |
|---------|--------------|--------|--------|
| Orchestrator skill | ✅ intended | ❌ leaking | ? |
| Session resume | ✅ intended | ❌ leaking | ✅ intended |
| Beads guidance | ✅ intended | ✅ intended | ✅ intended |
| Worker skill | ❌ | ✅ intended | ❌ |

---

## Probes Needed

### Probe 1: Hook Audit
**Question:** What does each SessionStart hook inject?
**Method:** Run each hook in isolation, capture output, measure size
**Deliverable:** Table of hook → content → size → overlap with spawn context

### Probe 2: OpenCode Plugin Audit
**Question:** What does OpenCode inject at session start?
**Method:** Audit ~/.config/opencode/plugin/ and session startup
**Deliverable:** Comparison table: Claude Code hooks vs OpenCode plugins

### Probe 3: Spawn Context Audit
**Question:** What does SPAWN_CONTEXT.md contain for orchestrator vs worker?
**Method:** Generate sample contexts for each role, compare
**Deliverable:** Role-specific content matrix

### Probe 4: Usage Analysis
**Question:** Which injected content is actually referenced in sessions?
**Method:** Analyze transcripts for what content gets used vs ignored
**Deliverable:** Used vs unused content breakdown

---

## Probe Results

### Probe 1: Hook Audit (COMPLETE)

**Investigation:** `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md`

**Hook Output Matrix:**

| # | Hook | Output (bytes) | Est. Tokens | Condition |
|---|------|----------------|-------------|-----------|
| 1 | session-start.sh | 4,246 | ~1,060 | Session resume exists |
| 2 | load-orchestration-context.py | **93,631** | **~23,408** | Not spawned (CLAUDE_CONTEXT unset) |
| 3 | bd prime | 2,961 | ~740 | Always |
| 4 | inject-orch-patterns.sh | 0 | 0 | CWD in `.orch/` + patterns file exists |
| 5 | agentlog-inject.sh | 0 | 0 | `.agentlog/` has errors |
| 6 | usage-warning.sh | 0 | 0 | Max usage > 80% |
| 7 | reflect-suggestions-hook.py | 538 | ~135 | Suggestions file exists |

**Totals:**
- **Worst-case (manual session):** ~101KB (~25K tokens)
- **Typical spawned worker:** ~3KB (~750 tokens) - only bd prime runs

**Key Findings:**

1. **load-orchestration-context.py is 93% of context consumption** - The orchestrator skill (86KB) dominates
2. **Spawn detection exists via CLAUDE_CONTEXT** - But only load-orchestration-context.py uses it
3. **session-start.sh has NO spawn detection** - Workers receive session resume (wrong content)
4. **Beads guidance is triple-duplicated:**
   - bd prime (~3KB)
   - Orchestrator skill (embedded)
   - SPAWN_CONTEXT.md (embedded)
5. **4 hooks rarely fire** - inject-orch-patterns.sh, agentlog-inject.sh, usage-warning.sh, reflect-suggestions-hook.py

**Quick Wins Identified:**
1. Add CLAUDE_CONTEXT check to session-start.sh (~5 min fix)
2. Deduplicate beads guidance between sources
3. Consider lazy-loading orchestrator skill on-demand

### Probe 2: OpenCode Plugin Audit (COMPLETE)

**Investigation:** `.kb/investigations/2026-01-16-inv-audit-opencode-session-start-injection.md`

**Key Finding:** OpenCode is architecturally leaner (~4KB direct injection vs ~25KB for Claude Code)

**Why the difference:**
- **OpenCode:** Loads orchestrator skill via `config.instructions` (file path reference)
- **Claude Code:** Injects full skill content via hook output

**Plugin inventory:**
- 4 global plugins: session-resume.js, guarded-files.ts, session-compaction.ts, friction-capture.ts
- 4 project plugins: session-context.ts, agentlog-inject.ts, usage-warning.ts, bd-close-gate.ts

**Worker Detection:**
| System | Mechanism |
|--------|-----------|
| OpenCode | ORCH_WORKER env var + SPAWN_CONTEXT.md presence |
| Claude Code | CLAUDE_CONTEXT env var |

### Probe 3: SPAWN_CONTEXT.md Role Audit (COMPLETE)

**Investigation:** `.kb/investigations/2026-01-16-inv-audit-spawn-context-md-content.md`

**Key Finding:** Clean separation across three templates - no duplication concern

**Role Matrix:**

| Content | Worker | Orchestrator | Meta-Orchestrator |
|---------|--------|--------------|-------------------|
| Beads ID + tracking | ✅ | ❌ | ❌ |
| Skill content (embedded) | ✅ | ✅ | ✅ |
| Cross-project info | ❌ | ✅ | ✅ |
| SESSION_HANDOFF.md required | ❌ | ✅ | ❌ |
| /exit behavior | Use it | Wait for level above | Stay interactive |

**Clarification:** bd prime (~3KB general commands) and SPAWN_CONTEXT.md beads section (progress tracking) are **complementary**, not duplicative.

---

## Potential Approaches (Not Yet Decided)

### Option A: Hooks for Manual, Spawn Context for Spawned
- Keep hooks minimal for manual `claude` sessions
- Disable hooks when spawned (detect via marker file)
- SPAWN_CONTEXT.md is authoritative for spawned sessions

### Option B: Migrate Everything to Spawn Context Machinery
- Create `orch session start` for manual sessions that generates context
- Eliminate hooks entirely
- Single injection mechanism

### Option C: Unified Context Service
- Central service that generates appropriate context
- Both hooks and spawn context call the same service
- Role and session-type aware

---

## Mental Model (To Build)

```
┌─────────────────────────────────────────────────────────────┐
│                    SESSION START                             │
├─────────────────────────────────────────────────────────────┤
│  Manual (`claude`)     │  Spawned (orch spawn)              │
│  ─────────────────     │  ────────────────────              │
│  Hooks fire            │  SPAWN_CONTEXT.md created          │
│  No role detection     │  + Hooks also fire (!)             │
│  Full orchestrator     │  Role-specific content             │
│  context always        │  intended                          │
│                        │                                     │
│  → Invisible           │  → Visible in SPAWN_CONTEXT.md     │
│  → Can't control       │  → But hooks still add invisibly   │
└─────────────────────────────────────────────────────────────┘

Current problem: No coherent model for who gets what and why.
```

---

## Progress Log

**2026-01-16 15:00:** All Probes Complete - Moving to Forming
- Probe 2 complete: OpenCode uses file refs (~4KB) vs Claude Code's injection (~25KB)
- Probe 3 complete: SPAWN_CONTEXT.md has clean role separation
- **Key insight:** The problem is Claude Code's hook architecture, not SPAWN_CONTEXT.md
- **Solution direction:** Make Claude Code hooks role-aware (use CLAUDE_CONTEXT consistently)
- Phase changed: 🔴 Probing → 🟡 Forming

**2026-01-16 13:05:** Probe 1 Complete - Hook Audit
- Investigation: `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md`
- **Major finding:** load-orchestration-context.py is 93% of context consumption (23K tokens)
- **Spawn detection:** CLAUDE_CONTEXT env var exists, only used by one hook
- **Role leakage confirmed:** session-start.sh injects resume for workers (wrong)
- **Triple duplication:** beads guidance in bd prime, orchestrator skill, SPAWN_CONTEXT.md
- Answered: Question 2 (spawn detection) and Discovery Question A (hook audit)

**2026-01-16 12:00:** Epic created
- Identified 7 SessionStart hooks
- Found ~25K+ tokens injected at start
- Found duplication: beads guidance in both `bd prime` and orchestrator skill
- Found role leakage: workers getting orchestrator context
- Key architectural question: hooks vs spawn context machinery

---

## Ready Checklist

Before implementing, we must be able to answer:
- [ ] What problem we're solving (not symptoms)
- [ ] Why previous approaches failed (or: why current state exists)
- [ ] What the key constraints are
- [ ] Where the risks live
- [ ] What "done" looks like

**Current status:** Cannot answer most of these. Need probes.

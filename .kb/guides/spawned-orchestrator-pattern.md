# Spawned Orchestrator Pattern

**Purpose:** Guide for hierarchical orchestration using `orch spawn orchestrator` - when to delegate to autonomous orchestrator agents instead of managing work directly.

**Scope:** This guide covers SPAWNED orchestrator agents (hierarchical delegation).

**Created:** 2026-01-13
**Based on:** `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md`

---

## Quick Reference

```bash
# Spawn autonomous orchestrator to handle an epic
orch spawn orchestrator "accomplish goal X" --issue epic-id

# Check on spawned orchestrator progress
orch status

# Review synthesis when orchestrator completes
cat .orch/workspace/{name}/SYNTHESIS.md

# Complete the orchestrator (meta-orchestrator only)
orch complete {workspace-name}
```

**Key distinction:** Spawned orchestrators wait for external completion via `orch complete`. They produce SYNTHESIS.md (same artifact as workers).

---

## The Problem: When One Orchestrator Isn't Enough

**Scenario 1: Concurrent epics**
- Epic A requires sustained orchestration (3+ days)
- Epic B also needs attention
- Single orchestrator can't focus on both simultaneously

**Scenario 2: Parallel investigation clusters**
- Multiple independent problem areas need exploration
- Each requires spawning/coordinating workers
- Single orchestrator would serialize work that could run concurrently

**Scenario 3: Delegation boundary**
- Meta-orchestrator defines strategic goals
- Each goal needs tactical orchestration
- Meta-orchestrator shouldn't drop into tactical execution

**What you need:** Hierarchical orchestration - spawn autonomous orchestrators to handle goals while you (meta-orchestrator) maintain strategic oversight.

---

## The Pattern: Hierarchical Orchestration

### Architecture

```
┌─────────────────────────────┐
│   Meta-Orchestrator         │
│   (Dylan or spawned)        │
│   - Strategic decisions     │
│   - Multiple epic oversight │
│   - Goal definition         │
└──────────┬──────────────────┘
           │ spawns (orch spawn orchestrator)
           ▼
┌─────────────────────────────┐
│   Orchestrator Agent 1      │
│   Goal: Ship auth epic      │
│   - Spawn workers           │
│   - Tactical decisions      │
│   - Produces SYNTHESIS.md   │
└──────────┬──────────────────┘
           │ spawns workers
           ▼
┌─────────────────────────────┐
│   Worker Agents             │
│   - Implement features      │
│   - Fix bugs                │
│   - Produce SYNTHESIS.md    │
└─────────────────────────────┘

           AND (concurrent)

┌─────────────────────────────┐
│   Orchestrator Agent 2      │
│   Goal: Dashboard reliability│
│   - Spawn workers           │
│   - Tactical decisions      │
│   - Produces SYNTHESIS.md   │
└──────────┬──────────────────┘
           │ spawns workers
           ▼
┌─────────────────────────────┐
│   Worker Agents             │
│   - Implement features      │
│   - Fix bugs                │
│   - Produce SYNTHESIS.md    │
└─────────────────────────────┘
```

**Key insight:** Each orchestrator is autonomous. Meta-orchestrator spawns them with goals, they work toward those goals, they signal completion via SYNTHESIS.md. Meta-orchestrator reviews synthesis and completes them.

---

## How It Works

### Spawned Orchestrator Lifecycle

```
1. Meta-orchestrator spawns:
   orch spawn orchestrator "ship auth epic" --issue epic-123

2. System creates:
   - Workspace: .orch/workspace/og-orch-auth-13jan-a1b2/
   - ORCHESTRATOR_CONTEXT.md (skill context + goal)
   - Registry entry: ~/.orch/sessions.json

3. Orchestrator agent works:
   - Reads ORCHESTRATOR_CONTEXT.md
   - Spawns workers via orch spawn
   - Completes workers via orch complete
   - Fills SYNTHESIS.md progressively

4. Orchestrator signals completion:
   - Fills SYNTHESIS.md completely
   - WAITS (doesn't call /exit)

5. Meta-orchestrator reviews and completes:
   - Reads SYNTHESIS.md
   - Reviews spawned work
   - Runs: orch complete og-orch-auth-13jan-a1b2
```

**Critical distinction:** Spawned orchestrators WAIT for level above. They don't self-terminate.

---

## When to Use Spawned Orchestrators

### Use spawned orchestrators when:

1. **Concurrent epic management** - Need multiple goals in flight simultaneously
   ```bash
   orch spawn orchestrator "ship auth epic" --issue auth-epic
   orch spawn orchestrator "improve dashboard reliability" --issue dashboard-epic
   # Both run concurrently, you oversee both
   ```

2. **Delegation boundary** - Goal is clear, execution is tactical
   ```bash
   # You define WHAT (ship feature X), orchestrator decides HOW
   orch spawn orchestrator "implement user settings dashboard"
   ```

3. **Overnight processing** - Goal requires sustained work beyond single session
   ```bash
   # Spawn before EOD, orchestrator works overnight
   orch spawn orchestrator "resolve all P1 bugs in backlog"
   ```

4. **Parallel exploration** - Multiple independent areas need investigation
   ```bash
   orch spawn orchestrator "investigate performance bottlenecks"
   orch spawn orchestrator "audit security vulnerabilities"
   # Each spawns investigation workers independently
   ```

### Use interactive sessions when:

1. **You ARE the orchestrator** - Not delegating, actively coordinating work yourself
2. **Real-time goal refinement** - Goals emerge through conversation, not defined upfront
3. **Single-threaded focus** - One epic at a time, no concurrency needed

**Decision tree:**

```
Are you delegating orchestration to an agent?
├─ YES → Use orch spawn orchestrator
│         (Hierarchical delegation)
└─ NO → You're working interactively
          (Start with bd ready, orch status)
```

---

## SYNTHESIS.md: Orchestrator Completion Artifact

Spawned orchestrators produce SYNTHESIS.md (same artifact as workers), filled progressively during work.

**Sections:**
- **TLDR:** One-sentence summary of session accomplishments
- **Spawns:** Table of all agents spawned (workspace, skill, outcome)
- **Evidence:** Concrete findings with sources
- **Knowledge:** Insights and learnings
- **Friction:** What was harder than expected
- **Focus Progress:** How far did we get on stated goal
- **Next:** Recommended follow-up actions
- **Unexplored Questions:** What wasn't addressed

**Key behavior:** Fill AS YOU WORK (progressive documentation). Context decays — don't defer to end.

---

## Common Patterns

### Pattern 1: Epic Delegation

**Scenario:** Meta-orchestrator has multiple epics in backlog

```bash
# Review epics
bd list --type epic --status open

# Spawn orchestrator for each epic
orch spawn orchestrator "ship user authentication epic" --issue auth-epic
orch spawn orchestrator "improve test coverage epic" --issue testing-epic

# Monitor progress
orch status

# Review handoffs when complete
orch complete og-orch-auth-13jan-a1b2
orch complete og-orch-testing-13jan-c3d4
```

### Pattern 2: Parallel Investigation

**Scenario:** Multiple problem areas need exploration

```bash
# Spawn orchestrators for each area
orch spawn orchestrator "investigate dashboard performance bottlenecks"
orch spawn orchestrator "audit security vulnerabilities in auth flow"

# Each orchestrator spawns investigation workers independently
# Meta-orchestrator reviews synthesis from both
```

### Pattern 3: Overnight Batch

**Scenario:** Sustained work beyond single session

```bash
# Before EOD
orch spawn orchestrator "clear all P1 bugs in backlog"

# Next morning
orch status  # Check progress
cat .orch/workspace/{name}/SYNTHESIS.md  # Review synthesis
orch complete {workspace-name}  # If complete
```

---

## Completion Protocol

### Spawned Orchestrators (External Completion)

**What orchestrator does:**
1. Fill SYNTHESIS.md completely
2. WAIT (stay in session, don't exit)
3. Wait for `orch complete` from level above

**What meta-orchestrator does:**
1. Notice synthesis is complete (via orch status or monitoring)
2. Read SYNTHESIS.md
3. Review spawned work quality
4. Run `orch complete {workspace-name}`

**Orchestrator MUST NOT:**
- Call `/exit` (would close session before meta-orchestrator reviews)
- Self-complete (violates hierarchical model)

---

## Integration with Meta-Orchestrator Role

### Frame Shift

**Meta-orchestrator perspective:**
- WHICH goals to pursue (strategic)
- WHICH orchestrators to spawn (resource allocation)
- WHICH handoffs to review first (prioritization)
- WHETHER work meets goals (quality assessment)

**Orchestrator perspective:**
- HOW to accomplish goal (tactical)
- HOW to break into spawnable work (decomposition)
- HOW to coordinate workers (execution)
- HOW to synthesize results (synthesis)

**The shift:** Moving from "should I do X?" (orchestrator frame) to "should someone do X?" (meta-orchestrator frame)

### WHICH vs HOW Test

```
"Should I fix this bug?" → Orchestrator question (doing)
"Should we fix this class of bugs?" → Meta-orchestrator question (prioritizing)

"How do I implement feature X?" → Orchestrator question (execution)
"Which features should we implement?" → Meta-orchestrator question (strategy)
```

---

## Common Problems

### "Spawned orchestrator tried to self-terminate"

**Cause:** Confusion about completion model

**Fix:** Spawned orchestrators write SYNTHESIS.md and WAIT. ORCHESTRATOR_CONTEXT.md explicitly states: "When done, fill SYNTHESIS.md and WAIT for level above to run orch complete."

### "Meta-orchestrator doing tactical work (level collapse)"

**Cause:** Vague goals cause exploration mode which leads to hands-on debugging

**Fix:** Provide specific goals with:
- Action verbs (ship, implement, resolve, investigate)
- Concrete deliverables (epic complete, all P1 bugs closed)
- Success criteria (tests pass, dashboard loads <2s)

**Example:**
- ❌ "Work on auth"
- ✅ "Ship auth epic: JWT refresh, logout, session management"

### "How do I know when to spawn orchestrator vs do it myself?"

**Use the delegation test:**

1. **Is the goal defined?** If yes, can delegate. If no, refine first.
2. **Do I need to stay hands-on?** If yes, interactive. If no, spawn.
3. **Is this one epic or many?** If many, spawn orchestrators for each.
4. **Can this run while I work on something else?** If yes, spawn.

**When unclear:** Default to spawning. You can always `orch complete` early if it's not working.

### "Multiple orchestrators stepped on each other"

**Cause:** Spawned orchestrators with overlapping goals

**Prevention:**
- Define clear, non-overlapping goals per orchestrator
- Use different projects/directories when possible
- Review handoffs to detect overlap before spawning more

**Example:**
- ❌ Both orchestrators working on "auth" (collision likely)
- ✅ One on "auth implementation", one on "auth testing" (clean boundary)

---

## Troubleshooting

**Check orchestrator status:**
```bash
orch status  # Shows all active orchestrators
cat ~/.orch/sessions.json  # Raw session registry
```

**Check synthesis progress:**
```bash
# See if orchestrator has filled synthesis
cat .orch/workspace/{name}/SYNTHESIS.md
```

**Check spawned workers:**
```bash
# See what orchestrator has spawned
orch status | grep "spawned by"
```

**Abandon stuck orchestrator:**
```bash
# If orchestrator is stuck/infinite loop
orch abandon {workspace-name}
```

---

## Key Decisions

These are settled. Don't re-investigate:

- **Spawned orchestrators produce SYNTHESIS.md** - Same artifact as workers, unified completion verification.
- **Spawned orchestrators wait for external completion** - Hierarchical model requires level above to complete. No self-termination.
- **Beads tracking inappropriate for orchestrators** - Session registry replaces beads. Orchestrators manage sessions, not issues.
- **Tmux default for orchestrator spawns** - Orchestrators need visibility; workers default to headless.

---

## References

- **Investigation:** `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` - Complementary mechanisms analysis
- **Decision:** `.kb/decisions/2026-01-19-remove-session-handoff-machinery.md` - Session handoff removal rationale
- **Source code:** `pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md template
- **Source code:** `pkg/session/registry.go` - Session registry
- **Source code:** `cmd/orch/complete_cmd.go` - Completion flow

---

## Examples

### Example 1: Concurrent Epic Work

```bash
# Meta-orchestrator (Dylan) has 3 open epics
bd list --type epic --status open
# auth-epic: User authentication system
# dashboard-epic: Performance dashboard
# testing-epic: Test coverage improvements

# Spawn orchestrator for each epic
orch spawn orchestrator "ship auth epic: JWT refresh, logout, session mgmt" --issue auth-epic
orch spawn orchestrator "ship dashboard epic: <2s load, real-time updates" --issue dashboard-epic
orch spawn orchestrator "ship testing epic: 80% coverage, integration tests" --issue testing-epic

# All three run concurrently
# Meta-orchestrator monitors progress via orch status
# Reviews handoffs as they complete
# Synthesizes across epics (e.g., auth affects dashboard)
```

### Example 2: Delegation with Refinement

```bash
# Unclear goal initially
orch spawn design-session "explore auth architecture options"
# Design session produces decision document

# Now goal is clear, delegate to orchestrator
orch spawn orchestrator "implement OAuth2 flow per design decision" --issue auth-epic

# Orchestrator spawns workers:
# - feature-impl for OAuth implementation
# - systematic-debugging for integration issues
# - investigation for third-party library evaluation
```

### Example 3: Overnight Processing

```bash
# Friday 5pm - spawn orchestrator for weekend batch work
orch spawn orchestrator "clear all P1 bugs: auth, dashboard, API"

# Monday 9am - check progress
orch status
# og-orch-p1-bugs-10jan-a1b2: running, 6 workers spawned

# Read synthesis to see what's done
cat .orch/workspace/og-orch-p1-bugs-10jan-a1b2/SYNTHESIS.md

# Complete if done
orch complete og-orch-p1-bugs-10jan-a1b2
```

---

## History

- **2026-02-06:** Updated for session handoff removal - SESSION_HANDOFF.md → SYNTHESIS.md, removed orch session start/end references
- **2026-01-13:** Created from architect analysis (orch-go-lvrzc) - hierarchical orchestration pattern documented

## Summary (D.E.K.N.)

**Delta:** 4 orchestrator sessions on 2025-12-28 exhibited three failure modes: (1) stale binary inheritance across sessions causing 30+ min debugging the same fixed bug, (2) documentation/context gaps where features existed but weren't surfaced, (3) circular progress where sessions re-discovered what prior sessions had already found.

**Evidence:** Git log shows Session A fixed serve.go at 12:23:58 but Session B (started 12:43:14) ran stale binary missing that fix. Investigation artifacts show 12+ orch commands undocumented. Prior investigation documented port mismatch (3333 vs 3348) that would have confused agents.

**Knowledge:** The root cause is "features exist but aren't surfaced at session start." Stale binary detection exists (`orch version --source`), server context exists (GenerateServerContext), but neither is injected into orchestrator SessionStart. Documentation drift is systematic - as features are added, they're not added to session context.

**Next:** Implement 3 systemic fixes: (1) SessionStart hook adds stale binary warning, (2) Orchestrator sessions get server context like workers do, (3) `orch lint --skills` catches documentation drift. Priority-ranked action items created.

---

# Investigation: Post-Mortem Orchestrator Session Inefficiency on 2025-12-28

**Question:** What caused 4 orchestrator sessions to experience circular progress and wasted effort, and what systemic improvements would prevent this pattern?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** og-inv-post-mortem-orchestrator-28dec
**Phase:** Complete
**Next Step:** None - action items created
**Status:** Complete

---

## Sessions Analyzed

| Session ID | Title | Time | Key Issue |
|------------|-------|------|-----------|
| ses_49913ff06ffeqSmBkMgHVBSkbz | Reading dashboard status mismatch investigation | ~13:23 | SSE proxy missing header, status logic divergence |
| ses_499281294ffe8MF9wyRMa1h3Hs | Investigating orchestrator session sequence issue | ~13:36 | Circular progress analysis itself |
| ses_4994c8ebfffe9lrn4E9IH1yKp7 | Reviewing orch-go session issues | ~12:56 | Stale binary discovery |
| ses_4996eb903ffeIzpHmmThxUIm4j | Headless spawn HTTP API migration | ~12:19 | Directory header bug, serve.go fix |

---

## Findings

### Finding 1: Stale Binary Inheritance Caused 30+ Minute Debugging Loop

**Evidence:** Timeline reconstruction from git log shows:
- Session A (ses_4996...) fixed serve.go at commit d948e5d6 (12:23:58)
- Session A did NOT run `make install`
- Session B (ses_4994...) started at 12:43:14 - only 6 minutes later
- Session B's binary was from c06db83c, MISSING d948e5d6 fix
- Session B spent 22+ minutes debugging "sessions died silently" - a phantom problem
- Session B's eureka moment: comparing `go run ./cmd/orch status` vs `orch status`

The commit that fixed this (f0d8b823 - auto-rebuild) was implemented by Session B *after* discovering it was the victim of the very problem it was now solving.

**Source:**
- `git log --oneline --since="2025-12-28 12:00" --until="2025-12-28 14:00"`
- `.kb/investigations/2025-12-28-inv-circular-progress-between-orchestrator-sessions.md`

**Significance:** ~30 minutes wasted on a problem that was already fixed. The stale binary is "self-hiding" - you can't see the fix exists when running the stale binary.

---

### Finding 2: Documentation Drift Left 12+ Features Undiscovered

**Evidence:** Investigation `2025-12-28-inv-critical-meta-gap-orch-features.md` found 12+ significant orch commands undocumented in CLAUDE.md and orchestrator skill:

| Command | Purpose | Priority |
|---------|---------|----------|
| `orch servers init/up/down` | launchd dev servers | P0 |
| `orch kb ask` | 5-10 second inline knowledge queries | P0 |
| `orch sessions search` | Full-text search of past sessions | P0 |
| `orch doctor --fix` | Health checks with auto-fix | P0 |
| `orch lint --skills` | Validate CLI references in skills | P1 |
| `orch swarm` | Batch spawning with concurrency | P1 |
| `orch tokens` | Token usage visibility | P1 |

The gap includes entire command groups (servers, sessions, port) that would have helped orchestrators.

**Source:** 
- `.kb/investigations/2025-12-28-inv-critical-meta-gap-orch-features.md`
- Comparison of `orch --help` output vs CLAUDE.md and orchestrator skill

**Significance:** An orchestrator trying to check server health or search past sessions wouldn't know these capabilities existed. The "meta-gap problem" - documentation gaps prevent discovery of documentation gaps.

---

### Finding 3: Port Mismatch in Orchestrator Skill (3333 vs 3348)

**Evidence:** The orchestrator skill referenced port 3333 in 3 places, but `orch serve` runs on port 3348:
- Line 360: "Dashboard at http://127.0.0.1:3333"
- Line 367: "beads-ui at http://127.0.0.1:3333"
- Line 432: "Dashboard visibility at http://127.0.0.1:3333"

Test results:
- `curl http://127.0.0.1:3333/health` → Nothing listening
- `curl http://127.0.0.1:3348/health` → `{"status":"ok"}`

This was fixed in commit cadeb6fb (12:27:28).

**Source:**
- `.kb/investigations/2025-12-28-inv-gaps-exist-session-start-context.md`
- `~/.claude/skills/meta/orchestrator/SKILL.md` (before fix)

**Significance:** Any orchestrator following skill guidance would waste time on wrong URL, then question whether dashboard is running at all.

---

### Finding 4: Three-Layer Visibility Stack With Divergent State

**Evidence:** Investigation `2025-12-28-inv-dashboard-status-mismatch-orch-status-vs-api.md` found:
```
orch status: 6 active (1 running, 5 idle)
API /api/agents: 1 active, 2 idle, 643 completed  
Dashboard: 1 active agent shown
```

Three layers with different state models:
1. **OpenCode sessions** - Source of truth for session state
2. **orch serve API** - Uses time-based heuristics + beads phase
3. **Dashboard** - Uses SSE events, was missing directory header

SSE proxy was missing `x-opencode-directory` header (fixed in 8ccc3af2), causing dashboard to never receive activity events.

**Source:**
- `.kb/investigations/2025-12-28-inv-dashboard-status-mismatch-orch-status-vs-api.md`
- `cmd/orch/serve.go:1054-1056` (pre-fix)

**Significance:** Same agent could appear "running" in CLI but "completed" in dashboard. This caused confusion about whether agents were actually working.

---

### Finding 5: Workers Get Context That Orchestrators Don't

**Evidence:** `pkg/spawn/context.go:858` contains `GenerateServerContext()` which creates a LOCAL SERVERS section for spawned agents:
```go
func GenerateServerContext(projectDir string) string {
    // Returns formatted section with:
    // - Project name, status (running/stopped)
    // - Port list (e.g., web: 5188, api: 3348)
    // - Quick commands (start/stop/open)
}
```

This is injected into SPAWN_CONTEXT.md for workers. Orchestrators have no equivalent - they rely on static CLAUDE.md that lacks runtime details.

**Source:**
- `pkg/spawn/context.go:858-902`
- `.kb/investigations/2025-12-28-inv-gaps-exist-session-start-context.md`

**Significance:** Structural asymmetry: workers get project-specific operational context automatically, orchestrators get static documentation. This explains why orchestrators repeatedly hit "how do I start the web UI?" friction.

---

### Finding 6: Stale Binary Detection Exists But Isn't Surfaced

**Evidence:** `orch version --source` correctly detects binary staleness:
```
$ orch version --source
status: ✓ UP TO DATE
# or when stale:
status: ⚠️  STALE
binary hash:  877943f1
current HEAD: abc12345
rebuild: cd /path && make install
```

The mechanism is complete (compares embedded git hash to HEAD) but never runs automatically. SessionStart hook focuses on workspaces, not operational context.

**Source:**
- `cmd/orch/main.go:115-156` (runVersionSource implementation)
- `~/.claude/hooks/session-start.sh` (no staleness check)

**Significance:** The fix for stale binary detection EXISTS, it just isn't surfaced at session start. Pattern: build feature → forget to add to session context → hit the gap later.

---

## Synthesis

**Key Insights:**

1. **Features Exist, Surfacing Doesn't** - The common pattern across all 4 sessions: Dylan builds features (staleness detection, server context, documentation commands) but doesn't add them to SessionStart context. The features work but orchestrators don't know they exist or they run stale versions that don't have the fix.

2. **Stale Binary is Self-Hiding** - When you run `orch` with a stale binary, you can't see the fix that would show you the problem exists. Session B couldn't see Session A's serve.go fix because it was running pre-fix binary. This creates circular debugging patterns.

3. **Documentation Drift is Systematic** - As orch-go adds features, CLAUDE.md and the orchestrator skill aren't updated. The gap isn't random - it accumulates over time. 12+ commands were undocumented, including entire command groups.

4. **Asymmetric Context Injection** - Workers get project-specific context via code (GenerateServerContext), orchestrators get static documentation. This design choice means orchestrator context lags behind by design.

5. **Three-Layer State Divergence** - OpenCode sessions, orch serve API, and dashboard all have different state models. Fixes to one layer don't propagate to others. Multiple sessions per beads ID add confusion about which session is "the" agent.

**Answer to Investigation Questions:**

1. **What was each session trying to accomplish?**
   - ses_4996...: Fix dashboard not showing spawned agents (HTTP API migration)
   - ses_4994...: Investigate why sessions appeared dead (discovered stale binary)
   - ses_499281...: Analyze the circular progress pattern
   - ses_49913...: Fix dashboard status mismatch

2. **Where did sessions get stuck or go in circles?**
   - Session A fixed serve.go but didn't deploy → Session B ran stale binary → 30 min wasted
   - Session B re-discovered same stale binary problem Session A had conceptually addressed
   - Dashboard showed wrong state because SSE proxy lacked header that other code expected

3. **What knowledge was lost between sessions that caused rework?**
   - Session A's fix wasn't in Session B's binary
   - Neither session knew `orch version --source` could detect staleness
   - Neither session knew about undocumented features that could help

4. **Were there violations of orchestrator principles?**
   - Yes: Session A "completed" without verifying deployment
   - Yes: No SessionStart check for binary freshness
   - Yes: Orchestrators did investigation work that should have been delegated earlier

5. **What systemic improvements would prevent this pattern?**
   - See Implementation Recommendations below

---

## Structured Uncertainty

**What's tested:**

- ✅ Stale binary caused 30 min waste (verified: git log timeline reconstruction)
- ✅ Port 3333 vs 3348 mismatch existed (verified: curl commands)
- ✅ `orch version --source` correctly detects staleness (verified: ran command)
- ✅ 12+ commands undocumented (verified: compared --help to CLAUDE.md)
- ✅ SSE proxy missing header caused visibility issues (verified: code review + fix)

**What's untested:**

- ⚠️ Whether SessionStart staleness check would actually prevent pattern (not implemented yet)
- ⚠️ Whether orchestrator server context injection would help (not implemented yet)
- ⚠️ Whether documentation gaps directly caused session inefficiency (correlation, not causation proven)

**What would change this:**

- If Session A had run `make install`, Session B would not have hit stale binary
- If SessionStart warned about staleness, Session B would have rebuilt immediately
- If orchestrators got server context, web UI startup would be obvious

---

## Implementation Recommendations

### Recommended Approach ⭐

**Three-Tier Prevention System** - Address stale binary, context asymmetry, and documentation drift

**Why this approach:**
- Each tier addresses a distinct failure mode from the post-mortem
- Uses existing mechanisms (SessionStart hooks, GenerateServerContext, lint)
- Progressive improvement - can implement in phases

**Implementation sequence:**

1. **SessionStart Staleness Warning (P0)** - Add check to SessionStart hook
   ```bash
   orch version --source --json | jq -e '.stale' && echo "⚠️ STALE BINARY"
   ```
   - Surfaces existing staleness detection automatically
   - Catches the stale binary problem BEFORE wasting time

2. **Orchestrator Server Context (P1)** - Add server info to SessionStart
   - Call GenerateServerContext() equivalent for orchestrators
   - Surface `orch doctor` summary
   - Shows port numbers, server status, startup commands

3. **Documentation Drift Detection (P2)** - Enhance `orch lint --skills`
   - Compare command catalog to skill/CLAUDE.md content
   - Flag undocumented commands as warnings
   - Run in CI to catch drift before it accumulates

### Alternative Approaches Considered

**Option B: Pre-commit hook enforces `make install`**
- **Pros:** Prevents stale binary at source
- **Cons:** Doesn't help orchestrators who start fresh sessions
- **When to use instead:** As additional layer, not replacement

**Option C: Daily reconciliation daemon**
- **Pros:** Catches drift across all projects
- **Cons:** Reactive not proactive; doesn't help in-session
- **When to use instead:** For overnight cleanup, not prevention

**Rationale for recommendation:** Three-tier approach addresses each failure mode with appropriate mechanism. SessionStart is proactive (catches at session start), server context fills structural gap, lint catches drift before it compounds.

---

### Implementation Details

**What to implement first:**
1. SessionStart staleness check (addresses Finding 1, 6)
2. Orchestrator server context injection (addresses Finding 5)
3. Documentation drift lint rule (addresses Finding 2)

**Things to watch out for:**
- ⚠️ SessionStart hook is shared across projects - changes affect everything
- ⚠️ Staleness check adds latency (~100ms for git hash comparison)
- ⚠️ GenerateServerContext uses config files that may not exist in all projects

**Areas needing further investigation:**
- Should `orch spawn` refuse to run with stale binary?
- Should there be a "session overlap warning" when starting in project with recent changes?
- Why didn't pre-commit hook prevent Session B staleness?

**Success criteria:**
- ✅ SessionStart warns when binary is stale (test: modify code, start session without rebuild)
- ✅ Orchestrators see server/port info at session start (test: new session shows LOCAL SERVERS)
- ✅ `orch lint --skills` catches undocumented commands (test: add command, run lint before documenting)

---

## Priority-Ranked Action Items

| Priority | Action | Addresses | Mechanism |
|----------|--------|-----------|-----------|
| **P0** | Add stale binary warning to SessionStart hook | Finding 1, 6 | Hook calls `orch version --source` |
| **P0** | Create unified status determination in `pkg/state/reconcile.go` | Finding 4 | Extract logic, use in both CLI and API |
| **P1** | Add server context to orchestrator SessionStart | Finding 5 | Inject GenerateServerContext output |
| **P1** | Document missing commands in CLAUDE.md and skill | Finding 2, 3 | Manual update + lint enforcement |
| **P2** | Add `orch lint --skills --check-commands` | Finding 2 | Automated drift detection |
| **P2** | Add session overlap detection | Finding 1 | Warn when starting in project with uncommitted changes |

---

## References

**Investigations Analyzed:**
- `.kb/investigations/2025-12-28-inv-circular-progress-between-orchestrator-sessions.md` - Session A/B timeline
- `.kb/investigations/2025-12-28-inv-dashboard-status-mismatch-orch-status-vs-api.md` - Three-layer divergence
- `.kb/investigations/2025-12-28-inv-gaps-exist-session-start-context.md` - Port mismatch, context asymmetry
- `.kb/investigations/2025-12-28-inv-critical-meta-gap-orch-features.md` - 12+ undocumented commands
- `.kb/investigations/2025-12-28-inv-solve-stale-binary-problem-human.md` - Root causes, symlink solution

**Decisions Created:**
- `.kb/decisions/2025-12-28-stale-binary-solution.md` - Symlink-based install pattern

**Git Commits Analyzed:**
- d948e5d6 (12:23:58) - Session A's serve.go fix
- f0d8b823 (13:14:24) - Session B's auto-rebuild feature
- 8ccc3af2 (13:37:29) - SSE proxy header fix
- cadeb6fb (12:27:28) - Port 3333→3348 fix

---

## Self-Review

- [x] Real test performed (timeline reconstruction from git log)
- [x] Conclusion from evidence (specific commits, timestamps, investigation artifacts)
- [x] Question answered (5 investigation questions addressed with findings)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Discovered Work Check

| Type | Item | Created? |
|------|------|----------|
| **Enhancement** | SessionStart staleness warning | Tracked in action items |
| **Enhancement** | Orchestrator server context injection | Tracked in action items |
| **Enhancement** | Unified status determination | Tracked in action items |
| **Enhancement** | Documentation drift lint | Tracked in action items |

Note: Action items documented in this investigation rather than creating separate beads issues, as orchestrator will triage from this synthesis.

---

## Investigation History

**2025-12-28 ~15:00:** Investigation started
- Initial question: What caused 4 orchestrator sessions to waste effort?
- Context: Dylan frustrated after circular debugging sessions

**2025-12-28 ~15:20:** Analyzed prior investigations
- Found 5 related investigations from same day
- Identified stale binary, documentation drift, context asymmetry patterns

**2025-12-28 ~15:40:** Timeline reconstructed
- Confirmed Session A fix wasn't in Session B binary
- Mapped 30+ minute waste to stale binary inheritance

**2025-12-28 ~16:00:** Investigation completed
- Status: Complete
- Key outcome: Three failure modes identified, three-tier prevention system recommended

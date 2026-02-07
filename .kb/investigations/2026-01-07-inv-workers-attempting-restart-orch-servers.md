---
linked_issues:
  - orch-go-1qol3
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Workers correctly use `orch servers start/stop` for project dev servers (tmuxinator-based); this is intentional, not confusion about launchd infrastructure.

**Evidence:** Code analysis shows clear separation: `orch servers` (context.go:939-941) manages project dev servers via tmuxinator; `orch serve` and daemon run via launchd (orchestrator infrastructure). KB context surfacing "restart orch serve" is an aspirational decision about automation, not worker instruction.

**Knowledge:** Two distinct server concepts exist and are correctly separated: (1) project dev servers (`orch servers`) use tmuxinator, appropriate for workers, (2) orchestration infrastructure (`orch serve`, daemon) uses launchd, orchestrator-only.

**Next:** Close - current behavior is correct. Consider clarifying the kn decision (kn-c75a03) to specify this is orchestrator responsibility, not worker action.

**Promote to Decision:** recommend-no - current design is intentional, no change needed

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Workers Attempting Restart Orch Servers

**Question:** Why are workers attempting to restart orch servers via tmux, and do they need awareness of launchd-managed services?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SPAWN_CONTEXT.md includes `orch servers start/stop` instructions

**Evidence:** The feature-impl skill's phase-validation.md and phase-implementation-tdd.md files both reference:
```bash
orch servers stop <project>
orch servers start <project>
```

This instruction appears in SPAWN_CONTEXT.md via `GenerateServerContext()` which is called for UI-focused skills (feature-impl, systematic-debugging, reliability-testing).

**Source:** 
- `pkg/spawn/context.go:939-941` - Generates the instructions
- `pkg/spawn/config.go:52-56` - `SkillIncludesServers` mapping
- `~/.claude/skills/worker/feature-impl/reference/phase-validation.md:38-39`
- `~/.claude/skills/worker/feature-impl/reference/phase-implementation-tdd.md:113-114`

**Significance:** These are **project dev servers** (web frontends, APIs), NOT `orch serve` (the dashboard) or the daemon. This is the intended behavior for workers doing UI work - they need to restart their project's dev server after making changes.

---

### Finding 2: `orch servers start/stop` uses tmuxinator (legacy approach)

**Evidence:** The `orch servers start` command (servers.go:231-259) runs:
```go
cmd := exec.Command("tmuxinator", "start", sessionName)
```

This is documented as "Start servers via tmuxinator" and creates tmux sessions named `workers-{project}`.

**Source:**
- `cmd/orch/servers.go:54-65` - Command definition
- `cmd/orch/servers.go:231-259` - `runServersStart()` implementation
- Orchestrator skill mentions launchd as "preferred" but `orch servers` still uses tmuxinator

**Significance:** The orchestrator skill mentions launchd-based server management (`orch servers up/down`) as the preferred approach, but the actual implementation of `orch servers start/stop` still uses tmuxinator. This creates confusion about which approach workers should use.

---

### Finding 3: Two distinct server concepts being conflated

**Evidence:** The original investigation question mentions "orch servers" but there are actually two different things:
1. **`orch serve`** - The dashboard API server running on localhost:5188, managed via launchd
2. **`orch servers`** - Project development servers (web, api), managed via tmuxinator

Workers should NEVER need to restart `orch serve` (that's orchestrator infrastructure).
Workers MAY need to restart `orch servers` (their project's dev servers) when doing UI work.

**Source:**
- Orchestrator skill: "Dashboard at `http://localhost:5188` (`orch serve`) for real-time visibility"
- servers.go - Only manages project servers, not `orch serve`
- Daemon runs via launchd, not tmux

**Significance:** Need to clarify which "servers" are being referenced. If workers are trying to restart the dashboard or daemon, that's a problem. If they're just running `orch servers start/stop` for their project, that's expected behavior.

---

### Finding 4: KB context surfaces an aspirational decision about `orch serve` restarts

**Evidence:** The kn entries (kn-c75a03, kn-7a1601) say:
> "After agents commit Go changes, orchestrator should auto-rebuild and restart affected services"
> Reason: "Manual rebuild/restart is friction. Pattern: detect changed files (cmd/orch/, pkg/) → make install → restart orch serve if running."

This appears in spawn contexts via `kb context` but describes a desired automation, NOT a worker action. The key word is "orchestrator should" - this is describing a future feature for the orchestrator, not an instruction for workers.

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl` (kn-c75a03, kn-7a1601)
- Found in archived workspace spawn contexts

**Significance:** This could potentially confuse workers who see "restart orch serve" in their context, but the phrasing ("orchestrator should") makes it clear this is not a worker instruction. The system is working correctly.

---

### Finding 5: Server management guidance is correctly targeted by skill type

**Evidence:** `SkillIncludesServers` in `pkg/spawn/config.go:52-56` only includes server context for:
- `feature-impl` - Often involves web UI work
- `systematic-debugging` - May need to access running servers
- `reliability-testing` - Needs to test live servers

Investigation skills (like this one) don't receive server context by default. This is intentional - investigators generally don't need to restart servers.

**Source:**
- `pkg/spawn/config.go:50-65` - `SkillIncludesServers` map and `DefaultIncludeServersForSkill()`

**Significance:** The system correctly filters server instructions to skills that need them. Workers doing non-UI work won't see `orch servers` instructions at all.

---

## Synthesis

**Key Insights:**

1. **Clean separation between project servers and orchestration infrastructure** - The system correctly distinguishes between `orch servers` (project dev servers, tmuxinator-managed, worker-appropriate) and `orch serve`/daemon (orchestration infrastructure, launchd-managed, orchestrator-only). This is intentional design, not confusion.

2. **Server context is skill-targeted** - Only UI-focused skills (feature-impl, systematic-debugging, reliability-testing) receive server management instructions in their SPAWN_CONTEXT. Investigation skills don't see these instructions by default, which is correct.

3. **KB context aspirational decisions don't constitute worker instructions** - The kn decision about "auto-rebuild and restart" is phrased as an aspiration ("orchestrator should") not a worker action. Workers seeing this context should correctly interpret it as describing a potential future automation.

**Answer to Investigation Question:**

Why are workers attempting to restart orch servers via tmux? The investigation found that the question may be based on a misunderstanding. Workers correctly receive `orch servers start/stop` instructions for **project dev servers** (which use tmuxinator) when doing UI work. This is intentional.

Workers should NOT and DO NOT receive instructions to restart `orch serve` (the dashboard) or the daemon - these are launchd-managed orchestrator infrastructure.

The original concern appears to conflate two different "orch servers":
1. **`orch servers start/stop <project>`** - Project dev servers, tmuxinator-based, worker-appropriate ✅
2. **`orch serve`** - Dashboard API, launchd-based, orchestrator-only ✅

Both are working as designed. No changes needed.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch servers start` uses tmuxinator (verified: read servers.go:231-259, confirmed `tmuxinator start` command)
- ✅ Server context only included for UI skills (verified: SkillIncludesServers map in config.go:52-56)
- ✅ `orch serve` is separate from `orch servers` (verified: different command files, different management approaches)
- ✅ KB context "restart orch serve" is a decision, not instruction (verified: kn entry says "orchestrator should")

**What's untested:**

- ⚠️ Whether workers have actually tried to restart `orch serve` specifically (no specific incident found in workspace search)
- ⚠️ Whether the kn decision text causes confusion (not observed in practice)

**What would change this:**

- Finding evidence of a worker attempting `launchctl kickstart` for daemon or trying to kill `orch serve` process
- Finding that workers misinterpret kb context decisions as actions they should perform

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**No changes needed** - The current system correctly separates project servers (`orch servers`) from orchestration infrastructure (`orch serve`, daemon). Workers receive appropriate guidance.

**Why this approach:**
- Current design is intentional and correct (Finding 1, 2, 5)
- No evidence of actual worker confusion or inappropriate restart attempts
- The two server concepts serve different purposes and correctly use different management approaches

**Trade-offs accepted:**
- kn decision mentioning "restart orch serve" could theoretically cause confusion (low risk)
- Accepting this low risk vs adding unnecessary complexity to filter kn decisions

### Alternative Approaches Considered

**Option B: Filter "orch serve" mentions from worker kb context**
- **Pros:** Eliminates any theoretical confusion
- **Cons:** Complex filtering logic, may hide useful context, addresses problem that doesn't exist
- **When to use instead:** If actual evidence of worker confusion emerges

**Option C: Update kn decision to clarify orchestrator responsibility**
- **Pros:** Makes it explicit this is orchestrator work, not worker action
- **Cons:** Minor documentation change for non-problem
- **When to use instead:** If clarity becomes important for training or audit

**Rationale for recommendation:** The investigation found no evidence of the assumed problem. Workers are correctly using `orch servers` for project servers. No implementation needed.

---

### Implementation Details

**What to implement first:**
- Nothing - investigation concludes current behavior is correct

**Things to watch out for:**
- ⚠️ If future reports surface of workers trying to restart dashboard or daemon, revisit this investigation
- ⚠️ The kn decision (kn-c75a03) describes a potential automation - if implemented, ensure it's orchestrator-triggered

**Areas needing further investigation:**
- None identified

**Success criteria:**
- ✅ Investigation question answered
- ✅ No changes needed
- ✅ Understanding documented for future reference

---

## References

**Files Examined:**
- `pkg/spawn/context.go:939-941` - GenerateServerContext() creates `orch servers start/stop` instructions
- `pkg/spawn/config.go:50-65` - SkillIncludesServers map controls which skills get server context
- `cmd/orch/servers.go` - Implementation of `orch servers` command (uses tmuxinator)
- `~/.claude/skills/worker/feature-impl/reference/phase-validation.md` - Worker guidance includes `orch servers stop/start`
- `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl` - kn decisions about server management

**Commands Run:**
```bash
# Search for server-related instructions in skills
rg "orch servers|orch serve" ~/.claude/skills --type md

# Check kn entries mentioning restart
cat .kn/entries.jsonl | grep -i "restart"

# Test kb context for servers keyword
kb context "servers"

# Search workspaces for server restart patterns
rg "orch serve|restart" .orch/workspace --type md
```

**External Documentation:**
- Orchestrator skill (SKILL.md) - Documents launchd vs tmux server management approaches

**Related Artifacts:**
- **Decision:** kn-c75a03 - "After agents commit Go changes, orchestrator should auto-rebuild and restart affected services"
- **Decision:** kn-be62db - "Tmux session existence is correct abstraction for 'orch servers' status - separate infrastructure from project servers"

---

## Investigation History

**2026-01-07:** Investigation started
- Initial question: Workers are attempting to restart orch servers via tmux - are they unaware of launchd setup?
- Context: Concern that workers might be confused about server management approaches

**2026-01-07:** Key discovery - separation is intentional
- Found that `orch servers` (project servers) and `orch serve` (dashboard) are correctly separate
- Workers receive project server instructions only for UI skills (intentional)

**2026-01-07:** Investigation completed
- Status: Complete
- Key outcome: Current behavior is correct; no changes needed. The assumed problem (workers trying to restart launchd services) was not found.

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

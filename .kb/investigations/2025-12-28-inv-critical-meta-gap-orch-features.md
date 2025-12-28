<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** 12+ orch commands/features are undocumented or poorly documented in CLAUDE.md and orchestrator skill, causing agents to miss critical capabilities.

**Evidence:** Ran `orch --help` and all subcommands, compared systematically against CLAUDE.md and orchestrator skill content.

**Knowledge:** The gap includes entire command groups (servers, sessions, port), utility commands (fetch-md, lint, synthesis, transcript), and important subcommands (kb ask/extract, daemon reflect). Documentation drift is a systematic issue.

**Next:** Create feature-impl issue to update CLAUDE.md and orchestrator skill with missing commands. Prioritize server management and kb ask as high-value gaps.

---

# Investigation: Critical Meta Gap Orch Features

**Question:** What orch commands/features exist but aren't documented in CLAUDE.md or the orchestrator skill?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Complete Command Catalog from --help

**Evidence:** Running `orch --help` reveals 40+ commands across 9 categories:

| Category | Commands |
|----------|----------|
| **Agent Lifecycle** | abandon, clean, complete, resume, send, spawn, status |
| **Monitoring** | monitor, question, serve, tail, tokens, wait |
| **Daemon/Automation** | daemon (run/once/preview/reflect), swarm, work |
| **Strategic** | drift, focus, next |
| **Account/Usage** | account (add/list/remove/switch), usage |
| **Servers** | servers (init/up/down/gen-plist/list/start/stop/attach/open/status) |
| **Sessions** | sessions (list/search/show) |
| **Knowledge** | kb (ask/extract), learn, patterns |
| **Utilities** | completion, doctor, fetch-md, handoff, history, init, lint, logs, port, retries, stale, synthesis, transcript, version |

**Source:** `orch --help` and `orch <cmd> --help` for all 40+ commands

**Significance:** This is the authoritative source of truth. Any command here that isn't documented is a discoverability gap.

---

### Finding 2: Commands Missing from CLAUDE.md

**Evidence:** Comparing CLAUDE.md against the command catalog shows these gaps:

| Command | In CLAUDE.md? | Should Orchestrators Know? |
|---------|---------------|---------------------------|
| `orch servers init` | ❌ No | ✅ Yes - sets up launchd integration |
| `orch servers up/down` | ❌ No | ✅ Yes - manages dev servers via launchd |
| `orch servers gen-plist` | ❌ No | ✅ Yes - generates launchd plists |
| `orch init` | ❌ No (only mentioned tangentially) | ✅ Yes - scaffolds new projects |
| `orch doctor` | ❌ No | ✅ Yes - health checks for services |
| `orch kb ask` | ❌ No | ✅ Yes - quick inline KB queries |
| `orch kb extract` | ❌ No | ✅ Yes - moves artifacts between projects |
| `orch sessions list/search/show` | ❌ No | ✅ Yes - finds past session content |
| `orch fetch-md` | ❌ No | ⚠️ Maybe - utility for web content |
| `orch lint` | ❌ No | ✅ Yes - validates CLAUDE.md limits |
| `orch lint --skills` | ❌ No | ✅ Yes - validates skill CLI references |
| `orch lint --issues` | ❌ No | ✅ Yes - validates beads issues |
| `orch synthesis` | ❌ No | ✅ Yes - activity summary |
| `orch transcript format` | ❌ No | ⚠️ Maybe - converts session exports |
| `orch tokens` | ❌ No | ✅ Yes - token usage visibility |
| `orch history` | ❌ No | ✅ Yes - skill analytics |
| `orch port allocate/list/release` | ❌ No | ✅ Yes - prevents port conflicts |
| `orch version --source` | ❌ No | ⚠️ Maybe - binary staleness check |
| `orch daemon reflect` | ❌ No | ✅ Yes - runs kb reflect for SessionStart |
| `orch swarm` | ❌ No | ✅ Yes - batch spawn with concurrency |

**Source:** Comparison of CLAUDE.md content (222 lines) vs orch --help output

**Significance:** Agents spawned with only CLAUDE.md context will not know about server management, session search, or inline KB queries - all high-value capabilities.

---

### Finding 3: Commands Missing from Orchestrator Skill

**Evidence:** The orchestrator skill (1766 lines) mentions some commands but misses:

| Command | In Orchestrator Skill? | Coverage Quality |
|---------|------------------------|------------------|
| `orch servers init/up/down/gen-plist` | ❌ No | Not mentioned at all |
| `orch doctor` | ❌ No | Not mentioned |
| `orch kb ask` | ❌ No | Not mentioned |
| `orch kb extract` | ❌ No | Not mentioned |
| `orch sessions` | ❌ No | Not mentioned |
| `orch tokens` | ❌ No | Not mentioned |
| `orch history` | ❌ No | Not mentioned |
| `orch synthesis` | ❌ No | Not mentioned |
| `orch lint` | ❌ No | Not mentioned |
| `orch swarm` | ❌ No | Not mentioned |
| `orch port` | ❌ No | Not mentioned |
| `orch daemon reflect` | ❌ No | Not mentioned |
| `orch fetch-md` | ❌ No | Not mentioned |
| `orch transcript` | ❌ No | Not mentioned |
| `orch retries` | ❌ No | Not mentioned |
| `orch stale` | ❌ No | Not mentioned |
| `orch handoff` | ❌ No | Not mentioned |
| `orch version` | ❌ No | Not mentioned |

**Commands that ARE documented well:**
- spawn, complete, review, status, wait, tail, question, send, resume, abandon, clean
- daemon run/preview/once
- account list/switch
- focus, drift, next
- learn, patterns
- monitor, serve

**Source:** Searched orchestrator skill for each command name

**Significance:** The orchestrator skill has good coverage of core agent lifecycle but completely misses server management, session introspection, and several utility commands.

---

### Finding 4: Server Management is a Major Gap

**Evidence:** The `orch servers` command group is sophisticated:

```
orch servers init <project>      - Scan and generate servers.yaml
orch servers up <project>        - Start via launchd/Docker
orch servers down <project>      - Stop servers
orch servers gen-plist <project> - Generate launchd plist files
orch servers list                - Show all projects with ports
orch servers start <project>     - Start via tmuxinator (legacy)
orch servers stop <project>      - Stop servers (legacy)
orch servers attach <project>    - Attach to servers window
orch servers open <project>      - Open in browser
orch servers status              - Show summary
```

Yet neither CLAUDE.md nor the orchestrator skill mentions this. The only reference is a brief mention in CLAUDE.md commands list: "servers list, start, stop, attach, open, status" - missing init, up, down, gen-plist.

**Source:** `orch servers --help`

**Significance:** Server management via launchd is a major operational capability. Agents trying to start dev servers won't know about `orch servers up` or the launchd integration.

---

### Finding 5: kb ask is a Hidden High-Value Feature

**Evidence:** `orch kb ask` provides inline knowledge synthesis:

```
orch kb ask "how should we handle rate limiting?"
orch kb ask "what's our auth pattern?" --save
orch kb ask "config patterns" --global
```

This is a ~5-10 second inline query that saves spawning full investigation agents for quick questions.

**Source:** `orch kb ask --help`

**Significance:** This command could dramatically reduce the need to spawn investigation agents for simple questions. Orchestrators don't know it exists.

---

### Finding 6: Session Introspection is Undocumented

**Evidence:** The `orch sessions` command group provides full-text search of past sessions:

```
orch sessions list --limit 50
orch sessions search "error handling"
orch sessions search --regex "auth.*token"
orch sessions show ses_abc123
```

This allows finding past discussions, decisions, and implementations across all sessions.

**Source:** `orch sessions --help`

**Significance:** When an orchestrator needs to find past context or decisions, they could search sessions directly instead of only relying on kb artifacts.

---

## Synthesis

**Key Insights:**

1. **Documentation Drift is Systematic** - As orch-go added features (servers launchd integration, kb ask, sessions search), documentation wasn't updated. The gap isn't random - it's accumulated over time.

2. **Three Tiers of Documentation Gaps**:
   - **Tier 1 (Critical):** Server management, kb ask, sessions search - actively useful for orchestrators
   - **Tier 2 (Important):** lint, synthesis, tokens, history, port - operational visibility
   - **Tier 3 (Minor):** transcript, fetch-md, version --source - edge case utilities

3. **The Meta-Gap Problem** - An agent specifically investigating context gaps MISSED these features because they searched artifacts that don't document them. This is the recursive problem: documentation gaps prevent discovery of documentation gaps.

**Answer to Investigation Question:**

12+ significant orch features are undocumented in CLAUDE.md and the orchestrator skill. The most critical gaps are:

1. **Server Management** (`orch servers init/up/down/gen-plist`) - launchd integration for dev servers
2. **KB Quick Queries** (`orch kb ask`) - inline knowledge synthesis in 5-10 seconds
3. **Session Search** (`orch sessions search`) - full-text search across all past sessions
4. **Daemon Reflection** (`orch daemon reflect`) - runs kb reflect for SessionStart hook
5. **Linting** (`orch lint`, `orch lint --skills`, `orch lint --issues`) - validation tools
6. **Batch Spawning** (`orch swarm`) - parallel agent spawning with concurrency control
7. **Doctor** (`orch doctor --fix`) - health checks and auto-fix for services
8. **Token Visibility** (`orch tokens`) - detailed token usage per session
9. **Port Management** (`orch port`) - prevents port conflicts across projects
10. **Activity Synthesis** (`orch synthesis`) - summarizes recent commits/issues/investigations

---

## Structured Uncertainty

**What's tested:**

- ✅ All orch commands cataloged via `orch --help` (verified: ran command)
- ✅ Each subcommand's help text captured (verified: ran `orch <cmd> --help`)
- ✅ CLAUDE.md compared against command catalog (verified: searched document)
- ✅ Orchestrator skill compared against command catalog (verified: searched document)

**What's untested:**

- ⚠️ Some commands may be documented elsewhere (e.g., docs/ folder)
- ⚠️ Some "undocumented" commands might be intentionally hidden
- ⚠️ Actual usage frequency of missing commands unknown

**What would change this:**

- Finding would be wrong if CLAUDE.md or skill have been updated since reading
- Finding would be incomplete if there are other documentation sources not checked

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Update Documentation in Two Phases

**Why this approach:**
- Phase 1 handles critical gaps immediately
- Phase 2 can be batched with lower priority
- Avoids document bloat by grouping related commands

**Implementation sequence:**

1. **Add Server Management Section to CLAUDE.md and Orchestrator Skill**
   - Document `orch servers init/up/down/gen-plist`
   - Include launchd integration explanation
   - Add to "Orch Commands (Quick Reference)" section

2. **Add KB Quick Query Documentation**
   - Document `orch kb ask` as alternative to spawning investigations
   - Show --save flag for preserving results
   - Add to Knowledge section

3. **Add Session Introspection Documentation**
   - Document `orch sessions list/search/show`
   - Include use cases (finding past decisions, debugging)

4. **Add Remaining Tier 1/2 Commands**
   - lint, synthesis, tokens, history, port, swarm, doctor

### Alternative Approaches Considered

**Option B: Single comprehensive update**
- **Pros:** One pass, everything done
- **Cons:** Large change, harder to review
- **When to use instead:** If time is not constrained

**Option C: Add only to --help (docs = help)**
- **Pros:** Single source of truth
- **Cons:** Agents can't run --help at session start
- **When to use instead:** Never - agents need context in documents

**Rationale for recommendation:** Phased approach allows immediate fixes for critical gaps while managing document size.

---

### Implementation Details

**What to implement first:**
- Add `orch servers` section (init, up, down, gen-plist, launchd explanation)
- Add `orch kb ask` section
- Add `orch sessions` section

**Things to watch out for:**
- Document bloat - CLAUDE.md is already 222 lines, skill is 1766 lines
- Token limits - `orch lint` can verify CLAUDE.md stays under limits
- Consistency - keep command documentation format consistent

**Areas needing further investigation:**
- Should some commands be documented only in project CLAUDE.md, not skill?
- Are there docs/ files that should be updated instead?
- Should we create an orch-commands-reference.md to reduce main doc size?

**Success criteria:**
- ✅ `orch servers init/up/down/gen-plist` documented
- ✅ `orch kb ask` documented
- ✅ `orch sessions search` documented
- ✅ `orch lint` can validate the updates pass limits

---

## Gap Analysis Table (Full Catalog)

| Feature | In --help? | In CLAUDE.md? | In Orchestrator Skill? | Priority |
|---------|------------|---------------|------------------------|----------|
| **Server Management** | | | | |
| `orch servers init` | ✅ | ❌ | ❌ | **P0** |
| `orch servers up/down` | ✅ | ❌ | ❌ | **P0** |
| `orch servers gen-plist` | ✅ | ❌ | ❌ | **P0** |
| `orch servers list` | ✅ | ✅ partial | ❌ | P1 |
| `orch servers start/stop` | ✅ | ✅ | ❌ | P2 (legacy) |
| `orch servers attach/open/status` | ✅ | ✅ | ❌ | P2 |
| **Knowledge** | | | | |
| `orch kb ask` | ✅ | ❌ | ❌ | **P0** |
| `orch kb extract` | ✅ | ❌ | ❌ | P1 |
| `orch learn` | ✅ | ✅ | ✅ | ✅ Documented |
| `orch patterns` | ✅ | ✅ | ✅ | ✅ Documented |
| **Sessions** | | | | |
| `orch sessions list` | ✅ | ❌ | ❌ | **P0** |
| `orch sessions search` | ✅ | ❌ | ❌ | **P0** |
| `orch sessions show` | ✅ | ❌ | ❌ | P1 |
| **Utilities** | | | | |
| `orch doctor --fix` | ✅ | ❌ | ❌ | **P0** |
| `orch lint` | ✅ | ❌ | ❌ | P1 |
| `orch lint --skills/--issues` | ✅ | ❌ | ❌ | P1 |
| `orch synthesis` | ✅ | ❌ | ❌ | P1 |
| `orch tokens` | ✅ | ❌ | ❌ | P1 |
| `orch history` | ✅ | ❌ | ❌ | P2 |
| `orch swarm` | ✅ | ❌ | ❌ | P1 |
| `orch port` | ✅ | ❌ | ❌ | P1 |
| `orch retries` | ✅ | ❌ | ❌ | P2 |
| `orch stale` | ✅ | ❌ | ❌ | P2 |
| `orch handoff` | ✅ | ❌ | ❌ | P1 |
| `orch fetch-md` | ✅ | ❌ | ❌ | P2 |
| `orch transcript format` | ✅ | ❌ | ❌ | P2 |
| `orch version --source` | ✅ | ❌ | ❌ | P3 |
| `orch init` | ✅ | ❌ (tangential) | ❌ | P1 |
| **Daemon** | | | | |
| `orch daemon run/once/preview` | ✅ | ✅ | ✅ | ✅ Documented |
| `orch daemon reflect` | ✅ | ❌ | ✅ partial | P1 |
| **Lifecycle** (all documented) | ✅ | ✅ | ✅ | ✅ Documented |
| **Monitoring** (all documented) | ✅ | ✅ | ✅ | ✅ Documented |
| **Strategic** (all documented) | ✅ | ✅ | ✅ | ✅ Documented |
| **Account** (all documented) | ✅ | ✅ | ✅ | ✅ Documented |

**Legend:**
- **P0**: Critical - orchestrators actively need this
- **P1**: Important - should know about but less urgent
- **P2**: Nice to have
- **P3**: Low priority / edge cases

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` - Project documentation
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator skill (1766 lines)

**Commands Run:**
```bash
# Get all commands
~/bin/orch --help

# Get subcommand help
~/bin/orch <cmd> --help  # For all 40+ commands

# Search documentation
grep "servers" CLAUDE.md
grep "kb ask" SKILL.md
```

**Related Artifacts:**
- **Spawn Context:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-critical-meta-gap-28dec/SPAWN_CONTEXT.md`

---

## Investigation History

**2025-12-28 ~14:00:** Investigation started
- Initial question: What orch commands exist but aren't documented?
- Context: Prior agent missed major features while investigating session-start context gaps

**2025-12-28 ~14:30:** Command catalog complete
- Ran `orch --help` and all subcommand help
- Identified 40+ commands across 9 categories

**2025-12-28 ~14:45:** Gap analysis complete
- Compared against CLAUDE.md and orchestrator skill
- Found 12+ significant undocumented commands

**2025-12-28 ~15:00:** Investigation completed
- Status: Complete
- Key outcome: Server management, kb ask, and sessions search are major documentation gaps

---

## Self-Review

- [x] Real test performed (not code review) - ran actual help commands
- [x] Conclusion from evidence (not speculation) - based on command output vs doc content
- [x] Question answered - clear list of undocumented commands
- [x] File complete - all sections filled
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

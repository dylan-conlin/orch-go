<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Daemon launchd plist created at ~/Library/LaunchAgents/com.orch.daemon.plist using build/orch binary for installation isolation.

**Evidence:** Plist loaded successfully (launchctl list shows PID 63004), daemon polling and finding issues (logs show "Found 53 open issues"), process using correct binary path confirmed via ps aux.

**Knowledge:** Using build/orch instead of ~/bin/orch prevents SIGKILL during make install; minimal environment (PATH, BEADS_NO_DAEMON, HOME) sufficient for operation; KeepAlive ensures auto-restart on crash.

**Next:** Daemon is running; no further action needed. Consider testing auto-restart behavior if needed.

**Promote to Decision:** recommend-no (infrastructure setup following existing decision, not new architectural choice)

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

# Investigation: Set Up Daemon Launchd Plist

**Question:** How should the orch daemon launchd plist be configured for persistent operation?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** orch-go agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Must use build/orch not ~/bin/orch

**Evidence:** KB constraint states "Use build/orch for serve daemon - Reason: Prevents SIGKILL during make install". Verified build/orch exists at `/Users/dylanconlin/Documents/personal/orch-go/build/orch` (21MB binary, modified Jan 15 07:31).

**Source:** SPAWN_CONTEXT.md line 20, `ls -la` command output

**Significance:** Using the build directory binary prevents daemon crashes when running `make install`, as the installed binary at ~/bin/orch would be replaced during installation, causing SIGKILL to the running process.

---

### Finding 2: Daemon guide specifies configuration pattern

**Evidence:** Daemon guide (lines 209-234) provides example plist configuration including: ProgramArguments array with flags, WorkingDirectory set to project root, EnvironmentVariables dict with BEADS_NO_DAEMON=1, StandardOutPath/StandardErrorPath for logging.

**Source:** .kb/guides/daemon.md:209-234

**Significance:** Provides authoritative pattern for daemon configuration including all required environment setup and logging configuration.

---

### Finding 3: Individual launchd services is the chosen architecture

**Evidence:** Decision document 2026-01-10-individual-launchd-services.md states "Each service runs directly under launchd supervision" with KeepAlive: true for auto-restart. All existing services (opencode, orch serve, orch web, orch doctor) follow this pattern.

**Source:** .kb/decisions/2026-01-10-individual-launchd-services.md

**Significance:** Daemon plist should follow same pattern as other orch services for consistency: KeepAlive, RunAtLoad, direct execution (no overmind/tmux wrapper).

---

## Synthesis

**Key Insights:**

1. **Build binary isolation prevents installation disruption** - Using build/orch instead of ~/bin/orch means daemon continues running when `make install` replaces the installed binary, avoiding SIGKILL during development cycles.

2. **Individual launchd services provide simple supervision** - Following the established pattern (per 2026-01-10 decision), daemon runs directly under launchd with KeepAlive for automatic restart, no tmux/overmind wrapper needed.

3. **Minimal environment configuration sufficient** - Daemon only needs PATH (for subcommands), BEADS_NO_DAEMON=1 (to prevent CLI spawning daemon), and HOME (for config access).

**Answer to Investigation Question:**

The orch daemon launchd plist should use `/Users/dylanconlin/Documents/personal/orch-go/build/orch daemon run --verbose` as ProgramArguments, working directory set to project root, environment variables for PATH/BEADS_NO_DAEMON/HOME, and KeepAlive/RunAtLoad both true for persistent operation with auto-restart. Logging configured to ~/.orch/daemon.log for stdout and stderr.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plist XML syntax valid (verified: plutil -lint returned OK)
- ✅ Daemon loads and runs via launchd (verified: launchctl list shows PID 63004, status 0)
- ✅ Daemon uses build/orch binary (verified: ps aux shows /Users/dylanconlin/Documents/personal/orch-go/build/orch in command)
- ✅ Logging works (verified: ~/.orch/daemon.log contains poll output)
- ✅ Daemon polls and finds issues (verified: logs show "Found 53 open issues", "Selected orch-go-pi2k2")

**What's untested:**

- ⚠️ Auto-restart behavior on daemon crash (not tested by killing process)
- ⚠️ Behavior during make install (haven't run installation while daemon running)
- ⚠️ RunAtLoad on system reboot (not tested by rebooting)

**What would change this:**

- Finding would be wrong if `kill -9 63004` doesn't result in launchd restarting daemon
- Binary path choice would be wrong if make install with daemon running still causes issues
- Configuration would be incomplete if daemon fails to spawn agents due to missing environment variables

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

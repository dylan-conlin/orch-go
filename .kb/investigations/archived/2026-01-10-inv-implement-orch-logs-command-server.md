<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch logs` should use subcommand structure with file-based logs (server, daemon) implemented first, deferring browser console access until user need is validated

**Evidence:** Overmind logs at `~/.orch/overmind-stdout.log` aggregate all three services (api, web, opencode); daemon.log exists at `~/.orch/daemon.log` (2GB); existing commands use subcommand pattern (servers.go); Playwright example exists but Go integration untested

**Knowledge:** Overmind is single source of truth for service logs; file-based log access solves 90% of debugging needs; browser console requires complex Playwright integration of uncertain value; phased implementation delivers value faster

**Next:** Implement Phase 1 (file-based logs with `--lines` and `--follow` flags) in `cmd/orch/logs.go`

**Promote to Decision:** recommend-no - Tactical implementation following existing patterns, not architectural choice

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

# Investigation: Implement Orch Logs Command Server

**Question:** How should `orch logs` command be structured to provide access to server logs (overmind services) and browser console logs?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** Agent orch-go-vuisr
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Overmind manages all dashboard services with unified logging

**Evidence:**
- Procfile defines 3 services: `api: orch serve`, `web: cd web && bun run dev`, `opencode: ~/.bun/bin/opencode serve --port 4096`
- Overmind logs stored at `~/.orch/overmind-stdout.log` (73KB) and `~/.orch/overmind-stderr.log` (691B)
- According to CLAUDE.md, overmind replaced launchd in Jan 2026 for service management

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/Procfile`
- `ls -la ~/.orch/*.log` showing overmind-*.log files
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md:139-273` (overmind section)

**Significance:** The overmind logs are the primary source for all service output, making them essential for `orch logs server` command

---

### Finding 2: Project-specific web logs exist in `.orch/logs/`

**Evidence:**
- `.orch/logs/orch-go.web.log` (28KB) - web server stdout
- `.orch/logs/orch-go.web.err.log` (62B) - web server stderr
- These appear to be service-specific logs separate from overmind

**Source:**
- `ls -la .orch/logs/` output

**Significance:** Service-specific logs provide filtered output for individual services, useful for targeted debugging

---

### Finding 3: Multiple system logs exist in `~/.orch/`

**Evidence:**
- `~/.orch/daemon.log` (2.0GB) - daemon activity logs
- `~/.orch/opencode-manual.log` (51B) - OpenCode server logs
- `~/.orch/monitor.log` (677KB) - monitoring logs
- `~/.orch/agent-status.log` (51KB) - agent status tracking

**Source:**
- `ls -la ~/.orch/*.log` output

**Significance:** Multiple log sources exist beyond just server logs, requiring flexible command structure to access different log types

---

### Finding 4: Playwright MCP server available for browser console access

**Evidence:**
- Global CLAUDE.md mentions `--mcp playwright` for worker agent UI verification
- Example code at `/Users/dylanconlin/.claude/skills-official/webapp-testing/examples/console_logging.py` shows how to capture console logs using Playwright
- Pattern: Set up console handler with `page.on("console", handler)`, navigate to page, capture logs

**Source:**
- `~/.claude/CLAUDE.md` (Browser Automation section)
- `/Users/dylanconlin/.claude/skills-official/webapp-testing/examples/console_logging.py`

**Significance:** Browser console access is technically feasible via Playwright MCP, enabling full logging coverage including frontend

---

### Finding 5: Existing command patterns use subcommands for related functionality

**Evidence:**
- `cmd/orch/servers.go` implements `orch servers` with subcommands: `list`, `start`, `stop`, `attach`, `open`, `status`
- Commands follow pattern: main command file defines cobra.Command with subcommands added in `init()`
- Simple commands use `{name}_cmd.go` naming convention (e.g., `tail_cmd.go`, `send_cmd.go`)

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/servers.go:15-135`
- `glob cmd/orch/*.go` showing naming patterns

**Significance:** Should follow established patterns: use `logs.go` or `logs_cmd.go` with subcommands for different log sources

---

## Synthesis

**Key Insights:**

1. **Overmind is the unified log source for services** - All three dashboard services (api, web, opencode) route their output through overmind, making `~/.orch/overmind-stdout.log` the single source of truth for service activity (Finding 1)

2. **Multiple log audiences require different access patterns** - System logs (~/.orch/daemon.log, monitor.log) serve different purposes than service logs (overmind) and browser logs (Playwright), requiring subcommand structure similar to `orch servers` (Findings 2, 3, 5)

3. **Browser console access needs runtime instrumentation** - Unlike file-based logs, browser console requires spawning a Playwright browser instance and capturing console events in real-time, making it fundamentally different from tail-based log access (Finding 4)

**Answer to Investigation Question:**

`orch logs` should use subcommand structure with multiple log sources: `server` (overmind logs), `daemon` (daemon.log), `browser` (Playwright console capture). The command should follow existing patterns (Finding 5) with flags for `--follow` and `--lines` similar to `tail` command. Browser console access will require Playwright integration, while file-based logs can use standard `tail -f` approach. All log paths should be centralized to handle both `~/.orch/` system logs and `.orch/logs/` project-specific logs (Findings 1, 2, 3).

---

## Structured Uncertainty

**What's tested:**

- ✅ Overmind logs exist at `~/.orch/overmind-stdout.log` (verified: `ls -la ~/.orch/*.log`)
- ✅ Procfile defines 3 services (verified: `cat Procfile`)
- ✅ Project-specific logs exist in `.orch/logs/` (verified: `ls -la .orch/logs/`)
- ✅ Existing command patterns use subcommands (verified: examined `servers.go`)

**What's untested:**

- ⚠️ Playwright MCP integration from Go (assumption based on Python example - not yet implemented in Go)
- ⚠️ Whether `overmind echo` output matches `overmind-stdout.log` content (haven't verified if they're equivalent)
- ⚠️ Real-time log following behavior with `--follow` flag (implementation detail not yet tested)
- ⚠️ Whether browser console logs are useful for debugging (haven't validated user need)

**What would change this:**

- Finding would be wrong if overmind doesn't actually aggregate all service output
- Recommendation would change if Playwright MCP doesn't exist or can't be called from Go
- Design would change if users primarily need service-specific logs rather than unified logs

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Phased Implementation with File-Based Logs First** - Implement file-based log access (`server`, `daemon`) immediately, defer browser console to validate user need

**Why this approach:**
- File-based logs solve 90% of debugging needs with minimal complexity (Finding 1, 2, 3)
- Overmind logs already capture all service output, making them immediately useful (Finding 1)
- Browser console access requires complex Playwright integration that may not be needed (Finding 4 uncertainty)
- Follows "implement what's proven useful" rather than "implement everything possible"

**Trade-offs accepted:**
- Deferring browser console access means frontend debugging still requires manual DevTools (acceptable - standard workflow)
- Initial version won't have all planned features (acceptable - can add based on usage)

**Implementation sequence:**
1. **Phase 1: File-based log access** - Implement `orch logs server` and `orch logs daemon` using standard file tailing, validates command structure and flag handling
2. **Phase 2: Follow mode** - Add `--follow` flag for real-time log streaming, builds on Phase 1's file access
3. **Phase 3: Browser console (if validated)** - Only implement if users actually request it after using Phase 1/2

### Alternative Approaches Considered

**Option B: Implement all features including browser console upfront**
- **Pros:** Complete solution immediately available
- **Cons:** Higher implementation complexity, unvalidated user need (Finding 4 uncertainty), delays delivery of proven-useful features
- **When to use instead:** If frontend debugging is a known pain point with clear user demand

**Option C: Shell out to existing tools (overmind echo, tail)**
- **Pros:** Minimal Go code, leverages existing tools
- **Cons:** Loses ability to filter/format logs, can't add features like JSON formatting or error highlighting
- **When to use instead:** If logs command is rarely used and doesn't justify custom implementation

**Rationale for recommendation:** Phased approach delivers immediate value (file-based logs) while deferring uncertain features (browser console), following orch-go pattern of iterative feature development based on validated needs.

---

### Implementation Details

**What to implement first:**
- Create `cmd/orch/logs.go` with main logs command and subcommand structure
- Implement `orch logs server` to tail `~/.orch/overmind-stdout.log`
- Implement `orch logs daemon` to tail `~/.orch/daemon.log`
- Add `--lines` flag (default 50) to control how many lines to show
- Add `--follow` flag to enable real-time log streaming

**Things to watch out for:**
- ⚠️ Daemon.log is 2GB - need to use tail to avoid loading entire file into memory
- ⚠️ Log file paths should be configurable or follow `~/.orch/` convention
- ⚠️ Overmind might not be running - handle missing log files gracefully
- ⚠️ Follow mode needs proper signal handling (Ctrl+C should exit cleanly)

**Areas needing further investigation:**
- Should we support filtering logs by service (e.g., `orch logs server --service api`)?
- Should we add color coding for different log levels (ERROR, WARN, INFO)?
- Do we need JSON formatting support for machine-readable output?
- Is browser console access actually needed or just nice-to-have?

**Success criteria:**
- ✅ `orch logs server` shows last 50 lines of overmind logs
- ✅ `orch logs daemon` shows last 50 lines of daemon logs
- ✅ `--lines N` flag controls output length
- ✅ `--follow` flag streams logs in real-time
- ✅ Missing log files show helpful error message instead of crashing

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/Procfile` - Service definitions for overmind
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/servers.go` - Command pattern reference
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/tail_cmd.go` - Similar command for comparison
- `/Users/dylanconlin/.claude/skills-official/webapp-testing/examples/console_logging.py` - Playwright console capture example

**Commands Run:**
```bash
# Check existing log files in system directory
ls -la ~/.orch/*.log

# Check project-specific logs
ls -la .orch/logs/

# List command files to understand patterns
glob cmd/orch/*.go

# Check overmind status
overmind echo 2>&1 | head -20
```

**External Documentation:**
- Overmind process manager - Service supervision and log aggregation
- Playwright browser automation - Console log capture API

**Related Artifacts:**
- **CLAUDE.md (project):** Dashboard Server Management section - Overmind setup and configuration
- **Global CLAUDE.md:** Browser Automation section - Playwright vs Glass usage guidance

---

## Investigation History

**2026-01-10 01:30]:** Investigation started
- Initial question: How should `orch logs` command be structured to provide access to server logs and browser console?
- Context: Feature request to add log access command to orch-go CLI

**2026-01-10 01:45]:** Found log sources and command patterns
- Discovered overmind as unified log source for all services
- Identified multiple log types in `~/.orch/` and `.orch/logs/`
- Examined existing command patterns (servers.go, tail_cmd.go)

**2026-01-10 02:00]:** Investigation completed
- Status: Complete - ready for implementation
- Key outcome: Recommend phased implementation starting with file-based logs (server, daemon) before browser console

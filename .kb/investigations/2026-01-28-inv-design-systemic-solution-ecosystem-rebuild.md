<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Three-layer architecture recommended: (1) Version commands for all CLIs (enables staleness detection), (2) On-demand auto-rebuild (already exists in glass/agentlog, should be standardized), (3) `orch doctor` expansion for verification. Post-commit hooks are useful but secondary.

**Evidence:** Glass and agentlog already have sophisticated on-demand auto-rebuild comparing embedded git hash to HEAD; they just lack `version` command to expose this to external tools. 4 CLIs have version commands, all 6 have ldflags configured.

**Knowledge:** The "gaps" are smaller than the audit suggested - the auto-rebuild mechanism exists, just needs version commands added for observability. Post-commit hooks are redundant when on-demand rebuild exists.

**Next:** Add version commands to glass and agentlog (trivial - ~30 lines each), then extend `orch doctor` to verify all ecosystem binaries.

**Promote to Decision:** recommend-yes (architectural choice establishing patterns for ecosystem CLI consistency)

---

# Investigation: Design Systemic Solution for Ecosystem Rebuild-on-Change

**Question:** How should we unify the rebuild-on-change mechanisms across the orch ecosystem to ensure consistent staleness detection and auto-rebuild?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Claude (agent og-arch-design-systemic-solution-28jan-b84a)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Builds-On:** .kb/investigations/2026-01-28-inv-audit-rebuild-change-mechanisms-across.md

---

## Problem Framing

### Design Question

What unified architecture enables reliable staleness detection and auto-rebuild across all ecosystem CLIs, minimizing developer friction and preventing "fixed bug still reproduces" failures?

### Success Criteria

1. **100% staleness detection** - Every CLI binary can report whether it matches source
2. **Automatic recovery** - Stale binaries rebuild themselves or alert developers
3. **External verification** - `orch doctor` can verify all ecosystem binaries
4. **Minimal friction** - Developers don't need to remember `make install`
5. **Graceful degradation** - Works even when some components are unavailable

### Constraints

| Constraint | Source | Implication |
|------------|--------|-------------|
| All Go CLIs must embed git commit hash | `kn-8d192d` (ECOSYSTEM.md) | Foundation already established |
| Local-First | `~/.kb/principles.md` | No external services, git-based only |
| Compose Over Monolith | `~/.kb/principles.md` | Each CLI handles its own rebuild, shared pattern not shared tool |
| Graceful Degradation | `~/.kb/principles.md` | Must work if a component is unavailable |
| Share Patterns Not Tools | `~/.kb/principles.md` | Share the schema/format, not implementation |

### Scope

**In Scope:**
- Dylan's Go CLIs: orch, kb, glass, skillc, agentlog (5 binaries)
- Dylan's forked projects: beads (`bd`), opencode
- Version command standardization
- Auto-rebuild mechanism architecture
- `orch doctor` expansion

**Out of Scope:**
- Non-binary repos (orch-knowledge, beads-ui-svelte)
- Python legacy (`orch-cli`) - being deprecated
- Skills (handled by skillc, not binary staleness)

**Note:** beads and opencode are Dylan's forks, not upstream. They should be included in ecosystem rebuild architecture. See orch-go-20989 for follow-up.

---

## Findings

### Finding 1: On-Demand Auto-Rebuild Already Exists in 2 CLIs

**Evidence:** Both glass and agentlog have identical `autorebuild.go` implementing:
- Compare embedded `GitHash` to current `git rev-parse HEAD` in `SourceDir`
- If different, run `make install` and `syscall.Exec` to replace process
- Lock file mechanism to prevent concurrent rebuilds
- Environment variable to disable (`GLASS_NO_AUTOREBUILD=1`, `AGENTLOG_NO_AUTOREBUILD=1`)

**Source:**
- `/Users/dylanconlin/Documents/personal/glass/autorebuild.go`
- `/Users/dylanconlin/Documents/personal/agentlog/cmd/agentlog/autorebuild.go`

**Significance:** The "60% of repos lack auto-rebuild" finding from the audit was misleading. Glass and agentlog have the MOST sophisticated rebuild mechanism - on-demand rebuild that works for ANY invocation, not just post-commit. The gap is observability (no version command), not rebuild capability.

---

### Finding 2: Version Variables Are Already Defined in All CLIs

**Evidence:** All 5 of Dylan's Go CLIs have ldflags configured in Makefile and version variables in main.go:
- orch: `main.version`, `main.buildTime`, `main.sourceDir`, `main.gitHash`
- kb: similar pattern
- glass: `main.Version`, `main.BuildTime`, `main.SourceDir`, `main.GitHash`
- skillc: `main.Version`, `main.Commit`, `main.BuildTime`, `main.SourceDir`, `main.GitHash`
- agentlog: `main.Version`, `main.BuildTime`, `main.SourceDir`, `main.GitHash`

**Source:** Makefiles for all repos (examined glass, skillc, agentlog)

**Significance:** Adding version commands to glass and agentlog is trivial (~30 lines of code each) - the infrastructure is already in place.

---

### Finding 3: Post-Commit Hooks Are Redundant With On-Demand Rebuild

**Evidence:** Glass and agentlog don't have post-commit hooks, but they don't need them because:
1. On-demand rebuild fires on EVERY invocation when stale
2. This is MORE comprehensive than post-commit (covers manual git operations, branch switches, etc.)
3. Post-commit only triggers when you commit in THAT repo

**Source:** Comparison of hook-based vs on-demand approaches

**Significance:** The recommendation to "add post-commit hooks to glass and skillc" from the audit is suboptimal. On-demand rebuild is superior. The real recommendation should be to adopt the on-demand pattern from glass/agentlog in skillc.

---

### Finding 4: orch doctor Already Has Staleness Detection Logic

**Evidence:** `orch doctor --stale-only` compares binary git hash to repo HEAD for orch itself.

**Source:** ECOSYSTEM.md documentation, existing `orch doctor` behavior

**Significance:** Extending `orch doctor` to verify all ecosystem binaries is a natural evolution of existing capability. The infrastructure exists; just needs expansion.

---

## Decision Forks

### Fork 1: Rebuild Trigger Mechanism

**Question:** When should stale binaries be rebuilt?

**Options:**
- A: **Post-commit hooks** (rebuild after every commit in source repo)
- B: **On-demand** (rebuild at execution time when stale detected)
- C: **Centralized watcher** (launchd service watching all source dirs)
- D: **Hybrid** (post-commit + on-demand as fallback)

**Substrate says:**
- Principle: **Graceful Degradation** - on-demand works even if hooks aren't installed
- Principle: **Compose Over Monolith** - each CLI handles its own, not centralized
- Evidence: Glass/agentlog on-demand works well, no complaints
- Prior Decision: kn decision says hooks are a first-line defense but on-demand is backup

**RECOMMENDATION:** Option B (On-demand)

On-demand is superior because:
1. Works for ANY staleness source (branch switch, pull, reset, manual edit)
2. Doesn't require hook setup in each repo
3. Already proven in glass/agentlog
4. Self-healing - stale binary fixes itself on first use
5. Falls back gracefully if rebuild fails

**Trade-off accepted:** Slight startup latency when rebuilding (~2-3 seconds)

**When this would change:** If on-demand rebuild proves too slow or disruptive for interactive use

---

### Fork 2: Version Command Interface

**Question:** What should the version command output format be?

**Options:**
- A: **Human-readable** (e.g., "orch version e177d0ea, built 2026-01-28")
- B: **Machine-readable JSON** (e.g., `{"version": "e177d0ea", "build_time": "...", "source_dir": "..."}`)
- C: **Both with flag** (human default, `--json` for machine)
- D: **Parseable text** (key: value format)

**Substrate says:**
- Principle: **Surfacing Over Browsing** - `orch doctor` needs to consume this programmatically
- Pattern: Existing CLIs use human-readable by default
- Evidence: `orch version` outputs "orch version e177d0ea\nbuild time: 2026-01-28T19:24:41Z"

**RECOMMENDATION:** Option C (Both with flag)

Human-readable is default for developer experience, `--json` flag enables programmatic consumption by `orch doctor` and other tools.

**Trade-off accepted:** Slightly more code than human-only

**When this would change:** If JSON output proves unnecessary (all consumers can parse text)

---

### Fork 3: Staleness Detection Scope for orch doctor

**Question:** Which binaries should `orch doctor` verify?

**Options:**
- A: **Dylan's CLIs only** (orch, kb, glass, skillc, agentlog)
- B: **All ecosystem binaries** (include bd, kn)
- C: **Configurable** (whitelist in ECOSYSTEM.md or config)

**Substrate says:**
- Principle: **Observation Infrastructure** - "If the system can't observe it, the system can't manage it"
- Constraint: bd is upstream OSS, Dylan uses releases not source builds
- Evidence: kn binary location was "unknown" in audit (not installed or different path)

**RECOMMENDATION:** Option A (Dylan's CLIs only) → **UPDATED to include beads and opencode**

Focus on binaries Dylan builds from source. This includes beads (`bd`) and opencode since Dylan maintains forks of both.

**Trade-off accepted:** kn excluded (rarely used)

**When this would change:** If kn becomes frequently used

---

### Fork 4: Standardization Approach

**Question:** How do we standardize the rebuild pattern across repos?

**Options:**
- A: **Copy-paste pattern** (each repo has its own autorebuild.go)
- B: **Shared library** (import from common package)
- C: **Template/generator** (generate autorebuild.go from template)
- D: **Documentation only** (describe pattern, let devs implement)

**Substrate says:**
- Principle: **Share Patterns Not Tools** - "share the schema/format, not the implementation"
- Evidence: Glass and agentlog already have copy-pasted autorebuild.go (identical except env var name)
- Constraint: These are separate repos, shared library adds dependency complexity

**RECOMMENDATION:** Option A (Copy-paste pattern)

The autorebuild.go is ~150 lines, stable, and rarely needs changes. Copy-paste is pragmatic:
1. No cross-repo dependency management
2. Each repo can customize env var name
3. Pattern is simple enough that duplication is acceptable
4. Already working in 2 repos

**Trade-off accepted:** Code duplication across repos

**When this would change:** If autorebuild.go needs frequent updates (then consider template)

---

### Fork 5: Version Command Implementation

**Question:** How do we add version commands to glass and agentlog?

**Options:**
- A: **Standalone subcommand** (like orch's `orch version`)
- B: **Flag on all commands** (e.g., `glass --version`)
- C: **Both** (subcommand and flag)

**Substrate says:**
- Pattern: orch uses subcommand (`orch version`), kb uses subcommand (`kb version`)
- Consistency: Other CLIs in ecosystem use subcommand pattern

**RECOMMENDATION:** Option A (Standalone subcommand)

Consistent with ecosystem pattern (orch, kb, skillc all use subcommand).

**Trade-off accepted:** Can't get version with `glass --version` (would need Option C)

**When this would change:** If users frequently try `--version` flag

---

## Synthesis

### Three-Layer Architecture

The systemic solution is a **three-layer architecture** with clear responsibilities:

```
┌─────────────────────────────────────────────────────────────────┐
│                    LAYER 3: VERIFICATION                        │
│                                                                 │
│   orch doctor --stale-only                                     │
│   - Verifies all ecosystem binaries match source               │
│   - Consumes version commands (JSON output)                    │
│   - Reports which binaries need attention                      │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                    LAYER 2: OBSERVABILITY                       │
│                                                                 │
│   CLI version commands                                         │
│   - Every CLI exposes: version, build_time, source_dir, git_hash│
│   - Human-readable default, --json for tools                   │
│   - Enables external staleness detection                       │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                    LAYER 1: SELF-HEALING                        │
│                                                                 │
│   On-demand auto-rebuild (maybeAutoRebuild())                  │
│   - Compares embedded git_hash to current HEAD                 │
│   - Rebuilds and re-execs if stale                            │
│   - Lock file prevents concurrent rebuilds                     │
│   - Environment variable to disable                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Key Insights

1. **On-demand > Post-commit** - On-demand auto-rebuild (glass/agentlog pattern) is superior to post-commit hooks because it catches ALL staleness sources and self-heals without setup

2. **Gaps are smaller than reported** - The audit found "60% lack auto-rebuild" but glass/agentlog have the BEST rebuild mechanism. The actual gap is just missing version commands for observability.

3. **Three layers complement each other** - Self-healing handles the common case (stale binary fixes itself). Observability enables external verification. Verification provides audit capability.

4. **Pattern over tool** - Following Share Patterns Not Tools, copy-paste the autorebuild.go pattern rather than creating shared library. 150 lines of stable code is acceptable duplication.

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add version commands to glass and agentlog, then extend orch doctor**

This addresses the ACTUAL gaps (observability) while recognizing the auto-rebuild layer is already solid.

**Why this approach:**
- Glass and agentlog already have auto-rebuild - they just can't report version
- Version commands are ~30 lines each, trivial to add
- orch doctor extension provides unified verification
- Post-commit hooks become unnecessary with on-demand rebuild

**Trade-offs accepted:**
- skillc doesn't have auto-rebuild (but has version command, lower priority)
- Copy-paste pattern means potential drift (acceptable for stable code)

**Implementation sequence:**
1. **Add version command to glass** (~30 lines) - enables staleness detection
2. **Add version command to agentlog** (~30 lines) - same pattern
3. **Extend orch doctor** - add ecosystem binary verification
4. **(Optional) Add auto-rebuild to skillc** - if rebuild friction continues

### Alternative Approaches Considered

**Option B: Post-commit hooks everywhere**
- **Pros:** Proactive rebuild, no execution latency
- **Cons:** Hooks aren't installed by default, miss non-commit staleness, redundant with on-demand
- **When to use instead:** If on-demand rebuild latency becomes a problem

**Option C: Centralized watcher (launchd)**
- **Pros:** Single place to manage all rebuilds
- **Cons:** Violates Compose Over Monolith, adds complexity, single point of failure
- **When to use instead:** Never - this violates core principles

**Rationale for recommendation:** The on-demand pattern is proven, already deployed, and superior to alternatives. The actual work needed is small: add version commands (trivial) and extend orch doctor (moderate).

---

### Implementation Details

**What to implement first:**
1. Add `version` subcommand to glass main.go
2. Add `version` subcommand to agentlog (via cobra command)
3. Update ECOSYSTEM.md to reflect current state

**Things to watch out for:**
- ⚠️ Glass uses custom flag parsing, version command needs to fit that pattern
- ⚠️ Agentlog uses cobra, version command should use cobra pattern
- ⚠️ On-demand rebuild may cause unexpected delays in scripts - document env var to disable

**Success criteria:**
- ✅ `glass version` and `glass version --json` work
- ✅ `agentlog version` and `agentlog version --json` work
- ✅ `orch doctor --stale-only` reports status of all 5 Dylan CLIs
- ✅ ECOSYSTEM.md reflects actual state (auto-rebuild coverage is 80%, not 40%)

---

## References

**Files Examined:**
- `~/.orch/ECOSYSTEM.md` - Ecosystem documentation with staleness patterns
- `~/.kb/principles.md` - Design principles guiding the architecture
- `/Users/dylanconlin/Documents/personal/glass/main.go` - Glass CLI implementation
- `/Users/dylanconlin/Documents/personal/glass/autorebuild.go` - On-demand rebuild implementation
- `/Users/dylanconlin/Documents/personal/agentlog/cmd/agentlog/main.go` - Agentlog CLI
- `/Users/dylanconlin/Documents/personal/agentlog/cmd/agentlog/autorebuild.go` - Identical pattern
- `/Users/dylanconlin/Documents/personal/skillc/Makefile` - Ldflags configuration
- `/Users/dylanconlin/Documents/personal/orch-go/Makefile` - Reference implementation
- `/Users/dylanconlin/Documents/personal/orch-go/.git/hooks/post-commit` - Post-commit hook example

**Commands Run:**
```bash
# Check version command availability
glass version  # "unknown command: version"
agentlog version  # "unknown command"
orch version  # Works - "orch version e177d0ea"

# Examine autorebuild implementations
cat ~/Documents/personal/glass/autorebuild.go
cat ~/Documents/personal/agentlog/cmd/agentlog/autorebuild.go
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-28-inv-audit-rebuild-change-mechanisms-across.md - Audit this builds on
- **Decision:** This investigation should be promoted to decision (establishes ecosystem pattern)

---

## Investigation History

**2026-01-28 20:30:** Investigation started
- Initial question: Design systemic solution for rebuild-on-change
- Context: Audit found gaps, need unified architecture

**2026-01-28 21:00:** Key finding - on-demand rebuild exists
- Discovered glass/agentlog have sophisticated auto-rebuild
- Realized gaps are smaller than reported
- Changed recommendation from "add hooks" to "add version commands"

**2026-01-28 21:30:** Investigation completed
- Status: Complete
- Key outcome: Three-layer architecture (self-healing, observability, verification) with on-demand rebuild as foundation
